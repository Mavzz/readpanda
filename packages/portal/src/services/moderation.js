import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { api } from './api';

// Reports and suspensions (report-block R9), through /admin/reports and
// /admin/users in api-go, plus the overview tabs (portal-moderation). Admins
// only.

export const REPORTS_KEY = ['admin-reports'];
export const SUSPENSIONS_KEY = ['admin-suspensions'];
export const USER_SUSPENSIONS_KEY = ['admin-user-suspensions'];
export const STATS_KEY = ['admin-moderation-stats'];

// Everything a suspend, reinstate or resolve can change.
export const MODERATION_KEYS = [REPORTS_KEY, SUSPENSIONS_KEY, USER_SUSPENSIONS_KEY, STATS_KEY, ['admin-user'], ['admin-rows']];

export const NOTE_MAX = 500;
const PAGE = 50;

// The reasons the app offers (R1.2), as readers saw them.
export const REASON_LABELS = {
  harassment: 'Harassment or bullying',
  hate: 'Hate speech',
  sexual: 'Sexual content',
  violence: 'Violence or threats',
  self_harm: 'Self-harm',
  spam: 'Spam or scam',
  other: 'Something else',
};

export const ACTION_LABELS = {
  remove_comment: 'Removed the comment',
  close_room: 'Closed the room',
  delete_room: 'Deleted the room',
  suspend_user: 'Suspended the reader',
  dismiss: 'Dismissed',
};

// GET /admin/reports?status=&about_user=|about_room= → { reports, has_more, open_count }
// about is { user } or { room } to see the reports about one of them (R6.1).
export const useReports = (status, about = {}) =>
  useQuery({
    queryKey: [...REPORTS_KEY, status, about.user ?? null, about.room ?? null],
    queryFn: () => {
      const params = new URLSearchParams({ status, limit: '100' });
      if (about.user) params.set('about_user', about.user);
      if (about.room) params.set('about_room', about.room);
      return api.get(`/admin/reports?${params}`);
    },
  });

// A list that loads PAGE rows at a time, with "Load more" (R2.6, R4.7).
const usePaged = (key, path, listKey) =>
  useInfiniteQuery({
    queryKey: key,
    initialPageParam: 0,
    queryFn: ({ pageParam }) => api.get(`${path}${path.includes('?') ? '&' : '?'}limit=${PAGE}&offset=${pageParam}`),
    getNextPageParam: (last, pages) => (last.has_more ? pages.length * PAGE : undefined),
    select: (data) => ({ ...data.pages.at(-1), [listKey]: data.pages.flatMap((p) => p[listKey]) }),
  });

// GET /admin/reports/targets → one row per reader or room (R4.1, R5.1).
export const useReportTargets = (type, status, sort) =>
  usePaged([...REPORTS_KEY, 'targets', type, status, sort], `/admin/reports/targets?type=${type}&status=${status}&sort=${sort}`, 'targets');

// GET /admin/suspensions?state=current|previous (R2.1, R2.2).
export const useSuspensions = (state) =>
  usePaged([...SUSPENSIONS_KEY, state], `/admin/suspensions?state=${state}`, 'readers');

// GET /admin/users/{uuid}/suspensions → { count, suspensions } (R3.1).
export const useUserSuspensions = (uuid) =>
  useQuery({
    queryKey: [...USER_SUSPENSIONS_KEY, uuid],
    queryFn: () => api.get(`/admin/users/${uuid}/suspensions`),
  });

// GET /admin/moderation/stats?weeks= (R7).
export const useModerationStats = (weeks) =>
  useQuery({
    queryKey: [...STATS_KEY, weeks],
    queryFn: () => api.get(`/admin/moderation/stats?weeks=${weeks}`),
  });

// The nav badge. Quiet on failure: a writer without the admin role gets 403.
export const useOpenReportCount = () =>
  useQuery({
    queryKey: [...REPORTS_KEY, 'count'],
    queryFn: async () => (await api.get('/admin/reports?status=open&limit=1')).open_count ?? 0,
    retry: false,
    refetchInterval: 60 * 1000,
  });

// The note goes with a suspension only, and only when there is one.
const withNote = (body, note) => (note?.trim() ? { ...body, note: note.trim() } : body);

export const resolveReport = (id, action, note) => api.post(`/admin/reports/${id}/resolve`, withNote({ action }, note));

export const setUserActive = (uuid, isactive, note) => api.patch(`/admin/users/${uuid}`, withNote({ isactive }, note));

// The admin table routes send every column as text, so isactive arrives as
// "false"; PATCH answers a real boolean. The reserved Deleted reader is
// inactive by design, not suspended.
export const isSuspended = (user) =>
  (user?.isactive === false || user?.isactive === 'false') && user?.username !== 'deleted.reader';
