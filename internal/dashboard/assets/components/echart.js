// Generic ECharts wrapper. Owns the canvas, watches the option, resizes on viewport changes.
import { defineComponent, h, ref, onMounted, onBeforeUnmount, watch, shallowRef, markRaw } from '../vendor.js';
import { echarts } from '../vendor.js';

export const EChart = defineComponent({
  name: 'EChart',
  props: {
    option: { type: Object, required: true },
    height: { type: String, default: '260px' },
    klass:  { type: String, default: '' },
  },
  setup(props) {
    const el = ref(null);
    const chart = shallowRef(null);

    function ensure() {
      if (!el.value || !echarts) return null;
      if (!chart.value) {
        chart.value = markRaw(echarts.init(el.value, null, { renderer: 'canvas' }));
      }
      return chart.value;
    }

    function apply() {
      const c = ensure();
      if (!c) return;
      c.setOption(props.option, { notMerge: true });
    }

    function onResize() { if (chart.value) chart.value.resize(); }

    onMounted(() => { apply(); window.addEventListener('resize', onResize); });
    onBeforeUnmount(() => {
      window.removeEventListener('resize', onResize);
      if (chart.value) { chart.value.dispose(); chart.value = null; }
    });
    watch(() => props.option, apply, { deep: true });

    return () => h('div', {
      ref: el,
      class: ['chart-box', props.klass],
      style: { height: props.height },
    });
  },
});
