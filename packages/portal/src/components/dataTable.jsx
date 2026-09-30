import React, { useEffect, useState } from 'react';
import { useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query';
import { useGet } from '../services/useGet';
import { usePost } from '../services/usePost';
import { useDelete } from '../services/useDelete';
import { getBackendUrl } from '../utils/Helper';

// Shared by the Database, Users and User detail pages. Talks to the generic
// /admin/tables endpoints (internal/handlers/admin.go in api-go).

export const authHeaders = () => ({
  Authorization: `Bearer ${localStorage.getItem('token')}`,
});

export const tableUrl = (table, suffix = '') =>
  getBackendUrl(`/admin/tables/${encodeURIComponent(table)}${suffix}`);

export const useTableSchema = (table) =>
  useQuery({
    queryKey: ['admin-schema', table],
    queryFn: async () => (await useGet(await tableUrl(table), authHeaders())).response,
    enabled: !!table,
    staleTime: 5 * 60 * 1000,
  });

export const deleteRow = async (table, primaryKey, row) => {
  const key = Object.fromEntries(primaryKey.map((c) => [c, row[c]]));
  const label = primaryKey.map((c) => `${c}=${row[c]}`).join(', ');
  if (!window.confirm(`Delete row from "${table}"?\n\n${label}\n\nRows in other tables that cascade from it will also be deleted.`)) {
    return false;
  }
  try {
    await useDelete(await tableUrl(table, '/rows'), authHeaders(), null, { key });
    return true;
  } catch (err) {
    alert(err.message);
    return false;
  }
};

const Cell = ({ value }) => {
  if (value === null || value === undefined) {
    return <span className="text-gray-300 italic">null</span>;
  }
  const text = String(value);
  return (
    <span className="block max-w-xs truncate" title={text}>
      {text}
    </span>
  );
};

// Plain rows table: header, cells, optional per-row delete and click-through.
export const RowsTable = ({ columns, primaryKey = [], rows, onDelete, cellLink, sort, onSort }) => (
  <div className="overflow-x-auto border border-gray-200 rounded-lg bg-white">
    <table className="min-w-full text-sm">
      <thead className="bg-gray-50 border-b border-gray-200">
        <tr>
          {columns.map((c) => (
            <th
              key={c.name}
              onClick={onSort ? () => onSort(c.name) : undefined}
              className={`px-3 py-2 text-left font-semibold text-gray-600 whitespace-nowrap ${onSort ? 'cursor-pointer hover:text-gray-900' : ''}`}
            >
              {c.is_primary && <span className="text-amber-500 mr-1" title="Primary key">●</span>}
              {c.name}
              <span className="ml-1 font-normal text-xs text-gray-400">{c.type}</span>
              {sort?.column === c.name && <span className="ml-1">{sort.dir === 'asc' ? '▲' : '▼'}</span>}
            </th>
          ))}
          {onDelete && <th className="px-3 py-2" />}
        </tr>
      </thead>
      <tbody className="divide-y divide-gray-100">
        {rows.length === 0 && (
          <tr>
            <td colSpan={columns.length + 1} className="px-3 py-6 text-center text-gray-400">
              No rows
            </td>
          </tr>
        )}
        {rows.map((row, i) => (
          <tr key={primaryKey.length ? primaryKey.map((c) => row[c]).join('|') : i} className="hover:bg-gray-50">
            {columns.map((c) => {
              const link = cellLink?.(c.name, row);
              return (
                <td key={c.name} className="px-3 py-2 text-gray-700 align-top">
                  {link ? (
                    <button onClick={link} className="text-indigo-600 hover:underline text-left">
                      <Cell value={row[c.name]} />
                    </button>
                  ) : (
                    <Cell value={row[c.name]} />
                  )}
                </td>
              );
            })}
            {onDelete && (
              <td className="px-3 py-2 text-right">
                {primaryKey.length > 0 && (
                  <button onClick={() => onDelete(row)} className="text-red-600 hover:text-red-800 text-xs font-medium">
                    Delete
                  </button>
                )}
              </td>
            )}
          </tr>
        ))}
      </tbody>
    </table>
  </div>
);

const inputClass = 'w-full border border-gray-300 rounded-md px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500';

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
      const { response } = await usePost(await tableUrl(table, '/rows'), body, authHeaders());
      onCreated(response.row);
    } catch (err) {
      setError(err.message);
    } finally {
      setSaving(false);
    }
  };

  const field = (c) => {
    const v = values[c.name] ?? '';
    if (c.enum_values?.length) {
      return (
        <select className={inputClass} value={v} onChange={(e) => set(c.name, e.target.value)}>
          <option value="">{c.has_default ? '(default)' : '(null)'}</option>
          {c.enum_values.map((o) => <option key={o} value={o}>{o}</option>)}
        </select>
      );
    }
    if (c.type === 'boolean') {
      return (
        <select className={inputClass} value={v} onChange={(e) => set(c.name, e.target.value)}>
          <option value="">{c.has_default ? '(default)' : '(null)'}</option>
          <option value="true">true</option>
          <option value="false">false</option>
        </select>
      );
    }
    if (c.type === 'json' || c.type === 'jsonb' || c.type === 'text') {
      return <textarea rows={2} className={inputClass} value={v} onChange={(e) => set(c.name, e.target.value)} placeholder={c.has_default ? '(default)' : ''} />;
    }
    return <input className={inputClass} value={v} onChange={(e) => set(c.name, e.target.value)} placeholder={c.has_default ? '(default)' : c.type.startsWith('_') ? '{a,b,c}' : ''} />;
  };

  return (
    <form onSubmit={submit} className="bg-white border border-gray-200 rounded-lg p-4 mb-4">
      <h3 className="font-semibold text-gray-800 mb-3">New row in {table}</h3>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        {columns.map((c) => (
          <label key={c.name} className="block">
            <span className="text-xs font-medium text-gray-600">
              {c.name}
              <span className="ml-1 text-gray-400">{c.type}</span>
              {!c.nullable && !c.has_default && <span className="ml-1 text-red-500">*</span>}
            </span>
            {field(c)}
          </label>
        ))}
      </div>
      {error && <p className="mt-3 text-sm text-red-600 break-words">{error}</p>}
      <div className="mt-4 flex gap-2">
        <button type="submit" disabled={saving} className="px-4 py-2 bg-indigo-600 text-white text-sm rounded-md hover:bg-indigo-700 disabled:opacity-50">
          {saving ? 'Saving…' : 'Insert row'}
        </button>
        <button type="button" onClick={onCancel} className="px-4 py-2 text-sm text-gray-600 rounded-md hover:bg-gray-100">
          Cancel
        </button>
      </div>
    </form>
  );
};

const PAGE_SIZE = 50;

// Full browser for one table: search, sort, paginate, insert, delete.
const DataTable = ({ table, cellLink }) => {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [debounced, setDebounced] = useState('');
  const [page, setPage] = useState(0);
  const [sort, setSort] = useState(null);
  const [adding, setAdding] = useState(false);

  useEffect(() => {
    const t = setTimeout(() => { setDebounced(search); setPage(0); }, 300);
    return () => clearTimeout(t);
  }, [search]);

  useEffect(() => {
    setSearch(''); setDebounced(''); setPage(0); setSort(null); setAdding(false);
  }, [table]);

  const { data: schema, error: schemaError } = useTableSchema(table);

  const rowsKey = ['admin-rows', table, debounced, page, sort?.column, sort?.dir];
  const { data, isLoading, error } = useQuery({
    queryKey: rowsKey,
    queryFn: async () => {
      const params = new URLSearchParams({ limit: PAGE_SIZE, offset: page * PAGE_SIZE });
      if (debounced) params.set('search', debounced);
      if (sort) { params.set('sort', sort.column); params.set('dir', sort.dir); }
      return (await useGet(await tableUrl(table, `/rows?${params}`), authHeaders())).response;
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
    if (await deleteRow(table, schema.primary_key, row)) refresh();
  };

  const err = schemaError || error;
  if (err) return <p className="text-red-600 text-sm break-words">{err.message}</p>;
  if (!schema) return <p className="text-gray-500 text-sm">Loading…</p>;

  const total = data?.total ?? 0;
  const pages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <div>
      <div className="flex flex-wrap items-center gap-3 mb-4">
        <input
          className="flex-1 min-w-[200px] border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
          placeholder={`Search ${table}…`}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <button onClick={() => setAdding((a) => !a)} className="px-4 py-2 bg-indigo-600 text-white text-sm rounded-md hover:bg-indigo-700">
          {adding ? 'Close form' : '+ Add row'}
        </button>
        <button onClick={refresh} className="px-3 py-2 text-sm text-gray-600 border border-gray-300 rounded-md hover:bg-gray-50">
          Refresh
        </button>
      </div>

      {schema.primary_key.length === 0 && (
        <p className="mb-3 text-xs text-amber-700">This table has no primary key, so rows can't be deleted from here.</p>
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
        <p className="text-gray-500 text-sm">Loading…</p>
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

      <div className="flex items-center justify-between mt-3 text-sm text-gray-600">
        <span>{total} row{total === 1 ? '' : 's'}</span>
        <div className="flex items-center gap-2">
          <button disabled={page === 0} onClick={() => setPage((p) => p - 1)} className="px-3 py-1 border border-gray-300 rounded-md disabled:opacity-40">
            Prev
          </button>
          <span>Page {page + 1} of {pages}</span>
          <button disabled={page + 1 >= pages} onClick={() => setPage((p) => p + 1)} className="px-3 py-1 border border-gray-300 rounded-md disabled:opacity-40">
            Next
          </button>
        </div>
      </div>
    </div>
  );
};

export default DataTable;
