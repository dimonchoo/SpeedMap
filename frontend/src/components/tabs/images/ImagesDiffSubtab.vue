<template>
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
        </div>

</template>

<script>
import { useApp } from '@/store/app';

export default {
  name: 'ImagesDiffSubtab',
  setup() {
    return useApp();
  }
};
</script>
