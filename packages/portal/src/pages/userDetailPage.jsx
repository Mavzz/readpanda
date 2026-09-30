import React from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate, useParams } from 'react-router-dom';
import { useGet } from '../services/useGet';
import { getBackendUrl } from '../utils/Helper';
import { RowsTable, authHeaders, deleteRow } from '../components/dataTable';

const UserDetailPage = () => {
  const { uuid } = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useQuery({
    queryKey: ['admin-user', uuid],
    queryFn: async () => (await useGet(await getBackendUrl(`/admin/users/${uuid}`), authHeaders())).response,
  });

  const refresh = () => {
    queryClient.invalidateQueries({ queryKey: ['admin-user', uuid] });
    queryClient.invalidateQueries({ queryKey: ['admin-rows'] });
    queryClient.invalidateQueries({ queryKey: ['admin-tables'] });
  };

  if (isLoading) return <p className="text-gray-500 text-sm">Loading…</p>;
  if (error) return <p className="text-red-600 text-sm break-words">{error.message}</p>;

  const { user, related } = data;

  const deleteUser = async () => {
    if (await deleteRow('users', ['uuid'], user)) {
      refresh();
      navigate('/users');
    }
  };

  return (
    <div>
      <button onClick={() => navigate('/users')} className="text-sm text-indigo-600 hover:underline mb-4">
        ← All users
      </button>

      <div className="flex items-start justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-800">{user.username}</h1>
          <p className="text-gray-500">{user.email}</p>
        </div>
        <button onClick={deleteUser} className="px-4 py-2 text-sm text-red-600 border border-red-200 rounded-md hover:bg-red-50">
          Delete user
        </button>
      </div>

      <dl className="bg-white border border-gray-200 rounded-lg grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-x-6 gap-y-3 p-4 mb-8 text-sm">
        {Object.entries(user).map(([k, v]) => (
          <div key={k} className="min-w-0">
            <dt className="text-xs font-medium text-gray-500">{k}</dt>
            <dd className="text-gray-800 break-all">{v ?? <span className="text-gray-300 italic">null</span>}</dd>
          </div>
        ))}
      </dl>

      <div className="space-y-8">
        {related.map((r) => (
          <section key={`${r.table}.${r.column}`}>
            <h2 className="text-lg font-semibold text-gray-800 mb-2">
              {r.table}
              <span className="ml-2 text-sm font-normal text-gray-400">
                via {r.column} · {r.total} row{r.total === 1 ? '' : 's'}
                {r.total > r.rows.length && ` (showing first ${r.rows.length})`}
              </span>
            </h2>
            <RowsTable
              columns={r.columns}
              primaryKey={r.primary_key}
              rows={r.rows}
              onDelete={async (row) => { if (await deleteRow(r.table, r.primary_key, row)) refresh(); }}
            />
          </section>
        ))}
      </div>
    </div>
  );
};

export default UserDetailPage;
