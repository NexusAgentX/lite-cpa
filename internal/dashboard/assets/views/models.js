import { defineComponent, h, ref, computed, watch, onMounted } from '../vendor.js';
import { naive } from '../vendor.js';
import { useStore } from '../store.js';
import { fetchLogs } from '../api.js';
import { EChart } from '../components/echart.js';
import { fmtShort, fmtNum, fmtMs, chartBase } from '../utils.js';

const SAMPLE_SIZE = 1000;

function p95(arr) {
  if (!arr.length) return 0;
  const sorted = [...arr].sort((a, b) => a - b);
  const idx = Math.min(sorted.length - 1, Math.floor(sorted.length * 0.95));
  return sorted[idx];
}

export const ModelsView = defineComponent({
  name: 'ModelsView',
  props: { router: { type: Object, required: true } },
  setup(props) {
    const store = useStore();
    const loading = ref(false);
    const rows = ref([]);

    async function load() {
      if (!store.apiKey.value) return;
      loading.value = true;
      try {
        const data = await fetchLogs({
          page: 1, per_page: SAMPLE_SIZE,
          sort_by: 'timestamp', sort_dir: 'desc',
        });
        store.captureOptions(data);
        store.captureTotals(data);
        // Aggregate per model
        const groups = new Map();
        (data.entries || []).forEach(e => {
          const m = e.model || 'unknown';
          if (!groups.has(m)) groups.set(m, { model: m, req: 0, input: 0, output: 0, cache: 0, dur: [], errors: 0 });
          const g = groups.get(m);
          g.req++;
          g.input  += e.input_tokens  || 0;
          g.output += e.output_tokens || 0;
          g.cache  += e.cache_read    || 0;
          if (e.duration_ms != null) g.dur.push(e.duration_ms);
          if (e.status >= 400) g.errors++;
        });
        rows.value = [...groups.values()].map(g => ({
          ...g,
          avg: g.dur.length ? Math.round(g.dur.reduce((a, b) => a + b, 0) / g.dur.length) : 0,
          p95: p95(g.dur),
          errPct: g.req ? (g.errors / g.req * 100) : 0,
        })).sort((a, b) => b.req - a.req);
      } catch (_) {}
      finally { loading.value = false; }
    }

    onMounted(load);
    watch(() => store.tick.value, load);
    watch(() => store.apiKey.value, (v) => { if (v) load(); });

    const tokenChart = computed(() => {
      const t = store.chartTokens.value;
      const top = rows.value.slice(0, 10);
      return {
        ...chartBase(t),
        grid: { left: 60, right: 20, top: 36, bottom: 80 },
        legend: { textStyle: { color: t.textDim, fontSize: 11 }, top: 0, right: 0 },
        tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' }, backgroundColor: t.tooltipBg, borderColor: t.tooltipBd, textStyle: { color: t.text } },
        xAxis: {
          type: 'category',
          data: top.map(r => r.model),
          axisLine: { show: false },
          axisLabel: { color: t.text, fontSize: 10, rotate: 30 },
          axisTick: { show: false },
        },
        yAxis: {
          type: 'value',
          axisLine: { show: false },
          axisLabel: { color: t.textDim, fontSize: 10, formatter: (v) => fmtShort(v) },
          splitLine: { lineStyle: { color: t.splitLine } },
        },
        series: [
          {
            name: 'Input', type: 'bar', stack: 'tok', barWidth: 18,
            itemStyle: { color: t.palette[1] },
            data: top.map(r => r.input),
          },
          {
            name: 'Output', type: 'bar', stack: 'tok', barWidth: 18,
            itemStyle: { color: t.palette[3] },
            data: top.map(r => r.output),
          },
          {
            name: 'Cache', type: 'bar', stack: 'tok', barWidth: 18,
            itemStyle: { color: t.palette[2] },
            data: top.map(r => r.cache),
          },
        ],
      };
    });

    function jumpToRequests(model) {
      props.router.go('requests', { model });
    }

    const { NDataTable } = naive;

    const columns = [
      {
        title: 'Model', key: 'model',
        render: (row) => h('a', {
          href: 'javascript:void(0)',
          style: { color: 'var(--accent)', textDecoration: 'none', fontWeight: 500 },
          onClick: () => jumpToRequests(row.model),
        }, row.model),
      },
      { title: 'Req',    key: 'req', align: 'right', width: 80,  render: (r) => h('span', { class: 'cell-num' }, fmtNum(r.req)) },
      { title: 'Input',  key: 'input',  align: 'right', width: 100, render: (r) => h('span', { class: 'cell-num', style: { color: 'var(--ok)' } }, fmtShort(r.input)) },
      { title: 'Output', key: 'output', align: 'right', width: 100, render: (r) => h('span', { class: 'cell-num', style: { color: 'var(--purple)' } }, fmtShort(r.output)) },
      { title: 'Cache',  key: 'cache',  align: 'right', width: 100, render: (r) => h('span', { class: 'cell-num', style: { color: 'var(--warn)' } }, fmtShort(r.cache)) },
      { title: 'Avg',    key: 'avg', align: 'right', width: 90,  render: (r) => h('span', { class: 'cell-num cell-dim' }, fmtMs(r.avg)) },
      { title: 'P95',    key: 'p95', align: 'right', width: 90,  render: (r) => h('span', { class: 'cell-num cell-dim' }, fmtMs(r.p95)) },
      {
        title: 'Errors', key: 'errPct', align: 'right', width: 90,
        render: (r) => {
          const tone = r.errPct === 0 ? 'cell-dim' : r.errPct < 5 ? 'cell-num' : '';
          const color = r.errPct === 0 ? 'var(--text-mute)' : r.errPct < 5 ? 'var(--warn)' : 'var(--err)';
          return h('span', { class: ['cell-num', tone], style: { color } }, r.errPct.toFixed(1) + '%');
        },
      },
    ];

    return () => h('div', null, [
      h('div', { class: 'panel', style: { marginBottom: '14px' } }, [
        h('div', { class: 'panel-head' }, [
          h('span', { class: 'panel-title' }, 'Token usage by model'),
          h('span', { class: 'panel-meta' }, `top 10 of ${rows.value.length} · sample ${SAMPLE_SIZE}`),
        ]),
        h(EChart, { option: tokenChart.value, height: '320px' }),
      ]),
      h('div', { class: 'panel', style: { padding: 0, overflow: 'hidden' } }, [
        h(NDataTable, {
          columns,
          data: rows.value,
          loading: loading.value,
          size: 'small',
          bordered: false,
          singleLine: true,
          maxHeight: 480,
          rowKey: (row) => row.model,
        }),
      ]),
    ]);
  },
});
