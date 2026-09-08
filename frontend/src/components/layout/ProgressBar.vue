<template>
<!-- Live Scan Progress Header Bar (Shown during scan or complete) -->
        <template v-if="scanProgress || isScanning || scanResults.length > 0">
          <div class="bg-slate-950/60 border-b border-slate-800 px-6 py-4 space-y-3 shrink-0">
            
            <div class="flex items-center justify-between">
              <div class="flex items-center space-x-3">
                <template v-if="isScanning">
                  <span class="flex h-3 w-3 relative">
                    <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                    <span class="relative inline-flex rounded-full h-3 w-3 bg-emerald-500"></span>
                  </span>
                </template>
                <div>
                  <h2 class="text-sm font-semibold text-slate-200">
                    <span v-text="isScanning ? 'Сканування сторінок...' : 'Сканування завершено'"></span>
                  </h2>
                  <p class="text-xs text-slate-400 font-mono truncate max-w-lg" v-text="currentScanningUrl"></p>
                </div>
              </div>

              <div class="flex items-center space-x-4">
                <div class="text-xs text-slate-400">
                  Оброблено: <span class="font-bold text-slate-100" v-text="processedCount"></span> / <span class="font-bold text-slate-100" v-text="totalToScan"></span>
                </div>
                <template v-if="isScanning">
                  <button @click="cancelScan()" class="bg-rose-500/10 hover:bg-rose-500/20 text-rose-400 border border-rose-500/30 font-medium px-3 py-1 rounded text-xs transition">
                    Скасувати
                  </button>
                </template>
              </div>
            </div>

            <!-- Progress Bar with Animated Rolling Pokéball -->
            <div class="w-full relative py-2">
              <div class="w-full bg-slate-800/90 rounded-full h-2.5 overflow-hidden shadow-inner border border-slate-700/40">
                <div class="bg-gradient-to-r from-emerald-500 via-teal-400 to-cyan-400 h-full rounded-full transition-all duration-300"
                  :style="`width: ${progressPercentage}%`"></div>
              </div>

              <!-- Animated Rolling Pokéball at the tip of progress line (clickable Easter Egg) -->
              <div class="absolute top-1/2 -translate-y-1/2 -translate-x-1/2 cursor-pointer transition-all duration-300 z-10 flex items-center justify-center hover:scale-125"
                @click.stop="openPokemonGame()"
                :style="`left: ${Math.max(1.5, Math.min(98.5, progressPercentage))}%;`">
                <div class="w-6 h-6 transition-transform duration-300 flex items-center justify-center filter drop-shadow-[0_2px_5px_rgba(0,0,0,0.8)]"
                  :style="`transform: rotate(${progressPercentage * 14.4}deg);`"
                  title="⚡ SpeedMap Easter Egg: Хто це за покемон? Клікніть!">
                  <svg class="w-full h-full" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <!-- Outer Dark Ring -->
                    <circle cx="16" cy="16" r="15" fill="#0f172a" stroke="#020617" stroke-width="1.5"/>
                    <!-- Red Top Dome -->
                    <path d="M 2.2 16 A 13.8 13.8 0 0 1 29.8 16 Z" fill="#ef4444"/>
                    <!-- Top Dome Highlight reflection -->
                    <path d="M 6 12 A 11 11 0 0 1 18 4.5 C 13 4.5 8 8 6 12 Z" fill="#fca5a5" opacity="0.6"/>
                    <!-- White Bottom Dome -->
                    <path d="M 2.2 16 A 13.8 13.8 0 0 0 29.8 16 Z" fill="#f8fafc"/>
                    <!-- Black Center Dividing Belt -->
                    <rect x="2" y="14.2" width="28" height="3.6" fill="#0f172a"/>
                    <!-- Center Outer Ring -->
                    <circle cx="16" cy="16" r="5.2" fill="#0f172a"/>
                    <!-- Center Button White Ring -->
                    <circle cx="16" cy="16" r="3.4" fill="#ffffff" stroke="#cbd5e1" stroke-width="0.8"/>
                    <!-- Center Core Lens Button -->
                    <circle cx="16" cy="16" r="1.8" fill="#38bdf8" />
                  </svg>
                </div>
              </div>
            </div>
          </div>
        </template>
</template>

<script>
import { useApp } from '@/store/app';

export default {
  name: 'ProgressBar',
  setup() {
    return useApp();
  }
};
</script>
