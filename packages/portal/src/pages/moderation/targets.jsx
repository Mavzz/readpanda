import { Banner, Button, Chip, EmptyState, Field, Loading } from '../../components/ui';
import { REASON_LABELS, useReportTargets } from '../../services/moderation';

// What the Readers and Rooms tabs share (portal-moderation R4, R5): the
// open/all chips, the sort, the table and "Load more".

const FILTERS = [
  { id: 'open', label: 'Has open reports' },
  { id: 'all', label: 'All' },
];

const SORTS = [
  { id: 'open', label: 'Most open' },
  { id: 'total', label: 'Most reports' },
  { id: 'latest', label: 'Latest' },
];

export const When = ({ iso }) => <span className="whitespace-nowrap text-xs text-ink-body">{new Date(iso).toLocaleString()}</span>;

// The top three reasons, with counts (R4.1).
export const Reasons = ({ reasons }) => (
  <ul className="m-0 p-0 list-none flex flex-col gap-1">
    {reasons.map((r) => (
      <li key={r.reason} className="text-[13px] text-ink-body whitespace-nowrap">
        {REASON_LABELS[r.reason] ?? r.reason} <span className="text-ink-holder">· {r.count}</span>
      </li>
    ))}
  </ul>
);

export const Counts = ({ total, open }) => (
  <div className="flex flex-col gap-0.5 whitespace-nowrap">
    <span className="font-bold text-ink-title">{total} report{total === 1 ? '' : 's'}</span>
    <span className={open ? 'text-link font-bold text-xs' : 'text-ink-holder text-xs'}>{open} open</span>
  </div>
);

const TargetsTab = ({ type, params, update, columns, empty, renderRow }) => {
  const filter = params.get('filter') === 'all' ? 'all' : 'open';
  const sort = SORTS.some((s) => s.id === params.get('sort')) ? params.get('sort') : 'open';
  const { data, isLoading, error, refetch, fetchNextPage, hasNextPage, isFetchingNextPage } = useReportTargets(type, filter, sort);
  const targets = data?.targets ?? [];

  return (
    <>
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div className="flex flex-wrap gap-2" role="group" aria-label="Which to show">
          {FILTERS.map((f) => (
            <Chip key={f.id} selected={filter === f.id} onClick={() => update({ filter: f.id === 'open' ? null : f.id })}>
              {f.label}
            </Chip>
          ))}
        </div>
        <Field as="select" label="Sort" value={sort} onChange={(e) => update({ sort: e.target.value === 'open' ? null : e.target.value })} className="w-44">
          {SORTS.map((s) => <option key={s.id} value={s.id}>{s.label}</option>)}
        </Field>
      </div>

      {isLoading && <Loading />}
      {error && <Banner message={error.message} onRetry={refetch} />}
      {!isLoading && !error && targets.length === 0 && <EmptyState title={empty[filter]} />}
      {targets.length > 0 && (
        <div className="overflow-x-auto bg-surface-1 rounded-card">
          <table className="min-w-full text-[13px] border-collapse">
            <thead>
              <tr className="hairline-b">
                {columns.map((h) => (
                  <th key={h} className="px-4 py-3 text-left whitespace-nowrap text-[11px] font-bold tracking-[1px] uppercase text-ink-meta">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>{targets.map(renderRow)}</tbody>
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

export default TargetsTab;
