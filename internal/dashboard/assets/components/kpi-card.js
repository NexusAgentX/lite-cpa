import { defineComponent, h, ref, onMounted, onBeforeUnmount, watch, shallowRef, markRaw } from '../vendor.js';
import { echarts } from '../vendor.js';

function fmtShort(n) {
  n = Number(n) || 0;
  const abs = Math.abs(n);
  if (abs >= 1e9) return (n / 1e9).toFixed(2) + 'B';
  if (abs >= 1e6) return (n / 1e6).toFixed(2) + 'M';
  if (abs >= 1e3) return (n / 1e3).toFixed(1) + 'K';
  return n.toLocaleString();
}

export const KpiCard = defineComponent({
  name: 'KpiCard',
  props: {
    label: String,
    value: [Number, String],
    sub: { type: String, default: '' },
    spark: { type: Array, default: () => [] },  // optional array of numbers
    accent: { type: String, default: '#6b8aff' },
    format: { type: String, default: 'short' }, // 'short' | 'raw' | 'percent'
  },
  setup(props) {
    const sparkEl = ref(null);
    const chart = shallowRef(null);

    function render() {
      if (!sparkEl.value || !echarts || !props.spark || props.spark.length < 2) return;
      if (!chart.value) {
        chart.value = markRaw(echarts.init(sparkEl.value, null, { renderer: 'svg' }));
      }
      chart.value.setOption({
        animation: false,
        grid: { left: 0, right: 0, top: 2, bottom: 0 },
        xAxis: { type: 'category', show: false, data: props.spark.map((_, i) => i) },
        yAxis: { type: 'value', show: false, min: 'dataMin', max: 'dataMax' },
        tooltip: { show: false },
        series: [{
          type: 'line',
          data: props.spark,
          smooth: true,
          symbol: 'none',
          lineStyle: { color: props.accent, width: 1.4 },
          areaStyle: { color: props.accent, opacity: 0.12 },
        }],
      });
    }

    onMounted(render);
    watch(() => [props.spark, props.accent], render);
    onBeforeUnmount(() => { if (chart.value) chart.value.dispose(); });

    const fmt = (v) => {
      if (props.format === 'raw') return Number(v || 0).toLocaleString();
      if (props.format === 'percent') return v;
      return fmtShort(v);
    };

    return () => h('div', { class: 'kpi' }, [
      h('div', { class: 'kpi-label' }, props.label),
      h('div', { class: 'kpi-value', style: { color: props.accent } }, fmt(props.value)),
      props.sub ? h('div', { class: 'kpi-sub' }, props.sub) : null,
      props.spark && props.spark.length >= 2
        ? h('div', { class: 'kpi-spark', ref: sparkEl })
        : null,
    ]);
  },
});
