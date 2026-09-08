import { createConfigModule } from './modules/config.js';
import { createNotificationsModule } from './modules/notifications.js';
import { createScannerModule } from './modules/scanner.js';
import { createAnalyticsModule } from './modules/analytics.js';
import { createImagesModule } from './modules/images.js';
import { createStudioModule } from './modules/studio.js';
import { createFormsModule } from './modules/forms.js';
import { createFontsModule } from './modules/fonts.js';
import { createIframesModule } from './modules/iframes.js';
import { createDiffModule } from './modules/diff.js';
import { createProfilesModule } from './modules/profiles.js';
import { createCloudModule } from './modules/cloud.js';
import { createShortcutsModule } from './modules/shortcuts.js';

/**
 * Safely merges module objects preserving property descriptors (getters/setters)
 * without triggering eager evaluation or converting getters into static properties.
 */
function mergeModules(target, ...modules) {
  for (const mod of modules) {
    if (mod) {
      Object.defineProperties(target, Object.getOwnPropertyDescriptors(mod));
    }
  }
  return target;
}

/**
 * speedMapApp - Root application store composition.
 * Aggregates domain-driven feature modules following the Single Responsibility Principle.
 */
export function speedMapApp() {
  const store = {};
  mergeModules(
    store,
    createConfigModule(),
    createNotificationsModule(),
    createScannerModule(),
    createAnalyticsModule(),
    createImagesModule(),
    createStudioModule(),
    createFormsModule(),
    createFontsModule(),
    createIframesModule(),
    createDiffModule(),
    createProfilesModule(),
    createCloudModule(),
    createShortcutsModule()
  );

  store.initApp = function() {
    console.log("[JS LOG] speedMapApp modularized and initialized.");
    this.loadSavedConfig?.();
    this.loadSiteProfiles?.();
    this.checkGDriveStatus?.();
    this.initShortcuts?.();
    this.initScannerEvents?.();
    this.addLog?.('info', 'SpeedMap додаток готовий до роботи.');
  };

  return store;
}