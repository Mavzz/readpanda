import { beforeEach, describe, expect, it, vi } from 'vitest';
import { api, ApiError, setUnauthorizedHandler } from './api';

const respond = (status, body) =>
  new Response(body === undefined ? null : typeof body === 'string' ? body : JSON.stringify(body), { status });

let fetchMock;
beforeEach(() => {
  fetchMock = vi.fn();
  vi.stubGlobal('fetch', fetchMock);
  setUnauthorizedHandler(null);
});

const lastCall = () => {
  const [url, init] = fetchMock.mock.calls.at(-1);
  return { url, init, headers: init.headers };
};

describe('api client', () => {
  it('sends the stored token and the portal header', async () => {
    localStorage.setItem('token', 'stored');
    fetchMock.mockResolvedValue(respond(200, { ok: true }));

    await expect(api.get('/books/all')).resolves.toEqual({ ok: true });

    const { url, init, headers } = lastCall();
    expect(url).toMatch(/\/api\/v1\/books\/all$/);
    expect(init.method).toBe('GET');
    expect(headers.Authorization).toBe('Bearer stored');
    expect(headers['X-Application-Type']).toBe('portal');
    expect(headers['Content-Type']).toBeUndefined();
  });

  it('JSON-encodes bodies', async () => {
    fetchMock.mockResolvedValue(respond(200, {}));
    await api.post('/room/create', { name: 'Club' });
    const { init, headers } = lastCall();
    expect(headers['Content-Type']).toBe('application/json');
    expect(init.body).toBe('{"name":"Club"}');
  });

  it('lets FormData set its own content type', async () => {
    fetchMock.mockResolvedValue(respond(200, {}));
    const form = new FormData();
    await api.upload('/books/seed', form);
    const { init, headers } = lastCall();
    expect(init.body).toBe(form);
    expect(headers['Content-Type']).toBeUndefined();
  });

  it('an explicit empty token sends no Authorization header', async () => {
    localStorage.setItem('token', 'stored');
    fetchMock.mockResolvedValue(respond(200, {}));
    await api.post('/auth/login', {}, { token: '' });
    expect(lastCall().headers.Authorization).toBeUndefined();
  });

  it('returns null for 204', async () => {
    fetchMock.mockResolvedValue(respond(204));
    await expect(api.del('/highlights/1')).resolves.toBeNull();
  });

  it.each([
    ['the {"error"} text', { error: 'Book not found' }, 'Book not found'],
    ['the {"message"} text', { message: 'Nope' }, 'Nope'],
    ['a plain-text body', 'Bad gateway', 'Bad gateway'],
  ])('turns failures into ApiError with %s', async (_, body, message) => {
    fetchMock.mockResolvedValue(respond(404, body));
    const err = await api.get('/x').catch((e) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect(err.status).toBe(404);
    expect(err.message).toBe(message);
  });

  it('signs out when the stored session token is rejected', async () => {
    const onUnauthorized = vi.fn();
    setUnauthorizedHandler(onUnauthorized);
    localStorage.setItem('token', 'expired');
    fetchMock.mockResolvedValue(respond(401, { error: 'Unauthorized' }));

    await expect(api.get('/books')).rejects.toThrow('Unauthorized');
    expect(onUnauthorized).toHaveBeenCalledOnce();
  });

  it('does not sign out on a 401 for an explicit token (e.g. a wrong password)', async () => {
    const onUnauthorized = vi.fn();
    setUnauthorizedHandler(onUnauthorized);
    localStorage.setItem('token', 'stored');
    fetchMock.mockResolvedValue(respond(401, { error: 'Invalid username or password' }));

    await expect(api.post('/auth/login', {}, { token: '' })).rejects.toThrow();
    expect(onUnauthorized).not.toHaveBeenCalled();
  });
});
