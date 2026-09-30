import React from 'react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate, useParams } from 'react-router-dom';
import { useGet } from '../services/useGet';
import { getBackendUrl } from '../utils/Helper';
import DataTable, { authHeaders } from '../components/dataTable';

// Columns that hold a user's uuid — clicking one opens that user's detail page.
const USER_COLUMNS = new Set(['user_id', 'admin_id']);

const DatabasePage = () => {
  const { table } = useParams();
  const navigate = useNavigate();

  const { data: tables = [], error } = useQuery({
    queryKey: ['admin-tables'],
    queryFn: async () => (await useGet(await getBackendUrl('/admin/tables'), authHeaders())).response.tables,
  });

  const cellLink = (column, row) => {
    const isUser = (table === 'users' && column === 'uuid') || USER_COLUMNS.has(column);
    return isUser && row[column] ? () => navigate(`/users/${row[column]}`) : null;
  };

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-800 mb-6">Database</h1>
      {error && <p className="text-red-600 text-sm mb-4 break-words">{error.message}</p>}
      <div className="flex flex-col lg:flex-row gap-6">
        <aside className="lg:w-56 shrink-0">
          <ul className="bg-white border border-gray-200 rounded-lg divide-y divide-gray-100 text-sm">
            {tables.map((t) => (
              <li key={t.name}>
                <button
                  onClick={() => navigate(`/database/${t.name}`)}
                  className={`w-full flex justify-between px-3 py-2 text-left hover:bg-gray-50 ${t.name === table ? 'bg-indigo-50 text-indigo-700 font-medium' : 'text-gray-700'}`}
                >
                  <span className="truncate">{t.name}</span>
                  <span className="text-gray-400 ml-2">{t.row_count}</span>
                </button>
              </li>
            ))}
          </ul>
        </aside>
        <section className="flex-1 min-w-0">
          {table ? (
            <DataTable table={table} cellLink={cellLink} />
          ) : (
            <p className="text-gray-500 text-sm">Pick a table on the left.</p>
          )}
        </section>
      </div>
    </div>
  );
};

export default DatabasePage;
