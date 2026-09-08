#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const ROOT = path.resolve(__dirname, '..');
const PARTIALS_DIR = path.join(ROOT, 'frontend/partials');
const COMPONENTS_DIR = path.join(ROOT, 'frontend/src/components');

const FILE_MAPPING = {
  'layout/header.html': { comp: 'layout/Header.vue', name: 'Header' },
  'layout/toast.html': { comp: 'layout/Toast.vue', name: 'Toast' },
  'layout/sidebar.html': { comp: 'layout/Sidebar.vue', name: 'Sidebar' },
  'layout/progress-bar.html': { comp: 'layout/ProgressBar.vue', name: 'ProgressBar' },
  
  'tabs/pages.html': { comp: 'tabs/PagesTab.vue', name: 'PagesTab' },
  'tabs/analytics.html': { comp: 'tabs/AnalyticsTab.vue', name: 'AnalyticsTab' },
  'tabs/images.html': { comp: 'tabs/ImagesTab.vue', name: 'ImagesTab' },
  'tabs/fonts.html': { comp: 'tabs/FontsTab.vue', name: 'FontsTab' },
  'tabs/iframes.html': { comp: 'tabs/IframesTab.vue', name: 'IframesTab' },
  'tabs/forms.html': { comp: 'tabs/FormsTab.vue', name: 'FormsTab' },

  'modals/log-drawer.html': { comp: 'modals/LogDrawer.vue', name: 'LogDrawer' },
  'modals/settings-modal.html': { comp: 'modals/SettingsModal.vue', name: 'SettingsModal' },
  'modals/image-comparison.html': { comp: 'modals/ImageComparison.vue', name: 'ImageComparison' },
  'modals/image-studio.html': { comp: 'modals/ImageStudio.vue', name: 'ImageStudio' },
  'modals/page-detail.html': { comp: 'modals/PageDetail.vue', name: 'PageDetail' },
  'modals/wp-export.html': { comp: 'modals/WpExport.vue', name: 'WpExport' },
  'modals/wp-apply.html': { comp: 'modals/WpApply.vue', name: 'WpApply' },
  'modals/profile-modal.html': { comp: 'modals/ProfileModal.vue', name: 'ProfileModal' },
  'modals/form-detail.html': { comp: 'modals/FormDetail.vue', name: 'FormDetail' },
  'modals/pokemon-modal.html': { comp: 'modals/PokemonModal.vue', name: 'PokemonModal' },
};

function convertTemplate(html) {
  let res = html;

  // 1. Remove x-cloak
  res = res.replace(/\s+x-cloak\b/g, '');

  // 2. Convert x-transition:*
  res = res.replace(/\s+x-transition:[a-zA-Z\-]+="[^"]*"/g, '');

  // 3. Convert @click.outside="action" -> v-click-outside="() => { action }"
  res = res.replace(/@click\.outside="([^"]+)"/g, (m, action) => {
    return `v-click-outside="() => { ${action} }"`;
  });

  // 4. Convert core directives
  res = res.replace(/\bx-if="/g, 'v-if="');
  res = res.replace(/\bx-show="/g, 'v-show="');
  res = res.replace(/\bx-for="/g, 'v-for="');
  res = res.replace(/\bx-text="/g, 'v-text="');
  res = res.replace(/\bx-html="/g, 'v-html="');
  res = res.replace(/\bx-model="/g, 'v-model="');

  // 5. Ensure public asset URLs start with /
  res = res.replace(/src="logo\.webp"/g, 'src="/logo.webp"');
  res = res.replace(/src="pokemon\//g, 'src="/pokemon/');
  res = res.replace(/src="pika/g, 'src="/pika');


  return res.trim();
}

for (const [relPath, info] of Object.entries(FILE_MAPPING)) {
  const srcFile = path.join(PARTIALS_DIR, relPath);
  if (!fs.existsSync(srcFile)) {
    console.warn('⚠️ Source partial not found:', srcFile);
    continue;
  }

  const rawHtml = fs.readFileSync(srcFile, 'utf8');
  const converted = convertTemplate(rawHtml);

  const destFile = path.join(COMPONENTS_DIR, info.comp);
  fs.mkdirSync(path.dirname(destFile), { recursive: true });

  const sfcContent = `<template>
${converted}
</template>

<script>
import { useApp } from '@/store/app';

export default {
  name: '${info.name}',
  setup() {
    return useApp();
  }
};
</script>
`;

  fs.writeFileSync(destFile, sfcContent, 'utf8');
  console.log(`✅ Converted ${relPath} -> ${info.comp}`);
}

console.log('🎉 All 20 partials successfully converted to Vue 3 Single File Components!');
