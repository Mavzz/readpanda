import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

vi.mock('../../services/moderation', async (importOriginal) => ({
  ...(await importOriginal()),
  useModerationStats: vi.fn(),
}));

import { useModerationStats } from '../../services/moderation';
import { formatHours } from './formatHours';
import ReportsPage from '../reportsPage';

// Recharts' ResponsiveContainer measures with ResizeObserver, which jsdom lacks.
beforeAll(() => {
  globalThis.ResizeObserver ??= class { observe() {} unobserve() {} disconnect() {} };
});

const zeroWeeks = (n) => Array.from({ length: n }, (_, i) => ({
  week_start: new Date(Date.UTC(2026, 7, 24 + i * 7)).toISOString().slice(0, 10),
  received: 0, resolved: 0, suspended: 0, reinstated: 0,
}));
const allZero = (keys) => Object.fromEntries(keys.map((k) => [k, 0]));
const ACTIONS = ['remove_comment', 'close_room', 'delete_room', 'suspend_user', 'dismiss'];
const REASONS = ['harassment', 'hate', 'sexual', 'violence', 'self_harm', 'spam', 'other'];

const empty = {
  range_start: '2026-08-24', weeks: zeroWeeks(12),
  totals: { received: 0, resolved: 0, suspended: 0, reinstated: 0 },
  open_now: 0, oldest_open_at: null,
  actions: allZero(ACTIONS), reasons: allZero(REASONS),
  resolve_hours: { median: null, p90: null }, undated_suspensions: 0,
};
const busy = {
  ...empty,
  weeks: zeroWeeks(12).map((w, i) => (i === 10 ? { ...w, received: 12, resolved: 4, suspended: 1 } : w)),
  totals: { received: 74, resolved: 61, suspended: 6, reinstated: 2 },
  open_now: 13, oldest_open_at: new Date(Date.now() - 3 * 24 * 36e5).toISOString(),
  actions: { ...allZero(ACTIONS), dismiss: 30, remove_comment: 20 },
  reasons: { ...allZero(REASONS), spam: 30, harassment: 18 },
  resolve_hours: { median: 18, p90: 80 }, undated_suspensions: 1,
};

const Where = () => {
  const loc = useLocation();
  return <output data-testid="where">{loc.pathname + loc.search}</output>;
};

const renderTab = (result, entry = '/reports?tab=stats') => {
  useModerationStats.mockReturnValue(result);
  render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter initialEntries={[entry]}><ReportsPage /><Where /></MemoryRouter>
    </QueryClientProvider>,
  );
  return userEvent.setup();
};

const ok = (data) => ({ data, isLoading: false, error: null, refetch: vi.fn() });
const tile = (label) => within(screen.getByText(label).parentElement);

beforeEach(() => vi.clearAllMocks());

describe('Stats tab (portal-moderation R7)', () => {
  it('defaults to 12 weeks and switches range in the URL (R7.1)', async () => {
    const user = renderTab(ok(busy));
    expect(useModerationStats).toHaveBeenLastCalledWith(12);
    expect(screen.getByRole('button', { name: '12 weeks' })).toHaveAttribute('aria-pressed', 'true');
    await user.click(screen.getByRole('button', { name: '4 weeks' }));
    expect(useModerationStats).toHaveBeenLastCalledWith(4);
    expect(screen.getByTestId('where').textContent).toBe('/reports?tab=stats&weeks=4');
  });

  it('shows the headline numbers (R7.2, R7.8)', () => {
    renderTab(ok(busy));
    expect(tile('Reports received').getByText('74')).toBeInTheDocument();
    expect(tile('Open now').getByText('13')).toBeInTheDocument();
    expect(tile('Open now').getByText('Oldest waiting 3 days')).toBeInTheDocument();
    expect(tile('Median time to resolve').getByText('18 h')).toBeInTheDocument();
    expect(tile('Median time to resolve').getByText('90% within 3.3 days')).toBeInTheDocument();
    expect(tile('Suspensions').getByText('6')).toBeInTheDocument();
    expect(tile('Suspensions').getByText('2 reinstated')).toBeInTheDocument();
  });

  it('gives each chart a table of its numbers, with this week last (R7.3, R7.4, R7.7, R9.2)', () => {
    renderTab(ok(busy));
    const reports = screen.getByRole('table', { name: 'Reports received and resolved, per week' });
    const rows = within(reports).getAllByRole('row');
    expect(rows).toHaveLength(13);
    expect(within(rows[0]).getAllByRole('columnheader').map((c) => c.textContent)).toEqual(['Week', 'Received', 'Resolved']);
    expect(within(rows[11]).getAllByRole('cell').map((c) => c.textContent)).toEqual(['12', '4']);
    expect(within(rows[12]).getByRole('rowheader')).toHaveTextContent('This week');

    const sus = screen.getByRole('table', { name: 'Suspensions and reinstatements, per week' });
    expect(within(within(sus).getAllByRole('row')[11]).getAllByRole('cell').map((c) => c.textContent)).toEqual(['1', '0']);
    expect(screen.getByText('1 suspension is from before history began, so it isn\'t shown.')).toBeInTheDocument();
  });

  it('names each series in a legend, not by colour alone (R9.3)', () => {
    renderTab(ok(busy));
    for (const name of ['Received', 'Resolved', 'Suspended', 'Reinstated']) {
      expect(screen.getAllByText(name).some((el) => el.closest('li'))).toBe(true);
    }
  });

  it('lists every action and reason with its count (R7.5)', () => {
    renderTab(ok(busy));
    const actions = within(screen.getByText('What reports came to').parentElement);
    expect(actions.getAllByRole('listitem')).toHaveLength(5);
    expect(within(actions.getByText('Dismissed').closest('li')).getByText('30')).toBeInTheDocument();
    expect(within(actions.getByText('Closed the room').closest('li')).getByText('0')).toBeInTheDocument();
    const reasons = within(screen.getByText('Why readers reported').parentElement);
    expect(reasons.getAllByRole('listitem')).toHaveLength(7);
    expect(within(reasons.getByText('Spam or scam').closest('li')).getByText('30')).toBeInTheDocument();
  });

  it('shows every week at zero, and dashes with nothing closed (R7.6, R7.8)', () => {
    renderTab(ok(empty));
    const rows = within(screen.getByRole('table', { name: 'Reports received and resolved, per week' })).getAllByRole('row').slice(1);
    expect(rows).toHaveLength(12);
    rows.forEach((r) => expect(within(r).getAllByRole('cell').map((c) => c.textContent)).toEqual(['0', '0']));
    expect(tile('Median time to resolve').getByText('–')).toBeInTheDocument();
    expect(tile('Median time to resolve').getByText('Nothing closed in this range')).toBeInTheDocument();
    expect(tile('Open now').getByText('Nothing waiting')).toBeInTheDocument();
    expect(screen.queryByText(/before history began/)).toBeNull();
  });

  it('offers to try again when it fails (R7.9)', async () => {
    const refetch = vi.fn();
    const user = renderTab({ data: undefined, isLoading: false, error: new Error('Admin access required'), refetch });
    expect(screen.getByRole('alert')).toHaveTextContent("Couldn't load the stats: Admin access required");
    await user.click(screen.getByRole('button', { name: 'Try again' }));
    expect(refetch).toHaveBeenCalled();
  });
});

describe('formatHours', () => {
  it('says hours the way a person would', () => {
    expect(formatHours(null)).toBe('–');
    expect(formatHours(0.25)).toBe('15 min');
    expect(formatHours(0.001)).toBe('1 min');
    expect(formatHours(4)).toBe('4 h');
    expect(formatHours(18.25)).toBe('18.3 h');
    expect(formatHours(80)).toBe('3.3 days');
  });
});
