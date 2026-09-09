<template>
<!-- VIEW 3: IMAGE OPTIMIZATION & COMPARISON HUB (SEOAEO-235) -->
  <template v-if="activeTab === 'images'">
    <div class="flex-1 overflow-y-auto p-6 space-y-6">

      <!-- Sub-Navigation for Images Hub: Catalog vs Regressions & Diff vs QA Verifier -->
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-800 pb-3">
        <div class="flex items-center bg-slate-900 p-1 rounded-xl border border-slate-800 text-xs font-semibold">
          <button @click="imageSectionTab = 'catalog'"
            :class="imageSectionTab === 'catalog' ? 'bg-slate-800 text-white shadow-sm' : 'text-slate-400 hover:text-slate-200'"
            class="px-3.5 py-1.5 rounded-lg transition flex items-center space-x-2">
            <svg class="w-3.5 h-3.5 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/>
            </svg>
            <span>Каталог та Аналіз</span>
            <template v-if="siteAnalytics && siteAnalytics.totalImageCount > 0">
              <span class="px-1.5 py-0.5 rounded-md bg-slate-700/80 text-amber-300 text-[10px] font-mono" v-text="siteAnalytics.totalImageCount"></span>
            </template>
          </button>
          <button @click="imageSectionTab = 'diff'; diffViewMode = 'packages'; loadExportHistoryForDiff()"
            :class="imageSectionTab === 'diff' ? 'bg-slate-800 text-white shadow-sm' : 'text-slate-400 hover:text-slate-200'"
            class="px-3.5 py-1.5 rounded-lg transition flex items-center space-x-2">
            <svg class="w-3.5 h-3.5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/>
            </svg>
            <span>Порівняння регресій (Diff)</span>
            <template v-if="exportDiffReport && exportDiffReport.degradedCount > 0">
              <span class="px-1.5 py-0.5 rounded-md bg-rose-500/20 text-rose-300 text-[10px] font-mono" v-text="exportDiffReport.degradedCount"></span>
            </template>
          </button>

          <button @click="openExportPackageInStudio()"
            class="px-3.5 py-1.5 rounded-lg transition flex items-center space-x-2 text-fuchsia-400 hover:text-fuchsia-200 hover:bg-slate-800"
            title="Відкрити збережену папку speedmap-webp-... для точкового доналаштування зображень">
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/>
            </svg>
            <span>Точкове коригування пакету</span>
          </button>

          <button @click="imageSectionTab = 'verify'"
            :class="imageSectionTab === 'verify' ? 'bg-slate-800 text-white shadow-sm' : 'text-slate-400 hover:text-slate-200'"
            class="px-3.5 py-1.5 rounded-lg transition flex items-center space-x-2"
            title="Автоматична перевірка доступності WebP файлів та заміни на цільовому сервері">
            <svg class="w-3.5 h-3.5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
            </svg>
            <span>Звірка з маніфестом (QA)</span>
            <template v-if="verifyState?.result?.summary">
              <span class="px-1.5 py-0.5 rounded-md text-[10px] font-mono font-bold"
                :class="verifyState.result.summary.allPassed ? 'bg-emerald-500/20 text-emerald-300' : 'bg-rose-500/20 text-rose-300'"
                v-text="verifyState.result.summary.passedImages + '/' + verifyState.result.summary.totalImages"></span>
            </template>
          </button>
        </div>

        <!-- Quick summary badge when in Diff mode -->
        <template v-if="imageSectionTab === 'diff' && exportDiffReport">
          <div class="flex items-center space-x-3 text-xs font-mono">
            <span class="text-slate-400">Файлів: <strong class="text-slate-200" v-text="exportDiffReport.totalFiles"></strong></span>
            <span class="text-emerald-400">Покращень: <strong v-text="exportDiffReport.improvedCount"></strong></span>
            <span class="text-rose-400">Регресій: <strong v-text="exportDiffReport.degradedCount"></strong></span>
          </div>
        </template>
      </div>

      <!-- SUB-TAB 1: Catalog & Analytics -->
      <ImagesCatalogSubtab />

      <!-- SUB-TAB 2: Runs Diff & Regression Tracker -->
      <ImagesDiffSubtab />

      <!-- SUB-TAB 3: Manifest Verification & Staging QA -->
      <ImagesVerifySubtab />

    </div>
  </template>
</template>

<script>
import { useApp } from '@/store/app';
import ImagesCatalogSubtab from './images/ImagesCatalogSubtab.vue';
import ImagesDiffSubtab from './images/ImagesDiffSubtab.vue';
import ImagesVerifySubtab from './images/ImagesVerifySubtab.vue';

export default {
  name: 'ImagesTab',
  components: {
    ImagesCatalogSubtab,
    ImagesDiffSubtab,
    ImagesVerifySubtab
  },
  setup() {
    return useApp();
  }
};
</script>
