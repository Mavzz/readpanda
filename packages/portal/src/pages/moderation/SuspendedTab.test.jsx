import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

vi.mock('../../services/moderation', async (importOriginal) => ({
  ...(await importOriginal()),
  useSuspensions: vi.fn(),
  setUserActive: vi.fn(),
}));

import { setUserActive, useSuspensions } from '../../services/moderation';
import ReportsPage from '../reportsPage';

const counts = { current_count: 3, previous_count: 2 };
const bookbot = {
  id: 'u-bot', username: 'bookbot.9000', suspension_count: 1,
  last: { suspended_at: '2026-11-13T12:00:00Z', suspended_by: { id: 'a', username: 'admin' }, note: 'Crypto spam room', ended: false, reinstated_at: null },
};
const king = {
  id: 'u-king', username: 'spam.king', suspension_count: 2,
  last: { suspended_at: '2026-10-25T12:00:00Z', suspended_by: { id: 'p', username: 'mod.priya' }, report_id: 'rp_4', ended: false, reinstated_at: null },
};
const mike = { id: 'u-mike', username: 'troll.mike', suspension_count: 1, last: { suspended_at: null, suspended_by: null, ended: false, reinstated_at: null } };
const ella = { id: 'u-ella', username: 'edgy.ella', suspension_count: 2, last: { suspended_at: '2026-11-01T00:00:00Z', ended: true, reinstated_at: '2026-11-11T12:00:00Z' } };
const quinn = { id: 'u-q', username: 'quiet.quinn', suspension_count: 1, last: { suspended_at: null, ended: true, reinstated_at: null } };

const Where = () => {
  const loc = useLocation();
  return <output data-testid="where">{loc.pathname + loc.search}</output>;
};

const lists = { current: [bookbot, king, mike], previous: [ella, quinn] };
const renderTab = (entry = '/reports?tab=suspended', override) => {
  useSuspensions.mockImplementation((state) => override ?? ({
    data: { readers: lists[state], ...counts }, isLoading: false, error: null, hasNextPage: false,
  }));
  render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter initialEntries={[entry]}><ReportsPage /><Where /></MemoryRouter>
    </QueryClientProvider>,
  );
  return userEvent.setup();
};

const rowOf = (name) => within(screen.getByRole('link', { name }).closest('tr'));

beforeEach(() => {
  vi.clearAllMocks();
  setUserActive.mockResolvedValue({});
});

describe('Suspended tab (portal-moderation R2)', () => {
  it('lists who is suspended now, with count, since, by, note and report (R2.1, R2.3)', () => {
    renderTab();
    expect(useSuspensions).toHaveBeenLastCalledWith('current');
    expect(screen.getByRole('button', { name: 'Suspended · 3' })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByRole('button', { name: 'Previously suspended · 2' })).toBeInTheDocument();

    const bot = rowOf('bookbot.9000');
    expect(bot.getByText('1 time')).toBeInTheDocument();
    expect(bot.getByText('admin')).toBeInTheDocument();
    expect(bot.getByText('Crypto spam room')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'bookbot.9000' })).toHaveAttribute('href', '/users/u-bot');

    const k = rowOf('spam.king');
    expect(k.getByText('2 times')).toBeInTheDocument();
    expect(k.getByRole('link', { name: 'From a report' })).toHaveAttribute('href', '/reports?tab=queue&status=all&about_user=u-king');

    expect(rowOf('troll.mike').getByText('Before history began')).toBeInTheDocument();
  });

  it('lists who has been reinstated (R2.2)', async () => {
    const user = renderTab();
    await user.click(screen.getByRole('button', { name: 'Previously suspended · 2' }));
    expect(useSuspensions).toHaveBeenLastCalledWith('previous');
    expect(screen.getByTestId('where').textContent).toBe('/reports?tab=suspended&state=previous');
    expect(rowOf('edgy.ella').getByText('2 times')).toBeInTheDocument();
    expect(rowOf('quiet.quinn').getByText('Date unknown')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Reinstate' })).toBeNull();
  });

  it('reinstates from a row (R2.4)', async () => {
    const user = renderTab();
    await user.click(rowOf('spam.king').getByRole('button', { name: 'Reinstate' }));
    const dialog = screen.getByRole('dialog', { name: 'Reinstate spam.king' });
    await user.type(within(dialog).getByLabelText('Note (optional)'), 'Appealed');
    await user.click(within(dialog).getByRole('button', { name: 'Reinstate' }));
    expect(setUserActive).toHaveBeenCalledWith('u-king', true, 'Appealed');
  });

  it('says when nobody is there (R2.5)', async () => {
    lists.current = [];
    const user = renderTab();
    expect(screen.getByText('Nobody is suspended')).toBeInTheDocument();
    lists.previous = [];
    await user.click(screen.getByRole('button', { name: /^Previously suspended/ }));
    expect(screen.getByText('Nobody has been reinstated yet')).toBeInTheDocument();
    lists.current = [bookbot, king, mike];
    lists.previous = [ella, quinn];
  });

  it('loads more (R2.6)', async () => {
    const fetchNextPage = vi.fn();
    const user = renderTab('/reports?tab=suspended', {
      data: { readers: [bookbot], ...counts }, isLoading: false, error: null, hasNextPage: true, fetchNextPage,
    });
    await user.click(screen.getByRole('button', { name: 'Load more' }));
    expect(fetchNextPage).toHaveBeenCalled();
  });

  it('shows the API\'s refusal to a writer (R8.2)', () => {
    renderTab('/reports?tab=suspended', { data: undefined, isLoading: false, error: new Error('Admin access required') });
    expect(screen.getByRole('alert')).toHaveTextContent('Admin access required');
  });
});
