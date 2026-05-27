// API wrapper. All calls go through here so 401/403 surface a single auth signal.

const AUTH_KEY = 'cpa_api_key';

export function getApiKey() { return localStorage.getItem(AUTH_KEY) || ''; }
export function setApiKey(k) { localStorage.setItem(AUTH_KEY, k || ''); }
export function clearApiKey() { localStorage.removeItem(AUTH_KEY); }

const listeners = new Set();
export function onAuthError(fn) { listeners.add(fn); return () => listeners.delete(fn); }
function emitAuthError() { listeners.forEach(fn => { try { fn(); } catch (_) {} }); }

function authHeaders() {
  const k = getApiKey();
  return k ? { 'x-api-key': k } : {};
}

async function request(url) {
  const r = await fetch(url, { headers: authHeaders() });
  if (r.status === 401 || r.status === 403) {
    emitAuthError();
    throw new Error('unauthorized');
  }
  if (!r.ok) throw new Error('HTTP ' + r.status);
  return r;
}

export function buildLogsQuery(p = {}) {
  const usp = new URLSearchParams();
  if (p.page != null) usp.set('page', p.page);
  if (p.per_page != null) usp.set('per_page', p.per_page);
  if (p.sort_by) usp.set('sort_by', p.sort_by);
  if (p.sort_dir) usp.set('sort_dir', p.sort_dir);
  if (p.search) usp.set('search', p.search);
  if (p.model && p.model.length) usp.set('model', p.model.join(','));
  if (p.protocol && p.protocol.length) usp.set('protocol', p.protocol.join(','));
  if (p.status_code && p.status_code.length) usp.set('status_code', p.status_code.join(','));
  if (p.method && p.method.length) usp.set('method', p.method.join(','));
  if (p.input_tokens_min) usp.set('input_tokens_min', p.input_tokens_min);
  if (p.input_tokens_max) usp.set('input_tokens_max', p.input_tokens_max);
  if (p.output_tokens_min) usp.set('output_tokens_min', p.output_tokens_min);
  if (p.output_tokens_max) usp.set('output_tokens_max', p.output_tokens_max);
  if (p.duration_min) usp.set('duration_min', p.duration_min);
  if (p.duration_max) usp.set('duration_max', p.duration_max);
  if (p.time_from) usp.set('time_from', p.time_from);
  if (p.time_to) usp.set('time_to', p.time_to);
  return usp;
}

export async function fetchLogs(params) {
  const r = await request('/dashboard-logs?' + buildLogsQuery(params).toString());
  return r.json();
}

export async function fetchLogDetail(id) {
  const r = await request('/dashboard-logs/' + encodeURIComponent(id));
  return r.json();
}

export async function downloadCsv(params) {
  const usp = buildLogsQuery(params);
  usp.delete('page');
  usp.delete('per_page');
  const r = await request('/dashboard-logs/export?' + usp.toString());
  const blob = await r.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = 'api-logs.csv';
  a.click();
  URL.revokeObjectURL(url);
}
