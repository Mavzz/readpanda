import { describe, expect, it, vi } from 'vitest';
import { isSuspended } from './moderation';

describe('isSuspended', () => {
  it('reads the admin table routes\' text as well as a boolean', () => {
    expect(isSuspended({ username: 'ada', isactive: 'false' })).toBe(true);
    expect(isSuspended({ username: 'ada', isactive: false })).toBe(true);
    expect(isSuspended({ username: 'ada', isactive: 'true' })).toBe(false);
    expect(isSuspended({ username: 'ada', isactive: null })).toBe(false);
  });

  it('never calls the Deleted reader suspended', () => {
    expect(isSuspended({ username: 'deleted.reader', isactive: 'false' })).toBe(false);
  });
});

describe('the note (portal-moderation R1.1)', () => {
  it('goes with a suspension, trimmed, and is left out when empty', async () => {
    const { api } = await import('./api');
    const post = vi.spyOn(api, 'post').mockResolvedValue({});
    const patch = vi.spyOn(api, 'patch').mockResolvedValue({});
    const { resolveReport, setUserActive } = await import('./moderation');

    await resolveReport('rp_1', 'suspend_user', '  why  ');
    await resolveReport('rp_2', 'dismiss');
    await setUserActive('u-1', true, '   ');
    expect(post).toHaveBeenNthCalledWith(1, '/admin/reports/rp_1/resolve', { action: 'suspend_user', note: 'why' });
    expect(post).toHaveBeenNthCalledWith(2, '/admin/reports/rp_2/resolve', { action: 'dismiss' });
    expect(patch).toHaveBeenCalledWith('/admin/users/u-1', { isactive: true });
  });
});
