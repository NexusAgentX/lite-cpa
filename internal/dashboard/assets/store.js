// App-wide reactive store. Provides auth, auto-refresh, options cache,
// and a tick counter that tabs subscribe to for synchronized refresh.
import { ref, reactive, computed, onMounted, onBeforeUnmount, watch } from './vendor.js';
import { getApiKey, setApiKey as apiSetKey, clearApiKey as apiClearKey, onAuthError, fetchLogs } from './api.js';

const PREF_KEY = 'cpa_dashboard_prefs';
const DEFAULT_PREFS = {
  autoRefresh: true,
  refreshSeconds: 5,
  perPage: 20,
  theme: 'dark',
};

// Chart palettes per theme. Kept in store so chart `computed`s re-render
// automatically when the user flips the toggle — the components don't need
// any extra wiring beyond reading `store.chartTokens.value`.
const CHART_TOKENS = {
  dark: {
    text:        '#abb2bf',
    textDim:     '#828997',
    textMute:    '#5c6370',
    axisLine:    '#3e4451',
    splitLine:   '#2c313a',
    tooltipBg:   '#21252b',
    tooltipBd:   '#3e4451',
    pieBorder:   '#21252b',
    palette: ['#61afef', '#98c379', '#e5c07b', '#c678dd', '#56b6c2', '#e06c75', '#abb2bf'],
    // Soft area-fill stops for line charts. RGBA derived from --accent.
    areaFill: [
      'rgba(97,175,239,.35)',
      'rgba(97,175,239,.02)',
    ],
  },
  light: {
    text:        '#383a42',
    textDim:     '#696c77',
    textMute:    '#a0a1a7',
    axisLine:    '#bfc0c2',
    splitLine:   '#e5e5e6',
    tooltipBg:   '#ffffff',
    tooltipBd:   '#d4d4d5',
    pieBorder:   '#ffffff',
    palette: ['#4078f2', '#50a14f', '#c18401', '#a626a4', '#0184bc', '#e45649', '#383a42'],
    areaFill: [
      'rgba(64,120,242,.25)',
      'rgba(64,120,242,.01)',
    ],
  },
};

function loadPrefs() {
  try {
    const raw = localStorage.getItem(PREF_KEY);
    if (!raw) return { ...DEFAULT_PREFS };
    return { ...DEFAULT_PREFS, ...JSON.parse(raw) };
  } catch (_) { return { ...DEFAULT_PREFS }; }
}
function savePrefs(p) { localStorage.setItem(PREF_KEY, JSON.stringify(p)); }

let _singleton = null;

export function createStore() {
  if (_singleton) return _singleton;

  const apiKey = ref(getApiKey());
  const authError = ref(false);
  const prefs = reactive(loadPrefs());
  const tick = ref(0);
  const lastUpdate = ref('');

  // Options cache (model/protocol/method) fed by any logs response that
  // includes them — backend only emits them on page=1 with no filters,
  // so we keep last good values around for the dropdowns.
  const options = reactive({
    models: [],
    protocols: [],
    methods: [],
  });

  // Stats cache for sider footer / topbar
  const totals = reactive({
    req: 0, input: 0, output: 0, cache: 0,
  });

  function captureOptions(resp) {
    if (resp.model_options) options.models = resp.model_options;
    if (resp.protocol_options) options.protocols = resp.protocol_options;
    if (resp.method_options) options.methods = resp.method_options;
  }
  function captureTotals(resp) {
    totals.req = resp.total_req || 0;
    totals.input = resp.total_input || 0;
    totals.output = resp.total_output || 0;
    totals.cache = resp.total_cache_read || 0;
  }

  function setApiKey(k) {
    apiSetKey(k);
    apiKey.value = k;
    authError.value = false;
  }
  function clearApiKey() {
    apiClearKey();
    apiKey.value = '';
  }
  onAuthError(() => {
    authError.value = true;
    apiKey.value = '';
  });

  // Auto-refresh global tick
  let timer = null;
  function startTimer() {
    stopTimer();
    if (!prefs.autoRefresh) return;
    const ms = Math.max(2, prefs.refreshSeconds | 0) * 1000;
    timer = setInterval(() => { tick.value++; }, ms);
  }
  function stopTimer() {
    if (timer) { clearInterval(timer); timer = null; }
  }
  watch(() => [prefs.autoRefresh, prefs.refreshSeconds], startTimer, { immediate: false });
  watch(prefs, () => savePrefs({ ...prefs }), { deep: true });

  // Reactive theme tokens — chart `computed`s subscribe to this and re-render
  // when the theme flips.
  const chartTokens = computed(() => CHART_TOKENS[prefs.theme] || CHART_TOKENS.dark);

  // Reflect theme onto <html data-theme="..."> so CSS variables pick it up.
  function applyTheme() {
    const t = prefs.theme === 'light' ? 'light' : 'dark';
    document.documentElement.setAttribute('data-theme', t);
  }
  watch(() => prefs.theme, applyTheme, { immediate: false });
  applyTheme();

  function toggleTheme() {
    prefs.theme = prefs.theme === 'light' ? 'dark' : 'light';
  }

  // Prime options + totals once at startup, and again whenever auth becomes
  // valid (so the sidebar/global stats appear without forcing the user to
  // visit Overview first).
  async function primeOptions() {
    if (!apiKey.value) return;
    try {
      const data = await fetchLogs({ page: 1, per_page: 1, sort_by: 'timestamp', sort_dir: 'desc' });
      captureOptions(data);
      captureTotals(data);
      lastUpdate.value = new Date().toLocaleTimeString();
    } catch (_) {}
  }

  watch(apiKey, (v) => { if (v) primeOptions(); });

  // Lifecycle hook outside Vue: kick off the timer when first consumed.
  startTimer();
  if (apiKey.value) primeOptions();

  _singleton = {
    apiKey, authError,
    prefs,
    options, totals,
    tick, lastUpdate,
    chartTokens, toggleTheme,
    setApiKey, clearApiKey,
    captureOptions, captureTotals,
    primeOptions,
  };
  return _singleton;
}

export function useStore() { return createStore(); }
