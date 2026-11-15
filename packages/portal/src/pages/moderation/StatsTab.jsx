import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { Banner, Card, Chip, Eyebrow, Loading } from '../../components/ui';
import { ACTION_LABELS, REASON_LABELS, useModerationStats } from '../../services/moderation';
import { chartTheme } from './chartTheme';
import { formatHours } from './formatHours';

// How much is reported and how fast it's dealt with (portal-moderation R7).

const RANGES = [4, 12, 52];

const ageOf = (iso) => formatHours((Date.now() - new Date(iso).getTime()) / 36e5);

// Weeks start on Monday (R7.7); the last one hasn't finished.
const weekLabel = (start, isLast) => (isLast
  ? 'This week'
  : new Date(`${start}T00:00:00Z`).toLocaleDateString(undefined, { day: 'numeric', month: 'short', timeZone: 'UTC' }));

const Stat = ({ label, value, sub }) => (
  <Card className="flex flex-col gap-2">
    <Eyebrow>{label}</Eyebrow>
    <span className="text-[30px] leading-none font-extrabold tracking-[-0.5px] text-ink-title">{value}</span>
    {sub && <span className="text-xs text-ink-holder">{sub}</span>}
  </Card>
);

// A swatch beside the name: the legend never relies on colour alone (R9.3).
const Legend = ({ series }) => (
  <ul className="m-0 p-0 list-none flex flex-wrap gap-4" aria-hidden="true">
    {series.map((s) => (
      <li key={s.key} className="inline-flex items-center gap-2 text-xs font-bold text-ink-body">
        <span className="w-2.5 h-2.5 rounded-[3px]" style={{ background: s.color }} />
        {s.label}
      </li>
    ))}
  </ul>
);

// Every series for the hovered week, values first.
const WeekTooltip = ({ active, payload, label, series }) => {
  if (!active || !payload?.length) return null;
  return (
    <div className="bg-surface-3 rounded-field px-3 py-2.5 flex flex-col gap-1.5 shadow-[0_20px_40px_rgba(11,19,38,0.4)]">
      <span className="text-[11px] font-bold text-ink-holder">{label}</span>
      {series.map((s) => {
        const value = payload.find((p) => p.dataKey === s.key)?.value ?? 0;
        return (
          <span key={s.key} className="inline-flex items-center gap-2 text-xs">
            <span className="w-3 h-0.5 rounded-full" style={{ background: s.color }} />
            <strong className="text-ink-title tabular-nums">{value}</strong>
            <span className="text-ink-holder">{s.label}</span>
          </span>
        );
      })}
    </div>
  );
};

// Two counts per week as grouped bars, with its numbers as a table for
// screen readers (R9.2).
const WeeklyChart = ({ title, rows, series, note }) => {
  const theme = chartTheme();
  return (
    <Card className="flex flex-col gap-4">
      <figure className="m-0 flex flex-col gap-4">
        <figcaption className="flex flex-wrap items-baseline justify-between gap-3">
          <span className="text-sm font-extrabold text-ink-title">{title}</span>
          <Legend series={series} />
        </figcaption>
        <div className="h-60" aria-hidden="true">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={rows} barGap={2} barCategoryGap="20%" margin={{ top: 4, right: 4, bottom: 0, left: -12 }} accessibilityLayer={false}>
              <CartesianGrid vertical={false} stroke={theme.grid} />
              <XAxis dataKey="label" tickLine={false} axisLine={{ stroke: theme.grid }} tick={{ fill: theme.axis, fontSize: 11 }} interval="preserveStartEnd" minTickGap={16} />
              {/* An empty range still shows its axis, every week at zero (R7.6). */}
              <YAxis allowDecimals={false} domain={[0, (max) => Math.max(max, 4)]} tickLine={false} axisLine={false} tick={{ fill: theme.axis, fontSize: 11 }} />
              <Tooltip cursor={{ fill: theme.cursor }} content={<WeekTooltip series={series} />} />
              {series.map((s) => (
                <Bar key={s.key} dataKey={s.key} name={s.label} fill={s.color} maxBarSize={24} radius={[4, 4, 0, 0]} isAnimationActive={false} />
              ))}
            </BarChart>
          </ResponsiveContainer>
        </div>
        <table className="sr-only">
          <caption>{title}</caption>
          <thead>
            <tr><th scope="col">Week</th>{series.map((s) => <th key={s.key} scope="col">{s.label}</th>)}</tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.week_start}><th scope="row">{r.label}</th>{series.map((s) => <td key={s.key}>{r[s.key]}</td>)}</tr>
            ))}
          </tbody>
        </table>
      </figure>
      {note && <p className="m-0 text-xs text-ink-holder">{note}</p>}
    </Card>
  );
};

// A short labelled list, each count with a bar in proportion. The numbers
// are printed, so it needs no table.
const BarList = ({ title, labels, counts }) => {
  const rows = Object.entries(labels).map(([key, label]) => ({ key, label, count: counts?.[key] ?? 0 }));
  const max = Math.max(1, ...rows.map((r) => r.count));
  return (
    <Card className="flex flex-col gap-3">
      <span className="text-sm font-extrabold text-ink-title">{title}</span>
      <ul className="m-0 p-0 list-none flex flex-col gap-2.5">
        {rows.map((r) => (
          <li key={r.key} className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-1">
            <span className="text-[13px] text-ink-body truncate">{r.label}</span>
            <span className="text-[13px] font-bold text-ink-title tabular-nums">{r.count}</span>
            <span className="col-span-2 h-2 rounded-full bg-surface-2 overflow-hidden">
              {r.count > 0 && <span className="block h-full rounded-full bg-series-1" style={{ width: `${(r.count / max) * 100}%` }} />}
            </span>
          </li>
        ))}
      </ul>
    </Card>
  );
};

const StatsTab = ({ params, update }) => {
  const weeks = RANGES.includes(Number(params.get('weeks'))) ? Number(params.get('weeks')) : 12;
  const { data, isLoading, error, refetch } = useModerationStats(weeks);
  const theme = chartTheme();

  const rows = (data?.weeks ?? []).map((w, i, all) => ({ ...w, label: weekLabel(w.week_start, i === all.length - 1) }));
  const reports = [
    { key: 'received', label: 'Received', color: theme.series1 },
    { key: 'resolved', label: 'Resolved', color: theme.series2 },
  ];
  const suspensions = [
    { key: 'suspended', label: 'Suspended', color: theme.series1 },
    { key: 'reinstated', label: 'Reinstated', color: theme.series2 },
  ];
  const undated = data?.undated_suspensions ?? 0;

  return (
    <>
      <div className="flex flex-wrap gap-2" role="group" aria-label="Range">
        {RANGES.map((n) => (
          <Chip key={n} selected={weeks === n} onClick={() => update({ weeks: n === 12 ? null : String(n) })}>
            {n} weeks
          </Chip>
        ))}
      </div>

      {isLoading && <Loading />}
      {error && <Banner message={`Couldn't load the stats: ${error.message}`} onRetry={refetch} />}
      {data && (
        <>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            <Stat label="Reports received" value={data.totals.received} sub={`In the last ${weeks} weeks`} />
            <Stat
              label="Open now"
              value={data.open_now}
              sub={data.oldest_open_at ? `Oldest waiting ${ageOf(data.oldest_open_at)}` : 'Nothing waiting'}
            />
            <Stat
              label="Median time to resolve"
              value={formatHours(data.resolve_hours.median)}
              sub={data.resolve_hours.p90 == null ? 'Nothing closed in this range' : `90% within ${formatHours(data.resolve_hours.p90)}`}
            />
            <Stat label="Suspensions" value={data.totals.suspended} sub={`${data.totals.reinstated} reinstated`} />
          </div>

          <WeeklyChart title="Reports received and resolved, per week" rows={rows} series={reports} />
          <WeeklyChart
            title="Suspensions and reinstatements, per week"
            rows={rows}
            series={suspensions}
            note={undated > 0
              ? `${undated} suspension${undated === 1 ? ' is' : 's are'} from before history began, so ${undated === 1 ? 'it isn\'t' : 'they aren\'t'} shown.`
              : null}
          />

          <div className="grid grid-cols-1 lg:grid-cols-[3fr_2fr] gap-4">
            <BarList title="What reports came to" labels={ACTION_LABELS} counts={data.actions} />
            <BarList title="Why readers reported" labels={REASON_LABELS} counts={data.reasons} />
          </div>
        </>
      )}
    </>
  );
};

export default StatsTab;
