import { defineComponent, ref, computed, h } from '../vendor.js';
import { useStore } from '../store.js';

const TABS = [
  { name: 'overview', label: 'Overview', icon: '◇', sub: 'Dashboard summary' },
  { name: 'requests', label: 'Requests', icon: '☰', sub: 'API call logs' },
  { name: 'models',   label: 'Models',   icon: '◈', sub: 'Per-model usage' },
  { name: 'errors',   label: 'Errors',   icon: '!',  sub: '4xx / 5xx responses' },
  { name: 'settings', label: 'Settings', icon: '⚙',  sub: 'Preferences' },
];

export const AppShell = defineComponent({
  name: 'AppShell',
  props: {
    current: { type: String, required: true },
    onNav:   { type: Function, required: true },
    onRefresh: { type: Function, required: true },
  },
  setup(props, { slots }) {
    const collapsed = ref(false);
    const store = useStore();

    const activeTab = computed(() => TABS.find(t => t.name === props.current) || TABS[0]);

    function fmtShort(n) {
      n = n || 0;
      if (n >= 1e9) return (n / 1e9).toFixed(1) + 'B';
      if (n >= 1e6) return (n / 1e6).toFixed(1) + 'M';
      if (n >= 1e3) return (n / 1e3).toFixed(1) + 'K';
      return String(n);
    }

    return () => h('div', { class: 'shell' }, [
      h('aside', { class: ['sider', { collapsed: collapsed.value }] }, [
        h('div', { class: 'sider-head' }, [
          h('div', { class: 'sider-logo' }, '◆'),
          h('span', { class: 'sider-title' }, 'CPA Dashboard'),
        ]),
        h('nav', { class: 'sider-nav' }, TABS.map(t =>
          h('div', {
            class: ['nav-item', { active: t.name === props.current }],
            onClick: () => props.onNav(t.name),
          }, [
            h('span', { class: 'icon' }, t.icon),
            h('span', { class: 'label' }, t.label),
          ])
        )),
        h('div', { class: 'sider-foot' }, [
          h('span', null, fmtShort(store.totals.req) + ' req'),
          h('span', null, store.lastUpdate.value || '—'),
        ]),
      ]),
      h('main', { class: 'main' }, [
        h('header', { class: 'topbar' }, [
          h('button', {
            class: 'icon-btn',
            title: collapsed.value ? 'Expand' : 'Collapse',
            onClick: () => { collapsed.value = !collapsed.value; },
          }, '☰'),
          h('div', null, [
            h('div', { class: 'topbar-title' }, activeTab.value.label),
            h('span', { class: 'topbar-sub' }, activeTab.value.sub),
          ]),
          h('div', { class: 'topbar-spacer' }),
          h('span', { class: 'topbar-meta' },
            store.prefs.autoRefresh
              ? `Auto · ${store.prefs.refreshSeconds}s`
              : 'Auto off'
          ),
          h('button', {
            class: 'icon-btn',
            title: store.prefs.theme === 'light' ? 'Switch to dark' : 'Switch to light',
            onClick: () => store.toggleTheme(),
          }, store.prefs.theme === 'light' ? '☾' : '☀'),
          h('button', {
            class: 'icon-btn',
            title: 'Refresh now',
            onClick: () => props.onRefresh(),
          }, '↻'),
        ]),
        h('section', { class: 'content' }, slots.default ? slots.default() : []),
      ]),
    ]);
  },
});
