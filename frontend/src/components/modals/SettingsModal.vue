<template>
<!-- SETTINGS MODAL / SLIDE-OVER -->
    <div v-show="showSettings" class="fixed inset-0 z-50 overflow-hidden" aria-labelledby="slide-over-title" role="dialog" aria-modal="true">
      <div class="absolute inset-0 bg-slate-950/70 backdrop-blur-sm transition-opacity" @click="showSettings = false"></div>

      <div class="fixed inset-y-0 right-0 pl-10 max-w-full flex">
        <div class="w-screen max-w-lg bg-slate-900 border-l border-slate-800 shadow-2xl flex flex-col">
          
          <!-- Modal Header & Tab Bar -->
          <div class="px-5 py-4 border-b border-slate-800 flex items-center justify-between">
            <div class="flex items-center space-x-2.5">
              <div class="w-7 h-7 rounded-lg bg-slate-800 border border-slate-700 flex items-center justify-center text-slate-300">
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
                </svg>
              </div>
              <h2 class="text-sm font-semibold text-slate-100">Налаштування сканера</h2>
            </div>
            <button @click="showSettings = false" class="text-slate-400 hover:text-slate-200 p-1.5 rounded-lg hover:bg-slate-800 transition">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>

          <!-- Category Navigation Tabs -->
          <div class="px-5 pt-2 bg-slate-950/40 border-b border-slate-800/80 flex space-x-1 overflow-x-auto text-xs">
            <button type="button" @click="settingsTab = 'general'"
              :class="settingsTab === 'general' ? 'border-b-2 border-emerald-400 text-slate-100 font-semibold bg-slate-800/50' : 'text-slate-400 hover:text-slate-200'"
              class="px-3 py-2 rounded-t-lg transition whitespace-nowrap flex items-center space-x-1.5">
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"/></svg>
              <span>Загальні</span>
            </button>
            <button type="button" @click="settingsTab = 'cloud'"
              :class="settingsTab === 'cloud' ? 'border-b-2 border-cyan-400 text-slate-100 font-semibold bg-slate-800/50' : 'text-slate-400 hover:text-slate-200'"
              class="px-3 py-2 rounded-t-lg transition whitespace-nowrap flex items-center space-x-1.5">
              <svg class="w-3.5 h-3.5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 00-9.78 2.096A4.001 4.001 0 003 15z"/></svg>
              <span>Google Drive</span>
            </button>
            <button type="button" @click="settingsTab = 'images'"
              :class="settingsTab === 'images' ? 'border-b-2 border-emerald-400 text-slate-100 font-semibold bg-slate-800/50' : 'text-slate-400 hover:text-slate-200'"
              class="px-3 py-2 rounded-t-lg transition whitespace-nowrap flex items-center space-x-1.5">
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>
              <span>Зображення</span>
            </button>
            <button type="button" @click="settingsTab = 'scanner'"
              :class="settingsTab === 'scanner' ? 'border-b-2 border-emerald-400 text-slate-100 font-semibold bg-slate-800/50' : 'text-slate-400 hover:text-slate-200'"
              class="px-3 py-2 rounded-t-lg transition whitespace-nowrap flex items-center space-x-1.5">
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"/></svg>
              <span>Сканер</span>
            </button>
            <button type="button" @click="settingsTab = 'auth'"
              :class="settingsTab === 'auth' ? 'border-b-2 border-emerald-400 text-slate-100 font-semibold bg-slate-800/50' : 'text-slate-400 hover:text-slate-200'"
              class="px-3 py-2 rounded-t-lg transition whitespace-nowrap flex items-center space-x-1.5">
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"/></svg>
              <span>Auth & Headers</span>
            </button>
          </div>

          <!-- Tab Contents -->
          <div class="flex-1 overflow-y-auto p-6 space-y-6 text-sm">
            
            
            <!-- TAB: GOOGLE DRIVE CLOUD INTEGRATION -->
            <template v-if="settingsTab === 'cloud'">
              <div class="space-y-6">
                
                <div class="p-5 bg-slate-950/90 rounded-2xl border border-slate-800 space-y-4 shadow-xl">
                  <div class="flex items-center justify-between border-b border-slate-800 pb-3">
                    <div class="flex items-center space-x-2.5">
                      <div class="w-8 h-8 rounded-lg bg-cyan-500/10 text-cyan-400 border border-cyan-500/20 flex items-center justify-center shrink-0">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 00-9.78 2.096A4.001 4.001 0 003 15z"/>
                        </svg>
                      </div>
                      <div>
                        <h3 class="text-sm font-bold text-slate-100 uppercase tracking-wider">Google Drive Хмарний Експорт</h3>
                        <p class="text-xs text-slate-400">Автоматичне вивантаження та публічні посилання у 1 клік</p>
                      </div>
                    </div>

                    <template v-if="gdriveStatus.connected">
                      <span class="px-2.5 py-1 rounded-md text-xs bg-emerald-500/15 text-emerald-300 border border-emerald-500/30 font-mono font-semibold flex items-center space-x-1.5">
                        <span class="w-2 h-2 rounded-full bg-emerald-400"></span>
                        <span v-text="gdriveStatus.email"></span>
                      </span>
                    </template>
                    <template v-if="!gdriveStatus.connected">
                      <span class="px-2.5 py-1 rounded-md text-xs bg-slate-800/80 text-slate-400 border border-slate-700/80 font-mono flex items-center space-x-1.5">
                        <span class="w-1.5 h-1.5 rounded-full bg-slate-500"></span>
                        <span>Не підключено</span>
                      </span>
                    </template>
                  </div>

                  <div class="space-y-3 text-xs text-slate-300 leading-relaxed">
                    <p>
                      Для підключення вашого Google Drive вкажіть ваш **Client ID** та **Client Secret** з вашої консолі Google Cloud:
                    </p>

                    <!-- Credentials Inputs -->
                    <template v-if="!gdriveStatus.connected">
                      <div class="space-y-3 p-3 bg-slate-900/90 rounded-xl border border-slate-800">
                        <div class="space-y-1">
                          <label class="block font-semibold text-slate-200 text-xs">Google OAuth Client ID:</label>
                          <input type="text" v-model="gdriveClientID" placeholder="xxxxxx.apps.googleusercontent.com"
                            class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-xs font-mono text-slate-100 placeholder-slate-600 focus:outline-none focus:border-cyan-500">
                        </div>

                        <div class="space-y-1">
                          <label class="block font-semibold text-slate-200 text-xs">Google OAuth Client Secret:</label>
                          <input type="password" v-model="gdriveClientSecret" placeholder="GOCSPX-xxxxxx"
                            class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-xs font-mono text-slate-100 placeholder-slate-600 focus:outline-none focus:border-cyan-500">
                        </div>
                      </div>
                    </template>

                    <div class="p-3 bg-slate-900/80 rounded-xl border border-slate-800/80 space-y-1.5 font-sans text-[11px] text-slate-400">
                      <div class="font-semibold text-slate-200 uppercase tracking-wider">Як отримати ключі (для Desktop app):</div>
                      <ol class="list-decimal list-inside space-y-1">
                        <li>Увімкніть <a @click.prevent="openUrlInBrowser('https://console.cloud.google.com/apis/library/drive.googleapis.com')" href="#" class="text-cyan-400 font-bold hover:underline">Google Drive API ↗</a> (натисніть <strong>Enable / Увімкнути</strong>)</li>
                        <li>Перейдіть у <a @click.prevent="openUrlInBrowser('https://console.cloud.google.com/apis/credentials')" href="#" class="text-cyan-400 hover:underline font-bold">Google Cloud Credentials ↗</a></li>
                        <li>Натисніть <strong>Create Credentials ➔ OAuth client ID</strong> та оберіть тип <strong>Desktop app</strong></li>
                        <li>Скопіюйте згенеровані <strong>Client ID</strong> та <strong>Client Secret</strong> у поля вище й натисніть «Авторизувати»!</li>
                      </ol>

                    </div>


                  </div>

                  <!-- Connect / Disconnect Action Buttons -->
                  <div class="pt-2 flex items-center justify-between border-t border-slate-800">
                    <template v-if="!gdriveStatus.connected">
                      <button @click="connectGDrive()" :disabled="isConnectingGDrive"
                        class="w-full bg-emerald-600 hover:bg-emerald-500 text-white font-semibold py-2.5 rounded-xl shadow-sm transition flex items-center justify-center space-x-2 text-xs disabled:opacity-50">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"/>
                        </svg>
                        <span v-show="!isConnectingGDrive">Авторизувати Google Drive</span>
                        <span v-show="isConnectingGDrive">Відкриваємо браузер...</span>
                      </button>
                    </template>

                    <template v-if="gdriveStatus.connected">
                      <div class="flex items-center justify-between w-full">
                        <span class="text-xs text-emerald-400 font-semibold flex items-center space-x-1.5">
                          <span class="w-2 h-2 rounded-full bg-emerald-400"></span>
                          <span>Статус: Готовий до вивантаження</span>
                        </span>
                        <button @click="disconnectGDrive()" class="px-3.5 py-1.5 bg-slate-800 hover:bg-rose-950 hover:text-rose-300 text-slate-300 font-semibold text-xs rounded-xl border border-slate-700 transition flex items-center space-x-1.5">
                          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"/>
                          </svg>
                          <span>Відключити акаунт</span>
                        </button>
                      </div>
                    </template>
                  </div>
                </div>

              </div>
            </template>


            <!-- TAB 1: GENERAL SETTINGS -->
            <template v-if="settingsTab === 'general'">
              <div class="space-y-6">
                <!-- Default Sitemap URL -->
                <div class="space-y-2">
                  <label class="font-semibold text-slate-200 block">Sitemap / Domain за замовчуванням</label>
                  <input type="text" v-model="config.sitemapUrl" placeholder="https://example.com/sitemap.xml"
                    class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-xs text-slate-100 placeholder-slate-500 focus:border-emerald-500 focus:outline-none font-mono">
                  <p class="text-xs text-slate-400">Вкажіть ваш постійний sitemap URL для автоматичного заповнення</p>
                </div>

                <hr class="border-slate-800">

                <!-- Concurrency Slider -->
                <div class="space-y-2">
                  <div class="flex justify-between items-center">
                    <label class="font-semibold text-slate-200">Паралелелізм (Concurrency)</label>
                    <span class="px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 font-mono font-bold" v-text="config.concurrency + ' Chrome'"></span>
                  </div>
                  <input type="range" min="1" max="10" x-model.number="config.concurrency"
                    class="w-full accent-emerald-500 bg-slate-800 rounded-lg cursor-pointer">
                  <p class="text-xs text-slate-400">Кількість одночасних процесів Chrome (за замовчуванням 1, щоб не навантажувати сайт)</p>
                </div>

                <hr class="border-slate-800">

                <!-- Audio Notifications Toggle -->
                <div class="flex items-center justify-between p-3.5 bg-slate-900 rounded-xl border border-slate-800">
                  <div class="space-y-0.5">
                    <label class="block text-xs font-semibold text-slate-200 uppercase tracking-wider flex items-center space-x-1.5">
                      <svg class="w-3.5 h-3.5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.536 8.464a5 5 0 010 7.072m2.828-9.9a9 9 0 010 12.728M5.586 15H4a1 1 0 01-1-1v-4a1 1 0 011-1h1.586l4.707-4.707C10.923 3.663 12 4.109 12 5v14c0 .891-1.077 1.337-1.707.707L5.586 15z"/>
                      </svg>
                      <span>Звукові сповіщення Pikachu</span>
                    </label>
                    <p class="text-xs text-slate-400">Увімкнути або вимкнути звукові сигнали при завершенні сканування</p>
                  </div>
                  <button type="button" @click="config.soundEnabled = !config.soundEnabled"
                    :class="config.soundEnabled ? 'bg-emerald-600' : 'bg-slate-700'"
                    class="relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none">
                    <span :class="config.soundEnabled ? 'translate-x-5' : 'translate-x-0'"
                      class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"></span>
                  </button>
                </div>

                <!-- System OS Notifications Toggle -->
                <div class="flex items-center justify-between p-3.5 bg-slate-900 rounded-xl border border-slate-800">
                  <div class="space-y-0.5">
                    <label class="block text-xs font-semibold text-slate-200 uppercase tracking-wider flex items-center space-x-1.5">
                      <svg class="w-3.5 h-3.5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"/>
                      </svg>
                      <span>Системні сповіщення (macOS Notification Center)</span>
                    </label>
                    <p class="text-xs text-slate-400">Показувати нативний банер з іконкою SpeedMap та звуком при завершенні сканування або експорту</p>
                  </div>
                  <button type="button" @click="config.systemNotifications = !config.systemNotifications"
                    :class="config.systemNotifications ? 'bg-emerald-600' : 'bg-slate-700'"
                    class="relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none">
                    <span :class="config.systemNotifications ? 'translate-x-5' : 'translate-x-0'"
                      class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"></span>
                  </button>
                </div>


                <hr class="border-slate-800">

                <!-- History & Storage Retention -->
                <div class="bg-slate-950 p-4 rounded-xl border border-slate-800 space-y-3.5">
                  <div class="flex items-start justify-between space-x-3">
                    <div class="space-y-0.5">
                      <label class="font-semibold text-slate-200 text-xs flex items-center space-x-1.5 cursor-pointer">
                        <svg class="w-3.5 h-3.5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"/>
                        </svg>
                        <span>Авто-ротація та очищення історії сканувань</span>
                      </label>
                      <p class="text-[11px] text-slate-400">Автоматично підчищає застарілі та надлишкові JSON-звіти в ~/.speedmap/history/, зберігаючи вільне місце на диску.</p>
                    </div>
                    <input type="checkbox" v-model="config.autoPruneHistory"
                      class="rounded border-slate-700 text-emerald-500 focus:ring-emerald-500/20 bg-slate-900 h-4 w-4 mt-0.5 shrink-0 cursor-pointer">
                  </div>

                  <div v-show="config.autoPruneHistory !== false" class="grid grid-cols-2 gap-3 pt-2 border-t border-slate-800/80">
                    <div class="space-y-1">
                      <label class="text-[11px] font-medium text-slate-300">Максимум звітів на домен</label>
                      <input type="number" min="0" max="500" step="5" x-model.number="config.historyRetentionRuns" placeholder="20"
                        class="w-full bg-slate-900 border border-slate-700 rounded-lg px-2.5 py-1.5 text-xs text-slate-100 focus:border-emerald-500 focus:outline-none font-mono">
                      <p class="text-[10px] text-slate-500">За замовч. 20 (0 = без обмежень)</p>
                    </div>
                    <div class="space-y-1">
                      <label class="text-[11px] font-medium text-slate-300">Зберігати історію (днів)</label>
                      <input type="number" min="0" max="1825" step="10" x-model.number="config.historyRetentionDays" placeholder="30"
                        class="w-full bg-slate-900 border border-slate-700 rounded-lg px-2.5 py-1.5 text-xs text-slate-100 focus:border-emerald-500 focus:outline-none font-mono">
                      <p class="text-[10px] text-slate-500">За замовч. 30 (365 = 1 рік, 0 = безстроково)</p>
                    </div>
                  </div>
                </div>

                <hr class="border-slate-800">

                <!-- Device Emulation Switch -->
                <div class="space-y-3">
                  <label class="font-semibold text-slate-200 block">Емуляція пристрою (Device Mode)</label>
                  <div class="grid grid-cols-2 gap-3">
                    <button type="button" @click="config.isMobile = false"
                      :class="!config.isMobile ? 'bg-cyan-500/15 border-cyan-500/60 text-cyan-300 font-bold' : 'bg-slate-900/60 border-slate-800 text-slate-400 hover:text-slate-200'"
                      class="border rounded-xl p-3 text-center transition flex flex-col items-center space-y-1.5">
                      <svg class="w-5 h-5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"/>
                      </svg>
                      <span class="text-xs">Desktop (1920x1080)</span>
                    </button>

                    <button type="button" @click="config.isMobile = true"
                      :class="config.isMobile ? 'bg-cyan-500/15 border-cyan-500/60 text-cyan-300 font-bold' : 'bg-slate-900/60 border-slate-800 text-slate-400 hover:text-slate-200'"
                      class="border rounded-xl p-3 text-center transition flex flex-col items-center space-y-1.5">
                      <svg class="w-5 h-5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 18h.01M8 21h8a2 2 0 002-2V5a2 2 0 00-2-2H8a2 2 0 00-2 2v14a2 2 0 002 2z"/>
                      </svg>
                      <span class="text-xs">Mobile (375x812)</span>
                    </button>
                  </div>
                </div>
              </div>
            </template>

            <!-- TAB 2: IMAGES & WEBP SETTINGS -->
            <template v-if="settingsTab === 'images'">
              <div class="space-y-6">
                <!-- Heavy Image Threshold KB -->
                <div class="space-y-2">
                  <div class="flex justify-between items-center">
                    <label class="font-semibold text-slate-200">Поріг важкого зображення (Heavy Threshold)</label>
                    <span class="px-2 py-0.5 rounded bg-amber-500/10 text-amber-400 font-mono font-bold" v-text="(config.heavyImageThresholdKB || 100) + ' KB'"></span>
                  </div>
                  <input type="number" min="10" max="5000" step="10" x-model.number="config.heavyImageThresholdKB" placeholder="100"
                    class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-xs text-slate-100 focus:border-emerald-500 focus:outline-none font-mono">
                  <p class="text-xs text-slate-400">Розмір у КБ, вище якого зображення вважається важким (за замовчуванням 100 KB)</p>
                </div>

                <hr class="border-slate-800">

                <!-- WebP Compression Quality -->
                <div class="space-y-2">
                  <div class="flex justify-between items-center">
                    <label class="font-semibold text-slate-200">Базова якість стиснення WebP (Quality)</label>
                    <span class="px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 font-mono font-bold" v-text="(config.webpQuality || 80) + '%'"></span>
                  </div>
                  <input type="range" min="50" max="100" step="5" x-model.number="config.webpQuality"
                    class="w-full accent-emerald-500 bg-slate-800 rounded-lg cursor-pointer">
                  <p class="text-xs text-slate-400">Рівень якості для реального WebP конвертера (за замовчуванням 80% — золотий стандарт Google Lighthouse)</p>
                </div>

                <!-- Min WebP Quality Floor -->
                <div class="space-y-2">
                  <div class="flex justify-between items-center">
                    <label class="font-semibold text-slate-200">Мінімальний поріг якості WebP (Min Quality Floor)</label>
                    <span class="px-2 py-0.5 rounded bg-cyan-500/10 text-cyan-400 font-mono font-bold" v-text="(config.minWebPQuality || 80) + '%'"></span>
                  </div>
                  <input type="range" min="50" max="95" step="5" x-model.number="config.minWebPQuality"
                    class="w-full accent-cyan-500 bg-slate-800 rounded-lg cursor-pointer">
                  <p class="text-xs text-slate-400">Нижня межа якості: оптимізатор ніколи не опуститься нижче цього порогу для запобігання артефактам (за замовчуванням 80%)</p>
                </div>

                <!-- Skip if No WebP Savings Toggle -->
                <div class="flex items-start justify-between bg-slate-950/80 p-3 rounded-xl border border-slate-800 space-x-3">
                  <div class="space-y-0.5">
                    <label class="font-semibold text-slate-200 text-xs flex items-center space-x-1.5 cursor-pointer">
                      <svg class="w-3.5 h-3.5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"/>
                      </svg>
                      <span>Пропускати зображення, якщо WebP не дає економії</span>
                    </label>
                    <p class="text-[11px] text-slate-400">Якщо при допустимому порозі якості розмір WebP виходить більшим або рівним оригіналу (наприклад, складні індексовані PNG), SpeedMap залишає вихідний оригінал недоторканим і не включає його в пакет заміни.</p>
                  </div>
                  <input type="checkbox" v-model="config.skipIfNoWebPSavings"
                    class="rounded border-slate-700 text-cyan-500 focus:ring-cyan-500/20 bg-slate-900 h-4 w-4 mt-0.5 shrink-0 cursor-pointer">
                </div>

                <!-- Adaptive Smart Quality Toggle -->
                <div class="flex items-start justify-between bg-slate-950/80 p-3 rounded-xl border border-slate-800 space-x-3">
                  <div class="space-y-0.5">
                    <label class="font-semibold text-slate-200 text-xs flex items-center space-x-1.5 cursor-pointer">
                      <svg class="w-3.5 h-3.5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/>
                      </svg>
                      <span>Розумне адаптивне стиснення (Adaptive Quality)</span>
                    </label>
                    <p class="text-[11px] text-slate-400">Автоматично підвищує якість для дрібних та градієнтних фото (88–94%), усуває чорні краї на прозорих PNG (Lossless), запобігаючи артефактам перестискання.</p>
                  </div>
                  <input type="checkbox" v-model="config.adaptiveQuality"
                    class="rounded border-slate-700 text-emerald-500 focus:ring-emerald-500/20 bg-slate-900 h-4 w-4 mt-0.5 shrink-0 cursor-pointer">
                </div>

                <!-- Retina 2x Render Downscaling Toggle -->
                <div class="flex items-start justify-between bg-slate-950/80 p-3 rounded-xl border border-slate-800 space-x-3">
                  <div class="space-y-0.5">
                    <label class="font-semibold text-slate-200 text-xs flex items-center space-x-1.5 cursor-pointer">
                      <svg class="w-3.5 h-3.5 text-indigo-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4"/>
                      </svg>
                      <span>Ресайз до Retina 2x рендеру (Properly Size Images для Lighthouse)</span>
                    </label>
                    <p class="text-[11px] text-slate-400">Фізично масштабує надвеликі зображення під максимальний розмір контейнера на сайті з запасом Retina 2x. Вимкнено за замовчуванням, щоб гарантовано зберегти 100% оригінальну роздільну здатність на всіх екранах.</p>
                  </div>
                  <input type="checkbox" v-model="config.resizeToRetina"
                    class="rounded border-slate-700 text-indigo-500 focus:ring-indigo-500/20 bg-slate-900 h-4 w-4 mt-0.5 shrink-0 cursor-pointer">
                </div>

                <hr class="border-slate-800">

                <div class="space-y-4">
                  <h3 class="text-xs font-bold text-slate-300 uppercase tracking-wider flex items-center space-x-1.5">
                    <svg class="w-3.5 h-3.5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/>
                    </svg>
                    <span>Коефіцієнти Аналітичної Оцінки Розміру WebP</span>
                  </h3>
                  <p class="text-xs text-slate-400">Вкажіть, який відсоток від оригінального розміру очікується після стиснення в WebP для кожного формату:</p>

                  <!-- PNG WebP Ratio -->
                  <div class="bg-slate-950 p-3.5 rounded-xl border border-slate-800 space-y-2">
                    <div class="flex justify-between items-center">
                      <label class="text-xs font-bold text-slate-200">PNG WebP Ratio</label>
                      <span class="px-2 py-0.5 rounded text-xs font-mono font-bold bg-indigo-500/20 text-indigo-300" v-text="(config.pngWebPRatio || 30) + '% (' + (100 - (config.pngWebPRatio || 30)) + '% економії)'"></span>
                    </div>
                    <input type="range" min="10" max="95" step="5" x-model.number="config.pngWebPRatio"
                      class="w-full accent-indigo-500 bg-slate-800 rounded-lg cursor-pointer">
                  </div>

                  <!-- JPEG WebP Ratio -->
                  <div class="bg-slate-950 p-3.5 rounded-xl border border-slate-800 space-y-2">
                    <div class="flex justify-between items-center">
                      <label class="text-xs font-bold text-slate-200">JPEG WebP Ratio</label>
                      <span class="px-2 py-0.5 rounded text-xs font-mono font-bold bg-amber-500/20 text-amber-300" v-text="(config.jpgWebPRatio || 60) + '% (' + (100 - (config.jpgWebPRatio || 60)) + '% економії)'"></span>
                    </div>
                    <input type="range" min="10" max="95" step="5" x-model.number="config.jpgWebPRatio"
                      class="w-full accent-amber-500 bg-slate-800 rounded-lg cursor-pointer">
                  </div>

                  <!-- GIF WebP Ratio -->
                  <div class="bg-slate-950 p-3.5 rounded-xl border border-slate-800 space-y-2">
                    <div class="flex justify-between items-center">
                      <label class="text-xs font-bold text-slate-200">GIF WebP Ratio</label>
                      <span class="px-2 py-0.5 rounded text-xs font-mono font-bold bg-emerald-500/20 text-emerald-300" v-text="(config.gifWebPRatio || 50) + '% (' + (100 - (config.gifWebPRatio || 50)) + '% економії)'"></span>
                    </div>
                    <input type="range" min="10" max="95" step="5" x-model.number="config.gifWebPRatio"
                      class="w-full accent-emerald-500 bg-slate-800 rounded-lg cursor-pointer">
                  </div>
                </div>

                <hr class="border-slate-800">

                <!-- Tracking Beacons & URL Pattern Exclusions -->
                <div class="space-y-4">
                  <div class="flex items-center justify-between">
                    <h3 class="text-xs font-bold text-slate-300 uppercase tracking-wider flex items-center space-x-1.5">
                      <svg class="w-3.5 h-3.5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"/>
                      </svg>
                      <span>Фільтрація Трекерів та Виключення Зображень</span>
                    </h3>
                  </div>

                  <!-- Toggle Filter Tracking Beacons -->
                  <div class="flex items-start justify-between bg-slate-950/80 p-3 rounded-xl border border-slate-800 space-x-3">
                    <div class="space-y-0.5">
                      <label class="font-semibold text-slate-200 text-xs flex items-center space-x-1.5 cursor-pointer">
                        <span>Фільтрувати службові трекінгові пікселі (Ad / Analytics Beacons)</span>
                      </label>
                      <p class="text-[11px] text-slate-400">Автоматично відсікає 1x1 службові пікселі Google Ads, Facebook, DoubleClick, LinkedIn, Clarity з динамічними таймстемпами. Запити в браузері виконуються для чесних Web Vitals, але не потрапляють у список картинок сайту.</p>
                    </div>
                    <input type="checkbox" v-model="config.filterTrackingBeacons"
                      class="rounded border-slate-700 text-emerald-500 focus:ring-emerald-500/20 bg-slate-900 h-4 w-4 mt-0.5 shrink-0 cursor-pointer">
                  </div>

                  <!-- Custom Excluded Patterns Textarea -->
                  <div class="space-y-1.5">
                    <label class="block font-semibold text-slate-300 text-xs">Шаблони URL для виключення зі списку зображень (по одному на рядок)</label>
                    <textarea v-model="excludedPatternsText" rows="4" placeholder="googleadservices.com&#10;doubleclick.net&#10;facebook.com/tr&#10;/pagead/"
                      class="w-full bg-slate-950 border border-slate-700 rounded-lg p-2.5 text-xs text-slate-100 placeholder-slate-600 focus:border-emerald-500 focus:outline-none font-mono"></textarea>
                    <p class="text-[11px] text-slate-500">Якщо URL ресурсу містить будь-який із цих фрагментів, він не додаватиметься до списку медіа-ресурсів сайту.</p>
                  </div>
                </div>
              </div>
            </template>

            <!-- TAB 3: SCANNER & SCROLL SETTINGS -->
            <template v-if="settingsTab === 'scanner'">
              <div class="space-y-6">
                <!-- Auto-Scroll Toggle Switch -->
                <div class="space-y-2">
                  <div class="flex justify-between items-center">
                    <div>
                      <label class="font-semibold text-slate-200 block">Автоматична прокрутка (Full-Page Auto-Scroll)</label>
                      <p class="text-xs text-slate-400 max-w-xs mt-0.5">Прокручує кожну сторінку донизу для завантаження лінивих зображень (loading="lazy") та динамічних ресурсів</p>
                    </div>
                    <label class="relative inline-flex items-center cursor-pointer shrink-0 ml-2">
                      <input type="checkbox" v-model="config.autoScroll" class="sr-only peer">
                      <div class="w-11 h-6 bg-slate-800 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-emerald-600"></div>
                    </label>
                  </div>
                </div>

                <hr class="border-slate-800">

                <!-- Timeout Sec -->
                <div class="space-y-2">
                  <div class="flex justify-between items-center">
                    <label class="font-semibold text-slate-200">Таймаут завантаження сторінки (Timeout)</label>
                    <span class="px-2 py-0.5 rounded bg-cyan-500/10 text-cyan-400 font-mono font-bold" v-text="(config.timeoutSec || 30) + ' сек'"></span>
                  </div>
                  <input type="number" min="5" max="180" step="5" x-model.number="config.timeoutSec" placeholder="30"
                    class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-xs text-slate-100 focus:border-emerald-500 focus:outline-none font-mono">
                  <p class="text-xs text-slate-400">Максимальний час очікування завантаження однієї сторінки (за замовчуванням 30 сек)</p>
                </div>
              </div>
            </template>

            <!-- TAB 4: AUTH & HEADERS SETTINGS -->
            <template v-if="settingsTab === 'auth'">
              <div class="space-y-6">
                
                <!-- Global User-Agent -->
                <div class="space-y-2.5">
                  <div class="flex justify-between items-start">
                    <div>
                      <label class="font-semibold text-slate-200 block text-xs flex items-center space-x-1.5">
                        <svg class="w-3.5 h-3.5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"/>
                        </svg>
                        <span>Глобальний User-Agent</span>
                      </label>
                      <p class="text-[11px] text-slate-400 mt-0.5">Наскрізно діє на всі запити: браузерний сканер, парсер Sitemap, завантажувач картинок і W3C валідатор.</p>
                    </div>
                    <template v-if="config.userAgent && config.userAgent.trim()">
                      <button type="button" @click="resetUserAgent()" class="text-[11px] text-cyan-400 hover:text-cyan-300 underline font-medium shrink-0 ml-2">
                        Скинути на дефолт
                      </button>
                    </template>
                  </div>
                  
                  <div class="relative">
                    <input type="text" v-model="config.userAgent"
                      :placeholder="config.isMobile ? defaultMobileUA : defaultDesktopUA"
                      class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-xs text-slate-100 placeholder-slate-600 focus:border-emerald-500 focus:outline-none font-mono">
                  </div>

                  <div class="p-2.5 rounded-lg bg-slate-950/60 border border-slate-800 text-[11px] text-slate-400 flex flex-col space-y-1">
                    <div class="flex items-center justify-between text-slate-500">
                      <span>Активний User-Agent (відправляється на сервер):</span>
                      <span class="text-[10px] uppercase font-bold" :class="config.userAgent && config.userAgent.trim() ? 'text-amber-400' : 'text-emerald-400'" v-text="config.userAgent && config.userAgent.trim() ? 'Користувацький' : 'Стандартний'"></span>
                    </div>
                    <span class="font-mono text-slate-300 break-all select-all text-[10px]" v-text="getActiveUserAgent()"></span>
                  </div>
                </div>

                <hr class="border-slate-800">

                <!-- Basic Auth -->
                <div class="space-y-3">
                  <label class="font-semibold text-slate-200 block text-xs flex items-center space-x-1.5">
                    <svg class="w-3.5 h-3.5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"/>
                    </svg>
                    <span>HTTP Basic Authentication</span>
                  </label>
                  <div class="space-y-2">
                    <input type="text" v-model="config.authUser" placeholder="Username"
                      class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-xs text-slate-100 focus:border-emerald-500 focus:outline-none font-mono">
                    <input type="password" v-model="config.authPass" placeholder="Password"
                      class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 text-xs text-slate-100 focus:border-emerald-500 focus:outline-none font-mono">
                  </div>
                </div>

                <hr class="border-slate-800">

                <!-- Custom Headers -->
                <div class="space-y-3">
                  <div class="flex justify-between items-center">
                    <div>
                      <label class="font-semibold text-slate-200 text-xs">Додаткові HTTP заголовки (Custom Headers)</label>
                      <p class="text-[11px] text-slate-400">Довільні HTTP заголовки для специфічних API-ключів, bypass-токенів тощо.</p>
                    </div>
                    <button type="button" @click="addHeader()" class="text-xs text-emerald-400 hover:underline font-medium shrink-0 ml-2">+ Додати заголовок</button>
                  </div>
                  
                  <div class="space-y-2">
                    <template v-if="!config.headers || config.headers.length === 0">
                      <div class="text-xs text-slate-500 italic py-1">Немає додаткових заголовків (за замовчуванням список порожній)</div>
                    </template>
                    <template v-for="(h, idx) in config.headers" :key="idx">
                      <div class="flex space-x-2">
                        <input type="text" v-model="h.key" placeholder="Header Name (e.g. X-Custom-Key)"
                          class="flex-1 bg-slate-950 border border-slate-700 rounded px-2.5 py-1.5 text-xs text-slate-100 focus:border-emerald-500 focus:outline-none font-mono">
                        <input type="text" v-model="h.value" placeholder="Value"
                          class="flex-1 bg-slate-950 border border-slate-700 rounded px-2.5 py-1.5 text-xs text-slate-100 focus:border-emerald-500 focus:outline-none font-mono">
                        <button type="button" @click="removeHeader(idx)" class="text-rose-400 hover:text-rose-300 p-1">
                          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                          </svg>
                        </button>
                      </div>
                    </template>
                  </div>
                </div>
              </div>
            </template>

          </div>

          <div class="p-6 border-t border-slate-800">
            <button @click="saveConfig(); showSettings = false" class="w-full bg-emerald-600 hover:bg-emerald-500 text-white font-semibold py-2.5 rounded-xl transition">
              Зберегти та закрити
            </button>
          </div>




        </div>
      </div>
    </div>
</template>

<script>
import { useApp } from '@/store/app';

export default {
  name: 'SettingsModal',
  setup() {
    return useApp();
  }
};
</script>
