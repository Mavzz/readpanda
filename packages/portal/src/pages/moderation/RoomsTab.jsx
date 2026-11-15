import { Link } from 'react-router-dom';
import { Tag } from '../../components/ui';
import TargetsTab, { Counts, Reasons, When } from './targets';
import { queueLink, useOpenRow } from './useOpenRow';

// Reports grouped by room (portal-moderation R5). No close or delete here:
// those tell the creator one report's reason, so they stay on a report in
// the Queue (R5.4).

const COLUMNS = ['Room', 'Run by', 'Reports', 'Top reasons', 'Last reported'];
const EMPTY = { open: 'No open reports about any room', all: 'No room has been reported' };
const STATES = { open: 'Open', invite_only: 'Invite only', gone: 'Deleted room' };

const RoomsTab = ({ params, update }) => {
  const openRow = useOpenRow(update, 'about_room');

  const renderRow = (t) => (
    <tr key={t.room.id} className="hairline-b last:shadow-none align-top cursor-pointer hover:bg-surface-2" onClick={openRow(t.room.id)}>
      <td className="px-4 py-3 min-w-[200px]">
        <div className="flex flex-col items-start gap-1.5">
          {/* A deleted room's name is from the stored copy, as text. */}
          <Link to={queueLink('about_room', t.room.id)} className="text-link font-bold hover:underline break-words">
            {t.room.name || 'Unnamed room'}
          </Link>
          <Tag muted>{STATES[t.room.state]}</Tag>
        </div>
      </td>
      <td className="px-4 py-3">
        {t.admin ? (
          <span className="inline-flex flex-wrap items-center gap-1.5">
            <Link to={`/users/${t.admin.id}`} className="text-link font-bold hover:underline">{t.admin.username}</Link>
            {t.admin.suspended && <Tag muted>Suspended</Tag>}
          </span>
        ) : <span className="text-ink-holder italic">a deleted account</span>}
      </td>
      <td className="px-4 py-3"><Counts total={t.total} open={t.open} /></td>
      <td className="px-4 py-3"><Reasons reasons={t.top_reasons} /></td>
      <td className="px-4 py-3"><When iso={t.last_reported_at} /></td>
    </tr>
  );

  return <TargetsTab type="room" params={params} update={update} columns={COLUMNS} empty={EMPTY} renderRow={renderRow} />;
};

export default RoomsTab;
