import { useId, useRef } from 'react';
import { useSearchParams } from 'react-router-dom';
import { PageHeader } from '../components/ui';
import QueueTab from './moderation/QueueTab';
import ReadersTab from './moderation/ReadersTab';
import RoomsTab from './moderation/RoomsTab';
import SuspendedTab from './moderation/SuspendedTab';
import StatsTab from './moderation/StatsTab';

// Moderation, in tabs (portal-moderation R8.1): today's queue first, then
// the overviews. Which tab, and each tab's filters, live in the URL so links
// and the back button work (R6.2).

const TABS = [
  { id: 'queue', label: 'Queue', subtitle: 'What readers reported, newest first. Acting on one closes every open report on the same thing.' },
  { id: 'readers', label: 'Readers', subtitle: 'Reports grouped by the reader they\'re about: their profile, comments and rooms.' },
  { id: 'rooms', label: 'Rooms', subtitle: 'Reports grouped by room.' },
  { id: 'suspended', label: 'Suspended', subtitle: 'Who is suspended now, and who has been before.' },
  { id: 'stats', label: 'Stats', subtitle: 'How much is reported, and how fast it gets dealt with.' },
];

const PANELS = { queue: QueueTab, readers: ReadersTab, rooms: RoomsTab, suspended: SuspendedTab, stats: StatsTab };

// A tab list the keyboard can drive (R9.1): arrows, Home and End move
// between tabs and open them; only the current tab is in the Tab order.
const TabList = ({ current, onSelect, idFor }) => {
  const refs = useRef({});
  const onKeyDown = (e) => {
    const i = TABS.findIndex((t) => t.id === current);
    const next = {
      ArrowRight: (i + 1) % TABS.length,
      ArrowLeft: (i - 1 + TABS.length) % TABS.length,
      Home: 0,
      End: TABS.length - 1,
    }[e.key];
    if (next === undefined) return;
    e.preventDefault();
    onSelect(TABS[next].id);
    refs.current[TABS[next].id]?.focus();
  };
  return (
    <div role="tablist" aria-label="Moderation" className="flex gap-1 overflow-x-auto bg-well rounded-full p-1 self-start max-w-full" onKeyDown={onKeyDown}>
      {TABS.map((t) => {
        const selected = t.id === current;
        return (
          <button
            key={t.id}
            ref={(el) => { refs.current[t.id] = el; }}
            type="button"
            role="tab"
            id={idFor(`tab-${t.id}`)}
            aria-selected={selected}
            aria-controls={idFor('panel')}
            tabIndex={selected ? 0 : -1}
            onClick={() => onSelect(t.id)}
            className={`h-9 px-4 shrink-0 rounded-full text-sm ${selected ? 'bg-surface-2 text-ink-title font-extrabold' : 'text-ink-body font-bold hover:bg-surface-1'}`}
          >
            {t.label}
          </button>
        );
      })}
    </div>
  );
};

const ReportsPage = () => {
  const [params, setParams] = useSearchParams();
  const baseId = useId();
  const idFor = (part) => `${baseId}-${part}`;
  const current = TABS.some((t) => t.id === params.get('tab')) ? params.get('tab') : 'queue';
  const tab = TABS.find((t) => t.id === current);
  const Panel = PANELS[current];

  // A new tab is a new history entry, with that tab's defaults.
  const selectTab = (id) => {
    if (id !== current) setParams({ tab: id });
  };
  // A filter changes the entry in place, unless push is set (a row opening
  // the filtered queue, so back returns to the list). null removes a key.
  const update = (changes, { push = false } = {}) => {
    setParams((prev) => {
      const next = new URLSearchParams(prev);
      next.set('tab', changes.tab ?? current);
      Object.entries(changes).forEach(([k, v]) => (v == null ? next.delete(k) : next.set(k, v)));
      return next;
    }, { replace: !push });
  };

  return (
    <>
      <PageHeader title="Reports" subtitle={tab.subtitle} />
      <TabList current={current} onSelect={selectTab} idFor={idFor} />
      <div role="tabpanel" id={idFor('panel')} aria-labelledby={idFor(`tab-${current}`)} className="flex flex-col gap-6">
        <Panel key={current} params={params} update={update} />
      </div>
    </>
  );
};

export default ReportsPage;
