import { reactive } from 'vue';
import { speedMapApp } from '../app.js';

let instance = null;

export function createAppStore() {
  if (!instance) {
    const raw = speedMapApp();
    instance = reactive(raw);
    for (const key of Object.keys(raw)) {
      if (typeof raw[key] === 'function') {
        instance[key] = raw[key].bind(instance);
      }
    }
  }
  return instance;
}

export function useApp() {
  if (!instance) {
    return createAppStore();
  }
  return instance;
}
