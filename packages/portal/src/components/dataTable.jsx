import { useEffect, useState } from 'react';
import { useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query';
import { api } from '../services/api';
import { Banner, Button, Card, Field, Loading, SearchInput, TextAction } from './ui';
import { tablePath, useDeleteRow } from './useDeleteRow';

// Shared by the Database, Users and User detail pages. Talks to the generic
// /admin/tables endpoints (internal/handlers/admin.go in api-go).

const useTableSchema = (table) =>
  useQuery({
    queryKey: ['admin-schema', table],
    queryFn: () => api.get(tablePath(table)),
    enabled: !!table,
    staleTime: 5 * 60 * 1000,
  });

const Cell = ({ value }) => {
  if (value === null || value === undefined) {
    return <span className="text-ink-disabled italic">null</span>;
  }
  const text = typeof value === 'object' ? JSON.stringify(value) : String(value);
  return (
    <span className="block max-w-xs truncate" title={text}>
      {text}
    </span>
  );
};

// Plain rows table: header, cells, optional per-row delete and click-through.
export const RowsTable = ({ columns, primaryKey = [], rows, onDelete, cellLink, sort, onSort }) => (
  <div className="overflow-x-auto bg-surface-1 rounded-card">
    <table className="min-w-full text-[13px] border-collapse">
      <thead>
        <tr className="hairline-b">
          {columns.map((c) => (
            <th
              key={c.name}
              onClick={onSort ? () => onSort(c.name) : undefined}
              className={`px-4 py-3 text-left whitespace-nowrap text-[11px] font-bold tracking-[1px] uppercase text-ink-meta ${onSort ? 'cursor-pointer hover:text-ink-title' : ''}`}
            >
              {c.is_primary && <span className="text-gold mr-1" title="Primary key">●</span>}
              {c.name}
              <span className="ml-1.5 normal-case tracking-normal font-semibold text-ink-holder">{c.type}</span>
              {sort?.column === c.name && <span className="ml-1 text-link">{sort.dir === 'asc' ? '▲' : '▼'}</span>}
            </th>
          ))}
          {onDelete && <th className="px-4 py-3" />}
        </tr>
      </thead>
      <tbody>
        {rows.length === 0 && (
          <tr>
            <td colSpan={columns.length + 1} className="px-4 py-8 text-center text-ink-holder">
              No rows
            </td>
          </tr>
        )}
        {rows.map((row, i) => (
          <tr key={primaryKey.length ? primaryKey.map((c) => row[c]).join('|') : i} className="hairline-b last:shadow-none hover:bg-surface-2">
            {columns.map((c) => {
              const link = cellLink?.(c.name, row);
              return (
                <td key={c.name} className="px-4 py-2.5 text-ink-body align-top">
                  {link ? (
                    <button type="button" onClick={link} className="p-0 bg-transparent text-link font-bold hover:underline text-left">
                      <Cell value={row[c.name]} />
                    </button>
                  ) : (
                    <Cell value={row[c.name]} />
                  )}
                </td>
              );
            })}
            {onDelete && (
              <td className="px-4 py-2.5 text-right">
                {primaryKey.length > 0 && (
                  <TextAction tone="muted" className="text-xs" onClick={() => onDelete(row)}>Delete</TextAction>
                )}
              </td>
            )}
          </tr>
        ))}
      </tbody>
    </table>
  </div>
);

// Form built from the table schema. Blank fields are left out of the insert,
// so Postgres applies the column default (or NULL).
const AddRowForm = ({ table, columns, onCreated, onCancel }) => {
  const [values, setValues] = useState({});
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState(null);

  const set = (name, v) => setValues((prev) => ({ ...prev, [name]: v }));

  const submit = async (e) => {
    e.preventDefault();
    const body = Object.fromEntries(Object.entries(values).filter(([, v]) => v !== ''));
    setSaving(true);
    setError(null);
    try {
      const response = await api.post(tablePath(table, '/rows'), body);
      onCreated(response.row);
    } catch (err) {
      setError(err.message);
    } finally {
      setSaving(false);
    }
  };

  const field = (c) => {
    const v = values[c.name] ?? '';
    const required = !c.nullable && !c.has_default;
    const label = `${c.name} · ${c.type}${required ? ' *' : ''}`;
    const empty = c.has_default ? '(default)' : '(null)';
    const onChange = (e) => set(c.name, e.target.value);
    if (c.enum_values?.length || c.type === 'boolean') {
      const options = c.enum_values?.length ? c.enum_values : ['true', 'false'];
      return (
        <Field key={c.name} label={label} as="select" value={v} onChange={onChange}>
          <option value="">{empty}</option>
          {options.map((o) => <option key={o} value={o}>{o}</option>)}
        </Field>
      );
    }
    const multiline = c.type === 'json' || c.type === 'jsonb' || c.type === 'text';
    return (
      <Field
        key={c.name}
        label={label}
        multiline={multiline}
        rows={multiline ? 2 : undefined}
        value={v}
        onChange={onChange}
        placeholder={c.has_default ? '(default)' : c.type.startsWith('_') ? '{a,b,c}' : ''}
      />
    );
  };

  return (
    <Card className="p-6">
      <form onSubmit={submit} className="flex flex-col gap-4">
        <span className="text-[17px] font-extrabold text-ink-title">New row in {table}</span>
        <Banner message={error} />
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">{columns.map(field)}</div>
        <div className="flex gap-2.5">
          <Button type="submit" disabled={saving}>{saving ? 'Saving…' : 'Insert row'}</Button>
          <Button variant="secondary" onClick={onCancel}>Cancel</Button>
        </div>
      </form>
    </Card>
  );
};

const PAGE_SIZE = 50;

// Full browser for one table: search, sort, paginate, insert, delete.
const DataTable = ({ table, cellLink }) => {
  const queryClient = useQueryClient();
  const deleteRow = useDeleteRow();
  const [search, setSearch] = useState('');
  const [debounced, setDebounced] = useState('');
  const [page, setPage] = useState(0);
  const [sort, setSort] = useState(null);
  const [adding, setAdding] = useState(false);
  const [banner, setBanner] = useState('');

  useEffect(() => {
    const t = setTimeout(() => { setDebounced(search); setPage(0); }, 300);
    return () => clearTimeout(t);
  }, [search]);

  useEffect(() => {
    setSearch(''); setDebounced(''); setPage(0); setSort(null); setAdding(false); setBanner('');
  }, [table]);

  const { data: schema, error: schemaError } = useTableSchema(table);

  const { data, isLoading, error } = useQuery({
    queryKey: ['admin-rows', table, debounced, page, sort?.column, sort?.dir],
    queryFn: () => {
      const params = new URLSearchParams({ limit: PAGE_SIZE, offset: page * PAGE_SIZE });
      if (debounced) params.set('search', debounced);
      if (sort) { params.set('sort', sort.column); params.set('dir', sort.dir); }
      return api.get(tablePath(table, `/rows?${params}`));
    },
    enabled: !!table,
    placeholderData: keepPreviousData,
  });

  const refresh = () => {
    queryClient.invalidateQueries({ queryKey: ['admin-rows', table] });
    queryClient.invalidateQueries({ queryKey: ['admin-tables'] });
  };

  const onSort = (column) =>
    setSort((s) => (s?.column === column ? (s.dir === 'asc' ? { column, dir: 'desc' } : null) : { column, dir: 'asc' }));

  const onDelete = async (row) => {
    try {
      if (await deleteRow(table, schema.primary_key, row)) refresh();
    } catch (err) {
      setBanner(`Couldn't delete the row: ${err.message}`);
    }
  };

  const err = schemaError || error;
  if (err) return <Banner message={err.message} onRetry={refresh} />;
  if (!schema) return <Loading />;

  const total = data?.total ?? 0;
  const pages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-2.5">
        <SearchInput className="flex-1 min-w-[200px]" placeholder={`Search ${table}`} value={search} onChange={setSearch} />
        <Button variant="secondary" onClick={() => setAdding((a) => !a)}>{adding ? 'Close form' : 'Add row'}</Button>
        <Button variant="secondary" onClick={refresh}>Refresh</Button>
      </div>

      <Banner message={banner} onDismiss={() => setBanner('')} />

      {schema.primary_key.length === 0 && (
        <p className="m-0 text-xs font-semibold text-ink-meta">This table has no primary key, so rows can't be deleted from here.</p>
      )}

      {adding && (
        <AddRowForm
          table={table}
          columns={schema.columns}
          onCreated={() => { setAdding(false); refresh(); }}
          onCancel={() => setAdding(false)}
        />
      )}

      {isLoading ? (
        <Loading />
      ) : (
        <RowsTable
          columns={schema.columns}
          primaryKey={schema.primary_key}
          rows={data?.rows ?? []}
          onDelete={onDelete}
          cellLink={cellLink}
          sort={sort}
          onSort={onSort}
        />
      )}

      <div className="flex items-center justify-between text-xs font-semibold text-ink-meta">
        <span>{total} row{total === 1 ? '' : 's'}</span>
        <div className="flex items-center gap-4">
          <TextAction disabled={page === 0} onClick={() => setPage((p) => p - 1)}>Prev</TextAction>
          <span>Page {page + 1} of {pages}</span>
          <TextAction disabled={page + 1 >= pages} onClick={() => setPage((p) => p + 1)}>Next</TextAction>
        </div>
      </div>
    </div>
  );
};

export default DataTable;
