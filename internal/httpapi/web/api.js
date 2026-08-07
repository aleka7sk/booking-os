export class APIError extends Error {
  constructor(message, code = 'request_failed', status = 0) {
    super(message);
    this.name = 'APIError';
    this.code = code;
    this.status = status;
  }
}

async function request(path, options = {}) {
  const init = {
    credentials: 'same-origin',
    headers: { ...(options.body ? { 'Content-Type': 'application/json' } : {}), ...(options.headers || {}) },
    ...options,
  };
  if (options.body && typeof options.body !== 'string') init.body = JSON.stringify(options.body);
  let response;
  try {
    response = await fetch(path, init);
  } catch (error) {
    throw new APIError('Сервис недоступен. Проверьте соединение и повторите попытку.', 'network_error', 0);
  }
  const contentType = response.headers.get('content-type') || '';
  const payload = contentType.includes('application/json') ? await response.json() : null;
  if (!response.ok) {
    const error = payload?.error || {};
    throw new APIError(error.message || `Ошибка запроса (${response.status})`, error.code || 'request_failed', response.status);
  }
  return payload;
}

const query = (params = {}) => {
  const search = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') search.set(key, value);
  });
  const result = search.toString();
  return result ? `?${result}` : '';
};

export const api = {
  login: (email, password) => request('/api/auth/login', { method: 'POST', body: { email, password } }),
  logout: () => request('/api/auth/logout', { method: 'POST' }),
  me: () => request('/api/auth/me'),

  dashboard: () => request('/api/dashboard'),
  locations: () => request('/api/locations'),
  resources: () => request('/api/resources'),
  createResource: (body) => request('/api/resources', { method: 'POST', body }),
  offerings: () => request('/api/offerings'),
  createOffering: (body) => request('/api/offerings', { method: 'POST', body }),
  bookings: (params) => request(`/api/bookings${query(params)}`),
  booking: (id) => request(`/api/bookings/${encodeURIComponent(id)}`),
  createBooking: (body) => request('/api/bookings', { method: 'POST', body }),
  confirmBooking: (id) => request(`/api/bookings/${encodeURIComponent(id)}/confirm`, { method: 'POST' }),
  rejectBooking: (id, reason) => request(`/api/bookings/${encodeURIComponent(id)}/reject`, { method: 'POST', body: { reason } }),
  recordPayment: (id, amount, note = '') => request(`/api/bookings/${encodeURIComponent(id)}/payment`, { method: 'POST', body: { amount, note } }),
  cancelBooking: (id, reason) => request(`/api/bookings/${encodeURIComponent(id)}/cancel`, { method: 'POST', body: { reason } }),
  rescheduleBooking: (id, body) => request(`/api/bookings/${encodeURIComponent(id)}/reschedule`, { method: 'POST', body }),
  completeBooking: (id) => request(`/api/bookings/${encodeURIComponent(id)}/complete`, { method: 'POST' }),
  noShowBooking: (id) => request(`/api/bookings/${encodeURIComponent(id)}/no-show`, { method: 'POST' }),
  availability: (body) => request('/api/availability/search', { method: 'POST', body }),
  calendar: (from, to) => request(`/api/calendar${query({ from, to })}`),
  audit: () => request('/api/audit'),

  publicProfile: () => request('/api/public/profile'),
  publicOfferings: () => request('/api/public/offerings'),
  publicAvailability: (body) => request('/api/public/availability', { method: 'POST', body }),
  publicCreateBooking: (body) => request('/api/public/bookings', { method: 'POST', body }),
  publicBooking: (token) => request(`/api/public/bookings/${encodeURIComponent(token)}`),
};
