import { api } from '../services/api';

const SESSION_KEYS = ['token', 'refreshToken', 'username', 'email', 'avatar', 'genres', 'subgenres'];

export const hasSession = () => !!localStorage.getItem('token');

// Stores the auth response and the genre lists Upload needs. Every sign-in
// path (password, Google, sign-up) goes through here so they store the same
// things.
export const startSession = async (auth) => {
  const token = auth.accessToken;
  const [genres, subgenres] = await Promise.all([
    api.get('/genres', { token }),
    api.get('/subgenres', { token }),
  ]);

  localStorage.setItem('token', token);
  if (auth.refreshToken) localStorage.setItem('refreshToken', auth.refreshToken);
  localStorage.setItem('username', auth.username ?? '');
  if (auth.email) localStorage.setItem('email', auth.email);
  if (auth.picture) localStorage.setItem('avatar', auth.picture);
  localStorage.setItem('genres', JSON.stringify(genres.genre ?? []));
  localStorage.setItem('subgenres', JSON.stringify(subgenres.subgenre ?? []));
};

export const endSession = () => {
  SESSION_KEYS.forEach((k) => localStorage.removeItem(k));
};

export const readJSON = (key, fallback) => {
  try {
    return JSON.parse(localStorage.getItem(key)) ?? fallback;
  } catch {
    return fallback;
  }
};
