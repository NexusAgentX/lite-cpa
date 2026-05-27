export function fmtShort(n) {
  n = Number(n) || 0;
  const abs = Math.abs(n);
  if (abs >= 1e9) return (n / 1e9).toFixed(2) + 'B';
  if (abs >= 1e6) return (n / 1e6).toFixed(2) + 'M';
  if (abs >= 1e3) return (n / 1e3).toFixed(1) + 'K';
  return n.toLocaleString();
}

export function fmtNum(n) {
  return Number(n || 0).toLocaleString();
}

export function fmtMs(ms) {
  if (ms == null) return '—';
  if (ms < 1000) return ms + 'ms';
  return (ms / 1000).toFixed(2) + 's';
}

export function protoTone(p) {
  switch (p) {
    case 'anthropic': return 'tone-anthropic';
    case 'openai':    return 'tone-openai';
    case 'gemini':    return 'tone-gemini';
    case 'responses': return 'tone-responses';
    default:          return '';
  }
}

export function statusTone(s) {
  const n = Number(s);
  if (n < 300) return 'ok';
  if (n < 500) return 'warn';
  return 'err';
}

export function toDatetimeLocal(d) {
  const pad = n => String(n).padStart(2, '0');
  return d.getFullYear() + '-' + pad(d.getMonth() + 1) + '-' + pad(d.getDate())
       + 'T' + pad(d.getHours()) + ':' + pad(d.getMinutes());
}

export function presetRange(preset) {
  const now = new Date();
  let from = null;
  switch (preset) {
    case '1h':    from = new Date(now - 3600 * 1000); break;
    case '8h':    from = new Date(now - 8 * 3600 * 1000); break;
    case 'today': from = new Date(now.getFullYear(), now.getMonth(), now.getDate()); break;
    case '7d':    from = new Date(now - 7 * 86400 * 1000); break;
    case '30d':   from = new Date(now - 30 * 86400 * 1000); break;
    default:      return { time_from: '', time_to: '' };
  }
  return { time_from: toDatetimeLocal(from), time_to: toDatetimeLocal(now) };
}

export const STATUS_OPTIONS = [
  { value: '200', label: '200 OK' },
  { value: '201', label: '201 Created' },
  { value: '400', label: '400 Bad Request' },
  { value: '401', label: '401 Unauthorized' },
  { value: '403', label: '403 Forbidden' },
  { value: '429', label: '429 Rate Limited' },
  { value: '500', label: '500 Error' },
  { value: '502', label: '502 Bad Gateway' },
  { value: '503', label: '503 Unavailable' },
];

export function chartBase(tokens) {
  const t = tokens || {};
  return {
    backgroundColor: 'transparent',
    textStyle: { color: t.textDim || '#6b7280', fontFamily: 'inherit' },
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'line',
        lineStyle: { color: t.axisLine || '#3e4451', width: 1, type: 'dashed' },
        label: { show: false },
      },
      backgroundColor: t.tooltipBg || '#1b1f27',
      borderColor: t.tooltipBd || '#2d333f',
      borderWidth: 1,
      padding: [8, 12],
      textStyle: { color: t.text || '#c8ccd4', fontSize: 12 },
      valueFormatter: (v) => fmtShort(v),
    },
    grid: { left: 40, right: 16, top: 24, bottom: 28, containLabel: false },
  };
}