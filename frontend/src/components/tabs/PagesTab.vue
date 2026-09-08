<template>
<!-- VIEW 1: PAGES TABLE VIEW -->
        <template v-if="activeTab === 'pages'">
          <div class="flex-1 flex flex-col overflow-hidden">
            <!-- Stats Overview Cards -->
            <template v-if="scanResults.length > 0">
              <div class="grid grid-cols-2 xl:grid-cols-4 gap-3 p-4 xl:p-6 pb-2 shrink-0">
                <div class="bg-slate-950/60 border border-slate-800 rounded-xl p-3.5 xl:p-4">
                  <div class="text-[11px] font-medium text-slate-400 uppercase tracking-wider">Всього перевірено</div>
                  <div class="text-2xl font-bold font-mono text-slate-100 mt-1" v-text="scanResults.length"></div>
                </div>
                <div class="bg-slate-950/60 border border-slate-800 rounded-xl p-3.5 xl:p-4">
                  <div class="text-[11px] font-medium text-slate-400 uppercase tracking-wider flex items-center justify-between">
                    <span>Хороші</span>
                    <span class="w-2 h-2 rounded-full bg-emerald-400"></span>
                  </div>
                  <div class="text-2xl font-bold font-mono text-emerald-400 mt-1" v-text="countByStatus('good')"></div>
                </div>
                <div class="bg-slate-950/60 border border-slate-800 rounded-xl p-3.5 xl:p-4">
                  <div class="text-[11px] font-medium text-slate-400 uppercase tracking-wider flex items-center justify-between">
                    <span>Потребують уваги</span>
                    <span class="w-2 h-2 rounded-full bg-amber-400"></span>
                  </div>
                  <div class="text-2xl font-bold font-mono text-amber-400 mt-1" v-text="countByStatus('needs-improvement')"></div>
                </div>
                <div class="bg-slate-950/60 border border-slate-800 rounded-xl p-3.5 xl:p-4">
                  <div class="text-[11px] font-medium text-slate-400 uppercase tracking-wider flex items-center justify-between">
                    <span>Критичні / Помилки</span>
                    <span class="w-2 h-2 rounded-full bg-rose-400"></span>
                  </div>
                  <div class="text-2xl font-bold font-mono text-rose-400 mt-1" v-text="countByStatus('poor') + countByStatus('error')"></div>
                </div>
              </div>
            </template>

            <!-- Results Toolbar & Filters -->
            <div class="px-4 xl:px-6 py-3 flex flex-wrap items-center justify-between gap-3 border-b border-slate-800 shrink-0">
              <div class="flex items-center flex-wrap gap-2.5">
                <div class="flex items-center bg-slate-950/80 p-1 rounded-xl border border-slate-800 text-xs">
                  <button @click="statusFilter = 'all'" :class="statusFilter === 'all' ? 'bg-slate-800 text-white font-semibold shadow-sm' : 'text-slate-400 hover:text-slate-200'" class="px-3 py-1.5 rounded-lg transition">
                    Усі (<span v-text="scanResults.length"></span>)
                  </button>
                  <button @click="statusFilter = 'critical'" :class="statusFilter === 'critical' ? 'bg-slate-800 text-rose-300 font-semibold shadow-sm' : 'text-slate-400 hover:text-slate-200'" class="px-3 py-1.5 rounded-lg transition flex items-center space-x-1.5">
                    <span class="w-1.5 h-1.5 rounded-full bg-rose-400"></span>
                    <span>Критичні</span>
                  </button>
                  <button @click="statusFilter = 'needs-improvement'" :class="statusFilter === 'needs-improvement' ? 'bg-slate-800 text-amber-300 font-semibold shadow-sm' : 'text-slate-400 hover:text-slate-200'" class="px-3 py-1.5 rounded-lg transition flex items-center space-x-1.5">
                    <span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
                    <span>Потребують уваги</span>
                  </button>
                  <button @click="statusFilter = 'good'" :class="statusFilter === 'good' ? 'bg-slate-800 text-emerald-300 font-semibold shadow-sm' : 'text-slate-400 hover:text-slate-200'" class="px-3 py-1.5 rounded-lg transition flex items-center space-x-1.5">
                    <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                    <span>Хороші</span>
                  </button>
                </div>

                <!-- Live URL Search Input -->
                <div class="relative flex items-center">
                  <div class="absolute inset-y-0 left-0 pl-2.5 flex items-center pointer-events-none text-slate-500">
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
                    </svg>
                  </div>
                  <input type="text" v-model="pageSearchQuery" placeholder="Пошук за посиланням (URL)..."
                    class="bg-slate-900 border border-slate-700/80 hover:border-slate-600 focus:border-cyan-500 rounded-lg pl-8 pr-7 py-1.5 text-xs text-slate-100 placeholder-slate-500 focus:outline-none w-56 lg:w-72 font-mono transition shadow-inner">
                  <template v-if="pageSearchQuery">
                    <button @click="pageSearchQuery = ''" class="absolute inset-y-0 right-0 pr-2 flex items-center text-slate-400 hover:text-slate-200 text-xs font-bold" title="Очистити">
                      ✕
                    </button>
                  </template>
                </div>

                <!-- Matched Filter Count Badge -->
                <template v-if="pageSearchQuery || statusFilter !== 'all'">
                  <span class="text-[11px] px-2 py-0.5 rounded-md bg-slate-800 text-cyan-300 border border-slate-700 font-mono">
                    Знайдено: <strong v-text="filteredResults.length"></strong> з <span v-text="scanResults.length"></span>
                  </span>
                </template>
              </div>

              <div class="flex items-center space-x-2">
                <template v-if="siteAnalytics?.globalCandidateSelectors && siteAnalytics.globalCandidateSelectors.length > 0">
                  <div class="flex items-center space-x-1.5 mr-1">
                    <button @click="exportDOMVirtualizationCSS(siteAnalytics.globalVirtualizationCss || siteAnalytics.globalVirtualizationCSS, 'site-dom-virtualization.css')"
                      class="px-2.5 py-1.5 rounded-lg border text-xs font-semibold transition flex items-center space-x-1.5 bg-indigo-950/80 border-indigo-500/40 hover:bg-indigo-900 text-indigo-200 shadow-sm"
                      title="Експорт CSS віртуалізації DOM (content-visibility: auto)">
                      <svg class="w-3.5 h-3.5 text-indigo-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/></svg>
                      <span>CSS Віртуалізація</span>
                    </button>
                    <button @click="exportDOMVirtualizationPHP(siteAnalytics.globalVirtualizationPhp || siteAnalytics.globalVirtualizationPHP, 'site-dom-virtualization.php')"
                      class="px-2.5 py-1.5 rounded-lg border text-xs font-semibold transition flex items-center space-x-1.5 bg-slate-900 border-slate-700 hover:bg-slate-800 text-slate-300 shadow-sm"
                      title="Експорт PHP хука wp_head для WordPress">
                      <span>PHP wp_head</span>
                    </button>
                  </div>
                </template>
                <button @click="exportPagesCSV()" :disabled="scanResults.length === 0"
                  class="px-3 py-1.5 rounded-lg border text-xs font-medium transition flex items-center space-x-1.5 bg-slate-900 border-slate-700 hover:bg-slate-800 text-slate-200 disabled:opacity-40 disabled:cursor-not-allowed shadow-sm"
                  title="Експорт результатів сторінок у CSV файл">
                  <svg class="w-3.5 h-3.5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
                  </svg>
                  <span>Експорт CSV</span>
                </button>
                <div class="hidden lg:block text-xs text-slate-500">
                  Клікніть рядок для деталей
                </div>
              </div>
            </div>

            <!-- Results Table Scroll Container (Flush Sticky Header) -->
            <div class="flex-1 overflow-auto relative">
              <template v-if="scanResults.length === 0 && !isScanning">
                <div class="h-full flex flex-col items-center justify-center text-slate-500 space-y-3 p-6">
                  <svg class="w-16 h-16 stroke-1 text-slate-700" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/>
                  </svg>
                  <p class="text-sm font-medium">Результати сканування з'являться тут</p>
                </div>
              </template>

              <template v-if="scanResults.length > 0">
                <table class="w-full text-left border-collapse">
                  <thead class="sticky top-0 z-10 bg-slate-950 border-b border-slate-800 shadow-md">
                    <tr class="text-xs font-semibold text-slate-400 uppercase tracking-wider">
                      <th class="py-3 px-6">#</th>
                      <th class="py-3 px-3 max-w-[200px] xl:max-w-[240px]">URL сторінки</th>
                      <th class="py-3 px-3 text-center">Статус</th>
                      <th class="py-3 px-3 text-right">TTFB</th>
                      <th class="py-3 px-3 text-right">FCP</th>
                      <th class="py-3 px-3 text-right">LCP</th>
                      <th class="py-3 px-3 text-right">CLS</th>
                      <th class="py-3 px-3 text-right">TBT</th>
                      <th class="py-3 px-3 text-center">ШРИФТИ</th>
                      <th class="py-3 px-6 text-center">Дії</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-slate-800/60 text-xs font-mono">
                    <template v-if="filteredResults.length === 0">
                      <tr>
                        <td colspan="10" class="py-12 text-center text-slate-400">
                          <div class="flex flex-col items-center justify-center space-y-2">
                            <svg class="w-8 h-8 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
                            </svg>
                            <p class="text-sm font-medium">Сторінок за вказаним пошуковим запитом не знайдено</p>
                            <template v-if="pageSearchQuery">
                              <button @click="pageSearchQuery = ''" class="px-3 py-1 bg-slate-800 hover:bg-slate-700 text-xs text-cyan-400 rounded-lg transition">
                                Очистити пошуковий фільтр
                              </button>
                            </template>
                          </div>
                        </td>
                      </tr>
                    </template>
                    <template v-for="item in paginatedResults" :key="item.id">
                      <tr @click="selectedDetail = item" class="hover:bg-slate-800/50 cursor-pointer transition group relative"
                        :class="rescanLoadingMap[item.id] ? 'bg-cyan-950/30 ring-1 ring-cyan-500/50' : (item.overallStatus === 'error' ? 'bg-rose-950/10' : '')">
                        <td class="py-3 px-6 text-slate-500" v-text="item.id"></td>

                        <td class="py-3 px-3 font-mono max-w-[240px] xl:max-w-[300px]">
                          <div class="flex flex-col space-y-1">
                            <div class="flex items-center space-x-1.5 truncate">
                              <!-- Plugin / Redis Cache Badge -->
                              <template v-if="item.pluginCacheStatus && item.pluginCacheStatus !== 'NONE'">
                                <span class="px-1.5 py-0.5 rounded text-[10px] font-mono font-bold border shrink-0 inline-flex items-center space-x-1"
                                  :class="item.pluginCacheStatus === 'HIT' ? 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30' : (item.pluginCacheStatus === 'MISS' ? 'bg-amber-500/20 text-amber-300 border-amber-500/30' : 'bg-slate-800 text-slate-400 border-slate-700')"
                                  :title="'Plugin / Redis Cache Status: ' + item.pluginCacheStatus">
                                  <span class="w-1.5 h-1.5 rounded-full" :class="item.pluginCacheStatus === 'HIT' ? 'bg-emerald-400' : 'bg-amber-400'"></span>
                                  <span v-text="item.pluginCacheStatus"></span>
                                </span>
                              </template>

                              <!-- Cloudflare Cache Badge with PoP / Datacenter -->
                              <template v-if="item.cloudflareCacheStatus && item.cloudflareCacheStatus !== 'NONE'">
                                <span class="px-1.5 py-0.5 rounded text-[10px] font-mono font-bold border shrink-0 inline-flex items-center space-x-1"
                                  :class="item.cloudflareCacheStatus === 'HIT' ? 'bg-sky-500/20 text-sky-300 border-sky-500/30' : (item.cloudflareCacheStatus === 'MISS' ? 'bg-amber-500/20 text-amber-300 border-amber-500/30' : 'bg-indigo-500/20 text-indigo-300 border-indigo-500/30')"
                                  :title="'Cloudflare Cache: ' + item.cloudflareCacheStatus + (item.cloudflarePop ? ' · Дата-центр (PoP): ' + item.cloudflarePop : '') + (item.cloudflareRay ? '\nCF-Ray: ' + item.cloudflareRay : '')">
                                  <svg class="w-2.5 h-2.5 text-sky-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 00-9.78 2.096A4.001 4.001 0 003 15z"/>
                                  </svg>
                                  <span v-text="item.cloudflareCacheStatus + (item.cloudflarePop ? ' (' + item.cloudflarePop + ')' : '')"></span>
                                </span>
                              </template>
                            </div>

                            <button @click.stop="openUrlInBrowser(item.url)"
                              class="text-slate-200 hover:text-emerald-400 underline decoration-slate-700 hover:decoration-emerald-400 transition inline-flex items-center space-x-1 text-left group-hover:text-white truncate"
                              :title="'Відкрити сторінку у браузері:\n' + item.url">
                              <span class="truncate" v-text="item.url"></span>
                              <span class="text-slate-500 group-hover:text-emerald-400 text-[10px] shrink-0">↗</span>
                            </button>

                            <!-- Immediate Page Fonts Display directly in table row -->
                            <div class="pt-1 flex items-center flex-wrap gap-1 font-sans">
                              <template v-if="item.diagnostics?.fonts && item.diagnostics?.fonts.length > 0">
                                <div class="flex items-center flex-wrap gap-1">
                                  <svg class="w-3 h-3 text-purple-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 5h12M9 5v14m4 0h6m-3-7v7M16 12h6"/>
                                  </svg>
                                  <template v-for="f in item.diagnostics.fonts" :key="f.family">
                                    <span class="px-1.5 py-0.2 rounded text-[10px] bg-purple-950/80 text-purple-200 border border-purple-800/60 font-semibold shadow-sm" :title="f.family + ' (' + (f.type || 'font') + ')'" v-text="f.family"></span>
                                  </template>
                                </div>
                              </template>
                              <template v-if="!item.diagnostics?.fonts || item.diagnostics?.fonts.length === 0">
                                <span class="text-[10px] text-slate-500 italic flex items-center space-x-1">
                                  <svg class="w-2.5 h-2.5 text-slate-500 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 5h12M9 5v14m4 0h6m-3-7v7M16 12h6"/>
                                  </svg>
                                  <span>системні шрифти</span>
                                </span>
                              </template>
                              <template v-if="item.diagnostics?.iframes && item.diagnostics.iframes.length > 0">
                                <span class="px-1.5 py-0.2 rounded text-[10px] bg-rose-950/80 text-rose-200 border border-rose-800/60 font-semibold"
                                  :title="item.diagnostics.iframes.filter(f => !f.loadedDuringScan).length + ' пропущено під час load'"
                                  v-text="'⧉ ' + item.diagnostics.iframes.length + ' iframe' + (item.diagnostics.iframes.filter(f => !f.loadedDuringScan).length ? ' · ' + item.diagnostics.iframes.filter(f => !f.loadedDuringScan).length + ' missed' : '')"></span>
                              </template>
                            </div>
                          </div>
                        </td>

                        <td class="py-3 px-3 text-center">
                          <template v-if="rescanLoadingMap[item.id]">
                            <span class="px-2 py-0.5 rounded text-[11px] font-semibold inline-flex items-center space-x-1 bg-cyan-500/20 text-cyan-300 border border-cyan-500/40">
                              <svg class="animate-spin h-3 w-3 text-cyan-300" fill="none" viewBox="0 0 24 24">
                                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                              </svg>
                              <span>Сканується...</span>
                            </span>
                          </template>
                          <template v-if="!rescanLoadingMap[item.id]">
                            <span class="px-2 py-0.5 rounded text-[11px] font-semibold inline-flex items-center space-x-1"
                              :title="item.error || 'HTTP Status ' + item.statusCode"
                              :class="item.statusCode >= 200 && item.statusCode < 300 && !item.error ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20' : 'bg-rose-500/10 text-rose-400 border border-rose-500/20'">
                              <span v-text="item.statusCode || 'ERR'"></span>
                              <template v-if="item.error">
                                <svg class="w-3 h-3 text-rose-400 ml-0.5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
                                </svg>
                              </template>
                            </span>
                          </template>
                        </td>

                        <td class="py-3 px-3 text-right">
                          <span class="px-2 py-0.5 rounded text-[11px] font-medium" :class="badgeClass(item.grades?.ttfb?.status)" v-text="item.grades?.ttfb?.formatted || '-'"></span>
                        </td>

                        <td class="py-3 px-3 text-right">
                          <span class="px-2 py-0.5 rounded text-[11px] font-medium" :class="badgeClass(item.grades?.fcp?.status)" v-text="item.grades?.fcp?.formatted || '-'"></span>
                        </td>

                        <td class="py-3 px-3 text-right">
                          <span class="px-2 py-0.5 rounded text-[11px] font-medium" :class="badgeClass(item.grades?.lcp?.status)" v-text="item.grades?.lcp?.formatted || '-'"></span>
                        </td>

                        <td class="py-3 px-3 text-right">
                          <span class="px-2 py-0.5 rounded text-[11px] font-medium" :class="badgeClass(item.grades?.cls?.status)" v-text="item.grades?.cls?.formatted || '-'"></span>
                        </td>

                        <td class="py-3 px-3 text-right">
                          <span class="px-2 py-0.5 rounded text-[11px] font-medium" :class="badgeClass(item.grades?.tbt?.status)" v-text="item.grades?.tbt?.formatted || '-'"></span>
                        </td>

                        <td class="py-3 px-6 text-center">
                          <div class="flex items-center justify-center space-x-1.5">
                            <button type="button" @click.stop.prevent="rescanSingle(item)" :disabled="!!rescanLoadingMap[item.id]"
                              class="p-1 rounded-lg bg-slate-800/80 hover:bg-cyan-600 text-slate-300 hover:text-white transition disabled:opacity-50" title="Пріоритетно пересканувати цю сторінку">
                              <svg v-show="!rescanLoadingMap[item.id]" class="w-3.5 h-3.5 pointer-events-none" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                              </svg>
                              <svg v-show="!!rescanLoadingMap[item.id]" class="animate-spin h-3.5 w-3.5 text-cyan-400 pointer-events-none" fill="none" viewBox="0 0 24 24">
                                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                              </svg>
                            </button>
                            <button type="button" @click.stop="openUrlInBrowser(item.url)" class="p-1 rounded-lg bg-slate-800/80 hover:bg-slate-700 text-slate-400 hover:text-slate-200 transition" title="Відкрити сторінку у браузері">
                              <svg class="w-3.5 h-3.5 pointer-events-none" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/>
                              </svg>
                            </button>
                          </div>
                        </td>
                      </tr>
                    </template>
                  </tbody>
                </table>
              </template>
            </div>

            <!-- Pages High-Performance Pagination Bar -->
            <template v-if="filteredResults.length > 0">
              <div class="px-6 py-3 bg-slate-950 border-t border-slate-800 flex flex-col sm:flex-row items-center justify-between gap-3 text-xs text-slate-300 font-mono shrink-0 shadow-lg">
                <div>
                  Показано <span class="font-bold text-cyan-400" v-text="filteredResults.length > 0 ? ((pagesPage - 1) * pagesPerPage + 1) : 0"></span> - 
                  <span class="font-bold text-cyan-400" v-text="Math.min(pagesPage * pagesPerPage, filteredResults.length)"></span> з 
                  <span class="font-bold text-slate-100" v-text="filteredResults.length"></span> сторінок
                </div>

                <div class="flex items-center space-x-2">
                  <button @click="pagesPage = Math.max(1, pagesPage - 1)" :disabled="pagesPage === 1"
                    class="px-3 py-1.5 rounded-lg bg-slate-900 border border-slate-700/80 hover:bg-slate-800 disabled:opacity-40 disabled:cursor-not-allowed transition font-sans">
                    ◄ Попередня
                  </button>

                  <span class="px-2 font-bold text-slate-200">
                    Стор. <span v-text="pagesPage"></span> / <span v-text="totalPagesCount"></span>
                  </span>

                  <button @click="pagesPage = Math.min(totalPagesCount, pagesPage + 1)" :disabled="pagesPage >= totalPagesCount"
                    class="px-3 py-1.5 rounded-lg bg-slate-900 border border-slate-700/80 hover:bg-slate-800 disabled:opacity-40 disabled:cursor-not-allowed transition font-sans">
                    Наступна ►
                  </button>

                  <select x-model.number="pagesPerPage" @change="pagesPage = 1" class="bg-slate-900 border border-slate-700 rounded-lg px-2 py-1.5 text-xs text-slate-200 focus:outline-none">
                    <option value="25">25 / стор.</option>
                    <option value="50">50 / стор.</option>
                    <option value="100">100 / стор.</option>
                    <option value="250">250 / стор.</option>
                    <option value="500">500 / стор.</option>
                  </select>
                </div>
              </div>
            </template>
          </div>
        </template>
</template>

<script>
import { useApp } from '@/store/app';

export default {
  name: 'PagesTab',
  setup() {
    return useApp();
  }
};
</script>
