import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../services/api', () => ({ api: { get: vi.fn() } }));

import { api } from '../services/api';
import { endSession, hasSession, readJSON, startSession } from './session';

beforeEach(() => {
  api.get.mockImplementation(async (path) =>
    path === '/genres' ? { genre: ['Fiction'] } : { subgenre: ['Noir'] },
  );
});

describe('session', () => {
  it('stores the auth response and the genre lists', async () => {
    await startSession({ accessToken: 'a', refreshToken: 'r', username: 'ada', email: 'ada@x.com' });

    // The genre lists are fetched with the new token, before it is stored.
    expect(api.get).toHaveBeenCalledWith('/genres', { token: 'a' });
    expect(hasSession()).toBe(true);
    expect(localStorage.getItem('refreshToken')).toBe('r');
    expect(localStorage.getItem('username')).toBe('ada');
    expect(readJSON('genres', [])).toEqual(['Fiction']);
    expect(readJSON('subgenres', [])).toEqual(['Noir']);
  });

  it('stores nothing when the genre fetch fails', async () => {
    api.get.mockRejectedValue(new Error('down'));
    await expect(startSession({ accessToken: 'a' })).rejects.toThrow('down');
    expect(hasSession()).toBe(false);
  });

  it('endSession clears every session key and nothing else', async () => {
    await startSession({ accessToken: 'a', username: 'ada', picture: 'p.png' });
    localStorage.setItem('theme', 'dark');
    endSession();
    expect(hasSession()).toBe(false);
    expect(localStorage.getItem('avatar')).toBeNull();
    expect(localStorage.getItem('theme')).toBe('dark');
  });

  it('readJSON falls back on missing or corrupt values', () => {
    expect(readJSON('nope', [])).toEqual([]);
    localStorage.setItem('bad', '{not json');
    expect(readJSON('bad', 'fallback')).toBe('fallback');
  });
});
