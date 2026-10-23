import { useQuery } from '@tanstack/react-query';
import { api } from './api';

// Reports and suspensions (report-block R9), through /admin/reports and
// /admin/users in api-go. Admins only.

export const REPORTS_KEY = ['admin-reports'];

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

// GET /admin/reports?status= → { reports, has_more, open_count }
export const useReports = (status) =>
  useQuery({
    queryKey: [...REPORTS_KEY, status],
    queryFn: () => api.get(`/admin/reports?status=${status}&limit=100`),
  });

// The nav badge. Quiet on failure: a writer without the admin role gets 403.
export const useOpenReportCount = () =>
  useQuery({
    queryKey: [...REPORTS_KEY, 'count'],
    queryFn: async () => (await api.get('/admin/reports?status=open&limit=1')).open_count ?? 0,
    retry: false,
    refetchInterval: 60 * 1000,
  });

export const resolveReport = (id, action) => api.post(`/admin/reports/${id}/resolve`, { action });

export const setUserActive = (uuid, isactive) => api.patch(`/admin/users/${uuid}`, { isactive });

// The admin table routes send every column as text, so isactive arrives as
// "false"; PATCH answers a real boolean. The reserved Deleted reader is
// inactive by design, not suspended.
export const isSuspended = (user) =>
  (user?.isactive === false || user?.isactive === 'false') && user?.username !== 'deleted.reader';
