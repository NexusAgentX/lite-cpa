import { defineComponent, h, ref, computed } from '../vendor.js';
import { naive } from '../vendor.js';
import { useStore } from '../store.js';
import { fmtNum } from '../utils.js';

export const SettingsView = defineComponent({
  name: 'SettingsView',
  setup() {
    const store = useStore();
    const { NSwitch, NInputNumber, NSelect, NButton, NInput } = naive;

    const keyDraft = ref('');
    const keyMessage = ref('');

    function saveKey() {
      if (!keyDraft.value.trim()) return;
      store.setApiKey(keyDraft.value.trim());
      keyDraft.value = '';
      keyMessage.value = 'API key updated.';
      setTimeout(() => { keyMessage.value = ''; }, 2500);
    }
    function clearKey() {
      store.clearApiKey();
      keyMessage.value = 'API key cleared. Refresh to log in again.';
    }

    return () => h('div', { style: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(360px, 1fr))', gap: '14px', maxWidth: '900px' } }, [

      h('div', { class: 'panel' }, [
        h('div', { class: 'panel-head' }, [
          h('span', { class: 'panel-title' }, 'API Key'),
          store.apiKey.value
            ? h('span', { class: 'panel-meta', style: { color: 'var(--ok)' } }, '● Authenticated')
            : h('span', { class: 'panel-meta', style: { color: 'var(--err)' } }, '○ Not set'),
        ]),
        h('div', { class: 'stack' }, [
          h(NInput, {
            type: 'password', size: 'small', placeholder: 'sk-…',
            value: keyDraft.value,
            showPasswordOn: 'click',
            'onUpdate:value': (v) => { keyDraft.value = v; },
          }),
          h('div', { class: 'cluster' }, [
            h(NButton, { type: 'primary', size: 'small', onClick: saveKey }, () => 'Save'),
            h(NButton, { quaternary: true, size: 'small', onClick: clearKey }, () => 'Clear'),
            keyMessage.value ? h('span', { style: { fontSize: '12px', color: 'var(--text-dim)' } }, keyMessage.value) : null,
          ]),
        ]),
      ]),

      h('div', { class: 'panel' }, [
        h('div', { class: 'panel-head' }, [
          h('span', { class: 'panel-title' }, 'Auto refresh'),
          h('span', { class: 'panel-meta' }, store.prefs.autoRefresh ? 'enabled' : 'disabled'),
        ]),
        h('div', { class: 'stack' }, [
          h('div', { class: 'cluster' }, [
            h(NSwitch, {
              value: store.prefs.autoRefresh,
              'onUpdate:value': (v) => { store.prefs.autoRefresh = v; },
            }),
            h('span', { style: { fontSize: '12px', color: 'var(--text-dim)' } }, 'Refresh data periodically'),
          ]),
          h('div', { class: 'cluster' }, [
            h('span', { style: { fontSize: '12px', color: 'var(--text-dim)', minWidth: '90px' } }, 'Interval'),
            h(NSelect, {
              size: 'small',
              value: store.prefs.refreshSeconds,
              options: [
                { label: '5 seconds', value: 5 },
                { label: '15 seconds', value: 15 },
                { label: '30 seconds', value: 30 },
                { label: '1 minute', value: 60 },
                { label: '5 minutes', value: 300 },
              ],
              style: { minWidth: '160px' },
              'onUpdate:value': (v) => { store.prefs.refreshSeconds = v; },
            }),
          ]),
        ]),
      ]),

      h('div', { class: 'panel' }, [
        h('div', { class: 'panel-head' }, [
          h('span', { class: 'panel-title' }, 'Table defaults'),
        ]),
        h('div', { class: 'cluster' }, [
          h('span', { style: { fontSize: '12px', color: 'var(--text-dim)', minWidth: '90px' } }, 'Rows per page'),
          h(NSelect, {
            size: 'small',
            value: store.prefs.perPage,
            options: [
              { label: '20', value: 20 },
              { label: '50', value: 50 },
              { label: '100', value: 100 },
            ],
            style: { minWidth: '120px' },
            'onUpdate:value': (v) => { store.prefs.perPage = v; },
          }),
        ]),
      ]),

      h('div', { class: 'panel' }, [
        h('div', { class: 'panel-head' }, [
          h('span', { class: 'panel-title' }, 'Snapshot'),
        ]),
        h('div', { class: 'stack', style: { fontSize: '12px', fontFamily: 'var(--mono)', color: 'var(--text-dim)' } }, [
          h('div', null, ['Total requests: ', h('strong', { style: { color: 'var(--text)', marginLeft: '4px' } }, fmtNum(store.totals.req))]),
          h('div', null, ['Input tokens:   ',  h('strong', { style: { color: 'var(--text)', marginLeft: '4px' } }, fmtNum(store.totals.input))]),
          h('div', null, ['Output tokens:  ', h('strong', { style: { color: 'var(--text)', marginLeft: '4px' } }, fmtNum(store.totals.output))]),
          h('div', null, ['Cache read:     ',     h('strong', { style: { color: 'var(--text)', marginLeft: '4px' } }, fmtNum(store.totals.cache))]),
          h('div', null, ['Models known:   ',   h('strong', { style: { color: 'var(--text)', marginLeft: '4px' } }, String(store.options.models.length))]),
          h('div', null, ['Last update:    ',    h('strong', { style: { color: 'var(--text)', marginLeft: '4px' } }, store.lastUpdate.value || '—')]),
        ]),
      ]),
    ]);
  },
});
