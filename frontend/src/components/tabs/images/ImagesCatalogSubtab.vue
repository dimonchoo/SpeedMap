<template>
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

</template>

<script>
import { useApp } from '@/store/app';

export default {
  name: 'ImagesCatalogSubtab',
  setup() {
    return useApp();
  }
};
</script>
