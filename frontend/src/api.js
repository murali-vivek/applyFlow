const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';
const TOKEN_KEY = 'applyflow_token';

export function getToken() {
  return localStorage.getItem(TOKEN_KEY);
}

export function setToken(token) {
  localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY);
}

async function request(path, options = {}) {
  const headers = {
    ...(options.body instanceof FormData ? {} : { 'Content-Type': 'application/json' }),
    ...options.headers,
  };
  const token = getToken();
  if (token) headers.Authorization = `Bearer ${token}`;

  const res = await fetch(`${API_BASE}${path}`, {
    credentials: 'include',
    ...options,
    headers,
  });

  if (res.status === 204) return null;

  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    const msg = data?.error?.message || 'Request failed';
    throw new Error(msg);
  }
  return data;
}

export const api = {
  me: () => request('/auth/me'),
  templates: {
    list: () => request('/templates'),
    create: (body) => request('/templates', { method: 'POST', body: JSON.stringify(body) }),
    update: (id, body) => request(`/templates/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
    delete: (id) => request(`/templates/${id}`, { method: 'DELETE' }),
  },
  resume: {
    get: () => request('/resume'),
    upload: (file) => {
      const form = new FormData();
      form.append('file', file);
      return request('/resume', { method: 'POST', body: form });
    },
  },
  xlsx: {
    upload: (file) => {
      const form = new FormData();
      form.append('file', file);
      return request('/files/xlsx', { method: 'POST', body: form });
    },
  },
  campaigns: {
    list: () => request('/campaigns'),
    get: (id) => request(`/campaigns/${id}`),
    create: (body) => request('/campaigns', { method: 'POST', body: JSON.stringify(body) }),
    cancel: (id) => request(`/campaigns/${id}/cancel`, { method: 'POST' }),
  },
};

export function googleLoginUrl() {
  return `${API_BASE}/auth/google`;
}

export function logoutUrl() {
  return `${API_BASE}/auth/logout`;
}
