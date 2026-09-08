<template>
<!-- SIDE-BY-SIDE IMAGE QUALITY INSPECTOR MODAL -->
    <div v-show="selectedImageComparison" class="fixed inset-0 z-50 overflow-y-auto" role="dialog" aria-modal="true">
      <div class="flex items-center justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
        
        <div class="fixed inset-0 bg-slate-950/80 backdrop-blur-md transition-opacity" @click="selectedImageComparison = null"></div>
        <span class="hidden sm:inline-block sm:align-middle sm:h-screen" aria-hidden="true">&#8203;</span>

        <div class="inline-block align-bottom bg-slate-900 rounded-3xl border border-slate-800 text-left overflow-hidden shadow-2xl transform transition-all sm:my-8 sm:align-middle sm:max-w-5xl sm:w-full">
          
          <!-- Modal Header -->
          <div class="p-6 border-b border-slate-800 flex items-center justify-between bg-slate-950/50">
            <div class="flex items-center space-x-3 truncate pr-4">
              <div class="w-9 h-9 rounded-xl bg-cyan-500/10 text-cyan-400 border border-cyan-500/20 flex items-center justify-center shrink-0">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/>
                </svg>
              </div>
              <div class="truncate">
                <h3 class="text-base font-bold text-slate-100 flex items-center space-x-2">
                  <span>Порівняльний Аналіз Якості (Original vs WebP)</span>
                  <span class="px-2 py-0.5 rounded text-[10px] bg-emerald-500/20 text-emerald-300 font-mono border border-emerald-500/30" v-text="'Quality: ' + (config.webpQuality || 80) + '%'"></span>
                </h3>
                <p class="text-xs text-slate-400 font-mono truncate mt-0.5" v-text="selectedImageComparison?.url"></p>
              </div>
            </div>
            
            <button @click="selectedImageComparison = null" class="text-slate-400 hover:text-slate-200 p-2 rounded-lg bg-slate-800/60 hover:bg-slate-800 shrink-0">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>

          <!-- Modal Body -->
          <div class="p-6 space-y-6">
            
            <template v-if="selectedImageComparison?.isConverting">
              <div class="py-16 text-center space-y-4">
                <svg class="animate-spin h-10 w-10 text-emerald-500 mx-auto" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                <div class="text-sm font-semibold text-slate-200">Конвертація зображення в WebP у реальному часі...</div>
                <div class="text-xs text-slate-400 font-mono">Завантаження оригінального файлу та обробка компресором CGo/WebP (Quality: <span v-text="config.webpQuality || 80"></span>%)</div>
              </div>
            </template>

            <template v-if="selectedImageComparison?.error">
              <div class="bg-rose-950/30 border border-rose-800/50 p-6 rounded-2xl text-center space-y-2">
                <div class="text-rose-400 font-bold text-base flex items-center justify-center space-x-2">
                  <svg class="w-5 h-5 text-rose-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
                  </svg>
                  <span>Помилка обробки зображення</span>
                </div>
                <p class="text-xs text-rose-300 font-mono" v-text="selectedImageComparison?.error"></p>
              </div>
            </template>

            <template v-if="selectedImageComparison?.conversionResult">
              <div class="space-y-6">
                
                <!-- KPI Comparison Cards Header -->
                <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <div class="bg-slate-950/80 p-4 rounded-2xl border border-slate-800 space-y-1">
                    <div class="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">Оригінальний розмір</div>
                    <div class="text-xl font-extrabold text-slate-200 font-mono" v-text="selectedImageComparison.conversionResult.originalFormatted"></div>
                    <div class="text-[10px] text-slate-500 font-mono" v-text="selectedImageComparison.conversionResult.filename"></div>
                  </div>

                  <div class="bg-emerald-950/30 p-4 rounded-2xl border border-emerald-800/50 space-y-1">
                    <div class="text-[11px] font-semibold text-emerald-400 uppercase tracking-wider">Конвертований WebP</div>
                    <div class="text-xl font-extrabold text-emerald-400 font-mono" v-text="selectedImageComparison.conversionResult.optimizedFormatted"></div>
                    <div class="text-[10px] text-emerald-300/80 font-mono">Якість: <span v-text="(config.webpQuality || 80) + '%'"></span></div>
                  </div>

                  <div class="bg-emerald-500/10 p-4 rounded-2xl border border-emerald-500/30 space-y-1">
                    <div class="text-[11px] font-semibold text-emerald-300 uppercase tracking-wider">Реальна Економія</div>
                    <div class="text-xl font-extrabold text-emerald-300 font-mono" v-text="'-' + selectedImageComparison.conversionResult.savingsFormatted + ' (' + selectedImageComparison.conversionResult.savingsPercent.toFixed(1) + '%)'"></div>
                    <div class="text-[10px] text-emerald-400/80 font-mono">Розмір зменшено на <span v-text="selectedImageComparison.conversionResult.savingsPercent.toFixed(1) + '%'"></span></div>
                  </div>
                </div>

                <!-- Stacked Row-by-Row Enriched Visual Comparison Display -->
                <div class="flex flex-col space-y-8">
                  
                  <!-- Row 1: Original Image (Enlarged Full-Width Row) -->
                  <div class="bg-slate-950 p-6 rounded-2xl border border-slate-800 space-y-4 shadow-inner">
                    <div class="flex items-center justify-between border-b border-slate-800 pb-3">
                      <span class="text-sm font-bold text-slate-200 uppercase tracking-wider flex items-center space-x-2">
                        <span class="w-2.5 h-2.5 rounded-full bg-rose-500 inline-block"></span>
                        <span>Оригінальне Зображення (Original Image)</span>
                      </span>
                      <span class="px-3 py-1 rounded-lg text-xs font-mono bg-slate-800 text-slate-200 font-bold border border-slate-700" v-text="selectedImageComparison.conversionResult.originalFormatted"></span>
                    </div>

                    <div class="relative bg-slate-900/80 rounded-2xl overflow-auto min-h-[350px] max-h-[700px] flex items-center justify-center border border-slate-800 p-4 shadow-2xl">
                      <img :src="selectedImageComparison.conversionResult.originalDataBase64 || selectedImageComparison.url"
                        class="max-h-[660px] w-auto max-w-full object-contain rounded-lg shadow-2xl" alt="Original Image">
                    </div>
                  </div>

                  <!-- Row 2: Converted WebP Image (Enlarged Full-Width Row) -->
                  <div class="bg-slate-950 p-6 rounded-2xl border border-emerald-900/50 space-y-4 shadow-inner">
                    <div class="flex items-center justify-between border-b border-emerald-900/40 pb-3">
                      <span class="text-sm font-bold text-emerald-400 uppercase tracking-wider flex items-center space-x-2">
                        <span class="w-2.5 h-2.5 rounded-full bg-emerald-500 inline-block animate-pulse"></span>
                        <span>WebP Оптимізована Версія (Quality: <span v-text="(config.webpQuality || 80) + '%'"></span>)</span>
                      </span>
                      <div class="flex items-center space-x-3">
                        <span class="px-3 py-1 rounded-lg text-xs font-mono bg-emerald-500/20 text-emerald-300 font-bold border border-emerald-500/30" v-text="selectedImageComparison.conversionResult.optimizedFormatted"></span>
                        <span class="px-3 py-1 rounded-lg text-xs font-mono bg-emerald-600 text-white font-extrabold shadow" v-text="'-' + selectedImageComparison.conversionResult.savingsPercent.toFixed(1) + '%'"></span>
                      </div>
                    </div>

                    <div class="relative bg-slate-900/80 rounded-2xl overflow-auto min-h-[350px] max-h-[700px] flex items-center justify-center border border-emerald-900/50 p-4 shadow-2xl">
                      <img :src="selectedImageComparison.conversionResult.optimizedWebPBase64"
                        class="max-h-[660px] w-auto max-w-full object-contain rounded-lg shadow-2xl" alt="Converted WebP Image">
                    </div>
                  </div>

                </div>

              </div>
            </template>

          </div>

          <!-- Modal Footer -->
          <div class="p-6 border-t border-slate-800 bg-slate-950/50 flex items-center justify-between">
            <button @click="selectedImageComparison = null" class="bg-slate-800 hover:bg-slate-700 text-slate-300 font-medium py-2 px-5 rounded-xl text-xs transition">
              Закрити
            </button>

            <template v-if="selectedImageComparison?.conversionResult">
              <button @click="downloadSingleWebP(selectedImageComparison.url)"
                class="bg-emerald-600 hover:bg-emerald-500 text-white font-bold py-2 px-6 rounded-xl text-xs transition flex items-center space-x-2 shadow-lg">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/>
                </svg>
                <span>Завантажити цей WebP (<span v-text="selectedImageComparison.conversionResult.filename"></span>)</span>
              </button>
            </template>
          </div>

        </div>
      </div>
    </div>
</template>

<script>
import { useApp } from '@/store/app';

export default {
  name: 'ImageComparison',
  setup() {
    return useApp();
  }
};
</script>
