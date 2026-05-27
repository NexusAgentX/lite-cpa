import { defineComponent, h, ref, reactive, computed, watch, onMounted, onBeforeUnmount } from '../vendor.js';
import { naive } from '../vendor.js';
import { useStore } from '../store.js';
import { fetchLogs, downloadCsv } from '../api.js';
import { DetailDrawer } from '../components/detail-drawer.js';
import { fmtShort, fmtNum, fmtMs, protoTone, statusTone, presetRange, STATUS_OPTIONS } from '../utils.js';

const PRESETS = [
  { key: '1h', label: '1H' },
  { key: 'today', label: 'Today' },
  { key: '7d', label: '7D' },
  { key: '30d', label: '30D' },
];

// Parse the URL query object into the internal filter shape we send to the API.
function queryToFilters(q) {
  const csv = (s) => (s ? s.split(',').filter(Boolean) : []);
  return {
    search: q.search || '',
    model: csv(q.model),
    protocol: csv(q.protocol),
    status_code: csv(q.status_code),
    method: csv(q.method),
    input_tokens_min: q.input_tokens_min || '',
    input_tokens_max: q.input_tokens_max || '',
    output_tokens_min: q.output_tokens_min || '',
    output_tokens_max: q.output_tokens_max || '',
    duration_min: q.duration_min || '',
    duration_max: q.duration_max || '',
    time_from: q.time_from || '',
    time_to: q.time_to || '',
    sort_by: q.sort_by || 'timestamp',
    sort_dir: q.sort_dir || 'desc',
    page: q.page ? Number(q.page) : 1,
    per_page: q.per_page ? Number(q.per_page) : null,
  };
}

function filtersToQuery(f) {
  const out = {};
  if (f.search) out.search = f.search;
  if (f.model.length)       out.model       = f.model.join(',');
  if (f.protocol.length)    out.protocol    = f.protocol.join(',');
  if (f.status_code.length) out.status_code = f.status_code.join(',');
  if (f.method.length)      out.method      = f.method.join(',');
  if (f.input_tokens_min)  out.input_tokens_min  = f.input_tokens_min;
  if (f.input_tokens_max)  out.input_tokens_max  = f.input_tokens_max;
  if (f.output_tokens_min) out.output_tokens_min = f.output_tokens_min;
  if (f.output_tokens_max) out.output_tokens_max = f.output_tokens_max;
  if (f.duration_min) out.duration_min = f.duration_min;
  if (f.duration_max) out.duration_max = f.duration_max;
  if (f.time_from) out.time_from = f.time_from;
  if (f.time_to)   out.time_to   = f.time_to;
  if (f.sort_by && f.sort_by !== 'timestamp') out.sort_by = f.sort_by;
  if (f.sort_dir && f.sort_dir !== 'desc')    out.sort_dir = f.sort_dir;
  if (f.page && f.page !== 1) out.page = f.page;
  if (f.per_page) out.per_page = f.per_page;
  return out;
}

export function makeRequestsView({ forcedFilters = {}, lockedKeys = [] } = {}) {
  return defineComponent({
    name: 'RequestsView',
    props: { router: { type: Object, required: true } },
    setup(props) {
      const store = useStore();
      const filters = reactive(queryToFilters(props.router.query));
      // Apply any view-level locked filters (e.g. Errors tab forces status>=400)
      Object.assign(filters, forcedFilters);
      if (!filters.per_page) filters.per_page = store.prefs.perPage || 20;

      const entries = ref([]);
      const total = ref(0);
      const loading = ref(false);
      const drawerOpen = ref(false);
      const drawerId = ref(null);
      const advancedOpen = ref(false);
      const showCustomRange = ref(!!(filters.time_from || filters.time_to));

      let searchTimer = null;
      function debouncedReload() {
        clearTimeout(searchTimer);
        searchTimer = setTimeout(() => { filters.page = 1; load(); }, 280);
      }

      function syncURL() {
        const q = filtersToQuery(filters);
        // strip locked keys so user toggles don't leak into the URL
        lockedKeys.forEach(k => delete q[k]);
        props.router.setQuery(q);
      }

      async function load(opts = {}) {
        if (opts.resetPage) filters.page = 1;
        loading.value = true;
        try {
          const data = await fetchLogs(filters);
          total.value   = data.total || 0;
          entries.value = data.entries || [];
          store.captureOptions(data);
          store.captureTotals(data);
          store.lastUpdate.value = new Date().toLocaleTimeString();
        } catch (_) {}
        finally { loading.value = false; }
      }

      // Initial load + every tick from auto-refresh
      onMounted(() => { load(); syncURL(); });
      watch(() => store.tick.value, () => load());
      watch(() => store.apiKey.value, (v) => { if (v) load(); });

      function applyPreset(key) {
        const r = presetRange(key);
        filters.time_from = r.time_from;
        filters.time_to   = r.time_to;
        filters.page = 1;
        syncURL();
        load();
      }
      const activePreset = computed(() => {
        for (const p of PRESETS) {
          const r = presetRange(p.key);
          if (r.time_from === filters.time_from && r.time_to === filters.time_to) return p.key;
        }
        return null;
      });

      function clearAll() {
        filters.search = '';
        filters.model = []; filters.protocol = []; filters.status_code = []; filters.method = [];
        filters.input_tokens_min = ''; filters.input_tokens_max = '';
        filters.output_tokens_min = ''; filters.output_tokens_max = '';
        filters.duration_min = ''; filters.duration_max = '';
        filters.time_from = ''; filters.time_to = '';
        filters.page = 1;
        Object.assign(filters, forcedFilters);
        syncURL(); load();
      }

      const chips = computed(() => {
        const out = [];
        if (filters.search) out.push({ tone: 'tone-accent', label: `Search: ${filters.search}`, clear: () => { filters.search = ''; reload(); } });
        filters.model.forEach(m => out.push({ tone: 'tone-accent', label: `Model: ${m}`, clear: () => { filters.model = filters.model.filter(x => x !== m); reload(); } }));
        filters.protocol.forEach(p => out.push({ tone: 'tone-ok', label: `Protocol: ${p}`, clear: () => { filters.protocol = filters.protocol.filter(x => x !== p); reload(); } }));
        filters.status_code.forEach(s => out.push({ tone: 'tone-warn', label: `Status: ${s}`, clear: () => { filters.status_code = filters.status_code.filter(x => x !== s); reload(); } }));
        filters.method.forEach(m => out.push({ tone: '', label: `Method: ${m}`, clear: () => { filters.method = filters.method.filter(x => x !== m); reload(); } }));
        if (filters.input_tokens_min)  out.push({ tone: '', label: `In ≥ ${filters.input_tokens_min}`,  clear: () => { filters.input_tokens_min = '';  reload(); } });
        if (filters.input_tokens_max)  out.push({ tone: '', label: `In ≤ ${filters.input_tokens_max}`,  clear: () => { filters.input_tokens_max = '';  reload(); } });
        if (filters.output_tokens_min) out.push({ tone: '', label: `Out ≥ ${filters.output_tokens_min}`, clear: () => { filters.output_tokens_min = ''; reload(); } });
        if (filters.output_tokens_max) out.push({ tone: '', label: `Out ≤ ${filters.output_tokens_max}`, clear: () => { filters.output_tokens_max = ''; reload(); } });
        if (filters.duration_min) out.push({ tone: '', label: `Dur ≥ ${filters.duration_min}ms`, clear: () => { filters.duration_min = ''; reload(); } });
        if (filters.duration_max) out.push({ tone: '', label: `Dur ≤ ${filters.duration_max}ms`, clear: () => { filters.duration_max = ''; reload(); } });
        if (filters.time_from) out.push({ tone: 'tone-err', label: `From: ${filters.time_from}`, clear: () => { filters.time_from = ''; reload(); } });
        if (filters.time_to)   out.push({ tone: 'tone-err', label: `To: ${filters.time_to}`,   clear: () => { filters.time_to   = ''; reload(); } });
        return out;
      });
      function reload() { filters.page = 1; syncURL(); load(); }

      function toggleSort(col) {
        if (filters.sort_by === col) {
          filters.sort_dir = filters.sort_dir === 'desc' ? 'asc' : 'desc';
        } else {
          filters.sort_by = col; filters.sort_dir = 'desc';
        }
        syncURL(); load();
      }

      function showDetail(row) {
        drawerId.value = row.id;
        drawerOpen.value = true;
      }

      function exportCsv() { downloadCsv(filters); }

      const totalPages = computed(() => Math.max(1, Math.ceil(total.value / (filters.per_page || 20))));

      const { NSelect, NInput, NButton, NDataTable, NPagination, NDatePicker } = naive;

      function renderFilters() {
        const modelOpts = store.options.models.map(m => ({ label: m, value: m }));
        const protoOpts = store.options.protocols.map(m => ({ label: m, value: m }));
        const methodOpts = store.options.methods.map(m => ({ label: m, value: m }));
        return h('div', { class: 'panel', style: { marginBottom: '14px' } }, [
          h('div', { class: 'filter-row' }, [
            h('div', { class: 'search' }, [
              h('span', { class: 'ico' }, '⌕'),
              h('input', {
                value: filters.search,
                placeholder: 'Search model, URL, method, user agent…',
                onInput: (e) => { filters.search = e.target.value; debouncedReload(); syncURL(); },
              }),
            ]),
            h('div', { class: 'preset-group' }, PRESETS.map(p =>
              h('button', {
                class: { active: activePreset.value === p.key },
                onClick: () => applyPreset(p.key),
              }, p.label)
            )),
            h('button', { class: 'btn subtle', onClick: () => { advancedOpen.value = !advancedOpen.value; } },
              advancedOpen.value ? '− Filters' : '+ Filters'
            ),
            h('button', { class: 'btn primary', onClick: exportCsv }, ['↓ ', 'Export CSV']),
          ]),

          advancedOpen.value ? h('div', null, [
            h('div', { class: 'divider' }),
            h('div', { class: 'filter-row' }, [
              h(NSelect, {
                multiple: true, clearable: true, filterable: true,
                size: 'small',
                placeholder: 'All models',
                options: modelOpts,
                value: filters.model,
                style: { minWidth: '180px' },
                'onUpdate:value': (v) => { filters.model = v || []; reload(); },
              }),
              h(NSelect, {
                multiple: true, clearable: true,
                size: 'small',
                placeholder: 'All protocols',
                options: protoOpts,
                value: filters.protocol,
                style: { minWidth: '180px' },
                'onUpdate:value': (v) => { filters.protocol = v || []; reload(); },
              }),
              h(NSelect, {
                multiple: true, clearable: true,
                size: 'small',
                placeholder: 'All statuses',
                options: STATUS_OPTIONS,
                value: filters.status_code,
                style: { minWidth: '200px' },
                'onUpdate:value': (v) => { filters.status_code = v || []; reload(); },
              }),
              h(NSelect, {
                multiple: true, clearable: true,
                size: 'small',
                placeholder: 'All methods',
                options: methodOpts,
                value: filters.method,
                style: { minWidth: '160px' },
                'onUpdate:value': (v) => { filters.method = v || []; reload(); },
              }),
            ]),
            h('div', { class: 'filter-row' }, [
              h('div', { class: 'range' }, [
                h('label', null, 'Input'),
                h('input', { type: 'number', placeholder: 'min', value: filters.input_tokens_min,
                  onChange: (e) => { filters.input_tokens_min = e.target.value; reload(); } }),
                h('span', { class: 'sep' }, '–'),
                h('input', { type: 'number', placeholder: 'max', value: filters.input_tokens_max,
                  onChange: (e) => { filters.input_tokens_max = e.target.value; reload(); } }),
              ]),
              h('div', { class: 'range' }, [
                h('label', null, 'Output'),
                h('input', { type: 'number', placeholder: 'min', value: filters.output_tokens_min,
                  onChange: (e) => { filters.output_tokens_min = e.target.value; reload(); } }),
                h('span', { class: 'sep' }, '–'),
                h('input', { type: 'number', placeholder: 'max', value: filters.output_tokens_max,
                  onChange: (e) => { filters.output_tokens_max = e.target.value; reload(); } }),
              ]),
              h('div', { class: 'range' }, [
                h('label', null, 'Duration'),
                h('input', { type: 'number', placeholder: 'min', value: filters.duration_min,
                  onChange: (e) => { filters.duration_min = e.target.value; reload(); } }),
                h('span', { class: 'sep' }, '–'),
                h('input', { type: 'number', placeholder: 'max', value: filters.duration_max,
                  onChange: (e) => { filters.duration_max = e.target.value; reload(); } }),
                h('span', { class: 'unit' }, 'ms'),
              ]),
              h('button', { class: 'btn subtle', onClick: () => { showCustomRange.value = !showCustomRange.value; } },
                showCustomRange.value ? 'Hide custom range' : 'Custom time range'
              ),
            ]),
            showCustomRange.value ? h('div', { class: 'filter-row' }, [
              h('input', { type: 'datetime-local', value: filters.time_from,
                class: 'range', style: { padding: '5px 9px', border: '1px solid var(--border)', borderRadius: 'var(--radius-xs)', background: 'var(--bg)', color: 'var(--text)', fontFamily: 'var(--mono)', fontSize: '11.5px' },
                onChange: (e) => { filters.time_from = e.target.value; reload(); } }),
              h('span', { class: 'sep', style: { color: 'var(--text-mute)' } }, '→'),
              h('input', { type: 'datetime-local', value: filters.time_to,
                style: { padding: '5px 9px', border: '1px solid var(--border)', borderRadius: 'var(--radius-xs)', background: 'var(--bg)', color: 'var(--text)', fontFamily: 'var(--mono)', fontSize: '11.5px' },
                onChange: (e) => { filters.time_to = e.target.value; reload(); } }),
            ]) : null,
          ]) : null,

          chips.value.length ? h('div', null, [
            h('div', { class: 'divider' }),
            h('div', { class: 'chips' }, [
              ...chips.value.map(c => h('span', { class: ['chip', c.tone] }, [
                c.label,
                h('button', { class: 'x', onClick: c.clear }, '×'),
              ])),
              h('button', { class: 'btn subtle', style: { marginLeft: 'auto', color: 'var(--err)' }, onClick: clearAll }, 'Clear all'),
            ]),
          ]) : null,
        ]);
      }

      function sortHeader(label, col) {
        const active = filters.sort_by === col;
        return h('span', { style: { cursor: 'pointer', userSelect: 'none' }, onClick: () => toggleSort(col) }, [
          label,
          active ? h('span', { style: { marginLeft: '4px', color: 'var(--accent)', fontSize: '10px' } },
            filters.sort_dir === 'desc' ? '↓' : '↑'
          ) : null,
        ]);
      }

      const columns = computed(() => [
        {
          title: () => sortHeader('Time', 'timestamp'),
          key: 'time_str',
          width: 140,
          render: (row) => h('span', { class: 'cell-mono cell-dim' }, row.time_str),
        },
        {
          title: () => sortHeader('Model', 'model'),
          key: 'model',
          render: (row) => h('span', { class: 'badge tone-accent' }, row.model || '—'),
        },
        {
          title: () => sortHeader('Proto', 'protocol'),
          key: 'protocol',
          width: 110,
          render: (row) => h('span', { class: ['badge', protoTone(row.protocol)] }, row.protocol || '—'),
        },
        {
          title: 'UA', key: 'ua', width: 140, ellipsis: { tooltip: true },
          render: (row) => h('span', { class: 'cell-mono cell-dim' }, row.ua || '—'),
        },
        {
          title: () => sortHeader('Status', 'status_code'),
          key: 'status', width: 90,
          render: (row) => {
            const tone = statusTone(row.status);
            return h('span', { class: ['status-text', tone], style: { fontFamily: 'var(--mono)', fontSize: '12px' } }, [
              h('span', { class: ['status-dot', tone] }),
              String(row.status),
            ]);
          },
        },
        {
          title: () => sortHeader('Input', 'input_tokens'),
          key: 'input_tokens', width: 90, align: 'right',
          render: (row) => h('span', { class: 'cell-num', style: { color: 'var(--ok)' } }, fmtNum(row.input_tokens)),
        },
        {
          title: () => sortHeader('Output', 'output_tokens'),
          key: 'output_tokens', width: 90, align: 'right',
          render: (row) => h('span', { class: 'cell-num', style: { color: 'var(--purple)' } }, fmtNum(row.output_tokens)),
        },
        {
          title: 'Cache', key: 'cache_read', width: 90, align: 'right',
          render: (row) => h('span', { class: 'cell-num', style: { color: 'var(--warn)' } }, fmtNum(row.cache_read)),
        },
        {
          title: () => sortHeader('Dur', 'duration_ms'),
          key: 'duration_ms', width: 80, align: 'right',
          render: (row) => h('span', { class: 'cell-num cell-dim' }, fmtMs(row.duration_ms)),
        },
        {
          title: '', key: 'actions', width: 70,
          render: (row) => h('a', {
            href: 'javascript:void(0)',
            style: { color: 'var(--accent)', fontSize: '12px', textDecoration: 'none' },
            onClick: () => showDetail(row),
          }, 'view →'),
        },
      ]);

      return () => h('div', null, [
        renderFilters(),
        h('div', { class: 'panel', style: { padding: '0', overflow: 'hidden' } }, [
          h(NDataTable, {
            columns: columns.value,
            data: entries.value,
            loading: loading.value,
            size: 'small',
            bordered: false,
            singleLine: true,
            flexHeight: false,
            maxHeight: 560,
            rowKey: (row) => row.id,
            style: { '--n-merged-th-color': 'var(--surface-2)' },
          }),
        ]),
        h('div', { style: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '12px 4px' } }, [
          h('span', { style: { fontSize: '12px', color: 'var(--text-mute)', fontFamily: 'var(--mono)' } },
            `${fmtNum(total.value)} record${total.value === 1 ? '' : 's'} · page ${filters.page} / ${totalPages.value}`
          ),
          h(NPagination, {
            page: filters.page,
            pageCount: totalPages.value,
            pageSize: filters.per_page,
            pageSizes: [20, 50, 100],
            showSizePicker: true,
            'onUpdate:page': (p) => { filters.page = p; syncURL(); load(); },
            'onUpdate:pageSize': (s) => { filters.per_page = s; filters.page = 1; store.prefs.perPage = s; syncURL(); load(); },
          }),
        ]),
        h(DetailDrawer, {
          open: drawerOpen.value,
          id: drawerId.value,
          onClose: () => { drawerOpen.value = false; },
        }),
      ]);
    },
  });
}

export const RequestsView = makeRequestsView();
