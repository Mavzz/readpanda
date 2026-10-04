import { getBackendUrl } from '../utils/Helper';

// One client for every portal request. It adds the bearer token and the
// portal header, and turns a failed response into an Error whose message is
// the server's {"error"} / {"message"} text and whose .status is the HTTP code.

export class ApiError extends Error {
  constructor(status, message) {
    super(message);
    this.status = status;
  }
}

// App registers this to sign out when a stored token stops working.
let onUnauthorized = null;
export const setUnauthorizedHandler = (fn) => { onUnauthorized = fn; };

const readError = async (response) => {
  const text = await response.text();
  try {
    const body = JSON.parse(text);
    return body.error || body.message || text;
  } catch {
    return text || response.statusText;
  }
};

const request = async (method, path, { body, form, signal, token } = {}) => {
  const bearer = token ?? localStorage.getItem('token');
  const headers = {
    Accept: 'application/json',
    'X-Application-Type': 'portal',
    ...(bearer ? { Authorization: `Bearer ${bearer}` } : {}),
    // FormData sets its own multipart boundary, so no Content-Type for it.
    ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}),
  };

  const response = await fetch(await getBackendUrl(path), {
    method,
    headers,
    body: form ?? (body !== undefined ? JSON.stringify(body) : undefined),
    signal,
  });

  if (!response.ok) {
    // Only for the stored session token — a 401 from the login form is just a wrong password.
    if (response.status === 401 && bearer && token === undefined) onUnauthorized?.();
    throw new ApiError(response.status, await readError(response));
  }
  if (response.status === 204) return null;
  return response.json();
};

export const api = {
  get: (path, opts) => request('GET', path, opts),
  post: (path, body, opts) => request('POST', path, { ...opts, body }),
  put: (path, body, opts) => request('PUT', path, { ...opts, body }),
  del: (path, body, opts) => request('DELETE', path, { ...opts, body }),
  upload: (path, form, opts) => request('POST', path, { ...opts, form }),
};
