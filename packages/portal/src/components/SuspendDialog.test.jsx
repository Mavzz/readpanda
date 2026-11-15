import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

vi.mock('../services/moderation', async (importOriginal) => ({
  ...(await importOriginal()),
  resolveReport: vi.fn(),
  setUserActive: vi.fn(),
}));

import { resolveReport, setUserActive } from '../services/moderation';
import { useSuspendReader } from './useSuspendReader';

const outcome = vi.fn();

// A page with one button that opens the dialog for opts.
const Page = ({ opts }) => {
  const { suspendReader, dialog } = useSuspendReader();
  return (
    <>
      <button type="button" onClick={async () => outcome(await suspendReader(opts))}>Open</button>
      {dialog}
    </>
  );
};

const renderWith = (opts) => {
  const client = new QueryClient();
  const invalidate = vi.spyOn(client, 'invalidateQueries');
  render(<QueryClientProvider client={client}><Page opts={opts} /></QueryClientProvider>);
  return { user: userEvent.setup(), invalidate };
};

beforeEach(() => {
  vi.clearAllMocks();
  resolveReport.mockResolvedValue({});
  setUserActive.mockResolvedValue({});
});

describe('useSuspendReader (portal-moderation R3.2)', () => {
  it('suspends from a report with the note, and reloads every moderation list', async () => {
    const { user, invalidate } = renderWith({ username: 'alice', suspending: true, reportId: 'rp_1' });
    await user.click(screen.getByRole('button', { name: 'Open' }));

    expect(screen.getByRole('dialog', { name: 'Suspend alice' })).toBeInTheDocument();
    expect(screen.getByText('Admins only. The reader never sees this.')).toBeInTheDocument();
    expect(screen.getByText('0 of 500')).toBeInTheDocument();

    await user.type(screen.getByLabelText('Note (optional)'), 'Third spam wave');
    expect(screen.getByText('15 of 500')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Suspend' }));

    expect(resolveReport).toHaveBeenCalledWith('rp_1', 'suspend_user', 'Third spam wave');
    expect(setUserActive).not.toHaveBeenCalled();
    expect(outcome).toHaveBeenCalledWith(true);
    expect(screen.queryByRole('dialog')).toBeNull();
    const keys = invalidate.mock.calls.map(([{ queryKey }]) => queryKey[0]);
    expect(keys).toEqual(expect.arrayContaining(['admin-reports', 'admin-suspensions', 'admin-user-suspensions', 'admin-moderation-stats', 'admin-user']));
  });

  it('reinstates from the user page', async () => {
    const { user } = renderWith({ username: 'bob', suspending: false, uuid: 'u-bob' });
    await user.click(screen.getByRole('button', { name: 'Open' }));
    expect(screen.getByText('bob can sign in and post again.')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Reinstate' }));
    expect(setUserActive).toHaveBeenCalledWith('u-bob', true, '');
  });

  it('keeps the dialog and the note when it fails', async () => {
    setUserActive.mockRejectedValueOnce(new Error('Keep the note to 500 characters'));
    const { user } = renderWith({ username: 'bob', suspending: true, uuid: 'u-bob' });
    await user.click(screen.getByRole('button', { name: 'Open' }));
    await user.type(screen.getByLabelText('Note (optional)'), 'why');
    await user.click(screen.getByRole('button', { name: 'Suspend' }));

    expect(await screen.findByText("Couldn't do that: Keep the note to 500 characters")).toBeInTheDocument();
    expect(screen.getByLabelText('Note (optional)')).toHaveValue('why');
    expect(outcome).not.toHaveBeenCalled();
  });

  it('backs out without a call', async () => {
    const { user } = renderWith({ username: 'bob', suspending: true, uuid: 'u-bob' });
    await user.click(screen.getByRole('button', { name: 'Open' }));
    await user.click(screen.getByRole('button', { name: 'Close' }));
    expect(outcome).toHaveBeenCalledWith(false);
    expect(setUserActive).not.toHaveBeenCalled();
  });
});
