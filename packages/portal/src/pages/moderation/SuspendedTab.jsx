import { Link } from 'react-router-dom';
import { Banner, Button, Chip, EmptyState, Loading } from '../../components/ui';
import { useSuspendReader } from '../../components/useSuspendReader';
import { useSuspensions } from '../../services/moderation';
import { When } from './targets';
import { queueLink } from './useOpenRow';

// Who is suspended now, and who has been and is back (portal-moderation R2).

const STATES = [
  { id: 'current', label: 'Suspended', count: 'current_count', empty: 'Nobody is suspended' },
  { id: 'previous', label: 'Previously suspended', count: 'previous_count', empty: 'Nobody has been reinstated yet' },
];

const times = (n) => `${n} time${n === 1 ? '' : 's'}`;

const SuspendedTab = ({ params, update }) => {
  const state = STATES.find((s) => s.id === params.get('state')) ?? STATES[0];
  const current = state.id === 'current';
  const { data, isLoading, error, refetch, fetchNextPage, hasNextPage, isFetchingNextPage } = useSuspensions(state.id);
  const { suspendReader, dialog } = useSuspendReader();
  const readers = data?.readers ?? [];
  const columns = current
    ? ['Reader', 'Times suspended', 'Suspended since', 'By', 'Why', '']
    : ['Reader', 'Times suspended', 'Last reinstated', ''];

  return (
    <>
      {dialog}
      <div className="flex flex-wrap gap-2" role="group" aria-label="Which readers">
        {STATES.map((s) => (
          <Chip key={s.id} selected={s.id === state.id} onClick={() => update({ state: s.id === 'current' ? null : s.id })}>
            {s.label}{data ? ` · ${data[s.count]}` : ''}
          </Chip>
        ))}
      </div>

      {isLoading && <Loading />}
      {error && <Banner message={error.message} onRetry={refetch} />}
      {!isLoading && !error && readers.length === 0 && <EmptyState title={state.empty} />}
      {readers.length > 0 && (
        <div className="overflow-x-auto bg-surface-1 rounded-card">
          <table className="min-w-full text-[13px] border-collapse">
            <thead>
              <tr className="hairline-b">
                {columns.map((h) => (
                  <th key={h} className="px-4 py-3 text-left whitespace-nowrap text-[11px] font-bold tracking-[1px] uppercase text-ink-meta">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {readers.map((r) => {
                const last = r.last;
                return (
                  <tr key={r.id} className="hairline-b last:shadow-none align-top">
                    <td className="px-4 py-3">
                      <Link to={`/users/${r.id}`} className="text-link font-bold hover:underline">{r.username}</Link>
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap text-ink-title">{times(r.suspension_count)}</td>
                    {current ? (
                      <>
                        <td className="px-4 py-3">
                          {last?.suspended_at ? <When iso={last.suspended_at} /> : <span className="text-xs text-ink-holder">Before history began</span>}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-ink-body">{last?.suspended_by?.username ?? <span className="text-ink-holder">—</span>}</td>
                        <td className="px-4 py-3 min-w-[220px]">
                          <div className="flex flex-col gap-1">
                            {/* What an admin wrote, as text. */}
                            {last?.note && <span className="text-ink-body whitespace-pre-wrap break-words">{last.note}</span>}
                            {last?.report_id && (
                              <Link to={queueLink('about_user', r.id)} className="text-xs font-bold text-link hover:underline">From a report</Link>
                            )}
                            {!last?.note && !last?.report_id && <span className="text-ink-holder">—</span>}
                          </div>
                        </td>
                      </>
                    ) : (
                      <td className="px-4 py-3">
                        {last?.reinstated_at ? <When iso={last.reinstated_at} /> : <span className="text-xs text-ink-holder">Date unknown</span>}
                      </td>
                    )}
                    <td className="px-4 py-3">
                      {current && (
                        <Button
                          variant="secondary"
                          className="h-9 px-4 text-[13px]"
                          onClick={() => suspendReader({ username: r.username, suspending: false, uuid: r.id })}
                        >
                          Reinstate
                        </Button>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
      {hasNextPage && (
        <Button variant="secondary" className="self-start" disabled={isFetchingNextPage} onClick={() => fetchNextPage()}>
          {isFetchingNextPage ? 'Loading…' : 'Load more'}
        </Button>
      )}
    </>
  );
};

export default SuspendedTab;
