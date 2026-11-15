import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { api } from '../services/api';
import { RowsTable } from '../components/dataTable';
import { useDeleteRow } from '../components/useDeleteRow';
import { useSuspendReader } from '../components/useSuspendReader';
import { isSuspended, useUserSuspensions } from '../services/moderation';
import { Banner, Button, Card, Eyebrow, List, ListItem, Loading, PageHeader, Section, Tag, TextAction } from '../components/ui';

const when = (iso) => new Date(iso).toLocaleString();
const timesLabel = (n) => (n === 0 ? 'Never suspended' : `Suspended ${n} time${n === 1 ? '' : 's'}`);
const reportsAbout = (uuid) => `/reports?tab=queue&status=all&about_user=${uuid}`;

// Notes are what admins wrote, shown as text.
const Note = ({ children }) => (
  <span className="text-[13px] text-ink-body whitespace-pre-wrap break-words">“{children}”</span>
);

// One reader's suspensions, open one first, then newest first
// (portal-moderation R3.1).
const SuspensionHistory = ({ uuid }) => {
  const { data, isLoading, error, refetch } = useUserSuspensions(uuid);
  if (isLoading) return <Section title="Suspension history"><Loading /></Section>;
  if (error) return <Section title="Suspension history"><Banner message={error.message} onRetry={refetch} /></Section>;
  const list = data.suspensions;
  return (
    <Section title={timesLabel(data.count)}>
      {list.length > 0 && (
        <List>
          {list.map((s) => (
            <ListItem key={s.id} className="flex flex-col gap-1.5">
              <span className="flex flex-wrap items-center gap-2 text-[13px] text-ink-title">
                <span className="font-bold">
                  {s.suspended_at ? `Suspended ${when(s.suspended_at)}` : 'Suspended before history began'}
                </span>
                {s.suspended_by && <span className="text-ink-holder">by {s.suspended_by.username}</span>}
                {!s.ended && <Tag>Still suspended</Tag>}
                {s.report_id && (
                  <Link to={reportsAbout(uuid)} className="text-link font-bold hover:underline">From a report</Link>
                )}
              </span>
              {s.note && <Note>{s.note}</Note>}
              {s.ended && (
                <span className="flex flex-wrap items-center gap-2 text-[13px] text-ink-body">
                  {s.reinstated_at ? `Reinstated ${when(s.reinstated_at)}` : 'Reinstated, date unknown'}
                  {s.reinstated_by && <span className="text-ink-holder">by {s.reinstated_by.username}</span>}
                </span>
              )}
              {s.reinstate_note && <Note>{s.reinstate_note}</Note>}
            </ListItem>
          ))}
        </List>
      )}
    </Section>
  );
};

const UserDetailPage = () => {
  const { uuid } = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const deleteRow = useDeleteRow();
  const { suspendReader, dialog } = useSuspendReader();
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

  // Suspend or reinstate (report-block R9.6, R10.4), with an optional note
  // (portal-moderation R3.2). The dialog makes the call and reloads.
  const toggleSuspended = (user) =>
    suspendReader({ username: user.username, suspending: !isSuspended(user), uuid: user.uuid });

  const back = <TextAction onClick={() => navigate('/users')} className="self-start">← All users</TextAction>;

  if (isLoading) return <>{back}<Loading /></>;
  if (error) return <>{back}<Banner message={error.message} onRetry={refetch} /></>;

  const { user, related } = data;
  const suspended = isSuspended(user);

  return (
    <>
      {back}
      <PageHeader title={user.username} subtitle={user.email}>
        {suspended && <Tag muted>Suspended</Tag>}
        <Link to={reportsAbout(user.uuid)} className="text-[13px] font-bold text-link hover:underline">See reports about them</Link>
        {user.username !== 'deleted.reader' && (
          <Button variant="secondary" onClick={() => toggleSuspended(user)}>
            {suspended ? 'Reinstate' : 'Suspend'}
          </Button>
        )}
        <Button variant="secondary" onClick={() => tryDelete('users', ['uuid'], user, () => { refresh(); navigate('/users'); })}>
          Delete user
        </Button>
      </PageHeader>

      <Banner message={banner} onDismiss={() => setBanner('')} />
      {dialog}

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

      {user.username !== 'deleted.reader' && <SuspensionHistory uuid={user.uuid} />}

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
