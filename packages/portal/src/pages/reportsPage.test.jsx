import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const confirmMock = vi.fn();
vi.mock('../components/useConfirm', () => ({ useConfirm: () => confirmMock }));
vi.mock('../services/moderation', async (importOriginal) => ({
  ...(await importOriginal()),
  useReports: vi.fn(),
  resolveReport: vi.fn(),
}));

import { resolveReport, useReports } from '../services/moderation';
import ReportsPage from './reportsPage';

const base = {
  reason: 'harassment',
  details: '',
  status: 'open',
  created_at: '2026-10-22T10:00:00Z',
  reporter: { id: 'u-bob', username: 'bob', suspended: false },
  target_user: { id: 'u-alice', username: 'alice', suspended: false },
};
const commentReport = {
  ...base, id: 'rp_1', target_type: 'comment', target_id: 'cm_1',
  snapshot: '<b>not bold</b> mean words', comment_state: 'live',
  room: { id: 'rm_1', name: 'Club', state: 'invite_only' },
};
const roomReport = {
  ...base, id: 'rp_2', target_type: 'room', target_id: 'rm_2', reason: 'other', details: 'rude name',
  snapshot: 'Bad room', room: { id: 'rm_2', name: 'Bad room', state: 'open' },
};

const renderPage = (reports, extra = {}) => {
  useReports.mockReturnValue({ data: { reports, open_count: reports.length, has_more: false }, isLoading: false, error: null, ...extra });
  render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter><ReportsPage /></MemoryRouter>
    </QueryClientProvider>,
  );
  return userEvent.setup();
};

beforeEach(() => {
  vi.clearAllMocks();
  resolveReport.mockResolvedValue({});
});

describe('ReportsPage', () => {
  it('shows the stored copy as text, never as HTML (R9.1)', () => {
    renderPage([commentReport]);
    expect(screen.getByText('<b>not bold</b> mean words')).toBeInTheDocument();
    expect(document.querySelector('pre b')).toBeNull();
    expect(screen.getByText('Harassment or bullying')).toBeInTheDocument();
    expect(screen.getByText('Club')).toBeInTheDocument();
  });

  it('offers the actions that fit each report', () => {
    renderPage([commentReport, roomReport]);
    expect(screen.getByRole('button', { name: 'Remove comment' })).toBeInTheDocument();
    expect(screen.getAllByRole('button', { name: 'Suspend alice' })).toHaveLength(2);
    expect(screen.getByRole('button', { name: 'Close room' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Delete room' })).toBeInTheDocument();
    expect(screen.getAllByRole('button', { name: 'Dismiss' })).toHaveLength(2);
    expect(screen.getByText('rude name')).toBeInTheDocument();
  });

  it('asks first, then acts (R9.2)', async () => {
    const user = renderPage([commentReport]);

    confirmMock.mockResolvedValueOnce(false);
    await user.click(screen.getByRole('button', { name: 'Remove comment' }));
    expect(resolveReport).not.toHaveBeenCalled();

    confirmMock.mockResolvedValueOnce(true);
    await user.click(screen.getByRole('button', { name: 'Remove comment' }));
    expect(resolveReport).toHaveBeenCalledWith('rp_1', 'remove_comment');
  });

  it('says what went wrong', async () => {
    resolveReport.mockRejectedValue(new Error('This report is already closed'));
    confirmMock.mockResolvedValue(true);
    const user = renderPage([commentReport]);
    await user.click(screen.getByRole('button', { name: 'Dismiss' }));
    expect(await screen.findByText("Couldn't do that: This report is already closed")).toBeInTheDocument();
  });

  it('shows what was done on a closed report (R9.5)', () => {
    renderPage([{ ...commentReport, status: 'actioned', action: 'remove_comment', resolved_by: 'mod', resolved_at: '2026-10-22T11:00:00Z', comment_state: 'removed' }]);
    expect(screen.getByText('Removed the comment')).toBeInTheDocument();
    expect(screen.getByText('Removed by ReadPanda')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Dismiss' })).toBeNull();
  });
});
