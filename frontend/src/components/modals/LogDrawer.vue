<template>
<!-- ACTIVITY LOG DRAWER -->
    <div v-show="showLogDrawer" class="fixed inset-0 z-50 overflow-hidden">
      <div class="absolute inset-0 bg-slate-950/70 backdrop-blur-sm transition-opacity" @click="showLogDrawer = false"></div>

      <div class="fixed inset-y-0 right-0 pl-10 max-w-full flex">
        <div class="w-screen max-w-lg bg-slate-900 border-l border-slate-800 shadow-2xl flex flex-col">
          <div class="px-5 py-4 border-b border-slate-800 flex items-center justify-between">
            <div class="flex items-center space-x-2.5">
              <div class="w-7 h-7 rounded-lg bg-amber-500/10 border border-amber-500/20 flex items-center justify-center text-amber-400">
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01"/>
                </svg>
              </div>
              <h2 class="text-sm font-semibold text-slate-100">Лог активності та запитів</h2>
            </div>
            <button @click="showLogDrawer = false" class="text-slate-400 hover:text-slate-200 p-1.5 rounded-lg hover:bg-slate-800 transition">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>
            </button>
          </div>

          <div class="flex-1 overflow-y-auto p-4 space-y-2 font-mono text-xs">
            <template v-if="activityLogs.length === 0">
              <div class="text-center py-12 text-slate-500">Логи відсутні.</div>
            </template>

            <template v-for="(log, idx) in activityLogs" :key="idx">
              <div class="bg-slate-950/80 p-2.5 rounded-lg border space-y-1"
                :class="log.type === 'error' ? 'border-rose-500/40 text-rose-300' : (log.type === 'warning' ? 'border-amber-500/40 text-amber-300' : (log.type === 'success' ? 'border-emerald-500/40 text-emerald-300' : 'border-slate-800 text-slate-300'))">
                <div class="flex items-center justify-between text-[10px] text-slate-500 font-sans">
                  <span v-text="log.time"></span>
                  <span class="uppercase font-bold" v-text="log.type"></span>
                </div>
                <p class="break-words leading-relaxed" v-text="log.message"></p>
              </div>
            </template>
          </div>
        </div>
      </div>
    </div>
</template>

<script>
import { useApp } from '@/store/app';

export default {
  name: 'LogDrawer',
  setup() {
    return useApp();
  }
};
</script>
