<template>
<!-- Floating Live Toast Notifications (Bottom Right) -->
    <div v-show="toast.show"
      class="fixed bottom-6 right-6 z-50 max-w-sm w-full bg-slate-900 border p-4 rounded-xl shadow-2xl space-y-1"
      :class="toast.type === 'success' ? 'border-emerald-500/50 text-emerald-300' : (toast.type === 'error' ? 'border-rose-500/50 text-rose-300' : (toast.type === 'warning' ? 'border-amber-500/50 text-amber-300' : 'border-cyan-500/50 text-cyan-300'))">
      <div class="flex items-center justify-between font-bold text-xs">
        <span v-text="toast.title"></span>
        <button @click="toast.show = false" class="text-slate-400 hover:text-slate-200">✕</button>
      </div>
      <p class="text-xs text-slate-300 leading-relaxed font-mono break-words" v-text="toast.message"></p>
    </div>
</template>

<script>
import { useApp } from '@/store/app';

export default {
  name: 'Toast',
  setup() {
    return useApp();
  }
};
</script>
