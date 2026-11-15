import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

vi.mock('../../services/moderation', async (importOriginal) => ({
  ...(await importOriginal()),
  useReportTargets: vi.fn(),
  useReports: vi.fn(() => ({ data: { reports: [], open_count: 0 }, isLoading: false, error: null })),
  setUserActive: vi.fn(),
}));

import { setUserActive, useReportTargets } from '../../services/moderation';
import ReportsPage from '../reportsPage';

const king = {
  user: { id: 'u-king', username: 'spam.king', suspended: true, suspension_count: 2 },
  total: 18, open: 3, by_type: { user: 5, comment: 4, room: 9 },
  top_reasons: [{ reason: 'spam', count: 14 }, { reason: 'other', count: 4 }],
  last_reported_at: '2026-11-14T10:00:00Z',
};
const nina = {
  user: { id: 'u-nina', username: 'nightowl.nina', suspended: false, suspension_count: 0 },
  total: 12, open: 4, by_type: { user: 8, comment: 4, room: 0 },
  top_reasons: [{ reason: 'harassment', count: 7 }], last_reported_at: '2026-11-13T10:00:00Z',
};

const Where = () => {
  const loc = useLocation();
  return <output data-testid="where">{loc.pathname + loc.search}</output>;
};
const where = () => screen.getByTestId('where').textContent;

const renderTab = (targets, extra = {}, entry = '/reports?tab=readers') => {
  useReportTargets.mockReturnValue({ data: { targets }, isLoading: false, error: null, hasNextPage: false, ...extra });
  render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter initialEntries={[entry]}><ReportsPage /><Where /></MemoryRouter>
    </QueryClientProvider>,
  );
  return userEvent.setup();
};

const rowOf = (name) => screen.getByRole('link', { name }).closest('tr');

beforeEach(() => {
  vi.clearAllMocks();
  setUserActive.mockResolvedValue({});
});

describe('Readers tab (portal-moderation R4)', () => {
  it('shows one row per reader with counts, split, reasons and suspensions (R4.1)', () => {
    renderTab([king, nina]);
    expect(useReportTargets).toHaveBeenLastCalledWith('user', 'open', 'open');
    const row = within(rowOf('spam.king'));
    expect(row.getByText('Suspended')).toBeInTheDocument();
    expect(row.getByText('Suspended 2 times')).toBeInTheDocument();
    expect(row.getByText('18 reports')).toBeInTheDocument();
    expect(row.getByText('3 open')).toBeInTheDocument();
    expect(row.getByText('Profile 5 · Comments 4 · Rooms 9')).toBeInTheDocument();
    expect(row.getByText('Spam or scam')).toHaveTextContent('Spam or scam · 14');
    expect(within(rowOf('nightowl.nina')).getByText('Profile 8 · Comments 4')).toBeInTheDocument();
    expect(within(rowOf('nightowl.nina')).queryByText(/Suspended/)).toBeNull();
  });

  it('switches between open and all, and sorts, in the URL (R4.2, R4.3)', async () => {
    const user = renderTab([king]);
    await user.click(screen.getByRole('button', { name: 'All' }));
    expect(useReportTargets).toHaveBeenLastCalledWith('user', 'all', 'open');
    await user.selectOptions(screen.getByLabelText('Sort'), 'Most reports');
    expect(useReportTargets).toHaveBeenLastCalledWith('user', 'all', 'total');
    expect(where()).toBe('/reports?tab=readers&filter=all&sort=total');
  });

  it('opens the queue filtered to the reader when a row is clicked (R4.4)', async () => {
    const user = renderTab([king], {}, '/reports?tab=readers&filter=all');
    expect(screen.getByRole('link', { name: 'spam.king' })).toHaveAttribute('href', '/reports?tab=queue&status=all&about_user=u-king');
    await user.click(within(rowOf('spam.king')).getByText('18 reports'));
    expect(where()).toBe('/reports?tab=queue&status=all&about_user=u-king');
    expect(screen.getByRole('tab', { name: 'Queue' })).toHaveAttribute('aria-selected', 'true');
  });

  it('suspends or reinstates from the row with the note dialog (R4.5)', async () => {
    const user = renderTab([king, nina]);
    await user.click(within(rowOf('nightowl.nina')).getByRole('button', { name: 'Suspend' }));
    expect(screen.getByRole('dialog', { name: 'Suspend nightowl.nina' })).toBeInTheDocument();
    expect(where()).toBe('/reports?tab=readers');
    await user.type(screen.getByLabelText('Note (optional)'), 'Explicit DMs');
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Suspend' }));
    expect(setUserActive).toHaveBeenCalledWith('u-nina', false, 'Explicit DMs');

    await user.click(within(rowOf('spam.king')).getByRole('button', { name: 'Reinstate' }));
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Reinstate' }));
    expect(setUserActive).toHaveBeenLastCalledWith('u-king', true, '');
  });

  it('loads more (R4.7)', async () => {
    const fetchNextPage = vi.fn();
    const user = renderTab([king], { hasNextPage: true, fetchNextPage });
    await user.click(screen.getByRole('button', { name: 'Load more' }));
    expect(fetchNextPage).toHaveBeenCalled();
  });

  it('says when there is nobody', () => {
    renderTab([]);
    expect(screen.getByText('No open reports about anyone')).toBeInTheDocument();
  });

  it('shows the API\'s refusal to a writer (R8.2)', () => {
    renderTab([], { data: undefined, error: new Error('Admin access required') });
    expect(screen.getByRole('alert')).toHaveTextContent('Admin access required');
  });
});
