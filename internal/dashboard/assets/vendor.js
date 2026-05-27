// Single re-export point for globals loaded via <script> tags in index.html.
// This lets us write `import { ref } from './vendor.js'` and keeps the
// "where does Vue come from" answer in one file.
const V = window.Vue || {};
export const {
  createApp, defineComponent, h,
  ref, reactive, computed, watch, watchEffect,
  onMounted, onBeforeUnmount, nextTick, provide, inject, shallowRef, markRaw,
  Fragment, Teleport, Transition,
} = V;

export const naive = window.naive || {};
export const echarts = window.echarts || null;
