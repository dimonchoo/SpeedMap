import { reactive, watch } from 'vue';
import { speedMapApp } from '../app.js';

let instance = null;

export function createAppStore() {
  if (!instance) {
    const raw = speedMapApp();
    instance = reactive(raw);
    for (const key of Object.keys(raw)) {
      const desc = Object.getOwnPropertyDescriptor(raw, key);
      if (desc && typeof desc.value === 'function') {
        instance[key] = desc.value.bind(instance);
      }
    }

    // Vue 3 Watchers
    watch(() => instance.imageFilterTab, () => {
      instance.imagePage = 1;
      instance.updateFilteredImages();
    });

    watch(() => instance.imageSearchQuery, () => {
      instance.imagePage = 1;
      instance.updateFilteredImages();
    });

    watch(() => instance.imageSortKey, () => {
      instance.imagePage = 1;
      instance.updateFilteredImages();
    });

    watch(() => instance.activeTab, (tab) => {
      if (tab === 'images') {
        instance.imagePage = 1;
        instance.updateFilteredImages();
      }
    });

    watch(() => instance.fontFilterTab, () => { instance.fontPage = 1; });
    watch(() => instance.fontSortKey, () => { instance.fontPage = 1; });
    watch(() => instance.iframeSearchQuery, () => { instance.iframePage = 1; });
    watch(() => instance.formSearchQuery, () => { instance.formPage = 1; });
    watch(() => instance.formEngineFilter, () => { instance.formPage = 1; });
    watch(() => instance.runsDiffSearch, () => { instance.diffCurrentPage = 1; });
    watch(() => instance.runsDiffFilter, () => { instance.diffCurrentPage = 1; });
    watch(() => instance.diffViewMode, () => { instance.diffCurrentPage = 1; });
  }
  return instance;
}

export function useApp() {
  if (!instance) {
    return createAppStore();
  }
  return instance;
}
