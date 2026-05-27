import { defineComponent, h, ref, reactive, computed, watch, onMounted } from '../vendor.js';
import { useStore } from '../store.js';
import { fetchLogs } from '../api.js';
import { KpiCard } from '../components/kpi-card.js';
import { EChart } from '../components/echart.js';
import { fmtShort, fmtNum, chartBase, presetRange } from '../utils.js';

const SAMPLE_SIZE = 1000;

const TIME_PRESETS = [
  { key: '1h', label: '1H' },
  { key: '8h', label: '8H' },
  { key: 'today', label: 'Today' },
  { key: '7d', label: '7D' },
  { key: '30d', label: '1M' },
];

export const OverviewView = defineComponent({
  name: 'OverviewView',
  setup() {
    const store = useStore();
    const loading = ref(false);
    const totals = reactive({ req: 0, input: 0, output: 0, cache: 0 });
    const recent = ref([]);
    const timePreset = ref('1h');

    function timeParams() {
      const r = presetRange(timePreset.value);
      return r.time_from ? { time_from: r.time_from, time_to: r.time_to } : {};
    }

    async function load() {
      if (!store.apiKey.value) return;
      loading.value = true;
      try {
        const data = await fetchLogs({
          page: 1, per_page: SAMPLE_SIZE,
          sort_by: 'timestamp', sort_dir: 'desc',
          ...timeParams(),
        });
        totals.req    = data.total_req || 0;
        totals.input  = data.total_input || 0;
        totals.output = data.total_output || 0;
        totals.cache  = data.total_cache_read || 0;
        recent.value = data.entries || [];
        store.captureOptions(data);
        store.captureTotals(data);
        store.lastUpdate.value = new Date().toLocaleTimeString();
      } catch (_) {}
      finally { loading.value = false; }
    }

    onMounted(load);
    watch(() => store.tick.value, load);
    watch(() => store.apiKey.value, (v) => { if (v) load(); });
    watch(timePreset, load);

    const activePresetLabel = computed(() => TIME_PRESETS.find(p => p.key === timePreset.value)?.label || '');

    const cacheRate = computed(() => {
      const cr = totals.cache, ti = totals.input;
      if (!cr && !ti) return '0%';
      if (cr && !ti) return '100%';
      return (cr / (cr + ti) * 100).toFixed(1) + '%';
    });

    const hourly = computed(() => {
      const now = new Date();
      const preset = timePreset.value;

      let bucketSpan, bucketCount, labelFormat, base;
      switch (preset) {
        case '1h':
          bucketSpan = 60 * 1000;
          bucketCount = 60;
          base = new Date(now.getFullYear(), now.getMonth(), now.getDate(), now.getHours(), now.getMinutes());
          labelFormat = (d) => String(d.getHours()).padStart(2, '0') + ':' + String(d.getMinutes()).padStart(2, '0');
          break;
        case '8h':
          bucketSpan = 3600 * 1000;
          bucketCount = 8;
          base = new Date(now.getFullYear(), now.getMonth(), now.getDate(), now.getHours());
          labelFormat = (d) => String(d.getHours()).padStart(2, '0') + ':00';
          break;
        case 'today':
          bucketSpan = 3600 * 1000;
          bucketCount = now.getHours() + 1;
          base = new Date(now.getFullYear(), now.getMonth(), now.getDate(), now.getHours());
          labelFormat = (d) => String(d.getHours()).padStart(2, '0') + ':00';
          break;
        case '7d':
          bucketSpan = 86400 * 1000;
          bucketCount = 7;
          base = new Date(now.getFullYear(), now.getMonth(), now.getDate(), now.getHours());
          labelFormat = (d) => { const days = ['Sun','Mon','Tue','Wed','Thu','Fri','Sat']; return days[d.getDay()] + ' ' + d.getDate(); };
          break;
        case '30d':
          bucketSpan = 86400 * 1000;
          bucketCount = 30;
          base = new Date(now.getFullYear(), now.getMonth(), now.getDate(), now.getHours());
          labelFormat = (d) => { const m = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec']; return m[d.getMonth()] + ' ' + d.getDate(); };
          break;
        default:
          bucketSpan = 3600 * 1000;
          bucketCount = 24;
          base = new Date(now.getFullYear(), now.getMonth(), now.getDate(), now.getHours());
          labelFormat = (d) => String(d.getHours()).padStart(2, '0') + ':00';
      }

      const buckets = new Array(bucketCount).fill(0).map(() => ({ req: 0, input: 0, output: 0, models: {} }));
      const labels = [];
      for (let i = bucketCount - 1; i >= 0; i--) {
        const t = new Date(base.getTime() - i * bucketSpan);
        labels.push(labelFormat(t));
      }
      const earliest = base.getTime() - (bucketCount - 1) * bucketSpan;
      recent.value.forEach(e => {
        const ts = new Date(e.time || e.time_str);
        const t = ts.getTime();
        if (isNaN(t) || t < earliest) return;
        const idx = Math.floor((t - earliest) / bucketSpan);
        if (idx < 0 || idx >= bucketCount) return;
        const b = buckets[idx];
        b.req++;
        b.input  += e.input_tokens  || 0;
        b.output += e.output_tokens || 0;
        const m = e.model || 'unknown';
        b.models[m] = (b.models[m] || 0) + 1;
      });
      return { labels, buckets };
    });

    const requestSeriesOption = computed(() => {
      const t = store.chartTokens.value;
      const { labels, buckets } = hourly.value;

      const modelTotals = {};
      buckets.forEach(b => {
        Object.entries(b.models).forEach(([m, c]) => {
          modelTotals[m] = (modelTotals[m] || 0) + c;
        });
      });
      const sorted = Object.entries(modelTotals).sort((a, b) => b[1] - a[1]);
      const top5 = new Set(sorted.slice(0, 5).map(([m]) => m));

      const series = [];
      let ci = 0;
      top5.forEach(model => {
        series.push({
          name: model,
          type: 'line', smooth: true, symbol: 'none',
          stack: 'total',
          lineStyle: { color: t.palette[ci], width: 1.2 },
          areaStyle: { color: t.palette[ci], opacity: 0.2 },
          data: buckets.map(b => b.models[model] || 0),
        });
        ci++;
      });
      if (sorted.length > 5) {
        series.push({
          name: 'Others',
          type: 'line', smooth: true, symbol: 'none',
          stack: 'total',
          lineStyle: { color: t.palette[6], width: 1.2 },
          areaStyle: { color: t.palette[6], opacity: 0.2 },
          data: buckets.map(b => {
            let other = 0;
            Object.entries(b.models).forEach(([m, c]) => {
              if (!top5.has(m)) other += c;
            });
            return other;
          }),
        });
      }

      return {
        ...chartBase(t),
        grid: { left: 36, right: 12, top: 28, bottom: 26 },
        legend: { textStyle: { color: t.textDim, fontSize: 11 }, right: 0, top: 0 },
        xAxis: {
          type: 'category',
          data: labels,
          axisLine: { lineStyle: { color: t.axisLine } },
          axisLabel: { color: t.textDim, fontSize: 10, interval: 3 },
          axisTick: { show: false },
        },
        yAxis: {
          type: 'value',
          splitLine: { lineStyle: { color: t.splitLine } },
          axisLabel: { color: t.textDim, fontSize: 10 },
        },
        series,
      };
    });

    const topModelsOption = computed(() => {
      const t = store.chartTokens.value;
      const counts = {};
      recent.value.forEach(e => {
        const m = e.model || 'unknown';
        if (!counts[m]) counts[m] = { req: 0, input: 0, output: 0, cache: 0 };
        counts[m].req++;
        counts[m].input  += e.input_tokens  || 0;
        counts[m].output += e.output_tokens || 0;
        counts[m].cache  += e.cache_read    || 0;
      });
      const top = Object.entries(counts)
        .sort((a, b) => b[1].req - a[1].req)
        .slice(0, 10);
      const data = top.map(([name, v], i) => ({
        name,
        value: v.req,
        itemStyle: { color: t.palette[i % t.palette.length] },
      }));
      return {
        ...chartBase(t),
        grid: undefined,
        tooltip: {
          trigger: 'item',
          backgroundColor: t.tooltipBg, borderColor: t.tooltipBd,
          textStyle: { color: t.text, fontSize: 12 },
          formatter: (params) => {
            const v = top[params.dataIndex]?.[1];
            if (!v) return '';
            const total = v.input + v.cache;
            const rate = total ? (v.cache / total * 100).toFixed(1) : '0.0';
            return `<div style="font-weight:600;margin-bottom:4px">${params.name}</div>` +
              `<div>Requests: ${fmtNum(v.req)}</div>` +
              `<div>Input: ${fmtNum(v.input)}</div>` +
              `<div>Output: ${fmtNum(v.output)}</div>` +
              `<div>Cache: ${fmtNum(v.cache)}</div>` +
              `<div>Cache rate: ${rate}%</div>`;
          },
        },
        legend: { bottom: 0, textStyle: { color: t.textDim, fontSize: 11 } },
        series: [{
          type: 'pie',
          radius: ['52%', '76%'],
          center: ['50%', '44%'],
          avoidLabelOverlap: true,
          label: { show: false },
          labelLine: { show: false },
          itemStyle: { borderColor: t.pieBorder, borderWidth: 2 },
          data,
        }],
      };
    });

    function sparkOf(key) {
      return hourly.value.buckets.map(b => b[key]);
    }

    return () => {
      const t = store.chartTokens.value;
      const [cAccent, cOk, , cPurple] = t.palette;
      const cWarn = t.palette[2];
      return h('div', { class: 'page-grid', style: { gap: '16px' } }, [
      h('div', { style: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '4px' } }, [
        h('span', { style: { fontSize: '16px', fontWeight: 600, color: 'var(--text)' } }, 'Overview'),
        h('div', { class: 'preset-group' }, TIME_PRESETS.map(p =>
          h('button', {
            class: { active: timePreset.value === p.key },
            onClick: () => { timePreset.value = p.key; },
          }, p.label)
        )),
      ]),
      h('div', { style: { display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '12px' } }, [
        h(KpiCard, { label: 'Total Requests', value: totals.req, sub: `${recent.value.length} in last sample`, spark: sparkOf('req'), accent: cAccent }),
        h(KpiCard, { label: 'Input Tokens',   value: totals.input,  sub: fmtNum(totals.input)  + ' total', spark: sparkOf('input'),  accent: cOk }),
        h(KpiCard, { label: 'Output Tokens',  value: totals.output, sub: fmtNum(totals.output) + ' total', spark: sparkOf('output'), accent: cPurple }),
        h(KpiCard, { label: 'Cache Hit Rate', value: cacheRate.value, sub: fmtShort(totals.cache) + ' cache read', accent: cWarn, format: 'percent' }),
      ]),

      h('div', { class: 'panel' }, [
        h('div', { class: 'panel-head' }, [
          h('span', { class: 'panel-title' }, `Requests · ${activePresetLabel.value}`),
          h('span', { class: 'panel-meta' }, `stacked by model · sample ${recent.value.length}`),
        ]),
        h(EChart, { option: requestSeriesOption.value, height: '280px' }),
      ]),

      h('div', { class: 'panel' }, [
        h('div', { class: 'panel-head' }, [
          h('span', { class: 'panel-title' }, 'Top models'),
          h('span', { class: 'panel-meta' }, 'by request count · donut'),
        ]),
        h(EChart, { option: topModelsOption.value, height: '300px' }),
      ]),
    ]);
    };
  },
});