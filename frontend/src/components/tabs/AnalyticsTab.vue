<template>
<!-- VIEW 2: SITE ANALYTICS & RUN COMPARISON DASHBOARD -->
        <template v-if="activeTab === 'analytics'">
          <div class="flex-1 overflow-y-auto p-4 xl:p-6 space-y-6">
            
            <template v-if="!siteAnalytics">
              <div class="h-full flex flex-col items-center justify-center text-slate-500 space-y-3 py-16">
                <svg class="w-14 h-14 text-slate-700 animate-bounce" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/>
                </svg>
                <p class="text-sm font-medium">Запустіть або завершіть сканування для розрахунку загальної статистики сайту</p>
              </div>
            </template>

            <template v-if="siteAnalytics">
              <div class="space-y-6">
                
                <!-- Health Score & Run Comparison Delta Banner -->
                <div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
                  
                  <!-- Site Health Scorecard -->
                  <div class="bg-slate-950/60 p-5 rounded-xl border border-slate-800 flex flex-col justify-between">
                    <div class="space-y-1">
                      <span class="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">Загальний стан сайту</span>
                      <div class="flex items-baseline space-x-2 mt-1">
                        <span class="text-4xl font-extrabold text-emerald-400 font-mono" v-text="siteAnalytics.healthScore"></span>
                        <span class="text-lg font-medium text-slate-500 font-mono">/ 100</span>
                      </div>
                    </div>
                    
                    <div class="mt-4 pt-3 border-t border-slate-800/80 flex items-center justify-between text-xs">
                      <span class="text-slate-400">Перевірено сторінок:</span>
                      <span class="font-bold text-slate-200 font-mono" v-text="siteAnalytics.totalPages"></span>
                    </div>
                  </div>

                  <!-- Run Comparison Delta Card -->
                  <div class="col-span-1 lg:col-span-2 bg-slate-950/60 p-5 rounded-xl border border-slate-800 space-y-3 flex flex-col justify-between">
                    <div class="flex items-center justify-between">
                      <h3 class="text-[11px] font-semibold text-slate-300 uppercase tracking-wider flex items-center space-x-2">
                        <svg class="w-3.5 h-3.5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6"/>
                        </svg>
                        <span>Порівняння з попереднім скануванням</span>
                      </h3>
                      <template v-if="runComparison?.hasPrevious">
                        <span class="text-xs text-slate-400 font-mono" v-text="'Попередній: ' + runComparison.previousTime"></span>
                      </template>
                    </div>

                    <p class="text-xs font-semibold leading-relaxed"
                      :class="runComparison?.summaryStatus === 'improved' ? 'text-emerald-400' : (runComparison?.summaryStatus === 'degraded' ? 'text-rose-400' : 'text-slate-300')"
                      v-text="runComparison?.summaryText || 'Перший прогон для даного сайту'"></p>

                    <!-- Metric Comparison Deltas Grid -->
                    <template v-if="runComparison?.hasPrevious">
                      <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 gap-2 pt-2 border-t border-slate-800 text-xs font-mono">
                        <template v-for="(val, key) in runComparison.metricDeltas" :key="key">
                          <div class="bg-slate-900/80 p-2 rounded-lg border border-slate-800 text-center space-y-0.5">
                            <div class="text-[10px] text-slate-400 font-sans font-medium" v-text="key"></div>
                            <div class="font-bold text-xs" :class="val < 0 ? 'text-emerald-400' : (val > 0 ? 'text-rose-400' : 'text-slate-400')">
                              <span v-text="val > 0 ? '+' + val : val"></span>
                            </div>
                          </div>
                        </template>
                      </div>
                    </template>
                  </div>

                </div>

                <!-- Site-Wide Averages Grid -->
                <div class="bg-slate-950/60 p-5 rounded-xl border border-slate-800 space-y-3">
                  <h3 class="text-[11px] font-semibold text-slate-300 uppercase tracking-wider flex items-center space-x-2">
                    <svg class="w-3.5 h-3.5 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
                    </svg>
                    <span>Середні показники Web Vitals по всіх сторінках</span>
                  </h3>
                  <div class="grid grid-cols-2 sm:grid-cols-3 xl:grid-cols-5 gap-3 font-mono text-center">
                    <div class="bg-slate-900/80 p-3 rounded-lg border border-slate-800">
                      <div class="text-[10px] font-sans text-slate-400 font-medium mb-1">TTFB</div>
                      <div class="text-lg font-bold text-slate-100" v-text="(siteAnalytics.averageMetrics?.TTFB || 0) + 'ms'"></div>
                    </div>
                    <div class="bg-slate-900/80 p-3 rounded-lg border border-slate-800">
                      <div class="text-[10px] font-sans text-slate-400 font-medium mb-1">FCP</div>
                      <div class="text-lg font-bold text-slate-100" v-text="(siteAnalytics.averageMetrics?.FCP || 0) + 'ms'"></div>
                    </div>
                    <div class="bg-slate-900/80 p-3 rounded-lg border border-slate-800">
                      <div class="text-[10px] font-sans text-slate-400 font-medium mb-1">LCP</div>
                      <div class="text-lg font-bold text-slate-100" v-text="(siteAnalytics.averageMetrics?.LCP || 0) + 'ms'"></div>
                    </div>
                    <div class="bg-slate-900/80 p-3 rounded-lg border border-slate-800">
                      <div class="text-[10px] font-sans text-slate-400 font-medium mb-1">CLS</div>
                      <div class="text-lg font-bold text-slate-100" v-text="siteAnalytics.averageMetrics?.CLS || 0"></div>
                    </div>
                    <div class="bg-slate-900/80 p-3 rounded-lg border border-slate-800">
                      <div class="text-[10px] font-sans text-slate-400 font-medium mb-1">TBT</div>
                      <div class="text-lg font-bold text-slate-100" v-text="(siteAnalytics.averageMetrics?.TBT || 0) + 'ms'"></div>
                    </div>
                  </div>
                </div>

                <!-- Top Resource Bottlenecks Across Entire Site -->
                <template v-if="siteAnalytics.topResourceBottlenecks && siteAnalytics.topResourceBottlenecks.length > 0">
                  <div class="space-y-3">
                    <div class="flex items-center justify-between">
                      <h3 class="text-[11px] font-semibold text-slate-300 uppercase tracking-wider flex items-center space-x-2">
                        <svg class="w-3.5 h-3.5 text-rose-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
                        </svg>
                        <span>Найповільніші ресурси сайту</span>
                      </h3>
                      <span class="text-xs text-slate-400">Ранжовано за затримкою на всьому сайті</span>
                    </div>

                    <div class="bg-slate-950/60 rounded-2xl border border-slate-800 overflow-x-auto">
                      <table class="w-full text-xs text-left font-mono min-w-[600px]">
                        <thead class="bg-slate-900 text-slate-400 border-b border-slate-800">
                          <tr>
                            <th class="p-3.5">Ресурс / Файл</th>
                            <th class="p-3.5">Тип</th>
                            <th class="p-3.5 text-center">Зустрічається</th>
                            <th class="p-3.5 text-right">Сер. час (ms)</th>
                            <th class="p-3.5 text-right">Сумарна затримка</th>
                          </tr>
                        </thead>
                        <tbody class="divide-y divide-slate-800/60">
                          <template v-for="res in siteAnalytics.topResourceBottlenecks" :key="res.name">
                            <tr class="hover:bg-slate-800/40">
                              <td class="p-3.5 max-w-md truncate text-slate-200 font-medium" v-text="res.name" :title="res.name"></td>
                              <td class="p-3.5 text-slate-400 text-[11px]" v-text="res.type"></td>
                              <td class="p-3.5 text-center font-bold text-cyan-400">
                                <span v-text="res.occurrences + ' сторінок'"></span>
                              </td>
                              <td class="p-3.5 text-right text-slate-300" v-text="res.avgDurationMs + 'ms'"></td>
                              <td class="p-3.5 text-right font-bold text-rose-400" v-text="res.totalDurationMs + 'ms'"></td>
                            </tr>
                          </template>
                        </tbody>
                      </table>
                    </div>
                  </div>
                </template>

                <!-- Largest Images Across Site -->
                <template v-if="siteAnalytics.largestImages && siteAnalytics.largestImages.length > 0">
                  <div class="space-y-3">
                    <div class="flex items-center justify-between">
                      <h3 class="text-xs font-bold text-slate-300 uppercase tracking-wider flex items-center space-x-1.5">
                        <svg class="w-3.5 h-3.5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/>
                        </svg>
                        <span>Найбільші зображення сайту (Largest Images)</span>
                      </h3>
                      <span class="text-xs text-slate-400">Ранжовано за обсягом переданих даних</span>
                    </div>

                    <div class="bg-slate-950/60 rounded-2xl border border-slate-800 overflow-x-auto">
                      <table class="w-full text-xs text-left font-mono min-w-[600px]">
                        <thead class="bg-slate-900 text-slate-400 border-b border-slate-800">
                          <tr>
                            <th class="p-3.5">Зображення / URL</th>
                            <th class="p-3.5 text-center">Роздільна здатність</th>
                            <th class="p-3.5 text-center">Сторінок</th>
                            <th class="p-3.5 text-right">Розмір файлу</th>
                            <th class="p-3.5 text-right">Завантаження</th>
                          </tr>
                        </thead>
                        <tbody class="divide-y divide-slate-800/60">
                          <template v-for="img in siteAnalytics.largestImages" :key="img.url">
                            <tr class="hover:bg-slate-800/40">
                              <td class="p-3.5 max-w-xs text-slate-200 font-medium">
                                <div class="flex items-center space-x-1.5 min-w-0">
                                  <button type="button" @click.stop="openUrlInBrowser(img.url)" class="hover:text-emerald-400 underline font-mono text-left truncate flex-1" :title="'Відкрити у браузері:\n' + img.url" v-text="img.url"></button>
                                  <button type="button" @click.stop="copyToClipboard(img.url, 'Посилання скопійовано')" class="text-slate-500 hover:text-slate-300 p-0.5 rounded transition shrink-0" title="Скопіювати повне посилання">
                                    <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
                                  </button>
                                </div>
                              </td>
                              <td class="p-3.5 text-center text-slate-400 font-sans text-[11px]" v-text="(img.width && img.height) ? (img.width + 'x' + img.height + ' px') : '-'"></td>
                              <td class="p-3.5 text-center font-bold text-cyan-400">
                                <span v-text="img.pageCount + ' стор.'"></span>
                              </td>
                              <td class="p-3.5 text-right font-bold text-amber-400" v-text="img.formattedSize || '0 B'"></td>
                              <td class="p-3.5 text-right text-slate-300" v-text="img.avgDurationMs + 'ms'"></td>
                            </tr>
                          </template>
                        </tbody>
                      </table>
                    </div>
                  </div>
                </template>

                <!-- Font Usage & Frequency Across Site -->
                <template v-if="siteAnalytics.fontUsage && siteAnalytics.fontUsage.length > 0">
                  <div class="space-y-3">
                    <div class="flex items-center justify-between">
                      <h3 class="text-xs font-bold text-slate-300 uppercase tracking-wider flex items-center space-x-1.5">
                        <svg class="w-3.5 h-3.5 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 5h12M9 5v14m4 0h6m-3-7v7M16 12h6"/>
                        </svg>
                        <span>Аналіз шрифтів сайту (Font Usage & Frequency)</span>
                      </h3>
                      <span class="text-xs text-slate-400">Частота використання гарнітур на сторінках</span>
                    </div>

                    <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
                      <template v-for="font in siteAnalytics.fontUsage" :key="font.family">
                        <div class="bg-slate-950/60 p-4 rounded-xl border border-slate-800 space-y-2 relative overflow-hidden">
                          <div class="flex items-center justify-between">
                            <span class="font-bold text-sm text-slate-100 font-sans truncate" v-text="font.family"></span>
                            <span class="px-1.5 py-0.5 rounded text-[10px] font-mono uppercase bg-indigo-500/20 text-indigo-300 border border-indigo-500/30" v-text="font.type || 'font'"></span>
                          </div>
                          <div class="flex items-baseline justify-between text-xs pt-1 border-t border-slate-800/60">
                            <span class="text-slate-400 font-sans">Покриття:</span>
                            <span class="font-bold text-emerald-400 font-mono" v-text="font.occurrences + ' стор. (' + font.percentage + '%)'"></span>
                          </div>
                          <template v-if="font.avgDurationMs > 0">
                            <div class="flex items-baseline justify-between text-[11px] text-slate-400">
                              <span>Час завантаження:</span>
                              <span class="font-mono text-slate-300" v-text="font.avgDurationMs + 'ms'"></span>
                            </div>
                          </template>
                        </div>
                      </template>
                    </div>
                  </div>
                </template>

                <!-- DOM Virtualization & Rendering Optimization Card -->
                <template v-if="siteAnalytics.globalCandidateSelectors && siteAnalytics.globalCandidateSelectors.length > 0">
                  <div class="bg-gradient-to-br from-indigo-950/40 via-slate-950/70 to-slate-900/80 p-5 rounded-2xl border border-indigo-500/30 space-y-4 shadow-xl">
                    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-indigo-500/20 pb-3">
                      <div class="space-y-1">
                        <div class="flex items-center space-x-2">
                          <svg class="w-5 h-5 text-indigo-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                          </svg>
                          <h3 class="text-sm font-bold text-slate-100 tracking-wide flex items-center space-x-2">
                            <span>Глобальна віртуалізація DOM (content-visibility: auto)</span>
                            <span class="px-2 py-0.5 rounded-full text-[10px] font-mono font-medium bg-indigo-500/20 text-indigo-300 border border-indigo-500/30">LCP Boost ~30-40% ⚡</span>
                          </h3>
                        </div>
                        <p class="text-xs text-slate-400 leading-relaxed">
                          Відкладений рендеринг масивних блоків нижче першого екрана. Браузер пропускає розрахунок розкладки та відмальовку до моменту наближення до в'юпорту, суттєво зменшуючи блокування Main Thread та покращуючи LCP/FCP.
                        </p>
                      </div>

                      <!-- Stats Pills -->
                      <div class="flex items-center gap-2 shrink-0 text-xs font-mono">
                        <div class="bg-slate-900/90 px-3 py-1.5 rounded-lg border border-slate-800 text-center">
                          <span class="text-slate-400 text-[10px] block">Сер. вузлів:</span>
                          <span class="font-bold text-slate-100" v-text="siteAnalytics.averageDomNodesPerPage || siteAnalytics.averageDOMNodesPerPage || 0"></span>
                        </div>
                        <div class="bg-slate-900/90 px-3 py-1.5 rounded-lg border border-slate-800 text-center">
                          <span class="text-slate-400 text-[10px] block">Важких стор.:</span>
                          <span class="font-bold text-amber-400" v-text="siteAnalytics.heavyDomPagesCount || siteAnalytics.heavyDOMPagesCount || 0"></span>
                        </div>
                        <div class="bg-slate-900/90 px-3 py-1.5 rounded-lg border border-slate-800 text-center">
                          <span class="text-slate-400 text-[10px] block">Селекторів:</span>
                          <span class="font-bold text-indigo-400" v-text="siteAnalytics.globalCandidateSelectors?.length || 0"></span>
                        </div>
                      </div>
                    </div>

                    <!-- Discovered Selectors Badges -->
                    <div class="space-y-1.5">
                      <div class="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">Знайдені кандидати для оптимізації на сайті:</div>
                      <div class="flex flex-wrap gap-1.5">
                        <template v-for="sel in siteAnalytics.globalCandidateSelectors" :key="sel">
                          <span class="px-2.5 py-1 rounded-md text-xs font-mono bg-slate-900/90 border border-indigo-500/20 text-indigo-300 font-semibold" v-text="sel"></span>
                        </template>
                      </div>
                    </div>

                    <!-- Code Preview Tabs and Actions -->
                    <div class="bg-slate-950/80 rounded-xl border border-slate-800/80 overflow-hidden">
                      <div class="flex flex-col sm:flex-row sm:items-center justify-between px-3 py-2 bg-slate-900/80 border-b border-slate-800 gap-2 text-xs">
                        <div class="flex items-center space-x-2">
                          <button @click="domVirtTab = 'css'"
                            :class="domVirtTab === 'css' ? 'bg-indigo-600 text-white shadow-sm' : 'text-slate-400 hover:text-slate-200'"
                            class="px-2.5 py-1 rounded-md transition font-semibold">
                            CSS Стилі
                          </button>
                          <button @click="domVirtTab = 'php'"
                            :class="domVirtTab === 'php' ? 'bg-indigo-600 text-white shadow-sm' : 'text-slate-400 hover:text-slate-200'"
                            class="px-2.5 py-1 rounded-md transition font-semibold">
                            WordPress functions.php (wp_head)
                          </button>
                        </div>

                        <div class="flex items-center space-x-2">
                          <template v-if="domVirtTab === 'css'">
                            <div class="flex items-center space-x-2">
                              <button @click="copyToClipboard(siteAnalytics.globalVirtualizationCss || siteAnalytics.globalVirtualizationCSS, 'Глобальний CSS скопійовано! 📋')"
                                class="px-2.5 py-1 bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 rounded-md transition flex items-center space-x-1.5 text-[11px] font-semibold">
                                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
                                <span>Копіювати CSS</span>
                              </button>
                              <button @click="exportDOMVirtualizationCSS(siteAnalytics.globalVirtualizationCss || siteAnalytics.globalVirtualizationCSS, 'site-dom-virtualization.css')"
                                class="px-2.5 py-1 bg-indigo-600 hover:bg-indigo-500 text-white rounded-md transition flex items-center space-x-1.5 text-[11px] font-semibold">
                                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/></svg>
                                <span>Зберегти .css</span>
                              </button>
                            </div>
                          </template>
                          <template v-if="domVirtTab === 'php'">
                            <div class="flex items-center space-x-2">
                              <button @click="copyToClipboard(siteAnalytics.globalVirtualizationPhp || siteAnalytics.globalVirtualizationPHP, 'PHP wp_head хук скопійовано! 📋')"
                                class="px-2.5 py-1 bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 rounded-md transition flex items-center space-x-1.5 text-[11px] font-semibold">
                                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
                                <span>Копіювати PHP</span>
                              </button>
                              <button @click="exportDOMVirtualizationPHP(siteAnalytics.globalVirtualizationPhp || siteAnalytics.globalVirtualizationPHP, 'site-dom-virtualization.php')"
                                class="px-2.5 py-1 bg-indigo-600 hover:bg-indigo-500 text-white rounded-md transition flex items-center space-x-1.5 text-[11px] font-semibold">
                                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/></svg>
                                <span>Зберегти .php</span>
                              </button>
                            </div>
                          </template>
                        </div>
                      </div>

                      <div class="p-3 max-h-48 overflow-y-auto">
                        <pre class="text-[11px] font-mono text-indigo-200 whitespace-pre leading-relaxed select-all" v-text="domVirtTab === 'css' ? (siteAnalytics.globalVirtualizationCss || siteAnalytics.globalVirtualizationCSS) : (siteAnalytics.globalVirtualizationPhp || siteAnalytics.globalVirtualizationPHP)"></pre>
                      </div>
                    </div>
                  </div>
                </template>

                <!-- Site-Wide Action Plan -->
                <template v-if="siteAnalytics.globalFixes && siteAnalytics.globalFixes.length > 0">
                  <div class="space-y-3">
                    <h3 class="text-xs font-bold text-slate-300 uppercase tracking-wider flex items-center space-x-1.5">
                      <svg class="w-3.5 h-3.5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
                      </svg>
                      <span>Глобальний план дій для розробників</span>
                    </h3>
                    <div class="space-y-2">
                      <template v-for="fix in siteAnalytics.globalFixes" :key="fix">
                        <div class="bg-slate-950/60 p-4 rounded-xl border border-emerald-500/30 text-xs text-slate-200 leading-relaxed flex items-start space-x-3 shadow-sm">
                          <span class="text-emerald-400 text-lg leading-none shrink-0">•</span>
                          <span v-text="fix"></span>
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
  name: 'AnalyticsTab',
  setup() {
    return useApp();
  }
};
</script>
