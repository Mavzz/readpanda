import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { Banner, Button, Chip, EmptyState, Eyebrow, IconButton, Loading, Tag } from '../../components/ui';
import { CloseIcon } from '../../components/icons';
import { useConfirm } from '../../components/useConfirm';
import { useSuspendReader } from '../../components/useSuspendReader';
import { ACTION_LABELS, REASON_LABELS, REPORTS_KEY, resolveReport, useReports } from '../../services/moderation';

// Readers' reports and what was done about them (report-block R9): the
// Reports page's Queue tab, which can be narrowed to the reports about one
// reader or room (portal-moderation R6.1).

const STATUSES = [
  { status: 'open', label: 'Open' },
  { status: 'actioned', label: 'Actioned' },
  { status: 'dismissed', label: 'Dismissed' },
];

const TARGET_LABELS = { comment: 'Comment', user: 'Reader', room: 'Room' };

const COMMENT_STATES = {
  live: null,
  deleted: 'Deleted by its author',
  removed: 'Removed by ReadPanda',
  gone: 'No longer there',
};

const ROOM_STATES = { open: 'Open', invite_only: 'Invite only', gone: 'Deleted' };

const when = (iso) => (iso ? new Date(iso).toLocaleString() : '');

// What an admin can do, for this report as it stands now.
const actionsFor = (report) => {
  const actions = [];
  if (report.target_type === 'comment' && report.comment_state !== 'gone' && report.comment_state !== 'removed') {
    actions.push({ action: 'remove_comment', label: 'Remove comment', confirm: 'Everyone will see "Removed by ReadPanda" in its place. Replies stay. The author is told why.' });
  }
  if (report.target_type === 'room' && report.room?.state === 'open') {
    actions.push({ action: 'close_room', label: 'Close room', confirm: 'It becomes Invite only and leaves the open-room lists. Members stay. The creator is told why.' });
  }
  if (report.target_type === 'room' && report.room && report.room.state !== 'gone') {
    actions.push({ action: 'delete_room', label: 'Delete room', confirm: 'The room, its comments and its memberships are deleted. This can\'t be undone. The creator is told why, and the members that it\'s gone.' });
  }
  if (report.target_user && !report.target_user.suspended) {
    actions.push({
      action: 'suspend_user',
      label: `Suspend ${report.target_user.username}`,
      confirm: `${report.target_user.username} is signed out and can't sign in or post until you reinstate them on the Users page.`,
    });
  }
  actions.push({ action: 'dismiss', label: 'Dismiss', confirm: 'The report closes with nothing done.' });
  return actions;
};

const Person = ({ person, fallback }) => {
  if (!person) return <span className="text-ink-holder italic">{fallback}</span>;
  return (
    <span className="inline-flex items-center gap-1.5 flex-wrap">
      <Link to={`/users/${person.id}`} className="text-link font-bold hover:underline">{person.username}</Link>
      {person.suspended && <Tag muted>Suspended</Tag>}
    </span>
  );
};

const ReportRow = ({ report, onAct, busy }) => {
  const commentState = report.target_type === 'comment' ? COMMENT_STATES[report.comment_state] : null;
  return (
    <tr className="hairline-b last:shadow-none align-top">
      <td className="px-4 py-3 min-w-[260px]">
        <div className="flex flex-col gap-2">
          <span className="inline-flex items-center gap-1.5">
            <Eyebrow>{TARGET_LABELS[report.target_type]}</Eyebrow>
            {commentState && <Tag muted>{commentState}</Tag>}
          </span>
          {/* The stored copy, as text: nothing a reader wrote renders as HTML. */}
          <pre className="m-0 px-3 py-2.5 rounded-field bg-surface-2 text-[12px] leading-normal text-ink-title font-mono whitespace-pre-wrap break-words max-h-48 overflow-y-auto">
            {report.snapshot}
          </pre>
        </div>
      </td>
      <td className="px-4 py-3 min-w-[160px]">
        <div className="flex flex-col gap-1">
          <span className="font-bold text-ink-title">{REASON_LABELS[report.reason] ?? report.reason}</span>
          {report.details && <span className="text-ink-body whitespace-pre-wrap break-words">{report.details}</span>}
        </div>
      </td>
      <td className="px-4 py-3 min-w-[150px]">
        <div className="flex flex-col gap-1.5">
          <span className="text-ink-holder text-xs">By <Person person={report.reporter} fallback="a deleted account" /></span>
          <span className="text-ink-holder text-xs">About <Person person={report.target_user} fallback="a deleted account" /></span>
        </div>
      </td>
      <td className="px-4 py-3 min-w-[140px]">
        {report.room ? (
          <div className="flex flex-col gap-1">
            <span className="text-ink-title">{report.room.name || <span className="italic text-ink-holder">Deleted room</span>}</span>
            <Tag muted>{ROOM_STATES[report.room.state]}</Tag>
          </div>
        ) : <span className="text-ink-holder">—</span>}
      </td>
      <td className="px-4 py-3 whitespace-nowrap text-ink-body text-xs">{when(report.created_at)}</td>
      <td className="px-4 py-3 min-w-[170px]">
        {report.status === 'open' ? (
          <div className="flex flex-col items-start gap-2">
            {actionsFor(report).map((a) => (
              <Button
                key={a.action}
                variant="secondary"
                className="h-9 px-4 text-[13px]"
                disabled={busy}
                onClick={() => onAct(report, a)}
              >
                {a.label}
              </Button>
            ))}
          </div>
        ) : (
          <div className="flex flex-col gap-1 text-xs">
            <span className="font-bold text-ink-title">{ACTION_LABELS[report.action] ?? report.action}</span>
            <span className="text-ink-holder">{[report.resolved_by, when(report.resolved_at)].filter(Boolean).join(' · ')}</span>
          </div>
        )}
      </td>
    </tr>
  );
};

// Who or what the filtered queue is about, from the reports themselves:
// the URL only carries the id.
const aboutName = (about, reports) => {
  if (about.user) return reports.find((r) => r.target_user?.id === about.user)?.target_user.username ?? 'this reader';
  const room = reports.find((r) => r.target_type === 'room')?.room;
  return room?.name || reports.find((r) => r.target_type === 'room')?.snapshot.split('\n')[0] || 'this room';
};

const QueueTab = ({ params, update }) => {
  const about = { user: params.get('about_user') ?? undefined, room: params.get('about_room') ?? undefined };
  const filtered = Boolean(about.user || about.room);
  const status = params.get('status') ?? 'open';
  const [banner, setBanner] = useState('');
  const [busy, setBusy] = useState(false);
  const confirm = useConfirm();
  const { suspendReader, dialog } = useSuspendReader();
  const queryClient = useQueryClient();
  const { data, isLoading, error, refetch } = useReports(status, about);

  const act = async (report, a) => {
    // Suspending takes an optional note, so it has its own dialog, which
    // makes the call and reloads (portal-moderation R3.2).
    if (a.action === 'suspend_user') {
      setBanner('');
      await suspendReader({ username: report.target_user.username, suspending: true, reportId: report.id, message: a.confirm });
      return;
    }
    const ok = await confirm({ title: a.label, message: a.confirm, action: a.label });
    if (!ok) return;
    setBusy(true);
    setBanner('');
    try {
      await resolveReport(report.id, a.action);
    } catch (err) {
      setBanner(`Couldn't do that: ${err.message}`);
    } finally {
      setBusy(false);
      // Other reports on the same thing close with it, so reload every tab.
      queryClient.invalidateQueries({ queryKey: REPORTS_KEY });
    }
  };

  const reports = data?.reports ?? [];
  // "All" is for seeing everything about one reader or room at once.
  const statuses = filtered || status === 'all' ? [...STATUSES, { status: 'all', label: 'All' }] : STATUSES;
  const name = filtered ? aboutName(about, reports) : '';
  const empty = filtered
    ? { title: `No ${status === 'all' ? '' : `${status} `}reports about ${name}` }
    : {
      title: status === 'open' ? 'No open reports' : `Nothing ${status} yet`,
      body: status === 'open' ? 'Readers report from the ⋯ menu on a comment, a member or a room. New ones show up here.' : undefined,
    };

  return (
    <>
      <div className="flex flex-wrap gap-2" role="group" aria-label="Report status">
        {statuses.map((t) => (
          <Chip key={t.status} selected={status === t.status} onClick={() => update({ status: t.status })}>
            {t.label}{t.status === 'open' && !filtered && data?.open_count ? ` · ${data.open_count}` : ''}
          </Chip>
        ))}
        {filtered && (
          <span className="h-[34px] pl-3.5 pr-0.5 inline-flex items-center gap-1.5 rounded-full bg-surface-3 text-[13px] font-bold text-ink-title">
            About {name}
            <IconButton
              label="Show every report"
              className="w-[30px] h-[30px] bg-transparent"
              onClick={() => update({ about_user: null, about_room: null, status: status === 'all' ? null : status })}
            >
              <CloseIcon width={12} height={12} strokeWidth={2.5} />
            </IconButton>
          </span>
        )}
      </div>

      <Banner message={banner} onDismiss={() => setBanner('')} />
      {dialog}

      {isLoading && <Loading />}
      {error && <Banner message={error.message} onRetry={refetch} />}
      {!isLoading && !error && reports.length === 0 && <EmptyState title={empty.title} body={empty.body} />}
      {reports.length > 0 && (
        <div className="overflow-x-auto bg-surface-1 rounded-card">
          <table className="min-w-full text-[13px] border-collapse">
            <thead>
              <tr className="hairline-b">
                {['Reported', 'Reason', 'Who', 'Room', 'When', { open: 'Act', all: 'Act or outcome' }[status] ?? 'Outcome'].map((h) => (
                  <th key={h} className="px-4 py-3 text-left whitespace-nowrap text-[11px] font-bold tracking-[1px] uppercase text-ink-meta">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {reports.map((r) => <ReportRow key={r.id} report={r} onAct={act} busy={busy} />)}
            </tbody>
          </table>
        </div>
      )}
      {data?.has_more && <p className="m-0 text-[13px] text-ink-holder">Showing the newest 100.</p>}
    </>
  );
};

export default QueueTab;
