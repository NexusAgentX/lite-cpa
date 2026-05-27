import { defineComponent, h, ref, computed, watch, onMounted } from '../vendor.js';
import { useStore } from '../store.js';
import { fetchLogs } from '../api.js';
import { makeRequestsView } from './requests.js';
import { fmtNum } from '../utils.js';

// Common error codes we sample for the summary strip and force into the
// embedded requests view. Anything more exotic the user can still filter for
// from the Requests tab directly.
const ERROR_CODES = [
  '400', '401', '402', '403', '404', '405', '408', '409', '413',
  '422', '429', '500', '501', '502', '503', '504',
];

const InnerRequests = makeRequestsView({
  forcedFilters: { status_code: ERROR_CODES },
  lockedKeys: ['status_code'],
});

export const ErrorsView = defineComponent({
  name: 'ErrorsView',
  props: { router: { type: Object, required: true } },
  setup(props) {
    const store = useStore();
    const counts = ref({});
    const totalErr = ref(0);

    async function loadSummary() {
      if (!store.apiKey.value) return;
      try {
        const data = await fetchLogs({
          page: 1, per_page: 500,
          sort_by: 'timestamp', sort_dir: 'desc',
          status_code: ERROR_CODES,
        });
        const c = {};
        (data.entries || []).forEach(e => {
          const k = String(e.status);
          c[k] = (c[k] || 0) + 1;
        });
        counts.value = c;
        totalErr.value = data.total || 0;
      } catch (_) {}
    }

    onMounted(loadSummary);
    watch(() => store.tick.value, loadSummary);
    watch(() => store.apiKey.value, (v) => { if (v) loadSummary(); });

    const summaryChips = computed(() => {
      const tones = {
        '4': { class: 'tone-warn' },
        '5': { class: 'tone-err' },
      };
      return Object.entries(counts.value)
        .sort((a, b) => Number(a[0]) - Number(b[0]))
        .map(([code, n]) => ({
          code, count: n,
          tone: tones[code[0]] ? tones[code[0]].class : '',
        }));
    });

    return () => h('div', null, [
      h('div', { style: { display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(140px, 1fr))', gap: '10px', marginBottom: '14px' } },
        summaryChips.value.length
          ? summaryChips.value.map(c => h('div', { class: 'kpi', style: { minHeight: '76px', padding: '12px 14px' } }, [
              h('div', { class: 'kpi-label' }, `Status ${c.code}`),
              h('div', { class: 'kpi-value', style: { fontSize: '20px' } }, fmtNum(c.count)),
            ]))
          : h('div', { class: 'empty', style: { gridColumn: '1 / -1', padding: '40px 20px' } }, [
              h('div', { class: 'ico' }, '✓'),
              'No errors in recent sample',
            ])
      ),
      h(InnerRequests, { router: props.router }),
    ]);
  },
});
