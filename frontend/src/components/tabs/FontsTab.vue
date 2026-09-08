<template>
<!-- VIEW 4: DEDICATED FONT INSPECTOR HUB (SEOAEO-301) -->
        <template v-if="activeTab === 'fonts'">
          <div class="flex-1 overflow-y-auto p-4 xl:p-6 space-y-6">
            
            <!-- Font Hub Top Header Bar & Export Actions -->
            <div class="bg-slate-950/60 p-4 rounded-xl border border-slate-800 space-y-3">
              <div class="flex flex-col xl:flex-row xl:items-center justify-between gap-3 border-b border-slate-800 pb-3">
                <div class="space-y-0.5">
                  <h2 class="text-sm font-semibold text-slate-100 flex items-center space-x-2">
                    <svg class="w-4 h-4 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 5h12M9 5v14m4 0h6m-3-7v7"/>
                    </svg>
                    <span>Інспектор Шрифтів (Font Hub)</span>
                  </h2>
                  <p class="text-xs text-slate-400">Посторінковий аналіз гарнітур, типографіки та часу завантаження</p>
                </div>

                <!-- View Mode Toggle & Export Actions in wrapping container -->
                <div class="flex flex-wrap items-center gap-2.5">
                  <!-- View Mode Toggle: Page-by-Page vs By-Font-Family -->
                  <div class="flex items-center space-x-1 bg-slate-900 p-1 rounded-xl border border-slate-800 text-xs shrink-0">
                    <button @click="fontViewMode = 'by-page'"
                      :class="fontViewMode === 'by-page' ? 'bg-slate-800 text-purple-300 font-semibold shadow-sm' : 'text-slate-400 hover:text-slate-200'"
                      class="px-2.5 py-1.5 rounded-lg transition flex items-center space-x-1.5">
                      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
                      </svg>
                      <span>За сторінками</span>
                    </button>
                    <button @click="fontViewMode = 'by-font'"
                      :class="fontViewMode === 'by-font' ? 'bg-slate-800 text-purple-300 font-semibold shadow-sm' : 'text-slate-400 hover:text-slate-200'"
                      class="px-2.5 py-1.5 rounded-lg transition flex items-center space-x-1.5">
                      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 5h12M9 5v14m4 0h6m-3-7v7"/>
                      </svg>
                      <span>За гарнітурами</span>
                    </button>
                  </div>

                  <!-- Standard Export Action Buttons -->
                  <div class="flex items-center space-x-1.5 shrink-0">
                    <button @click="exportFontsCSV()" :disabled="!siteAnalytics || !siteAnalytics.fontUsage || siteAnalytics.fontUsage.length === 0"
                      class="px-2.5 py-1.5 rounded-lg border text-xs font-medium transition flex items-center space-x-1.5 bg-slate-900 border-slate-700/80 hover:bg-slate-800 text-slate-200 disabled:opacity-40 disabled:cursor-not-allowed shadow-sm">
                      <svg class="w-3.5 h-3.5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
                      </svg>
                      <span>CSV</span>
                    </button>
                    <button @click="exportFontsJSON()" :disabled="!siteAnalytics || !siteAnalytics.fontUsage || siteAnalytics.fontUsage.length === 0"
                      class="px-2.5 py-1.5 rounded-lg border text-xs font-medium transition flex items-center space-x-1.5 bg-slate-900 border-slate-700/80 hover:bg-slate-800 text-slate-200 disabled:opacity-40 disabled:cursor-not-allowed shadow-sm">
                      <svg class="w-3.5 h-3.5 text-indigo-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/>
                      </svg>
                      <span>JSON</span>
                    </button>
                    <button @click="uploadFontsToGDrive()" :disabled="!siteAnalytics || !siteAnalytics.fontUsage || siteAnalytics.fontUsage.length === 0 || isUploadingGDrive"
                      class="px-2.5 py-1.5 rounded-lg border text-xs font-medium transition flex items-center space-x-1.5 bg-slate-900 border-slate-700/80 hover:bg-slate-800 text-slate-200 disabled:opacity-40 disabled:cursor-not-allowed shadow-sm">
                      <svg class="w-3.5 h-3.5 text-sky-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 00-9.78 2.096A4.001 4.001 0 003 15z"/>
                      </svg>
                      <span v-show="!isUploadingGDrive">Drive</span>
                      <span v-show="isUploadingGDrive" class="flex items-center space-x-1">
                        <svg class="animate-spin h-3 w-3 text-cyan-400" fill="none" viewBox="0 0 24 24">
                          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                        <span>Вивантаження...</span>
                      </span>
                    </button>
                  </div>
                </div>
              </div>

              <!-- Search and Filter Controls -->
              <div class="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 text-xs">
                <div class="relative flex items-center flex-1 w-full sm:max-w-md">
                  <div class="absolute inset-y-0 left-0 pl-2.5 flex items-center pointer-events-none text-slate-500">
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
                    </svg>
                  </div>
                  <input type="text" v-model="fontSearchQuery" @keydown.escape="fontSearchQuery = ''" placeholder="Пошук за назвою шрифту (Inter, Roboto, FontAwesome)..."
                    class="w-full bg-slate-900 border border-slate-700/80 hover:border-slate-600 focus:border-cyan-500 rounded-lg pl-8 pr-7 py-1.5 text-xs text-slate-100 placeholder-slate-500 focus:outline-none font-mono transition shadow-inner">
                  <template v-if="fontSearchQuery">
                    <button @click="fontSearchQuery = ''" class="absolute inset-y-0 right-0 pr-2 flex items-center text-slate-400 hover:text-slate-200 text-xs font-bold" title="Очистити">✕</button>
                  </template>
                </div>

                <div class="flex items-center space-x-2 shrink-0 justify-end">
                  <span class="text-slate-400 font-semibold shrink-0">Сортування:</span>
                  <select v-model="fontSortKey" class="bg-slate-900 border border-slate-700 hover:border-slate-600 rounded-lg px-2.5 py-1.5 text-xs text-slate-200 focus:outline-none font-sans">
                    <option value="pages">За покриттям сторінок</option>
                    <option value="duration">За часом завантаження</option>
                    <option value="size">За обсягом файлу</option>
                  </select>
                </div>
              </div>
            </div>

            <!-- Empty State -->
            <template v-if="!siteAnalytics || !siteAnalytics.fontUsage || siteAnalytics.fontUsage.length === 0">
              <div class="p-12 text-center bg-slate-950/40 rounded-xl border border-slate-800/80 space-y-3">
                <div class="w-12 h-12 mx-auto rounded-xl bg-slate-900/80 border border-slate-800 flex items-center justify-center text-slate-500">
                  <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 5h12M9 5v14m6-7h6m-3-7v14"/>
                  </svg>
                </div>
                <h3 class="text-sm font-semibold text-slate-300">Не виявлено зовнішніх шрифтів</h3>
                <p class="text-xs text-slate-500 max-w-md mx-auto">Після сканування тут з'явиться детальний аналіз усіх гарнітур, завантажених на сторінках вашого сайту.</p>
              </div>
            </template>

            <!-- Page-by-Page Fonts View Mode -->
            <template v-if="fontViewMode === 'by-page'">
              <div class="space-y-4">
                <template v-if="pageFontsList.length === 0">
                  <div class="p-12 text-center bg-slate-950/40 rounded-xl border border-slate-800 space-y-3">
                    <div class="w-12 h-12 mx-auto rounded-xl bg-slate-900/80 border border-slate-800 flex items-center justify-center text-slate-500">
                      <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"/>
                      </svg>
                    </div>
                    <h3 class="text-sm font-semibold text-slate-300">Немає сканованих сторінок</h3>
                    <p class="text-xs text-slate-500">Запустіть сканування сторінок для формування посторінкового звіту про шрифти.</p>
                  </div>
                </template>

                <template v-for="item in paginatedFontPages" :key="item.id">
                  <div class="bg-slate-950/60 rounded-xl border border-slate-800/80 p-4 space-y-3 hover:border-slate-700 transition">
                    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b border-slate-800/80 pb-3">
                      <div class="flex items-center space-x-2.5 truncate">
                        <span class="px-2 py-0.5 rounded bg-amber-500/10 text-amber-400 border border-amber-500/20 font-mono text-xs shrink-0" v-text="`#${item.id}`"></span>
                        <button @click.stop="openUrlInBrowser(item.url)" class="font-mono font-medium text-xs text-slate-100 hover:text-amber-400 underline truncate text-left" :title="item.url" v-text="item.url"></button>
                      </div>
                      <div class="flex items-center gap-2 shrink-0 text-[10px] font-mono">
                        <span class="px-2 py-0.5 rounded-md bg-amber-500/10 text-amber-300 border border-amber-500/20 font-medium" v-text="item.fontCount + ' гарнітур'"></span>
                      </div>
                    </div>
                    <template v-if="item.fonts.length > 0">
                      <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3">
                        <template v-for="f in item.fonts" :key="f.family + f.url">
                          <div class="bg-slate-900/60 p-3 rounded-lg border border-slate-800/80 text-xs space-y-2">
                            <div class="flex items-center justify-between gap-2">
                              <span class="font-semibold text-slate-200 truncate" v-text="f.family"></span>
                              <span class="px-1.5 py-0.5 rounded text-[10px] font-mono bg-purple-950/80 text-purple-300 border border-purple-800/60 shrink-0" v-text="f.type || 'font'"></span>
                            </div>

                            <div class="flex items-center justify-between text-[11px] font-mono text-slate-400 pt-1 border-t border-slate-800/60">
                              <span>Час: <strong class="text-slate-200" v-text="(f.duration || 0) + ' ms'"></strong></span>
                              <template v-if="f.url">
                                <button @click="copyFontUrl(f.url)" class="text-amber-400 hover:text-amber-300 font-medium text-[10px] flex items-center space-x-1">
                                  <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
                                  <span>URL</span>
                                </button>
                              </template>
                            </div>
                          </div>
                        </template>
                      </div>
                    </template>
                    <template v-if="item.fonts.length === 0">
                      <p class="text-xs text-slate-500 italic">Зовнішніх файлів шрифтів на цій сторінці не виявлено (використовуються системні шрифти).</p>
                    </template>
                  </div>
                </template>

                <!-- Standard Reusable Pagination Bar -->
                <template v-if="pageFontsList.length > 0">
                  <div class="px-4 py-2.5 bg-slate-950/80 border border-slate-800 rounded-xl flex flex-col sm:flex-row items-center justify-between gap-3 text-xs text-slate-300 font-mono">
                    <div>
                      Показано <span class="font-bold text-cyan-400" v-text="pageFontsList.length > 0 ? ((fontPage - 1) * fontPerPage + 1) : 0"></span> - 
                      <span class="font-bold text-cyan-400" v-text="Math.min(fontPage * fontPerPage, pageFontsList.length)"></span> з 
                      <span class="font-bold text-slate-100" v-text="pageFontsList.length"></span> сторінок
                    </div>

                    <div class="flex items-center space-x-2">
                      <button @click="fontPage = Math.max(1, fontPage - 1)" :disabled="fontPage === 1"
                        class="px-2.5 py-1 rounded-md bg-slate-900 border border-slate-700/80 hover:bg-slate-800 disabled:opacity-40 disabled:cursor-not-allowed transition font-sans">
                        Попередня
                      </button>

                      <span class="px-2 font-medium text-slate-300">
                        <span v-text="fontPage"></span> / <span v-text="totalFontPagesCount"></span>
                      </span>

                      <button @click="fontPage = Math.min(totalFontPagesCount, fontPage + 1)" :disabled="fontPage >= totalFontPagesCount"
                        class="px-2.5 py-1 rounded-md bg-slate-900 border border-slate-700/80 hover:bg-slate-800 disabled:opacity-40 disabled:cursor-not-allowed transition font-sans">
                        Наступна
                      </button>

                      <select x-model.number="fontPerPage" @change="fontPage = 1" class="bg-slate-900 border border-slate-700 rounded-md px-2 py-1 text-xs text-slate-200 focus:outline-none focus:border-cyan-500 font-sans">
                        <option value="25">25 / стор.</option>
                        <option value="50">50 / стор.</option>
                        <option value="100">100 / стор.</option>
                        <option value="250">250 / стор.</option>
                      </select>
                    </div>
                  </div>
                </template>
              </div>
            </template>

            <!-- Fonts Grid Cards (Aggregated by Font Family) -->
            <template v-if="fontViewMode === 'by-font' && filteredFonts.length > 0">

              <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                <template v-for="font in filteredFonts" :key="font.family">
                  <div class="bg-slate-950/60 rounded-xl border border-slate-800/80 hover:border-slate-700 transition duration-200 overflow-hidden flex flex-col justify-between group">
                    
                    <!-- UPPER SECTION: Icon, Family Name, Format Badge & Visual Preview -->
                    <div class="p-4 space-y-3 border-b border-slate-800/80">
                      <div class="flex items-center justify-between">
                        <div class="flex items-center space-x-2.5 truncate">
                          <div class="w-7 h-7 rounded-lg bg-amber-500/10 border border-amber-500/20 flex items-center justify-center text-amber-400 font-bold shrink-0">
                            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 5h12M9 5v14m6-7h6m-3-7v14"/></svg>
                          </div>
                          <h3 class="font-semibold text-sm text-slate-100 truncate group-hover:text-amber-400 transition font-sans" :title="font.family" v-text="font.family"></h3>
                        </div>
                        <span class="px-2 py-0.5 rounded text-[10px] font-mono font-medium uppercase shrink-0"
                          :class="font.type === 'woff2' ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30' : (font.type === 'woff' ? 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/30' : 'bg-amber-500/20 text-amber-300 border border-amber-500/30')"
                          v-text="font.type || 'FONT'"></span>
                      </div>

                      <!-- Live Typography Preview Sandbox -->
                      <div class="p-2.5 bg-slate-900/80 rounded-lg border border-slate-800/80 space-y-1 font-sans overflow-hidden">
                        <div class="text-slate-100 text-xs font-medium truncate" :style="'font-family: ' + font.family + ', sans-serif'">
                          Aa Bb Cc 123 • Ag Qg
                        </div>
                        <div class="text-slate-400 text-[11px] truncate opacity-75" :style="'font-family: ' + font.family + ', sans-serif'">
                          The quick brown fox jumps over the lazy dog
                        </div>
                      </div>
                    </div>

                    <!-- LOWER SECTION: Characteristics & Optimization Metrics -->
                    <div class="p-4 space-y-3 text-xs bg-slate-950/40">
                      <div class="grid grid-cols-2 gap-2 text-[11px] font-mono">
                        <div class="bg-slate-900/60 p-2 rounded-lg border border-slate-800/80 space-y-0.5">
                          <span class="text-slate-500 block text-[10px] font-medium font-sans">Покриття сайту</span>
                          <span class="font-semibold text-emerald-400 block truncate" v-text="font.occurrences + ' стор. (' + font.percentage + '%)'"></span>
                        </div>

                        <div class="bg-slate-900/60 p-2 rounded-lg border border-slate-800/80 space-y-0.5">
                          <span class="text-slate-500 block text-[10px] font-medium font-sans">Розмір файлу</span>
                          <span class="font-semibold text-cyan-400 block truncate" v-text="font.formattedSize || 'н/д'"></span>
                        </div>
                      </div>

                      <div class="flex items-center justify-between text-[11px] font-mono pt-1 border-t border-slate-800/60">
                        <span class="text-slate-400 font-sans">Час завантаження:</span>
                        <span class="font-semibold text-slate-200" v-text="font.avgDurationMs + ' ms'"></span>
                      </div>

                      <!-- Font URL & Copy Button -->
                      <template v-if="font.url">
                        <div class="pt-2 border-t border-slate-800/60 flex items-center justify-between gap-2">
                          <button @click.stop="openUrlInBrowser(font.url)" class="text-[11px] text-slate-400 hover:text-amber-400 underline font-mono truncate block flex-1 text-left" :title="font.url" v-text="font.url"></button>
                          <button @click="copyFontUrl(font.url)" class="px-2 py-1 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded text-[10px] font-medium shrink-0 transition flex items-center space-x-1">
                            <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
                            <span>Копіювати</span>
                          </button>
                        </div>
                      </template>

                      <!-- Optimization Advice Badge -->
                      <div class="pt-1 flex items-center justify-between">
                        <template v-if="font.type === 'woff2'">
                          <span class="inline-flex items-center gap-1 text-[10px] text-emerald-400 font-medium bg-emerald-500/10 px-2 py-0.5 rounded border border-emerald-500/20">
                            <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                            <span>WOFF2 Сучасний формат</span>
                          </span>
                        </template>
                        <template v-if="font.type && font.type !== 'woff2'">
                          <span class="inline-flex items-center gap-1 text-[10px] text-amber-400 font-medium bg-amber-500/10 px-2 py-0.5 rounded border border-amber-500/20">
                            <span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
                            <span>Конвертувати у WOFF2</span>
                          </span>
                        </template>
                      </div>

                      <!-- Page URLs List Collapsible -->
                      <template v-if="font.pageUrls && font.pageUrls.length > 0">
                        <div x-data="{ open: false }" class="pt-2 border-t border-slate-800/60 font-sans space-y-1">
                          <button @click="open = !open" class="flex items-center justify-between w-full text-[11px] text-slate-400 hover:text-amber-400 font-medium transition">
                            <span>Використовується на <strong class="text-slate-200" v-text="font.pageUrls.length"></strong> сторінках</span>
                            <span v-text="open ? '▲ Сховати' : '▼ Переглянути URL'" class="text-[10px] text-amber-400 font-mono"></span>
                          </button>
                          <div v-show="open" class="max-h-28 overflow-y-auto space-y-1 bg-slate-900/90 p-2 rounded-lg border border-slate-800 text-[10px] font-mono">
                            <template v-for="pageUrl in font.pageUrls" :key="pageUrl">
                              <button @click.stop="openUrlInBrowser(pageUrl)" class="block text-slate-300 hover:text-emerald-400 truncate underline text-left w-full" :title="pageUrl" v-text="pageUrl"></button>
                            </template>
                          </div>
                        </div>
                      </template>
                    </div>

                  </div>
                </template>
              </div>
            </template>

          </div>
        </template>
</template>

<script>
import { useApp } from '@/store/app';

export default {
  name: 'FontsTab',
  setup() {
    return useApp();
  }
};
</script>
