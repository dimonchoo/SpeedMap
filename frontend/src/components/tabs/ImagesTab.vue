<template>
<!-- VIEW 3: IMAGE OPTIMIZATION & COMPARISON HUB (SEOAEO-235) -->
        <template v-if="activeTab === 'images'">
          <div class="flex-1 overflow-y-auto p-6 space-y-6">

            <!-- Sub-Navigation for Images Hub: Catalog vs Regressions & Diff -->
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
            <div v-show="imageSectionTab === 'catalog'" class="space-y-6">
              <template v-if="!siteAnalytics">
              <div class="h-full flex flex-col items-center justify-center text-slate-500 space-y-4 py-16">
                <svg class="w-14 h-14 text-slate-700 animate-bounce" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/>
                </svg>
                <p class="text-sm font-medium text-slate-400">Запустіть сканування сторінок або відкрийте збережений пакет експорту для точкового доналаштування</p>
                <button @click="openExportPackageInStudio()"
                  class="px-4 py-2 bg-fuchsia-600/20 hover:bg-fuchsia-600/30 text-fuchsia-300 border border-fuchsia-500/40 rounded-xl text-xs font-semibold flex items-center space-x-2 transition shadow">
                  <svg class="w-4 h-4 text-fuchsia-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/>
                  </svg>
                  <span>Відкрити існуючий пакет WebP (speedmap-webp-...)</span>
                </button>
              </div>
            </template>

            <template v-if="siteAnalytics">
              <div class="space-y-6">
                
                <!-- Header Stats Cards & Export HTML Button -->
                <div class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-3">
                  
                  <div class="bg-slate-950/60 p-4 rounded-xl border border-slate-800 space-y-1">
                    <div class="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">Загальний обсяг</div>
                    <div class="text-2xl font-extrabold text-amber-400 font-mono" v-text="siteAnalytics.totalImagePayloadFormatted || '0 B'"></div>
                    <div class="text-[10px] text-slate-500 font-mono">Ціль: &lt; 2.0 MB</div>
                  </div>

                  <div class="bg-slate-950/60 p-4 rounded-xl border border-slate-800 space-y-1">
                    <div class="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">Унікальних файлів</div>
                    <div class="text-2xl font-extrabold text-slate-100 font-mono" v-text="siteAnalytics.totalImageCount || 0"></div>
                    <div class="text-[11px] text-slate-400">
                      Не-WebP: <span class="font-bold text-amber-400" v-text="siteAnalytics.nonWebPCount || 0"></span>
                    </div>
                  </div>

                  <div class="bg-slate-950/60 p-4 rounded-xl border border-slate-800 space-y-1">
                    <div class="text-[11px] font-semibold text-slate-400 uppercase tracking-wider flex items-center justify-between">
                      <span>Важкі (&gt;<span v-text="config.heavyImageThresholdKB || 100"></span>KB)</span>
                      <span class="w-2 h-2 rounded-full bg-rose-400"></span>
                    </div>
                    <div class="text-2xl font-extrabold text-rose-400 font-mono" v-text="siteAnalytics?.heavyImagesCount || 0"></div>
                    <div class="text-[11px] text-rose-400/80">Потребують оптимізації</div>
                  </div>

                  <div class="bg-slate-950/60 p-4 rounded-xl border border-slate-800 space-y-2 flex flex-col justify-between">
                    <div>
                      <div class="text-[11px] font-semibold text-emerald-400 uppercase tracking-wider flex items-center justify-between">
                        <span>Потенційна економія</span>
                        <span class="w-2 h-2 rounded-full bg-emerald-400"></span>
                      </div>
                      <div class="text-2xl font-extrabold text-emerald-400 font-mono" v-text="'-' + (siteAnalytics.totalWebPSavingsFormatted || '0 B')"></div>
                    </div>
                    <div class="grid grid-cols-2 gap-1.5 mt-1">
                      <button @click="exportImageReportHTML()" :disabled="isExportingReport"
                        class="bg-slate-800 hover:bg-slate-700 disabled:opacity-50 text-slate-200 font-medium py-1 px-2 rounded-lg text-[10px] border border-slate-700 transition flex items-center justify-center space-x-1 col-span-2 shadow-xs"
                        title="Зберегти повний інтерактивний HTML звіт">
                        <template v-if="!isExportingReport">
                          <span class="flex items-center space-x-1">
                            <svg class="w-3 h-3 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/></svg>
                            <span>Звіт HTML</span>
                          </span>
                        </template>
                        <template v-if="isExportingReport">
                          <svg class="animate-spin h-3 w-3 text-white" fill="none" viewBox="0 0 24 24">
                            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                          </svg>
                        </template>
                      </button>

                      <button @click="downloadBatchWebPZIP()" :disabled="isBatchDownloadingZIP"
                        class="bg-amber-600 hover:bg-amber-500 disabled:opacity-50 text-white font-medium py-1 px-2 rounded-lg text-[10px] transition flex items-center justify-center space-x-1"
                        title="Конвертувати та завантажити ZIP архів WebP зображень">
                        <span v-show="!isBatchDownloadingZIP">ZIP WebP</span>
                        <svg v-show="isBatchDownloadingZIP" class="animate-spin h-3 w-3 text-white" fill="none" viewBox="0 0 24 24">
                          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                      </button>

                      <button @click="exportWordPressWebPApply()" :disabled="isExportingWPApply"
                        class="bg-slate-800 hover:bg-slate-700 disabled:opacity-50 text-emerald-400 font-medium py-1 px-2 rounded-lg text-[10px] border border-slate-700 transition flex items-center justify-center space-x-1"
                        title="Експорт WordPress WebP apply">
                        <span v-show="!isExportingWPApply">WP apply</span>
                        <svg v-show="isExportingWPApply" class="animate-spin h-3 w-3 text-white" fill="none" viewBox="0 0 24 24">
                          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                      </button>
                    </div>
                  </div>

                </div>

                <!-- LCP Heavy Image Spotlight (Ticket hero-8.png scenario) -->
                <template v-if="siteAnalytics.allImages && siteAnalytics.allImages.length > 0 && siteAnalytics.allImages[0].isHeavy">
                  <div class="bg-slate-950/60 p-5 rounded-xl border border-amber-500/40 space-y-3">
                    <div class="flex items-center justify-between">
                      <div class="flex items-center space-x-2">
                        <svg class="w-4 h-4 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
                        </svg>
                        <h3 class="text-xs font-bold text-amber-300 uppercase tracking-wider">Критичне зображення для LCP</h3>
                      </div>
                      <span class="px-2 py-0.5 rounded text-[10px] font-mono bg-rose-500/20 text-rose-300 border border-rose-500/30 font-bold uppercase">
                        Найбільший обсяг: <span v-text="siteAnalytics.allImages[0].formattedSize"></span>
                      </span>
                    </div>

                    <div class="flex flex-col xl:flex-row items-start xl:items-center justify-between gap-4 text-xs font-mono bg-slate-900/90 p-4 rounded-xl border border-slate-800">
                      <div class="space-y-1 max-w-xl truncate">
                        <div class="text-slate-400 font-sans">URL зображення:</div>
                        <div class="flex items-center space-x-2">
                          <button type="button" @click.stop="openUrlInBrowser(siteAnalytics.allImages[0].url)" class="text-amber-300 hover:underline font-bold text-sm truncate text-left" :title="'Відкрити у браузері:\n' + siteAnalytics.allImages[0].url" v-text="siteAnalytics.allImages[0].url"></button>
                          <button type="button" @click.stop="copyToClipboard(siteAnalytics.allImages[0].url, 'Посилання скопійовано')" class="text-slate-500 hover:text-slate-300 p-0.5 rounded transition shrink-0" title="Скопіювати повне посилання">
                            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
                          </button>
                        </div>
                        <div class="text-slate-500 font-sans text-[11px]">
                          Формат: <span class="text-slate-200 uppercase font-bold" v-text="siteAnalytics.allImages[0].format"></span> |
                          Роздільна здатність: <span class="text-slate-200" v-text="(siteAnalytics.allImages[0].width || 0) + 'x' + (siteAnalytics.allImages[0].height || 0) + ' px'"></span> |
                          Зустрічається на: <span class="text-cyan-400 font-bold" v-text="siteAnalytics.allImages[0].pageCount + ' стор.'"></span>
                        </div>
                      </div>

                      <div class="flex items-center space-x-6 text-right shrink-0">
                        <div>
                          <div class="text-slate-400 font-sans text-[11px]">Поточний розмір</div>
                          <div class="text-rose-400 font-bold text-base" v-text="siteAnalytics.allImages[0].formattedSize"></div>
                        </div>
                        <div class="text-slate-500 text-lg">➔</div>
                        <div>
                          <div class="text-emerald-400 font-sans text-[11px]">WebP Оцінка</div>
                          <div class="text-emerald-400 font-bold text-base" v-text="siteAnalytics.allImages[0].estimatedWebPFormatted"></div>
                        </div>
                        <div class="bg-emerald-500/10 border border-emerald-500/30 px-3 py-1.5 rounded-lg text-emerald-300 text-xs font-bold font-sans">
                          Економія <span v-text="siteAnalytics.allImages[0].estimatedSavingsPercent + '%'"></span>
                        </div>
                      </div>
                    </div>
                  </div>
                </template>

                <!-- Filter Toolbar & Search Bar -->
                <div class="bg-slate-950/60 p-3 rounded-xl border border-slate-800 flex flex-col 2xl:flex-row items-stretch 2xl:items-center justify-between gap-3 shrink-0">
                  
                  <!-- Filter Pills (No horizontal scroll, clean wrap if needed) -->
                  <div class="flex items-center flex-wrap gap-1 bg-slate-900 p-1 rounded-xl border border-slate-800 text-xs">
                    <button @click="setImageFilterTab('all')"
                      :class="imageFilterTab === 'all' ? 'bg-slate-800 text-white font-semibold shadow-sm' : 'text-slate-400 hover:text-slate-200'"
                      class="px-2.5 py-1.5 rounded-lg transition whitespace-nowrap">
                      Усі (<span v-text="siteAnalytics.allImages?.length || 0"></span>)
                    </button>
                    <button @click="setImageFilterTab('heavy')"
                      :title="'Важкі зображення понад ' + (config.heavyImageThresholdKB || 100) + ' KB'"
                      :class="imageFilterTab === 'heavy' ? 'bg-slate-800 text-rose-300 font-semibold shadow-sm' : 'text-slate-400 hover:text-slate-200'"
                      class="px-2.5 py-1.5 rounded-lg transition whitespace-nowrap flex items-center space-x-1.5">
                      <span class="w-1.5 h-1.5 rounded-full bg-rose-400"></span>
                      <span>Важкі (<span v-text="siteAnalytics?.heavyImagesCount || 0"></span>)</span>
                    </button>
                    <button @click="setImageFilterTab('non-webp')"
                      :class="imageFilterTab === 'non-webp' ? 'bg-slate-800 text-amber-300 font-semibold shadow-sm' : 'text-slate-400 hover:text-slate-200'"
                      class="px-2.5 py-1.5 rounded-lg transition whitespace-nowrap flex items-center space-x-1.5">
                      <span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
                      <span>Не-WebP (<span v-text="siteAnalytics?.nonWebPCount || 0"></span>)</span>
                    </button>
                    <button @click="setImageFilterTab('missing-lazy')"
                      :class="imageFilterTab === 'missing-lazy' ? 'bg-slate-800 text-indigo-300 font-semibold shadow-sm' : 'text-slate-400 hover:text-slate-200'"
                      class="px-2.5 py-1.5 rounded-lg transition whitespace-nowrap">
                      Без lazy (<span v-text="siteAnalytics.missingLazyCount || 0"></span>)
                    </button>
                    <button @click="setImageFilterTab('svg')"
                      :class="imageFilterTab === 'svg' ? 'bg-purple-900/60 text-purple-200 border border-purple-500/50 font-semibold shadow-sm' : 'text-slate-400 hover:text-purple-300'"
                      class="px-2.5 py-1.5 rounded-lg transition whitespace-nowrap flex items-center space-x-1.5">
                      <span class="w-1.5 h-1.5 rounded-full bg-purple-400"></span>
                      <span>SVG (<span v-text="siteAnalytics?.svgCount || 0"></span>)</span>
                    </button>
                    <button @click="setImageFilterTab('png')"
                      :class="imageFilterTab === 'png' ? 'bg-slate-800 text-cyan-400 font-semibold shadow-sm' : 'text-slate-400 hover:text-slate-200'"
                      class="px-2.5 py-1.5 rounded-lg transition whitespace-nowrap">
                      PNG
                    </button>
                    <button @click="setImageFilterTab('jpg')"
                      :class="imageFilterTab === 'jpg' ? 'bg-slate-800 text-cyan-400 font-semibold shadow-sm' : 'text-slate-400 hover:text-slate-200'"
                      class="px-2.5 py-1.5 rounded-lg transition whitespace-nowrap">
                      JPG / JPEG
                    </button>
                  </div>

                  <div class="flex items-center flex-wrap gap-2.5 w-full md:w-auto shrink-0">
                    <!-- Standard Search Input -->
                    <div class="relative flex items-center flex-1 md:flex-initial">
                      <div class="absolute inset-y-0 left-0 pl-2.5 flex items-center pointer-events-none text-slate-500">
                        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
                        </svg>
                      </div>
                      <input type="text" v-model="imageSearchQuery" @input="updateFilteredImages()" @keydown.escape="imageSearchQuery = ''; updateFilteredImages()"
                        placeholder="Пошук зображень..."
                        class="bg-slate-900 border border-slate-700/80 hover:border-slate-600 focus:border-cyan-500 rounded-lg pl-8 pr-7 py-1.5 text-xs text-slate-100 placeholder-slate-500 focus:outline-none w-48 sm:w-60 font-mono transition shadow-inner">
                      <template v-if="imageSearchQuery">
                        <button @click="imageSearchQuery = ''; updateFilteredImages()" class="absolute inset-y-0 right-0 pr-2 flex items-center text-slate-400 hover:text-slate-200 text-xs font-bold" title="Очистити">✕</button>
                      </template>
                    </div>

                    <!-- Matched Count Badge -->
                    <template v-if="imageSearchQuery || imageFilterTab !== 'all'">
                      <span class="text-[11px] px-2 py-0.5 rounded bg-cyan-950/60 text-cyan-300 border border-cyan-800/50 font-mono shrink-0">
                        Знайдено: <strong v-text="filteredImages.length"></strong> з <span v-text="siteAnalytics?.allImages?.length || 0"></span>
                      </span>
                    </template>

                    <select v-model="imageSortKey" @change="updateFilteredImages()" class="bg-slate-900 border border-slate-700 hover:border-slate-600 rounded-lg px-2.5 py-1.5 text-xs text-slate-200 focus:outline-none focus:border-cyan-500 font-sans">
                      <option value="size">Сортувати: За розміром</option>
                      <option value="savings">Сортувати: За економією</option>
                      <option value="duration">Сортувати: За часом</option>
                      <option value="pages">Сортувати: За сторінками</option>
                    </select>

                    <button @click="openImageStudio()"
                      class="bg-cyan-600 hover:bg-cyan-500 text-white font-semibold px-3 py-1.5 rounded-lg text-xs transition flex items-center space-x-1.5 shadow-sm"
                      title="Відкрити Image Studio для тонкого налаштування">
                      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"/>
                      </svg>
                      <span>Image Studio</span>
                      <template v-if="studioOverridesCount > 0">
                        <span class="px-1.5 py-0.5 bg-slate-950/70 rounded-md text-[10px] text-cyan-200 font-mono font-medium" v-text="studioOverridesCount"></span>
                      </template>
                    </button>
                  </div>

                </div>

                <!-- Site-Wide Image Comparison Table -->
                <div class="bg-slate-950/60 rounded-xl border border-slate-800 overflow-hidden">
                  <table class="w-full text-xs text-left font-mono border-collapse">
                    <thead class="bg-slate-950 text-slate-400 border-b border-slate-800">
                      <tr class="uppercase text-[11px] font-semibold tracking-wider">
                        <th class="p-3.5">#</th>
                        <th class="p-3.5 max-w-xs">Зображення / URL</th>
                        <th class="p-3.5 text-center">Формат</th>
                        <th class="p-3.5 text-center">Роздільна здатність</th>
                        <th class="p-3.5 text-center">Сторінок</th>
                        <th class="p-3.5 text-right">Поточний розмір</th>
                        <th class="p-3.5 text-right">WebP Оцінка</th>
                        <th class="p-3.5 text-right">Економія (KB / %)</th>
                        <th class="p-3.5 text-center">Lazy Loading</th>
                        <th class="p-3.5 text-center">Дії / Порівняння</th>
                      </tr>
                    </thead>
                    <tbody class="divide-y divide-slate-800/60">
                      <template v-if="filteredImages.length === 0">
                        <tr>
                          <td colspan="10" class="p-8 text-center text-slate-500 font-sans">
                            Не знайдено зображень за обраними фільтрами.
                          </td>
                        </tr>
                      </template>

                      <template v-for="(img, idx) in paginatedImages" :key="img.url">
                        <tr class="hover:bg-slate-800/40 transition" :class="img.isHeavy ? 'bg-rose-950/10' : ''">
                          <td class="p-3.5 text-slate-500" v-text="(imagePage - 1) * imagePerPage + idx + 1"></td>
                          
                          <td class="p-3.5 max-w-xs text-slate-200 font-medium">
                            <div class="flex items-center space-x-1.5 min-w-0">
                              <button type="button" @click.stop="openUrlInBrowser(img.url)"
                                class="hover:text-amber-400 underline font-mono text-xs truncate text-left flex-1"
                                :title="'Відкрити у браузері:\n' + img.url"
                                v-text="img.url">
                              </button>
                              <button type="button" @click.stop="copyToClipboard(img.url, 'Посилання скопійовано')"
                                class="text-slate-500 hover:text-slate-300 p-0.5 rounded transition shrink-0"
                                title="Скопіювати повне посилання">
                                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
                              </button>
                            </div>
                            <div class="flex items-center space-x-1 mt-0.5">
                              <template v-if="img.isLCP">
                                <span class="px-1.5 py-0.2 rounded text-[10px] bg-amber-500/15 text-amber-300 font-bold border border-amber-500/30 inline-flex items-center space-x-1">
                                  <span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
                                  <span>LCP Hero</span>
                                </span>
                              </template>
                              <template v-if="imageStudio.overrides[img.url]?.skip">
                                <span class="px-1.5 py-0.2 rounded text-[10px] bg-rose-500/15 text-rose-300 font-bold border border-rose-500/30 inline-flex items-center space-x-1">
                                  <span class="w-1.5 h-1.5 rounded-full bg-rose-400"></span>
                                  <span>Пропущено</span>
                                </span>
                              </template>
                              <template v-if="!imageStudio.overrides[img.url]?.skip && imageStudio.overrides[img.url]">
                                <span class="px-1.5 py-0.2 rounded text-[10px] bg-purple-500/15 text-purple-300 font-bold border border-purple-500/30 inline-flex items-center space-x-1">
                                  <span class="w-1.5 h-1.5 rounded-full bg-purple-400"></span>
                                  <span v-text="'Q: ' + (imageStudio.overrides[img.url].quality || 80) + '%'"></span>
                                </span>
                              </template>
                            </div>
                          </td>

                          <td class="p-3.5 text-center">
                            <span class="px-2 py-0.5 rounded text-[10px] font-bold uppercase"
                              :class="img.format === 'webp' || img.format === 'avif' ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30' : (img.format === 'svg' ? 'bg-purple-500/20 text-purple-300 border border-purple-500/30' : (img.format === 'png' || img.format === 'jpg' ? 'bg-amber-500/20 text-amber-300 border border-amber-500/30' : 'bg-slate-800 text-slate-300'))"
                              v-text="img.format || 'OTHER'"></span>
                          </td>

                          <td class="p-3.5 text-center text-slate-400 font-sans text-[11px]" v-text="(img.width && img.height) ? (img.width + 'x' + img.height + ' px') : '-'"></td>
                          
                          <td class="p-3.5 text-center font-bold text-cyan-400" v-text="img.pageCount + ' стор.'"></td>
                          
                          <td class="p-3.5 text-right font-bold" :class="img.isHeavy ? 'text-rose-400' : 'text-slate-200'" v-text="img.formattedSize || '0 B'"></td>
                          
                          <td class="p-3.5 text-right font-bold" :class="img.format === 'svg' ? 'text-purple-400' : 'text-emerald-400'" v-text="img.estimatedWebPFormatted || '0 B'"></td>
                          
                          <td class="p-3.5 text-right font-bold text-emerald-400">
                            <span v-text="'-' + img.estimatedSavingsFormatted + ' (' + img.estimatedSavingsPercent + '%)'"></span>
                          </td>

                          <td class="p-3.5 text-center">
                            <template v-if="img.isLazy">
                              <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-medium bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                                <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                                <span>loading="lazy"</span>
                              </span>
                            </template>
                            <template v-if="!img.isLazy">
                              <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-medium bg-rose-500/10 text-rose-400 border border-rose-500/20">
                                <span class="w-1.5 h-1.5 rounded-full bg-rose-400"></span>
                                <span>Відсутнє</span>
                              </span>
                            </template>
                          </td>

                          <td class="p-3 text-center space-x-1 whitespace-nowrap">
                            <button @click="openImageStudio(img)"
                              class="bg-cyan-600 hover:bg-cyan-500 text-white font-medium px-2 py-1 rounded text-[10px] transition shadow-xs inline-flex items-center space-x-1"
                              title="Відкрити Image Tuning Studio (Live Preview & Налаштування)">
                              <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01"/></svg>
                              <span>Studio</span>
                            </button>
                            <template v-if="img.format !== 'svg' || img.isHeavy">
                              <button @click="openImageQualityModal(img)"
                                class="bg-slate-800 hover:bg-slate-700 text-slate-200 font-medium px-2 py-1 rounded text-[10px] border border-slate-700 transition inline-flex items-center space-x-1"
                                title="Відкрити порівняння якості (Original vs Real WebP)">
                                <svg class="w-3 h-3 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/></svg>
                                <span>Якість</span>
                              </button>
                            </template>
                            <button @click="downloadSingleWebP(img.url)"
                              class="bg-slate-800 hover:bg-slate-700 text-slate-200 font-medium px-2 py-1 rounded text-[10px] border border-slate-700 transition inline-flex items-center space-x-1"
                              :title="img.format === 'svg' ? 'Зберегти оптимізований SVG' : 'Завантажити WebP файл'">
                              <svg class="w-3 h-3 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/></svg>
                              <span v-text="img.format === 'svg' ? 'SVG' : 'WebP'"></span>
                            </button>
                            <button @click="downloadOriginalImage(img.url)"
                              class="bg-slate-800 hover:bg-slate-700 text-slate-400 hover:text-slate-200 font-medium px-1.5 py-1 rounded text-[10px] border border-slate-700 transition inline-flex items-center"
                              title="Зберегти оригінальний файл на диск">
                              <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/></svg>
                            </button>
                          </td>
                        </tr>
                      </template>
                    </tbody>
                  </table>
                </div>

                <!-- Standard Reusable Pagination Bar -->
                <template v-if="filteredImages.length > 0">
                  <div class="px-6 py-3 bg-slate-950 border-t border-slate-800 rounded-b-2xl flex flex-col sm:flex-row items-center justify-between gap-3 text-xs text-slate-300 font-mono shadow-lg">
                    <div>
                      Показано <span class="font-bold text-cyan-400" v-text="filteredImages.length > 0 ? ((imagePage - 1) * imagePerPage + 1) : 0"></span> - 
                      <span class="font-bold text-cyan-400" v-text="Math.min(imagePage * imagePerPage, filteredImages.length)"></span> з 
                      <span class="font-bold text-slate-100" v-text="filteredImages.length"></span> зображень
                    </div>

                    <div class="flex items-center space-x-2">
                      <button @click="imagePage = Math.max(1, imagePage - 1)" :disabled="imagePage === 1"
                        class="px-3 py-1.5 rounded-lg bg-slate-900 border border-slate-700/80 hover:bg-slate-800 disabled:opacity-40 disabled:cursor-not-allowed transition font-sans">
                        ◄ Попередня
                      </button>

                      <span class="px-2 font-bold text-slate-200">
                        Стор. <span v-text="imagePage"></span> / <span v-text="totalImagePages"></span>
                      </span>

                      <button @click="imagePage = Math.min(totalImagePages, imagePage + 1)" :disabled="imagePage >= totalImagePages"
                        class="px-3 py-1.5 rounded-lg bg-slate-900 border border-slate-700/80 hover:bg-slate-800 disabled:opacity-40 disabled:cursor-not-allowed transition font-sans">
                        Наступна ►
                      </button>

                      <select x-model.number="imagePerPage" @change="imagePage = 1" class="bg-slate-900 border border-slate-700 rounded-lg px-2.5 py-1.5 text-xs text-slate-200 focus:outline-none focus:border-cyan-500 font-sans">
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
            </div>

            <!-- SUB-TAB 2: Runs Diff & Regression Tracker -->
            <div v-show="imageSectionTab === 'diff'" class="space-y-6">
              <div class="space-y-6">
            <!-- Header Bar with Mode Toggle & Pickers -->
            <div class="bg-slate-950/60 rounded-xl border border-slate-800 p-4 space-y-4">
              <div class="flex flex-col xl:flex-row xl:items-center justify-between gap-4">
                <div>
                  <div class="flex items-center space-x-3">
                    <div class="w-8 h-8 rounded-lg bg-fuchsia-500/10 border border-fuchsia-500/20 flex items-center justify-center text-fuchsia-400 shrink-0">
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/>
                      </svg>
                    </div>
                    <div>
                      <h2 class="text-sm font-semibold text-slate-100">Порівняння прогонів та Детектор регресій</h2>
                      <p class="text-xs text-slate-400" v-text="diffViewMode === 'packages' ? 'Пофайлове зіставлення розміру та пошук регресій між збереженими пакетами WebP на диску' : 'Порівняння реального мережевого трафіку зображень між прогонами сканування сайту'"></p>
                    </div>
                  </div>
                </div>

                <!-- View Mode Switcher -->
                <div class="inline-flex p-0.5 bg-slate-900/80 rounded-lg border border-slate-800 text-xs shrink-0">
                  <button @click="diffViewMode = 'packages'; loadExportHistoryForDiff()"
                    :class="diffViewMode === 'packages' ? 'bg-slate-800 text-slate-100 font-semibold shadow-xs' : 'text-slate-400 hover:text-slate-200'"
                    class="px-2.5 py-1 rounded-md transition-colors flex items-center space-x-1.5">
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"/></svg>
                    <span>Пакети експорту</span>
                  </button>
                  <button @click="diffViewMode = 'runs'; loadHistoryRunsForDiff()"
                    :class="diffViewMode === 'runs' ? 'bg-slate-800 text-slate-100 font-semibold shadow-xs' : 'text-slate-400 hover:text-slate-200'"
                    class="px-2.5 py-1 rounded-md transition-colors flex items-center space-x-1.5">
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/></svg>
                    <span>Сканування сайту</span>
                  </button>
                </div>
              </div>

              <!-- Pickers Bar -->
              <div class="flex flex-wrap items-center justify-between gap-3 pt-2 border-t border-slate-800/80">
                <!-- MODE 1: EXPORT PACKAGES PICKER -->
                <template v-if="diffViewMode === 'packages'">
                  <div class="flex flex-wrap items-center gap-2">
                    <div class="flex items-center space-x-2 bg-slate-900/80 px-2.5 py-1 rounded-lg border border-slate-800">
                      <span class="text-[10px] font-medium text-slate-400 uppercase">Базовий:</span>
                      <select v-model="selectedBaseExportPath" @change="runExportPackageDiff()"
                        class="bg-slate-950 border border-slate-700/80 rounded px-2 py-1 text-xs text-slate-200 outline-none focus:border-fuchsia-500 font-mono max-w-[220px] truncate">
                        <template v-for="p in exportRecordsList" :key="p.manifestPath">
                          <option :value="p.manifestPath" v-text="`${p.id} (${p.imageCount} WebP, ${(p.optimizedBytes/1024/1024).toFixed(1)} MB)`"></option>
                        </template>
                      </select>
                      <button @click="browseBaseExportFolder()" title="Обрати іншу папку через Finder"
                        class="p-1 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded border border-slate-700 transition">
                        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/></svg>
                      </button>
                    </div>

                    <svg class="w-3.5 h-3.5 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3"/></svg>

                    <div class="flex items-center space-x-2 bg-slate-900/80 px-2.5 py-1 rounded-lg border border-slate-800">
                      <span class="text-[10px] font-medium text-slate-400 uppercase">Поточний:</span>
                      <select v-model="selectedCurrentExportPath" @change="runExportPackageDiff()"
                        class="bg-slate-950 border border-slate-700/80 rounded px-2 py-1 text-xs text-slate-200 outline-none focus:border-fuchsia-500 font-mono max-w-[220px] truncate">
                        <template v-for="p in exportRecordsList" :key="p.manifestPath">
                          <option :value="p.manifestPath" v-text="`${p.id} (${p.imageCount} WebP, ${(p.optimizedBytes/1024/1024).toFixed(1)} MB)`"></option>
                        </template>
                      </select>
                      <button @click="browseCurrentExportFolder()" title="Обрати іншу папку через Finder"
                        class="p-1 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded border border-slate-700 transition">
                        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/></svg>
                      </button>
                    </div>

                    <button @click="runExportPackageDiff()" :disabled="isLoadingDiff"
                      class="px-3 py-1.5 rounded-lg bg-fuchsia-600 hover:bg-fuchsia-500 text-white text-xs font-semibold transition flex items-center space-x-1.5 shadow-xs">
                      <svg class="w-3.5 h-3.5" :class="isLoadingDiff ? 'animate-spin' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>
                      <span>Порівняти</span>
                    </button>
                  </div>
                </template>

                <!-- MODE 2: SCAN RUNS PICKER -->
                <template v-if="diffViewMode === 'runs'">
                  <div class="flex flex-wrap items-center gap-2">
                    <div class="flex items-center space-x-2 bg-slate-900/80 px-2.5 py-1 rounded-lg border border-slate-800">
                      <span class="text-[10px] font-medium text-slate-400 uppercase">Базовий:</span>
                      <select v-model="selectedBaseRunId" @change="runComparisonDiff()"
                        class="bg-slate-950 border border-slate-700/80 rounded px-2 py-1 text-xs text-slate-200 outline-none focus:border-fuchsia-500 font-mono">
                        <template v-for="r in historyRunsList" :key="r.id">
                          <option :value="r.id" v-text="`${r.formattedTime} (${r.totalUrls} стор, ${r.totalImages || 0} медіа)`"></option>
                        </template>
                      </select>
                    </div>

                    <svg class="w-3.5 h-3.5 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3"/></svg>

                    <div class="flex items-center space-x-2 bg-slate-900/80 px-2.5 py-1 rounded-lg border border-slate-800">
                      <span class="text-[10px] font-medium text-slate-400 uppercase">Поточний:</span>
                      <select v-model="selectedCurrentRunId" @change="runComparisonDiff()"
                        class="bg-slate-950 border border-slate-700/80 rounded px-2 py-1 text-xs text-slate-200 outline-none focus:border-fuchsia-500 font-mono">
                        <template v-for="r in historyRunsList" :key="r.id">
                          <option :value="r.id" v-text="`${r.formattedTime} (${r.totalUrls} стор, ${r.totalImages || 0} медіа)`"></option>
                        </template>
                      </select>
                    </div>

                    <button @click="runComparisonDiff()" :disabled="isLoadingDiff"
                      class="px-3 py-1.5 rounded-lg bg-fuchsia-600 hover:bg-fuchsia-500 text-white text-xs font-semibold transition flex items-center space-x-1.5 shadow-xs">
                      <svg class="w-3.5 h-3.5" :class="isLoadingDiff ? 'animate-spin' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>
                      <span>Порівняти</span>
                    </button>
                  </div>
                </template>
              </div>

              <!-- Summary Stat Cards -->
              <template v-if="(diffViewMode === 'packages' && exportDiffReport) || (diffViewMode === 'runs' && runsDiffResult)">
                <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 gap-3 pt-2 border-t border-slate-800/80">
                  <div class="bg-slate-900/60 rounded-lg p-3 border border-slate-800/80">
                    <div class="text-[10px] uppercase tracking-wider text-slate-500 font-medium">Всього файлів</div>
                    <div class="text-lg font-bold font-mono text-slate-100 mt-0.5" v-text="diffViewMode === 'packages' ? exportDiffReport.totalFiles : runsDiffResult.totalFiles"></div>
                  </div>

                  <div class="bg-slate-900/60 rounded-lg p-3 border border-slate-800/80">
                    <div class="text-[10px] uppercase tracking-wider text-rose-400 font-medium flex items-center space-x-1.5">
                      <span class="w-1.5 h-1.5 rounded-full bg-rose-400"></span>
                      <span>Регресії (+розмір)</span>
                    </div>
                    <div class="text-lg font-bold font-mono text-rose-400 mt-0.5" v-text="diffViewMode === 'packages' ? exportDiffReport.degradedCount : runsDiffResult.degradedCount"></div>
                  </div>

                  <div class="bg-slate-900/60 rounded-lg p-3 border border-slate-800/80">
                    <div class="text-[10px] uppercase tracking-wider text-emerald-400 font-medium flex items-center space-x-1.5">
                      <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                      <span>Покращення (-розмір)</span>
                    </div>
                    <div class="text-lg font-bold font-mono text-emerald-400 mt-0.5" v-text="diffViewMode === 'packages' ? exportDiffReport.improvedCount : runsDiffResult.improvedCount"></div>
                  </div>

                  <div class="bg-slate-900/60 rounded-lg p-3 border border-slate-800/80">
                    <div class="text-[10px] uppercase tracking-wider text-cyan-400 font-medium flex items-center space-x-1.5">
                      <span class="w-1.5 h-1.5 rounded-full bg-cyan-400"></span>
                      <span>Нові / Змінені</span>
                    </div>
                    <div class="text-lg font-bold font-mono text-cyan-400 mt-0.5" v-text="diffViewMode === 'packages' ? (exportDiffReport.newCount || 0) : (runsDiffResult.newCount || 0)"></div>
                  </div>

                  <div class="bg-slate-900/60 rounded-lg p-3 border border-slate-800/80">
                    <div class="text-[10px] uppercase tracking-wider text-slate-500 font-medium">Дельта ваги</div>
                    <div class="text-lg font-bold font-mono mt-0.5"
                      :class="(diffViewMode === 'packages' ? exportDiffReport.deltaTotalWebp : runsDiffResult.deltaTotalBytes) > 0 ? 'text-rose-400' : 'text-emerald-400'"
                      v-text="`${(diffViewMode === 'packages' ? exportDiffReport.deltaTotalWebp : runsDiffResult.deltaTotalBytes) > 0 ? '+' : ''}${((diffViewMode === 'packages' ? exportDiffReport.deltaTotalWebp : runsDiffResult.deltaTotalBytes) / 1024 / 1024).toFixed(2)} MB`"></div>
                  </div>
                </div>
              </template>
            </div>

            <!-- Toolbar Filters & Search -->
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 bg-slate-950/60 p-3 rounded-xl border border-slate-800">
              <div class="inline-flex flex-wrap items-center gap-1 bg-slate-900/80 p-1 rounded-lg border border-slate-800 text-xs">
                <button @click="runsDiffFilter = 'all'"
                  :class="runsDiffFilter === 'all' ? 'bg-slate-800 text-slate-100 font-medium' : 'text-slate-400 hover:text-slate-200'"
                  class="px-2.5 py-1 rounded-md transition-colors">
                  Всі файли
                </button>
                <button @click="runsDiffFilter = 'degraded'"
                  :class="runsDiffFilter === 'degraded' ? 'bg-rose-500/20 text-rose-300 font-medium' : 'text-slate-400 hover:text-rose-400'"
                  class="px-2.5 py-1 rounded-md transition-colors flex items-center space-x-1.5">
                  <span class="w-1.5 h-1.5 rounded-full bg-rose-400"></span>
                  <span>Регресії</span>
                  <template v-if="(diffViewMode === 'packages' ? exportDiffReport?.degradedCount : runsDiffResult?.degradedCount) > 0">
                    <span class="px-1.5 py-0.2 rounded-full bg-rose-500/30 text-rose-300 text-[10px]" v-text="diffViewMode === 'packages' ? exportDiffReport.degradedCount : runsDiffResult.degradedCount"></span>
                  </template>
                </button>
                <button @click="runsDiffFilter = 'improved'"
                  :class="runsDiffFilter === 'improved' ? 'bg-emerald-500/20 text-emerald-300 font-medium' : 'text-slate-400 hover:text-emerald-400'"
                  class="px-2.5 py-1 rounded-md transition-colors flex items-center space-x-1.5">
                  <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                  <span>Покращення</span>
                  <template v-if="(diffViewMode === 'packages' ? exportDiffReport?.improvedCount : runsDiffResult?.improvedCount) > 0">
                    <span class="px-1.5 py-0.2 rounded-full bg-emerald-500/30 text-emerald-300 text-[10px]" v-text="diffViewMode === 'packages' ? exportDiffReport.improvedCount : runsDiffResult.improvedCount"></span>
                  </template>
                </button>
                <button @click="runsDiffFilter = 'new'"
                  :class="runsDiffFilter === 'new' ? 'bg-cyan-500/20 text-cyan-300 font-medium' : 'text-slate-400 hover:text-cyan-400'"
                  class="px-2.5 py-1 rounded-md transition-colors flex items-center space-x-1.5">
                  <span class="w-1.5 h-1.5 rounded-full bg-cyan-400"></span>
                  <span>Нові</span>
                  <template v-if="(diffViewMode === 'packages' ? (exportDiffReport?.newCount || 0) : (runsDiffResult?.newCount || 0)) > 0">
                    <span class="px-1.5 py-0.2 rounded-full bg-cyan-500/30 text-cyan-300 text-[10px]" v-text="diffViewMode === 'packages' ? exportDiffReport.newCount : runsDiffResult.newCount"></span>
                  </template>
                </button>
                <button @click="runsDiffFilter = 'same'"
                  :class="runsDiffFilter === 'same' ? 'bg-slate-800 text-slate-100 font-medium' : 'text-slate-400 hover:text-slate-200'"
                  class="px-2.5 py-1 rounded-md transition-colors">
                  Без змін
                </button>
              </div>

              <div class="relative flex items-center w-full sm:w-72">
                <div class="absolute inset-y-0 left-0 pl-2.5 flex items-center pointer-events-none text-slate-500">
                  <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
                  </svg>
                </div>
                <input type="text" v-model="runsDiffSearch" @keydown.escape="runsDiffSearch = ''" placeholder="Пошук по назві або URL..."
                  class="w-full bg-slate-900/90 border border-slate-700/80 hover:border-slate-600 focus:border-cyan-500 rounded-lg pl-8 pr-7 py-1.5 text-xs text-slate-100 placeholder-slate-500 focus:outline-none font-mono transition">
                <template v-if="runsDiffSearch">
                  <button @click="runsDiffSearch = ''" class="absolute inset-y-0 right-0 pr-2 flex items-center text-slate-400 hover:text-slate-200 text-xs font-bold" title="Очистити">✕</button>
                </template>
              </div>
            </div>

            <!-- Top Pagination Bar (Visible immediately without scrolling) -->
            <div v-show="diffTotalItems > 0" class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 px-1 text-xs text-slate-400">
              <div class="flex items-center space-x-2">
                <span>Показано</span>
                <span class="font-bold text-slate-200" v-text="((diffCurrentPage - 1) * diffPageSize) + 1"></span>
                <span>–</span>
                <span class="font-bold text-slate-200" v-text="Math.min(diffCurrentPage * diffPageSize, diffTotalItems)"></span>
                <span>з</span>
                <span class="font-bold text-fuchsia-400 font-mono" v-text="diffTotalItems"></span>
                <span>файлів</span>
              </div>

              <div class="flex items-center space-x-2">
                <span class="text-[11px] text-slate-400">На сторінці:</span>
                <select x-model.number="diffPageSize" @change="diffCurrentPage = 1"
                  class="bg-slate-900 border border-slate-700 rounded-lg px-2 py-1 text-slate-200 outline-none text-xs font-mono">
                  <option :value="50">50</option>
                  <option :value="100">100</option>
                  <option :value="200">200</option>
                </select>

                <div class="flex items-center space-x-1">
                  <button @click="diffCurrentPage = 1" :disabled="diffCurrentPage === 1"
                    :class="diffCurrentPage === 1 ? 'opacity-30 cursor-not-allowed text-slate-500' : 'text-slate-300 hover:bg-slate-800'"
                    class="px-2 py-0.5 rounded border border-slate-700 font-mono text-xs">«</button>
                  <button @click="prevDiffPage()" :disabled="diffCurrentPage === 1"
                    :class="diffCurrentPage === 1 ? 'opacity-30 cursor-not-allowed text-slate-500' : 'text-slate-300 hover:bg-slate-800'"
                    class="px-2 py-0.5 rounded border border-slate-700 font-mono text-xs">‹</button>
                  <span class="px-1.5 text-slate-300 font-mono text-xs">
                    <span class="font-bold text-slate-100" v-text="diffCurrentPage"></span>/<span class="font-bold text-slate-100" v-text="diffTotalPages"></span>
                  </span>
                  <button @click="nextDiffPage()" :disabled="diffCurrentPage >= diffTotalPages"
                    :class="diffCurrentPage >= diffTotalPages ? 'opacity-30 cursor-not-allowed text-slate-500' : 'text-slate-300 hover:bg-slate-800'"
                    class="px-2 py-0.5 rounded border border-slate-700 font-mono text-xs">›</button>
                  <button @click="diffCurrentPage = diffTotalPages" :disabled="diffCurrentPage >= diffTotalPages"
                    :class="diffCurrentPage >= diffTotalPages ? 'opacity-30 cursor-not-allowed text-slate-500' : 'text-slate-300 hover:bg-slate-800'"
                    class="px-2 py-0.5 rounded border border-slate-700 font-mono text-xs">»</button>
                </div>
              </div>
            </div>

            <!-- Table of File Diffs -->
            <div class="bg-slate-950/60 rounded-xl border border-slate-800 overflow-hidden">
              <div class="overflow-x-auto max-h-[600px] overflow-y-auto">
                <table class="w-full text-left border-collapse text-xs">
                  <thead class="sticky top-0 bg-slate-900 border-b border-slate-800 text-slate-400 font-medium z-10">
                    <tr>
                      <th class="p-3 w-12 text-center">#</th>
                      <th class="p-3">Файл / Посилання</th>
                      <th class="p-3 text-right" v-text="diffViewMode === 'packages' ? 'Базовий WebP' : 'Базовий розмір'"></th>
                      <th class="p-3 text-right" v-text="diffViewMode === 'packages' ? 'Поточний WebP' : 'Поточний розмір'"></th>
                      <th class="p-3 text-right">Різниця (Дельта)</th>
                      <th class="p-3 text-center w-28">Статус</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-slate-800/60 font-mono">
                    <template v-if="activeDiffFiles.length === 0">
                      <tr>
                        <td colspan="6" class="p-8 text-center text-slate-500 font-sans">
                          Файлів за обраним фільтром не знайдено.
                        </td>
                      </tr>
                    </template>

                    <template v-for="(file, idx) in paginatedDiffFiles" :key="file.url || file.sourceUrl">
                      <tr class="hover:bg-slate-900/50 transition">
                        <td class="p-3 text-center text-slate-600" v-text="((diffCurrentPage - 1) * diffPageSize) + idx + 1"></td>
                        <td class="p-3 font-sans">
                          <button type="button" @click.stop="openUrlInBrowser(file.url || file.sourceUrl)" class="font-medium text-slate-200 hover:text-fuchsia-400 truncate block max-w-lg underline text-left" :title="'Відкрити у браузері:\n' + (file.url || file.sourceUrl)" v-text="file.basename"></button>
                          <div class="text-[10px] text-slate-500 font-mono truncate max-w-lg mt-0.5" v-text="file.url || file.sourceUrl"></div>
                        </td>
                        <td class="p-3 text-right text-slate-300" v-text="diffViewMode === 'packages' ? file.baseWebpFormatted : file.baseFormatted"></td>
                        <td class="p-3 text-right font-bold text-slate-100" v-text="diffViewMode === 'packages' ? file.currWebpFormatted : file.currentFormatted"></td>
                        <td class="p-3 text-right">
                          <span class="px-2 py-0.5 rounded-full font-medium text-[11px]"
                            :class="file.status === 'degraded' ? 'bg-rose-500/20 text-rose-300 border border-rose-500/30' : (file.status === 'improved' ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30' : (file.status === 'new' ? 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/30' : 'bg-slate-800 text-slate-400'))"
                            v-text="file.deltaFormatted"></span>
                        </td>
                        <td class="p-3 text-center font-sans">
                          <span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md text-[10px] font-medium uppercase tracking-wider"
                            :class="file.status === 'degraded' ? 'bg-rose-500/10 text-rose-400' : (file.status === 'improved' ? 'bg-emerald-500/10 text-emerald-400' : (file.status === 'new' ? 'bg-cyan-500/10 text-cyan-400' : (file.status === 'removed' ? 'bg-amber-500/10 text-amber-400' : 'bg-slate-800 text-slate-500')))">
                            <span class="w-1.5 h-1.5 rounded-full shrink-0"
                              :class="file.status === 'degraded' ? 'bg-rose-400' : (file.status === 'improved' ? 'bg-emerald-400' : (file.status === 'new' ? 'bg-cyan-400' : (file.status === 'removed' ? 'bg-amber-400' : 'bg-slate-500')))"></span>
                            <span v-text="file.status === 'degraded' ? 'Регресія' : (file.status === 'improved' ? 'Краще' : (file.status === 'new' ? 'Новий' : (file.status === 'removed' ? 'Не завантаж.' : 'Норма')))"></span>
                          </span>
                        </td>
                      </tr>
                    </template>
                  </tbody>
                </table>
              </div>

              <!-- Standard Reusable Pagination Bar -->
              <div v-show="diffTotalItems > 0" class="px-6 py-3 bg-slate-950 border-t border-slate-800 rounded-b-2xl flex flex-col sm:flex-row items-center justify-between gap-3 text-xs text-slate-300 font-mono shadow-lg">
                <div>
                  Показано <span class="font-bold text-cyan-400" v-text="diffTotalItems > 0 ? (((diffCurrentPage - 1) * diffPageSize) + 1) : 0"></span> - 
                  <span class="font-bold text-cyan-400" v-text="Math.min(diffCurrentPage * diffPageSize, diffTotalItems)"></span> з 
                  <span class="font-bold text-slate-100" v-text="diffTotalItems"></span> файлів
                </div>

                <div class="flex items-center space-x-2">
                  <button @click="prevDiffPage()" :disabled="diffCurrentPage === 1"
                    class="px-3 py-1.5 rounded-lg bg-slate-900 border border-slate-700/80 hover:bg-slate-800 disabled:opacity-40 disabled:cursor-not-allowed transition font-sans">
                    ◄ Попередня
                  </button>

                  <span class="px-2 font-bold text-slate-200">
                    Стор. <span v-text="diffCurrentPage"></span> / <span v-text="diffTotalPages"></span>
                  </span>

                  <button @click="nextDiffPage()" :disabled="diffCurrentPage >= diffTotalPages"
                    class="px-3 py-1.5 rounded-lg bg-slate-900 border border-slate-700/80 hover:bg-slate-800 disabled:opacity-40 disabled:cursor-not-allowed transition font-sans">
                    Наступна ►
                  </button>

                  <select x-model.number="diffPageSize" @change="diffCurrentPage = 1"
                    class="bg-slate-900 border border-slate-700 rounded-lg px-2.5 py-1.5 text-xs text-slate-200 focus:outline-none focus:border-cyan-500 font-sans">
                    <option :value="25">25 / стор.</option>
                    <option :value="50">50 / стор.</option>
                    <option :value="100">100 / стор.</option>
                    <option :value="250">250 / стор.</option>
                  </select>
                </div>
              </div>
            </div>
          </div>

            <!-- SUB-TAB 3: Manifest Verification & Staging QA -->
            <div v-show="imageSectionTab === 'verify'" class="space-y-6">
              <!-- Controls Header Card -->
              <div class="bg-slate-900/90 border border-slate-800 rounded-2xl p-5 shadow-xl space-y-4">
                <div class="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
                  <div class="space-y-1">
                    <h3 class="text-base font-bold text-white flex items-center space-x-2">
                      <span class="w-2.5 h-2.5 rounded-full bg-emerald-400"></span>
                      <span>Звірка з маніфестом (QA Verifier)</span>
                    </h3>
                    <p class="text-xs text-slate-400">
                      Перевіряє, що всі оптимізовані WebP-файли віддаються сервером (HTTP 200, валідні габарити) та чи замінилися старі посилання у вихідному коді сторінок.
                    </p>
                  </div>

                  <div class="flex items-center space-x-3">
                    <button @click="runManifestVerification()"
                      :disabled="verifyState?.isRunning"
                      class="px-5 py-2.5 bg-emerald-600 hover:bg-emerald-500 disabled:bg-slate-800 disabled:text-slate-500 text-white rounded-xl text-xs font-bold transition shadow-lg shadow-emerald-950/40 flex items-center space-x-2">
                      <template v-if="verifyState?.isRunning">
                        <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                        </svg>
                        <span>Звірка в процесі...</span>
                      </template>
                      <template v-else>
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"/>
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
                        </svg>
                        <span>Почати звірку</span>
                      </template>
                    </button>
                  </div>
                </div>

                <!-- Input form fields: Package Path, Target Domain, Check Pages toggle -->
                <div class="grid grid-cols-1 md:grid-cols-12 gap-4 pt-2 border-t border-slate-800/80">
                  <div class="md:col-span-6 space-y-1.5">
                    <label class="text-[11px] font-semibold text-slate-300 uppercase tracking-wider">Папка пакету або manifest.json</label>
                    <div class="flex items-center space-x-2">
                      <input v-model="verifyState.packagePath"
                        type="text"
                        placeholder="/Users/.../speedmap-webp-..."
                        class="flex-1 bg-slate-950 border border-slate-700/80 rounded-xl px-3.5 py-2 text-xs font-mono text-slate-100 placeholder-slate-600 focus:outline-none focus:border-emerald-500 transition"/>
                      <button @click="selectManifestPackageForVerify()"
                        class="px-3 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-xl text-xs font-semibold transition flex items-center space-x-1.5 whitespace-nowrap">
                        <svg class="w-3.5 h-3.5 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/>
                        </svg>
                        <span>Огляд</span>
                      </button>
                    </div>
                  </div>

                  <div class="md:col-span-4 space-y-1.5">
                    <label class="text-[11px] font-semibold text-slate-300 uppercase tracking-wider">Цільовий сайт (Target Domain)</label>
                    <input v-model="verifyState.targetURL"
                      type="text"
                      placeholder="https://uat.infuse.com"
                      class="w-full bg-slate-950 border border-slate-700/80 rounded-xl px-3.5 py-2 text-xs font-mono text-slate-100 placeholder-slate-600 focus:outline-none focus:border-emerald-500 transition"/>
                  </div>

                  <div class="md:col-span-2 flex items-end pb-2">
                    <label class="flex items-center space-x-2 cursor-pointer text-xs text-slate-300 select-none">
                      <input v-model="verifyState.checkPages" type="checkbox" class="rounded bg-slate-950 border-slate-700 text-emerald-500 focus:ring-emerald-500 h-4 w-4"/>
                      <span>Звіряти HTML сторінок</span>
                    </label>
                  </div>
                </div>

                <!-- Live Progress Bar -->
                <template v-if="verifyState?.isRunning">
                  <div class="space-y-2 pt-2 border-t border-slate-800">
                    <div class="flex items-center justify-between text-xs font-mono">
                      <span class="text-slate-400">
                        Перевірено: <strong class="text-emerald-400" v-text="verifyState.progress.current"></strong> / <span v-text="verifyState.progress.total"></span>
                        <span class="text-slate-500 ml-2">(<span v-text="verifyState.progress.currentItem"></span>)</span>
                      </span>
                      <div class="flex items-center space-x-3 text-[11px]">
                        <span class="text-emerald-400">OK: <span v-text="verifyState.progress.passedCount"></span></span>
                        <span class="text-amber-400">Увага: <span v-text="verifyState.progress.warnCount"></span></span>
                        <span class="text-rose-400">Помилки: <span v-text="verifyState.progress.failCount"></span></span>
                      </div>
                    </div>
                    <div class="w-full h-2 bg-slate-950 rounded-full overflow-hidden border border-slate-800">
                      <div class="h-full bg-emerald-500 transition-all duration-150"
                        :style="`width: ${verifyState.progress.total > 0 ? (verifyState.progress.current / verifyState.progress.total * 100) : 0}%`"></div>
                    </div>
                  </div>
                </template>
              </div>

              <!-- Summary KPI Cards -->
              <template v-if="verifyState?.result?.summary">
                <div class="grid grid-cols-2 lg:grid-cols-5 gap-3">
                  <div class="bg-slate-900/70 border border-slate-800 rounded-xl p-4 space-y-1">
                    <div class="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">Перевірено файлів</div>
                    <div class="text-2xl font-extrabold text-white font-mono" v-text="verifyState.result.summary.totalImages"></div>
                    <div class="text-[10px] text-slate-500">з маніфесту (<span v-text="verifyState.result.summary.manifestCount"></span>)</div>
                  </div>

                  <div class="bg-slate-900/70 border border-slate-800 rounded-xl p-4 space-y-1">
                    <div class="text-[11px] font-semibold text-emerald-400 uppercase tracking-wider">WebP віддаються (200 OK)</div>
                    <div class="text-2xl font-extrabold text-emerald-400 font-mono" v-text="verifyState.result.summary.passedImages"></div>
                    <div class="text-[10px] text-emerald-400/70 font-mono">
                      <span v-text="Math.round((verifyState.result.summary.passedImages / Math.max(1, verifyState.result.summary.totalImages)) * 100)"></span>% успішно
                    </div>
                  </div>

                  <div class="bg-slate-900/70 border border-slate-800 rounded-xl p-4 space-y-1">
                    <div class="text-[11px] font-semibold text-cyan-400 uppercase tracking-wider">Перевірено сторінок</div>
                    <div class="text-2xl font-extrabold text-cyan-400 font-mono" v-text="verifyState.result.summary.totalPagesChecked"></div>
                    <div class="text-[10px] text-slate-400">WebP у коді: <strong class="text-cyan-300" v-text="verifyState.result.summary.pagesWithWebp"></strong></div>
                  </div>

                  <div class="bg-slate-900/70 border border-slate-800 rounded-xl p-4 space-y-1">
                    <div class="text-[11px] font-semibold uppercase tracking-wider"
                      :class="verifyState.result.summary.failedImages > 0 ? 'text-rose-400' : 'text-slate-400'">
                      Помилки / Недоступні
                    </div>
                    <div class="text-2xl font-extrabold font-mono"
                      :class="verifyState.result.summary.failedImages > 0 ? 'text-rose-400' : 'text-slate-200'"
                      v-text="verifyState.result.summary.failedImages"></div>
                    <div class="text-[10px] text-slate-500">Увага: <span class="text-amber-400" v-text="verifyState.result.summary.warnedImages"></span></div>
                  </div>

                  <div class="bg-slate-900/70 border border-slate-800 rounded-xl p-4 space-y-1">
                    <div class="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">Час виконання</div>
                    <div class="text-2xl font-extrabold text-slate-200 font-mono">
                      <span v-text="(verifyState.result.summary.durationMs / 1000).toFixed(1)"></span>s
                    </div>
                    <div class="text-[10px] text-slate-500 font-mono" v-text="verifyState.result.summary.targetDomain"></div>
                  </div>
                </div>
              </template>

              <!-- Results Data Table -->
              <template v-if="verifyState?.result">
                <div class="bg-slate-900/80 border border-slate-800 rounded-2xl overflow-hidden shadow-xl space-y-3 p-4">
                  <!-- Table Filters & Search -->
                  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-2 border-b border-slate-800">
                    <div class="flex items-center space-x-2 text-xs">
                      <button @click="verifyState.filter = 'all'; verifyState.currentPage = 1"
                        :class="verifyState.filter === 'all' ? 'bg-slate-800 text-white font-bold' : 'text-slate-400 hover:text-slate-200'"
                        class="px-3 py-1.5 rounded-lg transition">
                        Всі (<span v-text="verifyState.result.items.length"></span>)
                      </button>
                      <button @click="verifyState.filter = 'fail'; verifyState.currentPage = 1"
                        :class="verifyState.filter === 'fail' ? 'bg-rose-500/20 text-rose-300 font-bold border border-rose-500/40' : 'text-slate-400 hover:text-rose-300'"
                        class="px-3 py-1.5 rounded-lg transition">
                        Помилки (<span v-text="verifyState.result.summary.failedImages"></span>)
                      </button>
                      <button @click="verifyState.filter = 'warn'; verifyState.currentPage = 1"
                        :class="verifyState.filter === 'warn' ? 'bg-amber-500/20 text-amber-300 font-bold border border-amber-500/40' : 'text-slate-400 hover:text-amber-300'"
                        class="px-3 py-1.5 rounded-lg transition">
                        Увага (<span v-text="verifyState.result.summary.warnedImages"></span>)
                      </button>
                      <button @click="verifyState.filter = 'pass'; verifyState.currentPage = 1"
                        :class="verifyState.filter === 'pass' ? 'bg-emerald-500/20 text-emerald-300 font-bold border border-emerald-500/40' : 'text-slate-400 hover:text-emerald-300'"
                        class="px-3 py-1.5 rounded-lg transition">
                        Успішні (<span v-text="verifyState.result.summary.passedImages"></span>)
                      </button>
                    </div>

                    <div class="flex items-center space-x-3">
                      <div class="relative">
                        <input v-model="verifyState.searchQuery"
                          type="text"
                          placeholder="Пошук за ID або назвою..."
                          class="w-60 bg-slate-950 border border-slate-800 rounded-lg pl-8 pr-3 py-1.5 text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-emerald-500"/>
                        <svg class="w-3.5 h-3.5 text-slate-500 absolute left-2.5 top-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
                        </svg>
                      </div>
                    </div>
                  </div>

                  <!-- Table Content -->
                  <div class="overflow-x-auto">
                    <table class="w-full text-left text-xs border-collapse">
                      <thead>
                        <tr class="border-b border-slate-800 text-[11px] font-semibold text-slate-400 uppercase tracking-wider">
                          <th class="py-2.5 px-3">#</th>
                          <th class="py-2.5 px-3">Файл</th>
                          <th class="py-2.5 px-3">Цільовий WebP URL</th>
                          <th class="py-2.5 px-3 text-center">HTTP</th>
                          <th class="py-2.5 px-3">Габарити (Отримано / Очікувалось)</th>
                          <th class="py-2.5 px-3">Статус на сторінках</th>
                          <th class="py-2.5 px-3">Примітки</th>
                        </tr>
                      </thead>
                      <tbody class="divide-y divide-slate-800/60 font-mono text-[11px]">
                        <tr v-for="item in paginatedVerifyItems" :key="item.id"
                          class="hover:bg-slate-800/30 transition"
                          :class="item.status === 'fail' ? 'bg-rose-950/10' : (item.status === 'warn' ? 'bg-amber-950/10' : '')">
                          <td class="py-2.5 px-3 text-slate-500 font-bold" v-text="item.id"></td>
                          <td class="py-2.5 px-3 font-sans font-medium text-slate-200 whitespace-nowrap">
                            <div class="truncate max-w-[180px]" :title="item.basename" v-text="item.basename"></div>
                          </td>
                          <td class="py-2.5 px-3">
                            <a :href="item.targetWebpUrl" target="_blank" rel="noopener"
                              class="text-emerald-400 hover:underline truncate max-w-[280px] block"
                              :title="item.targetWebpUrl" v-text="item.targetWebpUrl"></a>
                          </td>
                          <td class="py-2.5 px-3 text-center whitespace-nowrap">
                            <span class="px-2 py-0.5 rounded text-[10px] font-bold"
                              :class="item.httpStatus === 200 ? 'bg-emerald-500/20 text-emerald-300' : 'bg-rose-500/20 text-rose-300'">
                              <span v-text="item.httpStatus || 'ERR'"></span>
                            </span>
                          </td>
                          <td class="py-2.5 px-3 whitespace-nowrap">
                            <template v-if="item.width > 0">
                              <span :class="item.dimensionsMatch ? 'text-slate-300' : 'text-amber-400 font-bold'">
                                <span v-text="item.width"></span>×<span v-text="item.height"></span>
                              </span>
                              <template v-if="item.expectedWidth > 0 && !item.dimensionsMatch">
                                <span class="text-slate-500 text-[10px] ml-1">(очік. <span v-text="item.expectedWidth"></span>×<span v-text="item.expectedHeight"></span>)</span>
                              </template>
                            </template>
                            <template v-else>
                              <span class="text-slate-500">-</span>
                            </template>
                          </td>
                          <td class="py-2.5 px-3 whitespace-nowrap font-sans">
                            <template v-if="item.samplePageStatus === 'replaced'">
                              <span class="inline-flex items-center text-emerald-400 text-[11px]">
                                <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 mr-1.5"></span>
                                <span>WebP у HTML</span>
                              </span>
                            </template>
                            <template v-else-if="item.samplePageStatus === 'both'">
                              <span class="inline-flex items-center text-amber-400 text-[11px]" title="Знайдено і WebP, і старий raster (можливо у srcset)">
                                <span class="w-1.5 h-1.5 rounded-full bg-amber-400 mr-1.5"></span>
                                <span>WebP + Старий</span>
                              </span>
                            </template>
                            <template v-else-if="item.samplePageStatus === 'old_found'">
                              <span class="inline-flex items-center text-rose-400 text-[11px]" title="Старий файл знайдено, WebP відсутній у коді">
                                <span class="w-1.5 h-1.5 rounded-full bg-rose-400 mr-1.5"></span>
                                <span>Залишився старий</span>
                              </span>
                            </template>
                            <template v-else>
                              <span class="text-slate-500 text-[10px]">-</span>
                            </template>

                            <template v-if="item.samplePageUrl">
                              <a :href="item.samplePageUrl" target="_blank" rel="noopener"
                                class="text-slate-400 hover:text-slate-200 ml-2" title="Відкрити сторінку">
                                ↗
                              </a>
                            </template>
                          </td>
                          <td class="py-2.5 px-3 font-sans text-xs">
                            <template v-if="item.errors && item.errors.length">
                              <span class="text-rose-400 text-[11px]" v-text="item.errors[0]"></span>
                            </template>
                            <template v-else>
                              <span class="text-emerald-400/80 text-[11px]">OK</span>
                            </template>
                          </td>
                        </tr>
                      </tbody>
                    </table>
                  </div>

                  <!-- Pagination Controls -->
                  <div class="flex items-center justify-between pt-3 border-t border-slate-800 text-xs">
                    <div class="text-slate-400 font-mono">
                      Показано <span class="text-slate-200" v-text="paginatedVerifyItems.length"></span> з <span class="text-slate-200" v-text="filteredVerifyItems.length"></span>
                    </div>

                    <div class="flex items-center space-x-2">
                      <button @click="verifyState.currentPage = Math.max(1, verifyState.currentPage - 1)"
                        :disabled="verifyState.currentPage <= 1"
                        class="px-2.5 py-1 bg-slate-800 disabled:opacity-40 text-slate-300 rounded hover:bg-slate-700 transition">
                        ←
                      </button>
                      <span class="text-slate-300 font-mono text-xs">
                        <span v-text="verifyState.currentPage"></span> / <span v-text="totalVerifyPages"></span>
                      </span>
                      <button @click="verifyState.currentPage = Math.min(totalVerifyPages, verifyState.currentPage + 1)"
                        :disabled="verifyState.currentPage >= totalVerifyPages"
                        class="px-2.5 py-1 bg-slate-800 disabled:opacity-40 text-slate-300 rounded hover:bg-slate-700 transition">
                        →
                      </button>
                    </div>
                  </div>
                </div>
              </template>
            </div>
          </div>
        </div>
      </template>
</template>

<script>
import { useApp } from '@/store/app';

export default {
  name: 'ImagesTab',
  setup() {
    return useApp();
  }
};
</script>
