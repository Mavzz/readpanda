import { Link } from 'react-router-dom';
import { Button, Tag } from '../../components/ui';
import { useSuspendReader } from '../../components/useSuspendReader';
import TargetsTab, { Counts, Reasons, When } from './targets';
import { queueLink, useOpenRow } from './useOpenRow';

// Reports grouped by the reader they're about: their profile, their
// comments and the rooms they run (portal-moderation R4).

const COLUMNS = ['Reader', 'Reports', 'On', 'Top reasons', 'Last reported', ''];
const EMPTY = { open: 'No open reports about anyone', all: 'Nobody has been reported' };

// Profile · comments · rooms, leaving out the zeros.
const Split = ({ byType }) => {
  const parts = [['Profile', byType.user], ['Comments', byType.comment], ['Rooms', byType.room]].filter(([, n]) => n > 0);
  return (
    <span className="text-[13px] text-ink-body whitespace-nowrap">
      {parts.map(([label, n]) => `${label} ${n}`).join(' · ')}
    </span>
  );
};

const ReadersTab = ({ params, update }) => {
  const { suspendReader, dialog } = useSuspendReader();
  const openRow = useOpenRow(update, 'about_user');

  const renderRow = (t) => {
    const { user } = t;
    return (
      <tr key={user.id} className="hairline-b last:shadow-none align-top cursor-pointer hover:bg-surface-2" onClick={openRow(user.id)}>
        <td className="px-4 py-3 min-w-[180px]">
          <div className="flex flex-col gap-1.5">
            <Link to={queueLink('about_user', user.id)} className="text-link font-bold hover:underline">{user.username}</Link>
            <span className="inline-flex flex-wrap items-center gap-1.5">
              {user.suspended && <Tag muted>Suspended</Tag>}
              {user.suspension_count > 0 && (
                <span className="text-xs text-ink-holder">Suspended {user.suspension_count} time{user.suspension_count === 1 ? '' : 's'}</span>
              )}
            </span>
          </div>
        </td>
        <td className="px-4 py-3"><Counts total={t.total} open={t.open} /></td>
        <td className="px-4 py-3"><Split byType={t.by_type} /></td>
        <td className="px-4 py-3"><Reasons reasons={t.top_reasons} /></td>
        <td className="px-4 py-3"><When iso={t.last_reported_at} /></td>
        <td className="px-4 py-3">
          <div className="flex flex-col items-start gap-2">
            <Button
              variant="secondary"
              className="h-9 px-4 text-[13px]"
              onClick={() => suspendReader({ username: user.username, suspending: !user.suspended, uuid: user.id })}
            >
              {user.suspended ? 'Reinstate' : 'Suspend'}
            </Button>
            <Link to={`/users/${user.id}`} className="text-xs font-bold text-ink-holder hover:text-ink-body">Profile</Link>
          </div>
        </td>
      </tr>
    );
  };

  return (
    <>
      {dialog}
      <TargetsTab type="user" params={params} update={update} columns={COLUMNS} empty={EMPTY} renderRow={renderRow} />
    </>
  );
};

export default ReadersTab;
