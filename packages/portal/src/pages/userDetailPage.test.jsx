import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

vi.mock('../components/useDeleteRow', () => ({ useDeleteRow: () => vi.fn() }));
vi.mock('../services/api', async (importOriginal) => {
  const real = await importOriginal();
  return { ...real, api: { ...real.api, get: vi.fn() } };
});
vi.mock('../services/moderation', async (importOriginal) => ({
  ...(await importOriginal()),
  useUserSuspensions: vi.fn(),
}));

import { api } from '../services/api';
import { useUserSuspensions } from '../services/moderation';
import UserDetailPage from './userDetailPage';

const UUID = '11111111-0000-0000-0000-000000000000';

const renderPage = (suspensions) => {
  api.get.mockResolvedValue({ user: { uuid: UUID, username: 'spam.king', email: 's@x', isactive: 'false' }, related: [] });
  useUserSuspensions.mockReturnValue({ data: { count: suspensions.length, suspensions }, isLoading: false, error: null });
  render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter initialEntries={[`/users/${UUID}`]}>
        <Routes><Route path="/users/:uuid" element={<UserDetailPage />} /></Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => vi.clearAllMocks());

describe('UserDetailPage suspension history (portal-moderation R3.1)', () => {
  it('shows how many times and each suspension, newest first', async () => {
    renderPage([
      { id: 's3', suspended_at: '2026-11-13T10:00:00Z', suspended_by: { id: 'a', username: 'admin' }, note: 'Spam rooms', ended: false, report_id: 'rp_9', source: 'report' },
      { id: 's2', suspended_at: '2026-10-01T10:00:00Z', suspended_by: { id: 'p', username: 'mod.priya' }, ended: true,
        reinstated_at: '2026-10-03T10:00:00Z', reinstated_by: { id: 'a', username: 'admin' }, reinstate_note: 'Appealed', source: 'users' },
      { id: 's1', suspended_at: null, suspended_by: null, ended: true, reinstated_at: null, reinstated_by: null, source: 'backfill' },
    ]);

    expect(await screen.findByText('Suspended 3 times')).toBeInTheDocument();
    const items = screen.getByText('Suspended 3 times').closest('section').querySelectorAll('.hairline-b');
    expect(items).toHaveLength(3);

    const [open, past, undated] = [...items].map((el) => within(el));
    expect(open.getByText('Still suspended')).toBeInTheDocument();
    expect(open.getByText('by admin')).toBeInTheDocument();
    expect(open.getByText('“Spam rooms”')).toBeInTheDocument();
    expect(open.getByRole('link', { name: 'From a report' })).toHaveAttribute('href', `/reports?tab=queue&status=all&about_user=${UUID}`);

    expect(past.getByText('by mod.priya')).toBeInTheDocument();
    expect(past.getByText(/^Reinstated /)).toBeInTheDocument();
    expect(past.getByText('“Appealed”')).toBeInTheDocument();
    expect(past.queryByText('Still suspended')).toBeNull();

    expect(undated.getByText('Suspended before history began')).toBeInTheDocument();
    expect(undated.getByText('Reinstated, date unknown')).toBeInTheDocument();
  });

  it('says never, and links to the reports about them', async () => {
    renderPage([]);
    expect(await screen.findByText('Never suspended')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'See reports about them' })).toHaveAttribute('href', `/reports?tab=queue&status=all&about_user=${UUID}`);
  });
});
