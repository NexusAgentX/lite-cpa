import { createApp, defineComponent, h, ref, computed, watch } from './vendor.js';
import { naive } from './vendor.js';
import { useStore } from './store.js';
import { createRouter } from './router.js';
import { AppShell } from './components/app-shell.js';
import { OverviewView } from './views/overview.js';
import { RequestsView } from './views/requests.js';
import { ModelsView }   from './views/models.js';
import { ErrorsView }   from './views/errors.js';
import { SettingsView } from './views/settings.js';

const VIEWS = {
  overview: OverviewView,
  requests: RequestsView,
  models:   ModelsView,
  errors:   ErrorsView,
  settings: SettingsView,
};

const AuthScreen = defineComponent({
  name: 'AuthScreen',
  props: { error: Boolean },
  setup(props) {
    const draft = ref('');
    const store = useStore();
    function submit(e) {
      e.preventDefault();
      const k = draft.value.trim();
      if (!k) return;
      store.setApiKey(k);
    }
    return () => h('div', { class: 'auth-wrap' }, [
      h('div', { class: 'auth-card' }, [
        h('div', { class: 'auth-logo' }, '◆'),
        h('h1', null, 'CPA Dashboard'),
        h('p', null, 'Enter your API key to continue'),
        props.error ? h('div', { class: 'err-msg' }, 'Invalid key — try again') : null,
        h('form', { onSubmit: submit }, [
          h('input', {
            type: 'password',
            placeholder: 'sk-…',
            value: draft.value,
            onInput: (e) => { draft.value = e.target.value; },
            autofocus: true,
          }),
          h('button', { type: 'submit' }, 'Sign in'),
        ]),
      ]),
    ]);
  },
});

// Naive UI theme overrides. Mirror our CSS tokens so Naive's built-ins
// blend with the rest of the page in both One Dark and One Light.
const THEME_OVERRIDES = {
  dark: {
    common: {
      primaryColor: '#61afef',
      primaryColorHover: '#82c0ff',
      primaryColorPressed: '#528bcc',
      primaryColorSuppl: '#61afef',
      bodyColor: '#282c34',
      cardColor: '#21252b',
      modalColor: '#21252b',
      popoverColor: '#21252b',
      borderColor: '#181a1f',
      dividerColor: '#181a1f',
      textColorBase: '#abb2bf',
      textColor1: '#abb2bf',
      textColor2: '#abb2bf',
      textColor3: '#828997',
      placeholderColor: '#5c6370',
      fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Inter, system-ui, sans-serif',
      fontFamilyMono: '"SF Mono", "JetBrains Mono", ui-monospace, Menlo, Consolas, monospace',
      fontSize: '13px',
      borderRadius: '8px',
      borderRadiusSmall: '6px',
    },
    DataTable: {
      thColor: '#2c313a',
      tdColor: '#21252b',
      tdColorHover: '#2c313a',
      tdColorStriped: '#262a32',
      thTextColor: '#828997',
      thFontWeight: '600',
      borderColor: '#181a1f',
    },
    Drawer: {
      color: '#21252b',
      headerBorderBottom: '1px solid #181a1f',
    },
    Pagination: {
      itemColorHover: '#2c313a',
      itemColorActive: 'rgba(97,175,239,.14)',
      itemTextColorActive: '#61afef',
    },
    Select: {
      peers: {
        InternalSelection: {
          color: '#282c34',
          colorActive: '#282c34',
          border: '1px solid #181a1f',
          borderHover: '1px solid #3e4451',
          borderActive: '1px solid #61afef',
          borderFocus: '1px solid #61afef',
        },
      },
    },
    Input: {
      color: '#282c34',
      colorFocus: '#282c34',
      border: '1px solid #181a1f',
      borderHover: '1px solid #3e4451',
      borderFocus: '1px solid #61afef',
    },
  },
  light: {
    common: {
      primaryColor: '#4078f2',
      primaryColorHover: '#2a66e8',
      primaryColorPressed: '#2f5fc7',
      primaryColorSuppl: '#4078f2',
      bodyColor: '#fafafa',
      cardColor: '#ffffff',
      modalColor: '#ffffff',
      popoverColor: '#ffffff',
      borderColor: '#d4d4d5',
      dividerColor: '#d4d4d5',
      textColorBase: '#383a42',
      textColor1: '#383a42',
      textColor2: '#383a42',
      textColor3: '#696c77',
      placeholderColor: '#a0a1a7',
      fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Inter, system-ui, sans-serif',
      fontFamilyMono: '"SF Mono", "JetBrains Mono", ui-monospace, Menlo, Consolas, monospace',
      fontSize: '13px',
      borderRadius: '8px',
      borderRadiusSmall: '6px',
    },
    DataTable: {
      thColor: '#f0f0f1',
      tdColor: '#ffffff',
      tdColorHover: '#f0f0f1',
      tdColorStriped: '#f7f7f8',
      thTextColor: '#696c77',
      thFontWeight: '600',
      borderColor: '#d4d4d5',
    },
    Drawer: {
      color: '#ffffff',
      headerBorderBottom: '1px solid #d4d4d5',
    },
    Pagination: {
      itemColorHover: '#f0f0f1',
      itemColorActive: 'rgba(64,120,242,.10)',
      itemTextColorActive: '#4078f2',
    },
    Select: {
      peers: {
        InternalSelection: {
          color: '#ffffff',
          colorActive: '#ffffff',
          border: '1px solid #d4d4d5',
          borderHover: '1px solid #bfc0c2',
          borderActive: '1px solid #4078f2',
          borderFocus: '1px solid #4078f2',
        },
      },
    },
    Input: {
      color: '#ffffff',
      colorFocus: '#ffffff',
      border: '1px solid #d4d4d5',
      borderHover: '1px solid #bfc0c2',
      borderFocus: '1px solid #4078f2',
    },
  },
};

const Root = defineComponent({
  name: 'Root',
  setup() {
    const store = useStore();
    const router = createRouter();

    const showAuth = computed(() => !store.apiKey.value || store.authError.value);

    function triggerRefresh() { store.tick.value++; }

    const { NConfigProvider, darkTheme } = naive;

    const themeMode = computed(() => store.prefs.theme === 'light' ? 'light' : 'dark');
    const activeTheme = computed(() => themeMode.value === 'light' ? null : darkTheme);
    const activeOverrides = computed(() => THEME_OVERRIDES[themeMode.value]);

    return () => h(NConfigProvider, {
      theme: activeTheme.value,
      themeOverrides: activeOverrides.value,
    }, () => showAuth.value
      ? h(AuthScreen, { error: store.authError.value })
      : h(AppShell, {
          current: router.current.value,
          onNav: (n) => router.go(n),
          onRefresh: triggerRefresh,
        }, {
          default: () => {
            const View = VIEWS[router.current.value] || OverviewView;
            // key= forces re-mount on tab switch, which keeps each view's
            // internal state isolated and predictable.
            return h(View, { key: router.current.value, router });
          },
        })
    );
  },
});

const app = createApp(Root);
if (naive && naive.install) app.use(naive);
app.mount('#app');
