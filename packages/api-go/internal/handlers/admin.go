package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/lib/pq"

	"github.com/Mavzz/readpanda/api-go/internal/config"
	"github.com/Mavzz/readpanda/api-go/internal/database"
	"github.com/Mavzz/readpanda/api-go/internal/utils"
)

// AdminHandler is a generic browser over the public schema for the portal:
// list tables, page through rows, insert and delete. Every endpoint requires
// the admin role.
//
// Nothing from the request is ever spliced into SQL unless it first matched a
// real table or column name read back from the catalog, and even then it goes
// through pq.QuoteIdentifier. Values are always bind parameters.
type AdminHandler struct {
	Config *config.Config
}

func NewAdminHandler(cfg *config.Config) *AdminHandler {
	return &AdminHandler{Config: cfg}
}

// Columns whose values are never sent back to the portal.
var adminRedactedColumns = map[string]bool{"password": true}

const adminRedacted = "••••••"

type adminColumn struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Nullable   bool     `json:"nullable"`
	HasDefault bool     `json:"has_default"`
	IsPrimary  bool     `json:"is_primary"`
	EnumValues []string `json:"enum_values,omitempty"`
}

type adminTableSchema struct {
	Name       string        `json:"name"`
	Columns    []adminColumn `json:"columns"`
	PrimaryKey []string      `json:"primary_key"`
}

func adminJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func adminError(w http.ResponseWriter, status int, msg string) {
	adminJSON(w, status, map[string]string{"error": msg})
}

// adminDBError surfaces Postgres' own message (constraint names, bad input
// syntax, ...) since the only caller is an admin fixing data by hand.
func adminDBError(w http.ResponseWriter, err error) {
	if pqErr, ok := err.(*pq.Error); ok {
		msg := pqErr.Message
		if pqErr.Detail != "" {
			msg += " — " + pqErr.Detail
		}
		adminError(w, http.StatusBadRequest, msg)
		return
	}
	adminError(w, http.StatusInternalServerError, err.Error())
}

func adminTableNames() ([]string, error) {
	rows, err := database.DB.Query(
		`SELECT table_name FROM information_schema.tables
		  WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		  ORDER BY table_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	return names, rows.Err()
}

// loadAdminSchema returns the table's columns, or (nil, nil) if no such table
// exists in the public schema.
func loadAdminSchema(table string) (*adminTableSchema, error) {
	names, err := adminTableNames()
	if err != nil {
		return nil, err
	}
	found := false
	for _, n := range names {
		if n == table {
			found = true
			break
		}
	}
	if !found {
		return nil, nil
	}

	qualified := "public." + pq.QuoteIdentifier(table)

	pk := map[string]bool{}
	var pkOrder []string
	pkRows, err := database.DB.Query(
		`SELECT a.attname
		   FROM pg_index i
		   JOIN LATERAL unnest(i.indkey) WITH ORDINALITY AS k(attnum, ord) ON true
		   JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = k.attnum
		  WHERE i.indrelid = $1::regclass AND i.indisprimary
		  ORDER BY k.ord`, qualified)
	if err != nil {
		return nil, err
	}
	for pkRows.Next() {
		var n string
		if err := pkRows.Scan(&n); err != nil {
			pkRows.Close()
			return nil, err
		}
		pk[n] = true
		pkOrder = append(pkOrder, n)
	}
	pkRows.Close()

	rows, err := database.DB.Query(
		`SELECT c.column_name,
		        CASE WHEN c.data_type IN ('USER-DEFINED', 'ARRAY') THEN c.udt_name ELSE c.data_type END,
		        c.udt_name,
		        c.data_type = 'USER-DEFINED',
		        c.is_nullable = 'YES',
		        c.column_default IS NOT NULL OR c.is_identity = 'YES'
		   FROM information_schema.columns c
		  WHERE c.table_schema = 'public' AND c.table_name = $1
		  ORDER BY c.ordinal_position`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	schema := &adminTableSchema{Name: table, PrimaryKey: pkOrder}
	if schema.PrimaryKey == nil {
		schema.PrimaryKey = []string{}
	}
	var enumTypes []string
	for rows.Next() {
		var col adminColumn
		var udt string
		var userDefined bool
		if err := rows.Scan(&col.Name, &col.Type, &udt, &userDefined, &col.Nullable, &col.HasDefault); err != nil {
			return nil, err
		}
		col.IsPrimary = pk[col.Name]
		if userDefined {
			enumTypes = append(enumTypes, udt)
		}
		schema.Columns = append(schema.Columns, col)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Enum labels, so the portal can render a dropdown instead of a text box.
	if len(enumTypes) > 0 {
		enumRows, err := database.DB.Query(
			`SELECT t.typname, e.enumlabel
			   FROM pg_enum e JOIN pg_type t ON t.oid = e.enumtypid
			  WHERE t.typname = ANY($1)
			  ORDER BY t.typname, e.enumsortorder`, pq.Array(enumTypes))
		if err != nil {
			return nil, err
		}
		defer enumRows.Close()
		labels := map[string][]string{}
		for enumRows.Next() {
			var typ, label string
			if err := enumRows.Scan(&typ, &label); err != nil {
				return nil, err
			}
			labels[typ] = append(labels[typ], label)
		}
		for i := range schema.Columns {
			if l, ok := labels[schema.Columns[i].Type]; ok {
				schema.Columns[i].EnumValues = l
			}
		}
	}

	return schema, nil
}

func (s *adminTableSchema) hasColumn(name string) bool {
	for _, c := range s.Columns {
		if c.Name == name {
			return true
		}
	}
	return false
}

// selectList casts every column to text so rows scan generically and render
// losslessly in the portal (uuids, numerics, arrays, jsonb, timestamps...).
func (s *adminTableSchema) selectList(alias string) string {
	parts := make([]string, len(s.Columns))
	for i, c := range s.Columns {
		parts[i] = alias + "." + pq.QuoteIdentifier(c.Name) + "::text"
	}
	return strings.Join(parts, ", ")
}

func (s *adminTableSchema) scanRows(rows *sql.Rows) ([]map[string]interface{}, error) {
	out := []map[string]interface{}{}
	for rows.Next() {
		vals := make([]sql.NullString, len(s.Columns))
		ptrs := make([]interface{}, len(s.Columns))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]interface{}, len(s.Columns))
		for i, c := range s.Columns {
			switch {
			case !vals[i].Valid:
				row[c.Name] = nil
			case adminRedactedColumns[c.Name]:
				row[c.Name] = adminRedacted
			default:
				row[c.Name] = vals[i].String
			}
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// adminParam turns a JSON value into a bind parameter. Everything goes to
// Postgres as text and is cast by the target column's type.
func adminParam(v interface{}) interface{} {
	switch t := v.(type) {
	case nil:
		return nil
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default: // objects / arrays → jsonb
		b, _ := json.Marshal(t)
		return string(b)
	}
}

// ListTables — GET /admin/tables
func (h *AdminHandler) ListTables(w http.ResponseWriter, r *http.Request) {
	if _, ok := utils.RequireAdmin(w, r, h.Config.JWTSecret); !ok {
		return
	}
	names, err := adminTableNames()
	if err != nil {
		adminDBError(w, err)
		return
	}
	type tableInfo struct {
		Name     string `json:"name"`
		RowCount int64  `json:"row_count"`
	}
	tables := make([]tableInfo, 0, len(names))
	for _, n := range names {
		var count int64
		if err := database.DB.QueryRow("SELECT count(*) FROM public." + pq.QuoteIdentifier(n)).Scan(&count); err != nil {
			adminDBError(w, err)
			return
		}
		tables = append(tables, tableInfo{Name: n, RowCount: count})
	}
	adminJSON(w, http.StatusOK, map[string]interface{}{"tables": tables})
}

// GetTableSchema — GET /admin/tables/{table}
func (h *AdminHandler) GetTableSchema(w http.ResponseWriter, r *http.Request) {
	if _, ok := utils.RequireAdmin(w, r, h.Config.JWTSecret); !ok {
		return
	}
	schema, err := loadAdminSchema(mux.Vars(r)["table"])
	if err != nil {
		adminDBError(w, err)
		return
	}
	if schema == nil {
		adminError(w, http.StatusNotFound, "Table not found")
		return
	}
	adminJSON(w, http.StatusOK, schema)
}

// GetTableRows — GET /admin/tables/{table}/rows?limit=&offset=&sort=&dir=&search=
// search matches against the whole row rendered as text.
func (h *AdminHandler) GetTableRows(w http.ResponseWriter, r *http.Request) {
	if _, ok := utils.RequireAdmin(w, r, h.Config.JWTSecret); !ok {
		return
	}
	schema, err := loadAdminSchema(mux.Vars(r)["table"])
	if err != nil {
		adminDBError(w, err)
		return
	}
	if schema == nil {
		adminError(w, http.StatusNotFound, "Table not found")
		return
	}

	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	if offset < 0 {
		offset = 0
	}

	qualified := "public." + pq.QuoteIdentifier(schema.Name)
	where := ""
	args := []interface{}{}
	if search := strings.TrimSpace(q.Get("search")); search != "" {
		where = " WHERE t::text ILIKE $1"
		args = append(args, "%"+search+"%")
	}

	orderBy := ""
	if sort := q.Get("sort"); sort != "" && schema.hasColumn(sort) {
		dir := "ASC"
		if strings.EqualFold(q.Get("dir"), "desc") {
			dir = "DESC"
		}
		orderBy = fmt.Sprintf(" ORDER BY t.%s %s NULLS LAST", pq.QuoteIdentifier(sort), dir)
	} else if len(schema.PrimaryKey) > 0 {
		cols := make([]string, len(schema.PrimaryKey))
		for i, c := range schema.PrimaryKey {
			cols[i] = "t." + pq.QuoteIdentifier(c)
		}
		orderBy = " ORDER BY " + strings.Join(cols, ", ")
	}

	var total int64
	if err := database.DB.QueryRow("SELECT count(*) FROM "+qualified+" t"+where, args...).Scan(&total); err != nil {
		adminDBError(w, err)
		return
	}

	query := fmt.Sprintf("SELECT %s FROM %s t%s%s LIMIT %d OFFSET %d",
		schema.selectList("t"), qualified, where, orderBy, limit, offset)
	rows, err := database.DB.Query(query, args...)
	if err != nil {
		adminDBError(w, err)
		return
	}
	defer rows.Close()
	data, err := schema.scanRows(rows)
	if err != nil {
		adminDBError(w, err)
		return
	}
	adminJSON(w, http.StatusOK, map[string]interface{}{
		"rows": data, "total": total, "limit": limit, "offset": offset,
	})
}

// InsertTableRow — POST /admin/tables/{table}/rows
// Body: { "<column>": value, ... }. Columns left out get their default.
func (h *AdminHandler) InsertTableRow(w http.ResponseWriter, r *http.Request) {
	if _, ok := utils.RequireAdmin(w, r, h.Config.JWTSecret); !ok {
		return
	}
	schema, err := loadAdminSchema(mux.Vars(r)["table"])
	if err != nil {
		adminDBError(w, err)
		return
	}
	if schema == nil {
		adminError(w, http.StatusNotFound, "Table not found")
		return
	}

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		adminError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	var cols, placeholders []string
	var args []interface{}
	for _, c := range schema.Columns { // iterate the schema, not the body, so only real columns get through
		v, ok := body[c.Name]
		if !ok {
			continue
		}
		args = append(args, adminParam(v))
		cols = append(cols, pq.QuoteIdentifier(c.Name))
		placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
	}
	for k := range body {
		if !schema.hasColumn(k) {
			adminError(w, http.StatusBadRequest, "Unknown column: "+k)
			return
		}
	}

	qualified := "public." + pq.QuoteIdentifier(schema.Name)
	var query string
	if len(cols) == 0 {
		query = "INSERT INTO " + qualified + " AS t DEFAULT VALUES"
	} else {
		query = fmt.Sprintf("INSERT INTO %s AS t (%s) VALUES (%s)",
			qualified, strings.Join(cols, ", "), strings.Join(placeholders, ", "))
	}
	query += " RETURNING " + schema.selectList("t")

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		adminDBError(w, err)
		return
	}
	defer rows.Close()
	data, err := schema.scanRows(rows)
	if err != nil {
		adminDBError(w, err)
		return
	}
	var row map[string]interface{}
	if len(data) > 0 {
		row = data[0]
	}
	adminJSON(w, http.StatusCreated, map[string]interface{}{"row": row})
}

// DeleteTableRow — DELETE /admin/tables/{table}/rows
// Body: { "key": { "<pk column>": value, ... } }. Every primary-key column
// must be given, so exactly one row can match.
func (h *AdminHandler) DeleteTableRow(w http.ResponseWriter, r *http.Request) {
	if _, ok := utils.RequireAdmin(w, r, h.Config.JWTSecret); !ok {
		return
	}
	schema, err := loadAdminSchema(mux.Vars(r)["table"])
	if err != nil {
		adminDBError(w, err)
		return
	}
	if schema == nil {
		adminError(w, http.StatusNotFound, "Table not found")
		return
	}
	if len(schema.PrimaryKey) == 0 {
		adminError(w, http.StatusBadRequest, "Table has no primary key; rows can't be deleted safely from here")
		return
	}

	var body struct {
		Key map[string]interface{} `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		adminError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	conds := make([]string, len(schema.PrimaryKey))
	args := make([]interface{}, len(schema.PrimaryKey))
	for i, c := range schema.PrimaryKey {
		v, ok := body.Key[c]
		if !ok || v == nil {
			adminError(w, http.StatusBadRequest, "Missing primary key column: "+c)
			return
		}
		args[i] = adminParam(v)
		// Compare as text: the portal only ever has the ::text rendering.
		conds[i] = fmt.Sprintf("%s::text = $%d", pq.QuoteIdentifier(c), i+1)
	}

	res, err := database.DB.Exec(
		"DELETE FROM public."+pq.QuoteIdentifier(schema.Name)+" WHERE "+strings.Join(conds, " AND "),
		args...)
	if err != nil {
		adminDBError(w, err)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		adminError(w, http.StatusNotFound, "Row not found")
		return
	}
	adminJSON(w, http.StatusOK, map[string]interface{}{"deleted": n})
}

// GetUserDetail — GET /admin/users/{uuid}
// Returns the user row plus every row in any table with a foreign key to
// users(uuid), discovered from the catalog so new tables show up on their own.
func (h *AdminHandler) GetUserDetail(w http.ResponseWriter, r *http.Request) {
	if _, ok := utils.RequireAdmin(w, r, h.Config.JWTSecret); !ok {
		return
	}
	userID := mux.Vars(r)["uuid"]

	usersSchema, err := loadAdminSchema("users")
	if err != nil || usersSchema == nil {
		adminError(w, http.StatusInternalServerError, "users table not found")
		return
	}
	rows, err := database.DB.Query(
		"SELECT "+usersSchema.selectList("t")+" FROM public.users t WHERE t.uuid::text = $1", userID)
	if err != nil {
		adminDBError(w, err)
		return
	}
	users, err := usersSchema.scanRows(rows)
	rows.Close()
	if err != nil {
		adminDBError(w, err)
		return
	}
	if len(users) == 0 {
		adminError(w, http.StatusNotFound, "User not found")
		return
	}

	fkRows, err := database.DB.Query(
		`SELECT cl.relname, a.attname
		   FROM pg_constraint c
		   JOIN pg_class cl ON cl.oid = c.conrelid
		   JOIN pg_namespace n ON n.oid = cl.relnamespace AND n.nspname = 'public'
		   JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = c.conkey[1]
		  WHERE c.contype = 'f'
		    AND c.confrelid = 'public.users'::regclass
		    AND array_length(c.conkey, 1) = 1
		  ORDER BY cl.relname, a.attname`)
	if err != nil {
		adminDBError(w, err)
		return
	}
	type ref struct{ table, column string }
	var refs []ref
	for fkRows.Next() {
		var rf ref
		if err := fkRows.Scan(&rf.table, &rf.column); err != nil {
			fkRows.Close()
			adminDBError(w, err)
			return
		}
		refs = append(refs, rf)
	}
	fkRows.Close()

	type related struct {
		Table      string                   `json:"table"`
		Column     string                   `json:"column"`
		Total      int64                    `json:"total"`
		PrimaryKey []string                 `json:"primary_key"`
		Columns    []adminColumn            `json:"columns"`
		Rows       []map[string]interface{} `json:"rows"`
	}
	const relatedLimit = 100
	out := []related{}
	for _, rf := range refs {
		schema, err := loadAdminSchema(rf.table)
		if err != nil || schema == nil {
			continue
		}
		qualified := "public." + pq.QuoteIdentifier(rf.table)
		cond := " WHERE t." + pq.QuoteIdentifier(rf.column) + "::text = $1"
		var total int64
		if err := database.DB.QueryRow("SELECT count(*) FROM "+qualified+" t"+cond, userID).Scan(&total); err != nil {
			adminDBError(w, err)
			return
		}
		rs, err := database.DB.Query(
			fmt.Sprintf("SELECT %s FROM %s t%s LIMIT %d", schema.selectList("t"), qualified, cond, relatedLimit), userID)
		if err != nil {
			adminDBError(w, err)
			return
		}
		data, err := schema.scanRows(rs)
		rs.Close()
		if err != nil {
			adminDBError(w, err)
			return
		}
		out = append(out, related{
			Table: rf.table, Column: rf.column, Total: total,
			PrimaryKey: schema.PrimaryKey, Columns: schema.Columns, Rows: data,
		})
	}

	adminJSON(w, http.StatusOK, map[string]interface{}{"user": users[0], "related": out})
}
