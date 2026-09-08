<template>
<!-- Create / Edit Site Profile Modal -->
    <div v-show="showProfileModal"
      class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 backdrop-blur-sm p-4">
      
      <div v-click-outside="() => { showProfileModal = false }"
        class="bg-slate-900 border border-slate-800 w-full max-w-lg rounded-2xl shadow-2xl overflow-hidden flex flex-col space-y-4 p-6 relative">
        
        <div class="flex items-center justify-between border-b border-slate-800 pb-3">
          <h3 class="text-base font-bold text-slate-100 flex items-center space-x-2">
            <svg class="w-4 h-4 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"/>
            </svg>
            <span v-text="editingProfile.id ? 'Редагувати сайт' : 'Створити новий сайт'"></span>
          </h3>
          <button @click="showProfileModal = false" class="text-slate-400 hover:text-slate-200 text-sm font-bold">✕</button>
        </div>

        <div class="space-y-4 text-xs">
          <!-- Site Name -->
          <div class="space-y-1">
            <label class="block font-semibold text-slate-300">Назва сайту / Проєкту</label>
            <input type="text" v-model="editingProfile.name" placeholder="наприклад: Infuse Media Main"
              class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-slate-100 placeholder-slate-500 focus:outline-none focus:border-emerald-500 font-sans">
          </div>

          <!-- Sitemap URL -->
          <div class="space-y-1">
            <label class="block font-semibold text-slate-300">URL Sitemap XML</label>
            <input type="text" v-model="editingProfile.sitemapUrl" placeholder="https://example.com/sitemap.xml"
              class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-slate-100 placeholder-slate-500 focus:outline-none focus:border-emerald-500 font-mono">
          </div>

          <!-- Specific Scan Options for this site -->
          <template v-if="editingProfile.config">
            <div class="pt-3 border-t border-slate-800 space-y-3">
              <h4 class="font-bold text-slate-400 uppercase tracking-wider text-[11px] flex items-center space-x-1.5">
                <svg class="w-3.5 h-3.5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
                </svg>
                <span>Індивідуальні налаштування сайту</span>
              </h4>
              
              <div class="grid grid-cols-2 gap-3">
                <div class="space-y-1">
                  <label class="block text-slate-400">Паралельні потоки Chrome</label>
                  <select x-model.number="editingProfile.config.concurrency" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-2.5 py-1.5 text-slate-200">
                    <option value="1">1 потік (безпечно)</option>
                    <option value="2">2 потоки</option>
                    <option value="3">3 потоки (стандарт)</option>
                    <option value="5">5 потоків (швидко)</option>
                    <option value="10">10 потоків (максимум)</option>
                  </select>
                </div>

                <div class="space-y-1">
                  <label class="block text-slate-400">Режим емуляції пристрою</label>
                  <select v-model="editingProfile.config.isMobile" class="w-full bg-slate-950 border border-slate-700 rounded-lg px-2.5 py-1.5 text-slate-200">
                    <option :value="true">Mobile (Mobile Safari)</option>
                    <option :value="false">Desktop (Chrome Desktop)</option>
                  </select>
                </div>
              </div>

              <div class="grid grid-cols-2 gap-3">
                <div class="space-y-1">
                  <label class="block text-slate-400">Поріг важких зображень (KB)</label>
                  <input type="number" x-model.number="editingProfile.config.heavyImageThresholdKB" placeholder="100"
                    class="w-full bg-slate-950 border border-slate-700 rounded-lg px-2.5 py-1.5 text-slate-200 font-mono">
                </div>

                <div class="space-y-1">
                  <label class="block text-slate-400">Таймаут сторінки (сек)</label>
                  <input type="number" x-model.number="editingProfile.config.timeoutSec" placeholder="30"
                    class="w-full bg-slate-950 border border-slate-700 rounded-lg px-2.5 py-1.5 text-slate-200 font-mono">
                </div>
              </div>

              <!-- User-Agent per Profile -->
              <div class="pt-2 border-t border-slate-800 space-y-1">
                <label class="block text-slate-400 font-bold text-[11px] uppercase tracking-wider flex items-center space-x-1.5">
                  <svg class="w-3.5 h-3.5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"/>
                  </svg>
                  <span>User-Agent сайту</span>
                </label>
                <input type="text" v-model="editingProfile.config.userAgent" placeholder="За замовчуванням (Chrome SpeedMap/1.0)"
                  class="w-full bg-slate-950 border border-slate-700 rounded-lg px-2.5 py-1.5 text-slate-100 font-mono text-xs focus:border-emerald-500 focus:outline-none">
              </div>

              <!-- Basic Auth Section per Profile -->
              <div class="pt-2 border-t border-slate-800 space-y-2">
                <label class="block text-slate-400 font-bold text-[11px] uppercase tracking-wider flex items-center space-x-1.5">
                  <svg class="w-3.5 h-3.5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"/>
                  </svg>
                  <span>Базова автентифікація (Basic Auth)</span>
                </label>
                <div class="grid grid-cols-2 gap-3">
                  <div class="space-y-1">
                    <label class="block text-[11px] text-slate-400">Логін (Username)</label>
                    <input type="text" v-model="editingProfile.config.authUser" placeholder="наприклад: admin"
                      class="w-full bg-slate-950 border border-slate-700 rounded-lg px-2.5 py-1.5 text-slate-100 font-mono text-xs focus:border-emerald-500 focus:outline-none">
                  </div>
                  <div class="space-y-1">
                    <label class="block text-[11px] text-slate-400">Пароль (Password)</label>
                    <input type="password" v-model="editingProfile.config.authPass" placeholder="••••••••"
                      class="w-full bg-slate-950 border border-slate-700 rounded-lg px-2.5 py-1.5 text-slate-100 font-mono text-xs focus:border-emerald-500 focus:outline-none">
                  </div>
                </div>
              </div>

              <!-- Custom Headers per Profile -->
              <div class="pt-2 border-t border-slate-800 space-y-2">
                <div class="flex items-center justify-between">
                  <label class="block text-slate-400 font-semibold text-[11px] uppercase tracking-wider">Власні HTTP Заголовки (Headers)</label>
                  <button type="button" @click="addHeaderToEditingProfile()" class="text-emerald-400 hover:text-emerald-300 text-[11px] font-medium underline">+ Додати заголовок</button>
                </div>
                <div class="space-y-1.5 max-h-32 overflow-y-auto">
                  <template v-if="!editingProfile.config.headers || editingProfile.config.headers.length === 0">
                    <div class="text-[11px] text-slate-500 italic">Немає власних заголовків</div>
                  </template>
                  <template v-for="(h, idx) in editingProfile.config.headers" :key="idx">
                    <div class="flex space-x-2">
                      <input type="text" v-model="h.key" placeholder="Header Key (e.g. X-Site-Token)"
                        class="flex-1 bg-slate-950 border border-slate-700 rounded px-2 py-1 text-xs text-slate-100 font-mono focus:border-emerald-500 focus:outline-none">
                      <input type="text" v-model="h.value" placeholder="Header Value"
                        class="flex-1 bg-slate-950 border border-slate-700 rounded px-2 py-1 text-xs text-slate-100 font-mono focus:border-emerald-500 focus:outline-none">
                      <button type="button" @click="removeHeaderFromEditingProfile(idx)" class="text-rose-400 hover:text-rose-300 px-1">✕</button>
                    </div>
                  </template>
                </div>
              </div>
            </div>
          </template>

        </div>

        <div class="flex items-center justify-end space-x-2.5 pt-3 border-t border-slate-800">
          <button @click="showProfileModal = false" class="px-3.5 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-lg text-xs font-medium transition">
            Скасувати
          </button>
          <button @click="saveProfileFromModal()" class="px-3.5 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-xs font-semibold shadow-xs transition">
            Зберегти сайт
          </button>
        </div>

      </div>
    </div>
</template>

<script>
import { useApp } from '@/store/app';

export default {
  name: 'ProfileModal',
  setup() {
    return useApp();
  }
};
</script>
