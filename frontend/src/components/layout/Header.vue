<template>
<!-- Top Navigation Header (macOS draggable with native traffic lights inset & seamless background artwork) -->
    <header class="wails-drag h-[88px] bg-slate-950 border-b border-slate-800 px-4 xl:px-6 flex items-center justify-between z-20 shrink-0 relative overflow-hidden select-none">
      
      <!-- Seamless Background Artwork (Flush edge-to-edge on the left, fades out smoothly to the right) -->
      <div class="mask-fade-right absolute left-0 top-0 bottom-0 pointer-events-none overflow-hidden w-[420px]">
        <div class="w-full h-full relative" style="transform: scale(1.02); transform-origin: left 42%;">
          <img src="/logo.webp"
            class="w-full h-full object-cover select-none pointer-events-none"
            style="object-position: -48px 42%;"
            alt="SpeedMap Logo Artwork">
        </div>
      </div>

      <!-- Left: macOS Traffic Lights Spacing Clearance -->
      <div class="w-20 shrink-0 pointer-events-none"></div>

      <!-- CENTERED NAVIGATION TABS (macOS Segmented Control - Responsive flex container) -->
      <div class="flex-1 flex items-center justify-center min-w-0 px-2 z-20">
        <nav class="wails-no-drag flex items-center bg-slate-900/90 backdrop-blur-md p-1 rounded-xl border border-slate-800 text-xs shadow-lg max-w-full overflow-x-auto no-scrollbar space-x-0.5 shrink-0">
          <button @click="activeTab = 'pages'"
            :class="activeTab === 'pages' ? 'bg-slate-800 text-white font-semibold shadow-sm' : 'text-slate-400 hover:text-slate-200'"
            class="px-2.5 xl:px-3.5 py-1.5 rounded-lg transition-all flex items-center space-x-1.5 xl:space-x-2 shrink-0">
            <svg class="w-3.5 h-3.5 shrink-0" :class="activeTab === 'pages' ? 'text-emerald-400' : 'text-slate-400'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
            </svg>
            <span class="truncate">Сторінки</span>
            <template v-if="scanResults.length > 0">
              <span class="px-1.5 py-0.5 rounded-md bg-slate-700/80 text-emerald-300 text-[10px] font-mono font-medium shrink-0" v-text="scanResults.length"></span>
            </template>
          </button>
          
          <button @click="activeTab = 'analytics'; updateAnalytics()"
            :class="activeTab === 'analytics' ? 'bg-slate-800 text-white font-semibold shadow-sm' : 'text-slate-400 hover:text-slate-200'"
            class="px-2.5 xl:px-3.5 py-1.5 rounded-lg transition-all flex items-center space-x-1.5 xl:space-x-2 shrink-0">
            <svg class="w-3.5 h-3.5 shrink-0" :class="activeTab === 'analytics' ? 'text-cyan-400' : 'text-slate-400'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/>
            </svg>
            <span class="truncate">Статистика</span>
            <template v-if="siteAnalytics">
              <span class="px-1.5 py-0.5 rounded-md bg-slate-700/80 text-cyan-300 text-[10px] font-mono font-medium shrink-0" v-text="siteAnalytics.healthScore + '%'"></span>
            </template>
          </button>

          <button @click="activeTab = 'images'; updateAnalytics()"
            :class="activeTab === 'images' ? 'bg-slate-800 text-white font-semibold shadow-sm' : 'text-slate-400 hover:text-slate-200'"
            class="px-2.5 xl:px-3.5 py-1.5 rounded-lg transition-all flex items-center space-x-1.5 xl:space-x-2 shrink-0">
            <svg class="w-3.5 h-3.5 shrink-0" :class="activeTab === 'images' ? 'text-amber-400' : 'text-slate-400'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/>
            </svg>
            <span class="truncate">Зображення</span>
            <template v-if="siteAnalytics && siteAnalytics.totalImageCount > 0">
              <span class="px-1.5 py-0.5 rounded-md bg-slate-700/80 text-amber-300 text-[10px] font-mono font-medium shrink-0" v-text="siteAnalytics.totalImageCount"></span>
            </template>
            <template v-if="exportDiffReport && exportDiffReport.degradedCount > 0">
              <span class="px-1.5 py-0.5 rounded-md bg-rose-500/20 text-rose-300 text-[10px] font-mono font-medium shrink-0" title="Виявлено регресії" v-text="'!' + exportDiffReport.degradedCount"></span>
            </template>
          </button>

          <button @click="activeTab = 'fonts'; updateAnalytics()"
            :class="activeTab === 'fonts' ? 'bg-slate-800 text-white font-semibold shadow-sm' : 'text-slate-400 hover:text-slate-200'"
            class="px-2.5 xl:px-3.5 py-1.5 rounded-lg transition-all flex items-center space-x-1.5 xl:space-x-2 shrink-0">
            <svg class="w-3.5 h-3.5 shrink-0" :class="activeTab === 'fonts' ? 'text-purple-400' : 'text-slate-400'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 5h12M9 5v14m4 0h6m-3-7v7"/>
            </svg>
            <span class="truncate">Шрифти</span>
            <template v-if="siteAnalytics && siteAnalytics.fontUsage && siteAnalytics.fontUsage.length > 0">
              <span class="px-1.5 py-0.5 rounded-md bg-slate-700/80 text-purple-300 text-[10px] font-mono font-medium shrink-0" v-text="siteAnalytics.fontUsage.length"></span>
            </template>
          </button>

          <button @click="activeTab = 'iframes'; updateAnalytics()"
            :class="activeTab === 'iframes' ? 'bg-slate-800 text-white font-semibold shadow-sm' : 'text-slate-400 hover:text-slate-200'"
            class="px-2.5 xl:px-3.5 py-1.5 rounded-lg transition-all flex items-center space-x-1.5 xl:space-x-2 shrink-0">
            <svg class="w-3.5 h-3.5 shrink-0" :class="activeTab === 'iframes' ? 'text-rose-400' : 'text-slate-400'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/>
            </svg>
            <span class="truncate">iframe</span>
            <template v-if="siteAnalytics && siteAnalytics.totalIframeCount > 0">
              <span class="px-1.5 py-0.5 rounded-md bg-slate-700/80 text-rose-300 text-[10px] font-mono font-medium shrink-0" v-text="siteAnalytics.totalIframeCount"></span>
            </template>
          </button>

          <button @click="activeTab = 'forms'; updateAnalytics()"
            :class="activeTab === 'forms' ? 'bg-slate-800 text-white font-semibold shadow-sm' : 'text-slate-400 hover:text-slate-200'"
            class="px-2.5 xl:px-3.5 py-1.5 rounded-lg transition-all flex items-center space-x-1.5 xl:space-x-2 shrink-0">
            <svg class="w-3.5 h-3.5 shrink-0" :class="activeTab === 'forms' ? 'text-blue-400' : 'text-slate-400'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4"/>
            </svg>
            <span class="truncate">Форми</span>
            <template v-if="siteAnalytics && siteAnalytics.totalFormsCount > 0">
              <span class="px-1.5 py-0.5 rounded-md bg-slate-700/80 text-blue-300 text-[10px] font-mono font-medium shrink-0" v-text="siteAnalytics.totalFormsCount"></span>
            </template>
          </button>
        </nav>
      </div>

      <!-- Right Action Controls (Activity Log, Device toggle, Settings) -->
      <div class="wails-no-drag flex items-center space-x-1.5 sm:space-x-2 relative z-20 shrink-0">
        <!-- Activity Log: clean icon button -->
        <button @click="showLogDrawer = true"
          class="relative flex items-center justify-center w-8 h-8 bg-slate-900 hover:bg-slate-800 active:bg-slate-700 text-slate-300 border border-slate-700/80 rounded-lg transition shrink-0 shadow-xs"
          title="Лог подій">
          <svg class="w-4 h-4 text-slate-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 10h16M4 14h16M4 18h16"/>
          </svg>
          <template v-if="activityLogs.length > 0">
            <span class="absolute top-1.5 right-1.5 w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
          </template>
        </button>

        <!-- Device mode: macOS segmented toggle -->
        <button type="button" @click="toggleDeviceMode()"
          class="flex items-center justify-center px-2.5 h-8 bg-slate-900 hover:bg-slate-800 border rounded-lg transition text-xs font-medium space-x-1.5 shadow-xs shrink-0"
          :class="config.isMobile ? 'text-cyan-300 border-cyan-500/40 bg-cyan-500/10' : 'text-slate-300 border-slate-700'"
          :title="config.isMobile ? 'Активний режим: Mobile 4G (375×812). Клікніть для Desktop' : 'Активний режим: Desktop (1920×1080). Клікніть для Mobile'">
          <svg v-show="config.isMobile" class="w-3.5 h-3.5 text-cyan-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 18h.01M8 21h8a2 2 0 002-2V5a2 2 0 00-2-2H8a2 2 0 00-2 2v14a2 2 0 002 2z"/>
          </svg>
          <svg v-show="!config.isMobile" class="w-3.5 h-3.5 text-amber-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"/>
          </svg>
          <span class="hidden sm:inline" v-text="config.isMobile ? 'Mobile 4G' : 'Desktop'"></span>
        </button>

        <!-- Settings: icon button -->
        <button @click="showSettings = true"
          class="flex items-center justify-center w-8 h-8 bg-slate-900 hover:bg-slate-800 active:bg-slate-700 text-slate-300 border border-slate-700/80 rounded-lg transition shrink-0 shadow-xs"
          title="Налаштування">
          <svg class="w-4 h-4 text-slate-400 hover:text-slate-200" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
          </svg>
        </button>
      </div>
    </header>
</template>

<script>
import { useApp } from '@/store/app';

export default {
  name: 'Header',
  setup() {
    return useApp();
  }
};
</script>
