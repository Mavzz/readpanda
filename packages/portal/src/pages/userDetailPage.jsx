import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate, useParams } from 'react-router-dom';
import { api } from '../services/api';
import { RowsTable } from '../components/dataTable';
import { useDeleteRow } from '../components/useDeleteRow';
import { Banner, Button, Card, Eyebrow, Loading, PageHeader, Section, TextAction } from '../components/ui';

const UserDetailPage = () => {
  const { uuid } = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const deleteRow = useDeleteRow();
  const [banner, setBanner] = useState('');

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['admin-user', uuid],
    queryFn: () => api.get(`/admin/users/${uuid}`),
  });

  const refresh = () => {
    queryClient.invalidateQueries({ queryKey: ['admin-user', uuid] });
    queryClient.invalidateQueries({ queryKey: ['admin-rows'] });
    queryClient.invalidateQueries({ queryKey: ['admin-tables'] });
  };

  const tryDelete = async (table, primaryKey, row, after) => {
    try {
      if (await deleteRow(table, primaryKey, row)) after();
    } catch (err) {
      setBanner(`Couldn't delete: ${err.message}`);
    }
  };

  const back = <TextAction onClick={() => navigate('/users')} className="self-start">← All users</TextAction>;

  if (isLoading) return <>{back}<Loading /></>;
  if (error) return <>{back}<Banner message={error.message} onRetry={refetch} /></>;

  const { user, related } = data;

  return (
    <>
      {back}
      <PageHeader title={user.username} subtitle={user.email}>
        <Button variant="secondary" onClick={() => tryDelete('users', ['uuid'], user, () => { refresh(); navigate('/users'); })}>
          Delete user
        </Button>
      </PageHeader>

      <Banner message={banner} onDismiss={() => setBanner('')} />

      <Card>
        <dl className="m-0 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-x-6 gap-y-4">
          {Object.entries(user).map(([k, v]) => (
            <div key={k} className="min-w-0 flex flex-col gap-1.5">
              <dt><Eyebrow>{k}</Eyebrow></dt>
              <dd className="m-0 text-[13px] text-ink-title break-all">
                {v === null || v === undefined ? <span className="text-ink-disabled italic">null</span> : String(v)}
              </dd>
            </div>
          ))}
        </dl>
      </Card>

      {related.map((r) => (
        <Section
          key={`${r.table}.${r.column}`}
          title={`${r.table} · via ${r.column} · ${r.total} row${r.total === 1 ? '' : 's'}${r.total > r.rows.length ? ` (first ${r.rows.length})` : ''}`}
        >
          <RowsTable
            columns={r.columns}
            primaryKey={r.primary_key}
            rows={r.rows}
            onDelete={(row) => tryDelete(r.table, r.primary_key, row, refresh)}
          />
        </Section>
      ))}
    </>
  );
};

export default UserDetailPage;
