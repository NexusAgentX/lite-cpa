// Minimal hash router. Routes look like #/overview, #/requests?model=x
import { ref, reactive } from './vendor.js';

const ROUTES = ['overview', 'requests', 'models', 'errors', 'settings'];
const DEFAULT_ROUTE = 'overview';

function parseHash() {
  const raw = window.location.hash.replace(/^#\/?/, '');
  const [path, qs] = raw.split('?');
  const name = ROUTES.includes(path) ? path : DEFAULT_ROUTE;
  const query = {};
  if (qs) {
    new URLSearchParams(qs).forEach((v, k) => { query[k] = v; });
  }
  return { name, query };
}

export function createRouter() {
  const initial = parseHash();
  const current = ref(initial.name);
  const query = reactive({ ...initial.query });

  function sync() {
    const r = parseHash();
    current.value = r.name;
    Object.keys(query).forEach(k => delete query[k]);
    Object.assign(query, r.query);
  }

  window.addEventListener('hashchange', sync);

  function go(name, q) {
    let hash = '#/' + name;
    if (q && Object.keys(q).length) {
      const usp = new URLSearchParams();
      for (const k in q) {
        if (q[k] !== '' && q[k] !== null && q[k] !== undefined) usp.set(k, q[k]);
      }
      const s = usp.toString();
      if (s) hash += '?' + s;
    }
    window.location.hash = hash;
  }

  function setQuery(q) {
    go(current.value, q);
  }

  // Ensure URL has a hash on first load
  if (!window.location.hash) {
    window.location.hash = '#/' + DEFAULT_ROUTE;
  }

  return { current, query, go, setQuery, routes: ROUTES };
}
