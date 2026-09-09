<template>
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
                        Всі (<span v-text="verifyState.result?.items?.length || 0"></span>)
                      </button>
                      <button @click="verifyState.filter = 'fail'; verifyState.currentPage = 1"
                        :class="verifyState.filter === 'fail' ? 'bg-rose-500/20 text-rose-300 font-bold border border-rose-500/40' : 'text-slate-400 hover:text-rose-300'"
                        class="px-3 py-1.5 rounded-lg transition">
                        Помилки (<span v-text="verifyState.result?.summary?.failedImages || 0"></span>)
                      </button>
                      <button @click="verifyState.filter = 'warn'; verifyState.currentPage = 1"
                        :class="verifyState.filter === 'warn' ? 'bg-amber-500/20 text-amber-300 font-bold border border-amber-500/40' : 'text-slate-400 hover:text-amber-300'"
                        class="px-3 py-1.5 rounded-lg transition">
                        Увага (<span v-text="verifyState.result?.summary?.warnedImages || 0"></span>)
                      </button>
                      <button @click="verifyState.filter = 'pass'; verifyState.currentPage = 1"
                        :class="verifyState.filter === 'pass' ? 'bg-emerald-500/20 text-emerald-300 font-bold border border-emerald-500/40' : 'text-slate-400 hover:text-emerald-300'"
                        class="px-3 py-1.5 rounded-lg transition">
                        Успішні (<span v-text="verifyState.result?.summary?.passedImages || 0"></span>)
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

</template>

<script>
import { useApp } from '@/store/app';

export default {
  name: 'ImagesVerifySubtab',
  setup() {
    return useApp();
  }
};
</script>
