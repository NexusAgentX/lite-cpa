import { defineComponent, h, ref, computed, watch } from '../vendor.js';
import { naive } from '../vendor.js';
import { fetchLogDetail } from '../api.js';

function esc(s) {
  return (s == null ? '' : String(s))
    .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

function hlJson(obj, indent) {
  indent = indent || 0;
  const pad = '  '.repeat(indent);
  const pad2 = '  '.repeat(indent + 1);
  if (obj === null) return '<span class="j-null">null</span>';
  const t = typeof obj;
  if (t === 'boolean') return `<span class="j-bool">${obj}</span>`;
  if (t === 'number') return `<span class="j-num">${obj}</span>`;
  if (t === 'string') {
    if (obj.length > 600) {
      return `<span class="j-str">"${esc(obj.slice(0, 600))}</span><span style="color:var(--err)">…(${obj.length - 600} chars trimmed)</span><span class="j-str">"</span>`;
    }
    return `<span class="j-str">"${esc(obj)}"</span>`;
  }
  if (Array.isArray(obj)) {
    if (!obj.length) return '<span class="j-brace">[]</span>';
    let r = '<span class="j-brace">[</span>\n';
    obj.forEach((v, i) => { r += pad2 + hlJson(v, indent + 1) + (i < obj.length - 1 ? ',' : '') + '\n'; });
    return r + pad + '<span class="j-brace">]</span>';
  }
  const keys = Object.keys(obj);
  if (!keys.length) return '<span class="j-brace">{}</span>';
  let r = '<span class="j-brace">{</span>\n';
  keys.forEach((k, i) => {
    r += pad2 + `<span class="j-key">"${esc(k)}"</span>: ` + hlJson(obj[k], indent + 1) + (i < keys.length - 1 ? ',' : '') + '\n';
  });
  return r + pad + '<span class="j-brace">}</span>';
}

function fmtBody(s) {
  if (!s) return '<span class="j-null">(empty)</span>';
  try { return hlJson(JSON.parse(s)); } catch (_) { return esc(s); }
}

export const DetailDrawer = defineComponent({
  name: 'DetailDrawer',
  props: {
    open: Boolean,
    id: [Number, String, null],
    onClose: Function,
  },
  setup(props) {
    const detail = ref(null);
    const loading = ref(false);
    const activeKey = ref('req');

    async function load() {
      if (!props.id) { detail.value = null; return; }
      loading.value = true;
      detail.value = null;
      try {
        detail.value = await fetchLogDetail(props.id);
      } catch (_) {}
      finally { loading.value = false; }
    }

    watch(() => props.id, load);

    const sections = computed(() => {
      const d = detail.value;
      if (!d) return [];
      return [
        { key: 'req',     label: 'Request Body',  text: d.request_body },
        { key: 'resp',    label: 'Response Body', text: d.response_body },
        { key: 'apireq',  label: 'API Request',   text: d.api_request },
        { key: 'apiresp', label: 'API Response',  text: d.api_response },
      ];
    });

    function copy(text) {
      try { navigator.clipboard.writeText(text || ''); } catch (_) {}
    }

    const { NDrawer, NDrawerContent, NTabs, NTabPane, NSpin, NButton } = naive;

    function renderMeta(d) {
      const items = [
        ['Time', d.time_str],
        ['Model', d.model],
        ['Protocol', d.protocol],
        ['Method', d.method || 'N/A'],
        ['Status', d.status],
        ['Duration', d.duration_ms != null ? d.duration_ms + ' ms' : 'N/A'],
        ['Input', (d.input_tokens || 0).toLocaleString()],
        ['Output', (d.output_tokens || 0).toLocaleString()],
        ['Cache read', (d.cache_read || 0).toLocaleString()],
        ['Cache create', (d.cache_create || 0).toLocaleString()],
        ['URL', d.url],
        ['User-Agent', d.ua || 'N/A'],
      ];
      return h('div', { class: 'detail-meta' },
        items.map(([k, v]) => h('div', { class: 'detail-meta-item' }, [
          h('strong', null, k),
          h('span', null, String(v ?? '')),
        ]))
      );
    }

    return () => h(NDrawer, {
      show: props.open,
      width: 760,
      placement: 'right',
      'onUpdate:show': (v) => { if (!v) props.onClose && props.onClose(); },
    }, () => h(NDrawerContent, {
      closable: true,
      title: detail.value ? `Log Detail · #${detail.value.id}` : 'Log Detail',
    }, () => {
      if (loading.value) return h(NSpin, { size: 'medium' });
      if (!detail.value) return h('div', { class: 'empty' }, 'No log selected.');
      const d = detail.value;
      return h('div', null, [
        renderMeta(d),
        h(NTabs, {
          value: activeKey.value,
          'onUpdate:value': (v) => { activeKey.value = v; },
          type: 'line',
          animated: true,
          size: 'small',
        }, () => sections.value.map(sect =>
          h(NTabPane, { name: sect.key, tab: sect.label }, () => h('div', { class: 'detail-section' }, [
            h('div', { class: 'detail-section-head' }, [
              h('span', { class: 'detail-section-title' }, `${sect.text ? (sect.text.length.toLocaleString() + ' chars') : 'empty'}`),
              h(NButton, {
                size: 'tiny',
                ghost: true,
                onClick: () => copy(sect.text),
              }, () => 'Copy'),
            ]),
            h('div', { class: 'detail-pre', innerHTML: fmtBody(sect.text) }),
          ]))
        )),
      ]);
    }));
  },
});
