import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

vi.mock('../../services/moderation', async (importOriginal) => ({
  ...(await importOriginal()),
  useReportTargets: vi.fn(),
  useReports: vi.fn(() => ({ data: { reports: [], open_count: 0 }, isLoading: false, error: null })),
}));

import { useReportTargets } from '../../services/moderation';
import ReportsPage from '../reportsPage';

const night = {
  room: { id: 'rm_night', name: 'Night Owls', state: 'open' },
  admin: { id: 'u-nina', username: 'nightowl.nina', suspended: false },
  total: 2, open: 1, top_reasons: [{ reason: 'sexual', count: 2 }], last_reported_at: '2026-11-14T10:00:00Z',
};
const gone = {
  room: { id: 'rm_gone', name: 'Buy Followers Here', state: 'gone' },
  admin: { id: 'u-king', username: 'spam.king', suspended: true },
  total: 9, open: 1, top_reasons: [{ reason: 'spam', count: 9 }], last_reported_at: '2026-11-10T10:00:00Z',
};
const orphan = { ...night, room: { id: 'rm_x', name: 'Quiet', state: 'invite_only' }, admin: null, open: 0 };

const Where = () => {
  const loc = useLocation();
  return <output data-testid="where">{loc.pathname + loc.search}</output>;
};

const renderTab = (targets, entry = '/reports?tab=rooms') => {
  useReportTargets.mockReturnValue({ data: { targets }, isLoading: false, error: null, hasNextPage: false });
  render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter initialEntries={[entry]}><ReportsPage /><Where /></MemoryRouter>
    </QueryClientProvider>,
  );
  return userEvent.setup();
};

const rowOf = (name) => within(screen.getByRole('link', { name }).closest('tr'));

beforeEach(() => vi.clearAllMocks());

describe('Rooms tab (portal-moderation R5)', () => {
  it('shows each room with its state, admin, counts and reasons (R5.1)', () => {
    renderTab([night, gone, orphan]);
    expect(useReportTargets).toHaveBeenLastCalledWith('room', 'open', 'open');

    const n = rowOf('Night Owls');
    expect(n.getByText('Open')).toBeInTheDocument();
    expect(n.getByRole('link', { name: 'nightowl.nina' })).toHaveAttribute('href', '/users/u-nina');
    expect(n.getByText('2 reports')).toBeInTheDocument();
    expect(n.getByText('Sexual content')).toHaveTextContent('Sexual content · 2');

    const g = rowOf('Buy Followers Here');
    expect(g.getByText('Deleted room')).toBeInTheDocument();
    expect(g.getByText('Suspended')).toBeInTheDocument();

    expect(rowOf('Quiet').getByText('a deleted account')).toBeInTheDocument();
    expect(rowOf('Quiet').getByText('Invite only')).toBeInTheDocument();
  });

  it('has no close or delete buttons on a row (R5.4)', () => {
    renderTab([night, gone]);
    const table = screen.getByRole('table');
    expect(within(table).queryAllByRole('button')).toHaveLength(0);
  });

  it('uses the same chips and sort as Readers (R5.2)', async () => {
    const user = renderTab([night]);
    await user.click(screen.getByRole('button', { name: 'All' }));
    await user.selectOptions(screen.getByLabelText('Sort'), 'Latest');
    expect(useReportTargets).toHaveBeenLastCalledWith('room', 'all', 'latest');
  });

  it('opens the queue filtered to the room (R5.3)', async () => {
    const user = renderTab([gone]);
    expect(screen.getByRole('link', { name: 'Buy Followers Here' })).toHaveAttribute('href', '/reports?tab=queue&status=all&about_room=rm_gone');
    await user.click(rowOf('Buy Followers Here').getByText('9 reports'));
    expect(screen.getByTestId('where').textContent).toBe('/reports?tab=queue&status=all&about_room=rm_gone');
  });

  it('says when no room has open reports', () => {
    renderTab([]);
    expect(screen.getByText('No open reports about any room')).toBeInTheDocument();
  });
});
