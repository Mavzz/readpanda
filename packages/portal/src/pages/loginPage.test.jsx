import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';

vi.mock('@react-oauth/google', () => ({ GoogleLogin: () => null }));
vi.mock('../services/api', () => ({ api: { post: vi.fn() } }));
vi.mock('../utils/session', () => ({ startSession: vi.fn() }));

import { api } from '../services/api';
import { startSession } from '../utils/session';
import LoginPage from './loginPage';

const renderLogin = () => {
  const setIsLoggedIn = vi.fn();
  render(
    <MemoryRouter initialEntries={['/login']}>
      <Routes>
        <Route path="/login" element={<LoginPage setIsLoggedIn={setIsLoggedIn} onSwitchToSignUp={() => {}} />} />
        <Route path="/dashboard" element={<p>Dashboard</p>} />
      </Routes>
    </MemoryRouter>,
  );
  return { setIsLoggedIn, user: userEvent.setup() };
};

const signIn = async (user, username = 'ada', password = 'pw') => {
  if (username) await user.type(screen.getByLabelText(/username/i), username);
  if (password) await user.type(screen.getByLabelText(/password/i), password);
  await user.click(screen.getByRole('button', { name: /sign in/i }));
};

beforeEach(() => {
  vi.spyOn(console, 'error').mockImplementation(() => {});
});

describe('LoginPage', () => {
  it('asks for both fields before calling the API', async () => {
    const { user } = renderLogin();
    await signIn(user, 'ada', '');
    expect(await screen.findByText('Enter your username and password.')).toBeInTheDocument();
    expect(api.post).not.toHaveBeenCalled();
  });

  it('signs in and goes to the dashboard', async () => {
    api.post.mockResolvedValue({ accessToken: 'tok', username: 'ada' });
    const { user, setIsLoggedIn } = renderLogin();

    await signIn(user);

    expect(await screen.findByText('Dashboard')).toBeInTheDocument();
    expect(api.post).toHaveBeenCalledWith(
      '/auth/login',
      { username: 'ada', password: 'pw' },
      expect.objectContaining({ token: '' }),
    );
    expect(startSession).toHaveBeenCalledWith({ accessToken: 'tok', username: 'ada' });
    expect(setIsLoggedIn).toHaveBeenCalledWith(true);
  });

  it.each([
    [401, "That username and password don't match."],
    [403, "This account isn't an admin."],
    [500, "Couldn't reach the server. Check your connection and try again."],
  ])('explains a %i', async (status, message) => {
    api.post.mockRejectedValue(Object.assign(new Error('x'), { status }));
    const { user, setIsLoggedIn } = renderLogin();

    await signIn(user);

    expect(await screen.findByText(message)).toBeInTheDocument();
    expect(setIsLoggedIn).not.toHaveBeenCalled();
    expect(screen.getByRole('button', { name: /sign in/i })).toBeEnabled();
  });
});
