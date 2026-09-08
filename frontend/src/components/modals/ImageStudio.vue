<template>
<!-- 🎨 FULLSCREEN IMAGE TUNING STUDIO (SEOAEO-235 80/20 UI) -->
    <!-- ======================================================== -->
    <div v-show="imageStudio.isOpen"
      @mousemove.window="onSplitMouseMove($event)"
      @mouseup.window="stopSplitDrag()"
      @touchmove.window="onSplitMouseMove($event)"
      @touchend.window="stopSplitDrag()"
      @keydown.window="handleStudioKeydown($event)"
      @keyup.window="handleStudioKeyup($event)"
      class="fixed inset-0 z-[65] bg-slate-950 flex flex-col text-slate-100 overflow-hidden select-none font-sans"
      role="dialog" aria-modal="true">

      <!-- TOPBAR: Navigation, Modes, Zoom, Close (macOS draggable with native traffic lights inset) -->
      <div class="wails-drag h-14 border-b border-slate-800 bg-slate-900/90 pl-20 pr-4 flex items-center justify-between shrink-0 shadow-md select-none gap-2">
        
        <!-- Left: Image Nav & Title -->
        <div class="flex items-center space-x-2.5 min-w-0 flex-1 mr-2">
          <div class="flex items-center space-x-1 shrink-0">
            <button @click="studioPrevImage()" :disabled="imageStudio.currentIndex === 0"
              class="px-2 py-1 rounded-lg bg-slate-800 hover:bg-slate-700 disabled:opacity-30 disabled:cursor-not-allowed text-xs text-slate-200 font-mono transition"
              title="Попереднє зображення (Стрілка вліво ←)">
              ◄
            </button>
            <span class="text-xs font-mono text-slate-400 px-1 shrink-0">
              <strong class="text-slate-100" v-text="imageStudio.currentIndex + 1"></strong> / <span v-text="imageStudioImages.length"></span>
            </span>
            <button @click="studioNextImage()" :disabled="imageStudio.currentIndex >= imageStudioImages.length - 1"
              class="px-2 py-1 rounded-lg bg-slate-800 hover:bg-slate-700 disabled:opacity-30 disabled:cursor-not-allowed text-xs text-slate-200 font-mono transition"
              title="Наступне зображення (Стрілка вправо →)">
              ►
            </button>
          </div>

          <div class="min-w-0 flex items-center space-x-2">
            <!-- Package Mode Badge -->
            <template v-if="packageContext?.active">
              <div class="flex items-center space-x-1 px-1.5 py-0.5 rounded-lg bg-fuchsia-950/80 border border-fuchsia-700/60 text-fuchsia-300 text-[11px] font-mono shrink-0">
                <svg class="w-3 h-3 text-fuchsia-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/>
                </svg>
                <span class="font-bold">ПАКЕТ</span>
                <button @click="closePackageStudioContext()" class="hover:text-white ml-0.5 text-slate-400" title="Вийти з режиму пакету до скану">✕</button>
              </div>

              <!-- Search in Package -->
              <div class="relative flex items-center shrink-0">
                <input type="text"
                  v-model="packageContext.searchQuery"
                  @input="studioSelectImage(0)"
                  placeholder="🔍 Пошук..."
                  class="bg-slate-950 border border-slate-700 rounded-lg px-2 py-1 text-[11px] text-slate-200 placeholder-slate-500 font-mono w-24 sm:w-32 focus:w-44 transition-all outline-none focus:border-fuchsia-500">
                <button v-if="packageContext.searchQuery" @click="packageContext.searchQuery = ''; studioSelectImage(0)" class="absolute right-1.5 text-slate-400 hover:text-white text-[10px]">✕</button>
              </div>

              <!-- Open compare.html -->
              <button @click="openPackageCompareHTML()"
                class="px-2 py-1 rounded-lg bg-slate-800 hover:bg-slate-700 text-cyan-300 border border-slate-700 text-[11px] font-mono flex items-center space-x-1 transition shrink-0"
                title="Відкрити compare.html у браузері">
                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/></svg>
                <span class="hidden xl:inline">compare.html</span>
              </button>
            </template>

            <span class="text-sm font-bold text-amber-400 font-mono truncate max-w-[120px] sm:max-w-[160px] md:max-w-[220px]"
              :title="currentStudioImage?.basename"
              v-text="currentStudioImage?.basename || 'image'"></span>
            
            <button type="button" @click.stop="openUrlInBrowser(currentStudioImage?.url)"
              class="text-slate-400 hover:text-amber-400 p-1 rounded hover:bg-slate-800 transition shrink-0"
              title="Відкрити це зображення у браузері">
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/></svg>
            </button>
            <button type="button" @click.stop="copyToClipboard(currentStudioImage?.url, 'Посилання скопійовано')"
              class="text-slate-400 hover:text-slate-200 p-1 rounded hover:bg-slate-800 transition shrink-0"
              title="Скопіювати повне посилання">
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
            </button>

            <template v-if="currentStudioImage?.isModified || packageContext?.modifiedIds?.includes(currentStudioImage?.id)">
              <span class="px-2 py-0.5 rounded text-[10px] font-bold uppercase bg-emerald-950/80 text-emerald-300 border border-emerald-700 shadow-sm flex items-center space-x-1 shrink-0">
                <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                <span class="hidden sm:inline">Оновлено</span>
              </span>
            </template>
            <template v-if="currentStudioImage?.format === 'svg'">
              <span class="px-2 py-0.5 rounded text-[10px] font-bold uppercase bg-purple-950/60 text-purple-300 border border-purple-800 shrink-0">SVG</span>
            </template>
            <template v-if="currentStudioSkip && currentStudioImage?.format !== 'svg'">
              <span class="px-2 py-0.5 rounded text-[10px] font-bold uppercase bg-rose-950/60 text-rose-300 border border-rose-800 shrink-0">Пропущено</span>
            </template>
            <template v-if="!currentStudioSkip && isCurrentStudioOverridden">
              <span class="px-2 py-0.5 rounded text-[10px] font-bold uppercase bg-purple-950/60 text-purple-300 border border-purple-800 shrink-0 hidden md:inline">Кастом</span>
            </template>
          </div>
        </div>

        <!-- Center: Comparison View Modes & Zoom -->
        <div class="flex items-center space-x-2 shrink-0">
          
          <!-- View Modes -->
          <div class="bg-slate-950 p-1 rounded-xl border border-slate-800 flex items-center text-xs">
            <button @click="imageStudio.viewMode = 'split'"
              :class="imageStudio.viewMode === 'split' ? 'bg-cyan-600 text-white font-bold shadow' : 'text-slate-400 hover:text-slate-200'"
              class="px-2.5 py-1 rounded-lg transition flex items-center space-x-1.5"
              title="Інтерактивна шторка-роздільник (Split)">
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4"/>
              </svg>
              <span>Шторка</span>
            </button>

            <button @click="imageStudio.viewMode = 'toggle'"
              :class="imageStudio.viewMode === 'toggle' ? 'bg-cyan-600 text-white font-bold shadow' : 'text-slate-400 hover:text-slate-200'"
              class="px-2.5 py-1 rounded-lg transition flex items-center space-x-1.5"
              title="Швидкий перемикач: клавіша T або Space">
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
              </svg>
              <span>Toggle</span>
            </button>

            <button @click="imageStudio.viewMode = 'side'"
              :class="imageStudio.viewMode === 'side' ? 'bg-cyan-600 text-white font-bold shadow' : 'text-slate-400 hover:text-slate-200'"
              class="px-2.5 py-1 rounded-lg transition flex items-center space-x-1.5"
              title="Два зображення поруч">
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 4H5a2 2 0 00-2 2v12a2 2 0 002 2h4m6-16h4a2 2 0 012 2v12a2 2 0 01-2 2h-4"/>
              </svg>
              <span>Поруч</span>
            </button>
          </div>

          <!-- Zoom Modes -->
          <div class="hidden sm:flex bg-slate-950 p-1 rounded-xl border border-slate-800 items-center text-xs">
            <button @click="imageStudio.zoomMode = 'fit'"
              :class="imageStudio.zoomMode === 'fit' ? 'bg-cyan-600 text-white font-bold shadow' : 'text-slate-400 hover:text-slate-200'"
              class="px-2.5 py-1 rounded-lg transition" title="Вписати в екран">
              Fit
            </button>
            <button @click="imageStudio.zoomMode = '100'"
              :class="imageStudio.zoomMode === '100' ? 'bg-cyan-600 text-white font-bold shadow' : 'text-slate-400 hover:text-slate-200'"
              class="px-2.5 py-1 rounded-lg transition" title="100% (1:1 масштаб)">
              100%
            </button>
            <button @click="imageStudio.zoomMode = '200'"
              :class="imageStudio.zoomMode === '200' ? 'bg-cyan-600 text-white font-bold shadow' : 'text-slate-400 hover:text-slate-200'"
              class="px-2.5 py-1 rounded-lg transition" title="200% (2x збільшення)">
              200%
            </button>
            <button @click="imageStudio.zoomMode = '400'"
              :class="imageStudio.zoomMode === '400' ? 'bg-cyan-600 text-white font-bold shadow' : 'text-slate-400 hover:text-slate-200'"
              class="px-2.5 py-1 rounded-lg transition" title="400% (4x збільшення)">
              400%
            </button>
          </div>

          <!-- Canvas Background Switcher (Dark, Light/White, Checkerboard, Custom Color Picker) -->
          <div class="flex items-center bg-slate-950 p-1 rounded-xl border border-slate-800 text-xs space-x-1" title="Колір фону робочої області">
            <!-- Dark Theme -->
            <button @click="imageStudio.canvasBgMode = 'dark'"
              :class="imageStudio.canvasBgMode === 'dark' ? 'ring-2 ring-cyan-400 bg-slate-800 shadow' : 'opacity-60 hover:opacity-100'"
              class="w-6 h-6 rounded-lg bg-slate-950 border border-slate-700 flex items-center justify-center transition"
              title="Темний фон">
              <span class="w-2.5 h-2.5 rounded-full bg-slate-900 border border-slate-600"></span>
            </button>

            <!-- Pure White (for dark images and black text) -->
            <button @click="imageStudio.canvasBgMode = 'light'"
              :class="imageStudio.canvasBgMode === 'light' ? 'ring-2 ring-cyan-400 bg-slate-800 shadow' : 'opacity-60 hover:opacity-100'"
              class="w-6 h-6 rounded-lg bg-white border border-slate-300 flex items-center justify-center transition"
              title="Світлий / білий фон (для темних або прозорих фото)">
              <span class="w-2.5 h-2.5 rounded-full bg-white shadow-sm"></span>
            </button>

            <!-- Transparency Checkerboard -->
            <button @click="imageStudio.canvasBgMode = 'checker'"
              :class="imageStudio.canvasBgMode === 'checker' ? 'ring-2 ring-cyan-400 bg-slate-800 shadow' : 'opacity-60 hover:opacity-100'"
              class="w-6 h-6 rounded-lg border border-slate-700 flex items-center justify-center transition overflow-hidden"
              style="background-image: repeating-conic-gradient(#94a3b8 0% 25%, #ffffff 0% 50%); background-size: 8px 8px;"
              title="Прозорість (шахова сітка)">
            </button>

            <!-- Native Color Picker -->
            <label class="relative w-6 h-6 rounded-lg border border-slate-700 flex items-center justify-center cursor-pointer transition overflow-hidden group"
              :class="imageStudio.canvasBgMode === 'custom' ? 'ring-2 ring-cyan-400 shadow' : 'opacity-70 hover:opacity-100'"
              :style="'background-color: ' + (imageStudio.canvasCustomColor || '#ffffff')"
              title="Вибрати довільний колір фону (Color Picker)">
              <input type="color"
                v-model="imageStudio.canvasCustomColor"
                @input="imageStudio.canvasBgMode = 'custom'"
                class="opacity-0 absolute inset-0 w-full h-full cursor-pointer">
              <svg class="w-3.5 h-3.5 text-slate-300 pointer-events-none drop-shadow" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 21a4 4 0 01-4-4 8.014 8.014 0 014.004-6.936M7 21h10a4 4 0 004-4 8.014 8.014 0 00-4.004-6.936M7 21a4 4 0 01-4-4c0-2.485 2.015-4.5 4.5-4.5h.5M17 21a4 4 0 004-4c0-2.485-2.015-4.5-4.5-4.5h-.5"/>
              </svg>
            </label>
          </div>

        </div>

        <!-- Right: Actions & Close -->
        <div class="flex items-center space-x-2 shrink-0">
          <template v-if="packageContext?.active">
            <button @click="saveCurrentToPackage()" :disabled="!imageStudio.currentResult"
              class="bg-fuchsia-600 hover:bg-fuchsia-500 disabled:opacity-40 text-white font-bold text-xs px-3.5 py-1.5 rounded-xl transition flex items-center space-x-1.5 shadow-lg ring-1 ring-fuchsia-400/50"
              title="Зберегти та оновити файл прямо в пакеті (Cmd+S / Ctrl+S)">
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4"/>
              </svg>
              <span>Зберегти в пакет</span>
              <span class="text-[10px] opacity-80 font-mono hidden sm:inline">[⌘S]</span>
            </button>
          </template>
          <template v-else>
            <button @click="downloadCurrentStudioWebP()" :disabled="!imageStudio.currentResult"
              class="bg-emerald-600 hover:bg-emerald-500 disabled:opacity-40 text-white font-bold text-xs px-3 py-1.5 rounded-xl transition flex items-center space-x-1.5 shadow"
              :title="(currentStudioImage?.format === 'svg' && imageStudio.currentResult?.isSkipped) ? 'Зберегти векторний SVG' : 'Завантажити WebP'">
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/>
              </svg>
              <span v-text="(currentStudioImage?.format === 'svg' && imageStudio.currentResult?.isSkipped) ? 'SVG' : 'WebP'"></span>
            </button>
          </template>

          <button @click="closeImageStudio()"
            class="text-slate-400 hover:text-slate-100 p-2 rounded-xl bg-slate-800 hover:bg-slate-700 transition"
            title="Закрити Студію (Esc)">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

      </div>

      <!-- MAIN WORKSPACE: Preview Canvas & Floating Controls -->
      <div class="flex-1 flex overflow-hidden relative">
        
        <!-- CANVAS: Visual Comparator -->
        <div class="flex-1 min-w-0 flex flex-col bg-slate-950 overflow-hidden relative">
          
          <!-- Top Hint Banner for Toggle Mode -->
          <div v-show="imageStudio.viewMode === 'toggle'"
            class="bg-slate-900/90 border-b border-slate-800/80 px-4 py-2 flex items-center justify-between text-xs font-mono shrink-0">
            <div class="flex items-center space-x-3">
              <!-- Switcher: Click either button to toggle -->
              <div class="flex items-center bg-slate-950 p-0.5 rounded-xl border border-slate-800">
                <button @click="imageStudio.toggleShowOriginal = false"
                  :class="!imageStudio.toggleShowOriginal ? 'bg-emerald-600 text-white font-bold shadow' : 'text-slate-400 hover:text-slate-200'"
                  class="px-3 py-1 rounded-lg transition text-xs flex items-center space-x-1.5">
                  <span class="w-2 h-2 rounded-full bg-emerald-300"></span>
                  <span>WebP</span>
                </button>
                <button @click="imageStudio.toggleShowOriginal = true"
                  :class="imageStudio.toggleShowOriginal ? 'bg-rose-600 text-white font-bold shadow' : 'text-slate-400 hover:text-slate-200'"
                  class="px-3 py-1 rounded-lg transition text-xs flex items-center space-x-1.5">
                  <span class="w-2 h-2 rounded-full bg-rose-300"></span>
                  <span>Оригінал</span>
                </button>
              </div>

              <!-- Hold button for quick compare -->
              <button @mousedown="imageStudio.toggleShowOriginal = true" @mouseup="imageStudio.toggleShowOriginal = false"
                @touchstart="imageStudio.toggleShowOriginal = true" @touchend="imageStudio.toggleShowOriginal = false"
                class="px-3 py-1 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 font-bold border border-slate-700 transition active:bg-rose-600 active:text-white select-none text-xs">
                <span>Затиснути для Оригіналу</span>
              </button>

              <span class="text-[11px] text-slate-400 font-sans hidden md:inline">(Клік по зображенню або клавіша <kbd class="px-1.5 py-0.5 bg-slate-800 rounded border border-slate-700 text-amber-300">T</kbd> / <kbd class="px-1.5 py-0.5 bg-slate-800 rounded border border-slate-700 text-amber-300">Пробіл</kbd>)</span>
            </div>

            <div>
              <span v-show="!imageStudio.toggleShowOriginal" class="text-emerald-400 font-bold flex items-center space-x-1.5">
                <span class="w-2 h-2 rounded-full bg-emerald-400 inline-block animate-pulse"></span>
                <span>Показується Оптимізоване WebP</span>
              </span>
              <span v-show="imageStudio.toggleShowOriginal" class="text-rose-400 font-bold flex items-center space-x-1.5">
                <span class="w-2 h-2 rounded-full bg-rose-400 inline-block"></span>
                <span>Показується Оригінал</span>
              </span>
            </div>
          </div>

          <!-- Canvas Viewport -->
          <div class="flex-1 overflow-auto flex items-center justify-center p-4 transition-colors duration-150 relative"
            :style="studioCanvasBgStyle">
            
            <!-- Loading Spinner Overlay -->
            <template v-if="imageStudio.isConverting && !imageStudio.currentResult">
              <div class="flex flex-col items-center justify-center space-y-3">
                <svg class="animate-spin h-10 w-10 text-cyan-400" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                <span class="text-xs font-mono text-slate-400">Генерація WebP у реальному часі...</span>
              </div>
            </template>

            <!-- Conversion Error Display -->
            <template v-if="imageStudio.error">
              <div class="bg-rose-950/60 border border-rose-800 p-6 rounded-2xl max-w-md text-center space-y-2">
                <div class="text-rose-400 font-bold text-sm flex items-center justify-center space-x-2">
                  <svg class="w-4 h-4 text-rose-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
                  </svg>
                  <span>Помилка обробки зображення</span>
                </div>
                <div class="text-xs font-mono text-rose-300" v-text="imageStudio.error"></div>
              </div>
            </template>

            <!-- Active Visualizer (Each mode is in its own isolated wrapper div to prevent style collisions) -->
            <template v-if="imageStudio.currentResult && !imageStudio.error">
              <div class="min-w-full min-h-full relative flex items-center justify-center p-4">

                <!-- 1. SPLIT SLIDER VIEW -->
                <div v-show="imageStudio.viewMode === 'split'"
                  class="w-full h-full flex items-center justify-center">
                  <div id="studio-split-container"
                    @mousedown="startSplitDrag($event)"
                    @touchstart="startSplitDrag($event)"
                    class="relative overflow-hidden rounded-2xl border border-slate-700/60 shadow-2xl select-none cursor-ew-resize flex items-center justify-center m-auto transition-all duration-150"
                    :style="studioContainerStyle">
                    
                    <!-- Base Layer: Optimized WebP -->
                    <img :src="imageStudio.currentResult.optimizedWebPBase64"
                      class="w-full h-full object-contain block pointer-events-none"
                      alt="Optimized WebP">

                    <!-- Overlay Layer: Original Image clipped by splitPos -->
                    <div class="absolute inset-0 overflow-hidden pointer-events-none"
                      :style="'clip-path: inset(0 ' + (100 - imageStudio.splitPos) + '% 0 0);'">
                      <img :src="imageStudio.currentResult.originalDataBase64 || currentStudioImage?.url"
                        @error="onOriginalImgError($event)"
                        class="w-full h-full object-contain block"
                        alt="Original">
                    </div>

                    <!-- Split Handle / Divider Bar -->
                    <div class="absolute top-0 bottom-0 w-0.5 bg-cyan-400 shadow-[0_0_12px_rgba(34,211,238,0.9)] pointer-events-none"
                      :style="'left: ' + imageStudio.splitPos + '%;'">
                      <div class="absolute top-1/2 -translate-y-1/2 -translate-x-1/2 w-8 h-8 rounded-full bg-cyan-500 border-2 border-slate-900 shadow-2xl flex items-center justify-center text-xs text-slate-950 font-black">
                        ↔
                      </div>
                    </div>

                    <!-- Left / Right Floating Tags -->
                    <div class="absolute top-3 left-3 px-2.5 py-1 bg-slate-950/80 backdrop-blur rounded-lg border border-rose-500/50 text-[11px] font-mono font-bold text-rose-300 pointer-events-none shadow flex items-center space-x-1.5">
                      <span class="w-2 h-2 rounded-full bg-rose-400"></span>
                      <span>ОРИГІНАЛ</span>
                    </div>
                    <div class="absolute top-3 right-3 px-2.5 py-1 bg-slate-950/80 backdrop-blur rounded-lg border border-emerald-500/50 text-[11px] font-mono font-bold text-emerald-300 pointer-events-none shadow flex items-center space-x-1.5">
                      <span class="w-2 h-2 rounded-full bg-emerald-400"></span>
                      <span>WEBP</span>
                    </div>
                  </div>
                </div>

                <!-- 2. TOGGLE VIEW (Click image to toggle, or use top switch/keys) -->
                <div v-show="imageStudio.viewMode === 'toggle'"
                  class="w-full h-full flex items-center justify-center">
                  <div @click="imageStudio.toggleShowOriginal = !imageStudio.toggleShowOriginal"
                    class="relative overflow-hidden rounded-2xl border border-slate-800 shadow-2xl flex items-center justify-center cursor-pointer select-none group m-auto transition-all duration-150"
                    :style="studioContainerStyle"
                    title="Клікніть для перемикання між WebP та Оригіналом (або затисніть T / Пробіл)">
                    
                    <!-- WebP Image -->
                    <img v-show="!imageStudio.toggleShowOriginal"
                      :src="imageStudio.currentResult.optimizedWebPBase64"
                      class="w-full h-full object-contain block select-none pointer-events-none" alt="WebP">

                    <!-- Original Image -->
                    <img v-show="imageStudio.toggleShowOriginal"
                      :src="imageStudio.currentResult.originalDataBase64 || currentStudioImage?.url"
                      @error="onOriginalImgError($event)"
                      class="w-full h-full object-contain block select-none pointer-events-none" alt="Original">

                    <!-- Floating Pill Indicator -->
                    <div class="absolute bottom-3 right-3 px-3 py-1.5 rounded-xl backdrop-blur bg-slate-950/85 border text-xs font-mono font-bold select-none pointer-events-none shadow-lg flex items-center space-x-2"
                      :class="imageStudio.toggleShowOriginal ? 'border-rose-500/60 text-rose-300' : 'border-emerald-500/60 text-emerald-300'">
                      <span class="w-2 h-2 rounded-full" :class="imageStudio.toggleShowOriginal ? 'bg-rose-400' : 'bg-emerald-400 animate-pulse'"></span>
                      <span v-text="imageStudio.toggleShowOriginal ? 'ОРИГІНАЛ' : 'WEBP'"></span>
                    </div>

                    <!-- Top Left Hint -->
                    <div class="absolute top-3 left-3 px-2 py-1 rounded bg-slate-950/70 backdrop-blur text-[10px] text-slate-400 font-sans border border-slate-800 pointer-events-none opacity-80 group-hover:opacity-100 transition">
                      Клікніть або клавіша <kbd class="font-mono text-amber-300 font-bold">T</kbd>
                    </div>
                  </div>
                </div>

                <!-- 3. SIDE BY SIDE VIEW -->
                <div v-show="imageStudio.viewMode === 'side'"
                  class="w-full h-full flex items-center justify-center p-2">
                  <div class="grid grid-cols-1 md:grid-cols-2 gap-4 w-full h-full max-h-[calc(100vh-200px)]">
                    <!-- Left: Original -->
                    <div class="flex flex-col bg-slate-900/70 rounded-2xl border border-slate-800 p-3 overflow-hidden">
                      <div class="flex items-center justify-between text-xs font-mono text-rose-400 font-bold mb-2 px-1 shrink-0">
                        <span class="flex items-center space-x-1.5">
                          <span class="w-2 h-2 rounded-full bg-rose-400"></span>
                          <span>ОРИГІНАЛ</span>
                        </span>
                        <span class="text-slate-400 text-[11px]" v-text="imageStudio.currentResult.originalFormatted"></span>
                      </div>
                      <div class="flex-1 flex items-center justify-center overflow-auto rounded-xl p-2"
                        :style="studioCanvasBgStyle">
                        <template v-if="currentStudioImage?.format === 'svg' && (!imageStudio.currentResult.originalWidth || imageStudio.currentResult.originalWidth === 0)">
                          <div class="flex flex-col items-center justify-center p-6 text-center space-y-2">
                            <div class="w-12 h-12 rounded-xl bg-purple-950/80 border border-purple-700/60 flex items-center justify-center text-purple-400 mb-1">
                              <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/></svg>
                            </div>
                            <div class="text-xs font-bold text-slate-100">SVG Спрайт (Оригінал)</div>
                            <div class="text-[11px] text-slate-400 max-w-xs font-sans">Містить набір символів <code>&lt;symbol&gt;</code> без прямої кореневої геометрії.</div>
                          </div>
                        </template>
                        <template v-if="currentStudioImage?.format !== 'svg' || (imageStudio.currentResult.originalWidth && imageStudio.currentResult.originalWidth > 0)">
                          <img :src="imageStudio.currentResult.originalDataBase64 || currentStudioImage?.url"
                            @error="onOriginalImgError($event)"
                            :style="studioSideImageStyle"
                            class="rounded-lg select-none block m-auto" alt="Original">
                        </template>
                      </div>
                    </div>
                    <!-- Right: WebP / SVG -->
                    <div class="flex flex-col bg-slate-900/70 rounded-2xl border border-emerald-900/50 p-3 overflow-hidden">
                      <div class="flex items-center justify-between text-xs font-mono text-emerald-400 font-bold mb-2 px-1 shrink-0">
                        <span class="flex items-center space-x-1.5">
                          <span class="w-2 h-2 rounded-full bg-emerald-400"></span>
                          <span v-text="currentStudioImage?.format === 'svg' ? 'ОПТИМІЗОВАНИЙ SVG' : 'ОПТИМІЗОВАНЕ WEBP'"></span>
                        </span>
                        <span class="text-emerald-400 text-[11px]" v-text="imageStudio.currentResult.optimizedFormatted"></span>
                      </div>
                      <div class="flex-1 flex items-center justify-center overflow-auto rounded-xl p-2"
                        :style="studioCanvasBgStyle">
                        <template v-if="currentStudioImage?.format === 'svg' && (!imageStudio.currentResult.optimizedWidth || imageStudio.currentResult.optimizedWidth === 0)">
                          <div class="flex flex-col items-center justify-center p-6 text-center space-y-2">
                            <div class="w-12 h-12 rounded-xl bg-emerald-950/80 border border-emerald-700/60 flex items-center justify-center text-emerald-400 mb-1">
                              <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
                            </div>
                            <div class="text-xs font-bold text-slate-100">SVG Спрайт (Оптимізовано)</div>
                            <div class="text-[11px] text-slate-400 max-w-xs font-sans">Код очищено за допомогою SVGO зі збереженням структури спрайту.</div>
                          </div>
                        </template>
                        <template v-if="currentStudioImage?.format !== 'svg' || (imageStudio.currentResult.optimizedWidth && imageStudio.currentResult.optimizedWidth > 0)">
                          <img :src="imageStudio.currentResult.optimizedWebPBase64"
                            :style="studioSideImageStyle"
                            class="rounded-lg select-none block m-auto" alt="Optimized">
                        </template>
                      </div>
                    </div>
                  </div>
                </div>

              </div>
            </template>

          </div>

          <!-- Bottom Metrics Bar & Quick Controls -->
          <template v-if="imageStudio.currentResult">
            <div class="h-14 border-t border-slate-800 bg-slate-900/90 backdrop-blur px-4 flex items-center justify-between text-xs font-mono shrink-0 gap-3">
              
              <!-- Left: Image Dims & Format Compare -->
              <div class="flex items-center space-x-3 truncate">
                <div class="flex items-center space-x-1.5">
                  <span class="text-slate-400 font-sans">Оригінал:</span>
                  <span class="text-slate-200 font-bold" v-text="imageStudio.currentResult.originalFormatted"></span>
                  <span class="text-slate-500 text-[11px]" v-text="'(' + imageStudio.currentResult.originalWidth + '×' + imageStudio.currentResult.originalHeight + ')'"></span>
                </div>
                <span class="text-slate-600">➔</span>
                <div class="flex items-center space-x-1.5">
                  <span class="text-emerald-400 font-sans">WebP:</span>
                  <span class="text-emerald-400 font-bold" v-text="imageStudio.currentResult.optimizedFormatted"></span>
                  <span class="text-emerald-600 text-[11px]" v-text="'(' + imageStudio.currentResult.optimizedWidth + '×' + imageStudio.currentResult.optimizedHeight + ')'"></span>
                </div>

                <!-- Size Reduced (Savings) -->
                <template v-if="imageStudio.currentResult.savingsBytes >= 0">
                  <div class="px-2.5 py-1 rounded-lg bg-emerald-500/20 text-emerald-300 font-bold border border-emerald-500/30 flex items-center space-x-1.5 shadow">
                    <svg class="w-3.5 h-3.5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 17h8m0 0V9m0 8l-8-8-4 4-6-6"/>
                    </svg>
                    <span v-text="'-' + imageStudio.currentResult.savingsFormatted + ' (-' + imageStudio.currentResult.savingsPercent.toFixed(1) + '%)'"></span>
                  </div>
                </template>

                <!-- Size Increased (Warning) -->
                <template v-if="imageStudio.currentResult.savingsBytes < 0">
                  <div class="px-2.5 py-1 rounded-lg bg-rose-500/20 text-rose-300 font-bold border border-rose-500/40 flex items-center space-x-1.5 shadow"
                    title="WebP більший за оригінал. Рекомендовано зменшити якість або вимкнути Lossless.">
                    <svg class="w-3.5 h-3.5 text-rose-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
                    </svg>
                    <span v-text="'+' + imageStudio.currentResult.savingsFormatted.replace('-', '') + ' (+' + Math.abs(imageStudio.currentResult.savingsPercent).toFixed(1) + '%)'"></span>
                  </div>
                </template>

                <template v-if="imageStudio.isConverting">
                  <span class="text-[11px] text-cyan-400 animate-pulse font-sans">Оновлення...</span>
                </template>
              </div>

              <!-- Right: Quick Controls (Always accessible directly in the bar) -->
              <div class="flex items-center space-x-2.5 shrink-0">
                
                <!-- Quick Skip Toggle -->
                <button @click="toggleStudioSkip()"
                  :class="currentStudioSkip ? 'bg-rose-600 hover:bg-rose-500 text-white shadow-rose-900/30' : 'bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700'"
                  class="px-2.5 py-1.5 rounded-xl font-bold text-xs transition flex items-center space-x-1.5 shadow"
                  title="Включити або виключити з експорту (Клавіша S)">
                  <template v-if="!currentStudioSkip">
                    <span class="flex items-center space-x-1">
                      <svg class="w-3 h-3 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
                      </svg>
                      <span>Експорт</span>
                    </span>
                  </template>
                  <template v-if="currentStudioSkip">
                    <span class="flex items-center space-x-1">
                      <svg class="w-3 h-3 text-rose-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636"/>
                      </svg>
                      <span>Пропущено</span>
                    </span>
                  </template>
                </button>

                <!-- SVG Vector Badge (for SVGs) -->
                <template v-if="currentStudioImage?.format === 'svg'">
                  <div class="flex items-center space-x-2 bg-purple-950/60 border border-purple-800/80 px-3 py-1.5 rounded-xl text-xs font-mono text-purple-300 shadow">
                    <span class="w-2 h-2 rounded-full bg-purple-400 inline-block animate-pulse"></span>
                    <span>100% Векторний Lossless (SVGO)</span>
                  </div>
                </template>

                <!-- Quick Lossy / Lossless pills (for raster images) -->
                <template v-if="currentStudioImage?.format !== 'svg'">
                  <div class="flex items-center space-x-2">
                    <div class="hidden sm:flex items-center bg-slate-950 p-0.5 rounded-xl border border-slate-800 text-xs">
                      <button @click="setStudioLossless(false)"
                        :class="!currentStudioLossless ? 'bg-cyan-600 text-white font-bold shadow' : 'text-slate-400 hover:text-slate-200'"
                        class="px-2.5 py-1 rounded-lg transition">
                        Lossy
                      </button>
                      <button @click="setStudioLossless(true)"
                        :class="currentStudioLossless ? 'bg-cyan-600 text-white font-bold shadow' : 'text-slate-400 hover:text-slate-200'"
                        class="px-2.5 py-1 rounded-lg transition">
                        Lossless
                      </button>
                    </div>

                    <!-- Quick Quality Slider (Active when Lossy) -->
                    <div v-show="!currentStudioLossless" class="hidden md:flex items-center space-x-2 bg-slate-950 px-3 py-1 rounded-xl border border-slate-800">
                      <span class="text-[11px] text-slate-400 font-sans">Якість:</span>
                      <input type="range" min="30" max="98" step="1"
                        :value="currentStudioQuality"
                        @input="setStudioQuality($event.target.value)"
                        class="w-24 lg:w-32 h-1.5 bg-slate-700 rounded-lg appearance-none cursor-pointer accent-cyan-400">
                      <span class="text-xs font-bold text-cyan-400 w-8 text-right font-mono" v-text="currentStudioQuality + '%'"></span>
                    </div>
                  </div>
                </template>

                <!-- Open Detailed Floating Settings Drawer Button -->
                <button @click="toggleStudioSettingsDrawer()"
                  :class="imageStudio.showSettingsDrawer ? 'bg-cyan-600 text-white font-bold shadow ring-2 ring-cyan-400/50' : 'bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700'"
                  class="px-3 py-1.5 rounded-xl text-xs transition flex items-center space-x-1.5 shadow"
                  title="Відкрити повну панель опцій (Клавіша O)">
                  <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4"/>
                  </svg>
                  <span>Налаштування</span>
                  <span class="text-[10px] opacity-75 font-mono hidden sm:inline">[O]</span>
                  <template v-if="isCurrentStudioOverridden">
                    <span class="w-2 h-2 rounded-full bg-purple-400"></span>
                  </template>
                </button>

              </div>
            </div>
          </template>

        </div>

        <!-- FLOATING SETTINGS DRAWER (On-Demand Overlay over Canvas) -->
        <div v-show="imageStudio.showSettingsDrawer"
          class="absolute top-3 right-4 bottom-3 w-88 max-w-[calc(100vw-2rem)] bg-slate-900/95 backdrop-blur-xl border border-slate-700/80 rounded-2xl shadow-2xl z-30 p-5 flex flex-col justify-between overflow-y-auto space-y-5 text-xs">
          
          <div class="space-y-5">
            <!-- Header with Title and Close Button -->
            <div class="flex items-center justify-between pb-3 border-b border-slate-800">
              <div class="flex items-center space-x-2">
                <svg class="w-4 h-4 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
                </svg>
                <span class="text-sm font-bold text-slate-100">Налаштування</span>
                <template v-if="isCurrentStudioOverridden">
                  <span class="px-1.5 py-0.5 rounded text-[10px] font-mono bg-purple-900/60 text-purple-300 border border-purple-700">Custom</span>
                </template>
              </div>
              <button @click="imageStudio.showSettingsDrawer = false"
                class="text-slate-400 hover:text-slate-100 p-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 transition"
                title="Сховати панель (O або Esc)">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
                </svg>
              </button>
            </div>

            <!-- Size increase advisory banner -->
            <template v-if="imageStudio.currentResult && imageStudio.currentResult.savingsBytes < 0">
              <div class="p-3 rounded-xl bg-rose-950/40 border border-rose-500/30 text-rose-200 text-xs space-y-1">
                <div class="font-bold flex items-center space-x-1.5 text-rose-300">
                  <svg class="w-4 h-4 text-rose-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
                  </svg>
                  <span>WebP більший за оригінал</span>
                </div>
                <div class="text-[10px] text-rose-200/80 leading-relaxed">
                  Оригінал уже був сильно стиснений. Рекомендовано вибрати <strong>Lossy</strong> та якість <strong>75-80%</strong>, або натиснути <strong>«Пропустити»</strong> нижче.
                </div>
              </div>
            </template>
            
            <!-- Section 1: Override Status & Skip Toggle -->
            <div class="space-y-2">
              <div class="text-[11px] font-bold text-slate-400 uppercase tracking-wider">Статус для експорту</div>
              
              <button @click="toggleStudioSkip()"
                :class="currentStudioSkip ? 'bg-rose-600 hover:bg-rose-500 text-white shadow-rose-900/40' : 'bg-emerald-600/90 hover:bg-emerald-500 text-white shadow-emerald-900/40'"
                class="w-full py-2.5 px-3 rounded-xl font-bold transition flex items-center justify-center space-x-2 shadow-lg">
                <span v-show="!currentStudioSkip" class="flex items-center space-x-1.5">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
                  </svg>
                  <span>Включено в експорт</span>
                </span>
                <span v-show="currentStudioSkip" class="flex items-center space-x-1.5">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636"/>
                  </svg>
                  <span>Пропустити (Не експортувати)</span>
                </span>
              </button>

              <template v-if="isCurrentStudioOverridden">
                <div class="flex items-center justify-between pt-1">
                  <span class="text-[11px] text-purple-400 font-mono">Кастомні налаштування</span>
                  <button @click="resetStudioCurrent()" class="text-[11px] text-slate-400 hover:text-amber-300 underline font-sans">
                    Скинути до дефолту
                  </button>
                </div>
              </template>
            </div>

            <!-- SVG Details Section (for SVGs) -->
            <template v-if="currentStudioImage?.format === 'svg'">
              <div class="space-y-4 pt-2 border-t border-slate-800">
                <div class="text-[11px] font-bold text-purple-400 uppercase tracking-wider">Параметри Векторного SVG</div>
                <div class="p-3 rounded-xl bg-purple-950/40 border border-purple-700/40 text-purple-200 text-xs space-y-2">
                  <div class="font-bold flex items-center space-x-1.5 text-purple-300">
                    <span class="w-2 h-2 rounded-full bg-purple-400"></span>
                    <span>SVGO Безпечна Оптимізація</span>
                  </div>
                  <div class="text-[11px] text-slate-300 leading-relaxed space-y-1">
                    <div>✓ Збереження всіх <code>&lt;symbol id="..."&gt;</code> для спрайтів</div>
                    <div>✓ Видалення метаданих, коментарів та редакторського сміття</div>
                    <div>✓ 100% збереження векторної якості (Lossless)</div>
                    <div>✓ Збереження оригінального формату <code class="text-purple-300 font-bold">.svg</code></div>
                  </div>
                </div>
              </div>
            </template>

            <!-- Raster WebP Tuning Controls (for PNG / JPG / GIF) -->
            <template v-if="currentStudioImage?.format !== 'svg'">
              <div class="space-y-5">
                <!-- Section 2: Format (Lossy vs Lossless) -->
                <div class="space-y-2 pt-2 border-t border-slate-800">
                  <div class="text-[11px] font-bold text-slate-400 uppercase tracking-wider">Режим WebP</div>
                  
                  <div class="grid grid-cols-2 gap-2">
                    <button @click="setStudioLossless(false)"
                      :class="!currentStudioLossless ? 'bg-cyan-600 text-white font-bold shadow' : 'bg-slate-800 text-slate-400 hover:text-slate-200 border border-slate-700'"
                      class="py-2 px-2 rounded-xl transition text-center">
                      Lossy (Стиснення)
                    </button>
                    <button @click="setStudioLossless(true)"
                      :class="currentStudioLossless ? 'bg-cyan-600 text-white font-bold shadow' : 'bg-slate-800 text-slate-400 hover:text-slate-200 border border-slate-700'"
                      class="py-2 px-2 rounded-xl transition text-center">
                      Lossless (100%)
                    </button>
                  </div>
                </div>

                <!-- Section 3: Quality Slider (Active when Lossy) -->
                <div class="space-y-3 pt-2 border-t border-slate-800" :class="currentStudioLossless ? 'opacity-40 pointer-events-none' : ''">
                  <div class="flex items-center justify-between">
                    <span class="text-[11px] font-bold text-slate-400 uppercase tracking-wider">Якість (Quality)</span>
                    <span class="text-sm font-extrabold text-cyan-400 font-mono" v-text="currentStudioQuality + '%'"></span>
                  </div>

                  <input type="range" min="30" max="98" step="1"
                    :value="currentStudioQuality"
                    @input="setStudioQuality($event.target.value)"
                    class="w-full h-2 bg-slate-700 rounded-lg appearance-none cursor-pointer accent-cyan-400">

                  <!-- Quick Presets -->
                  <div class="flex items-center justify-between text-[11px] font-mono">
                    <template v-for="q in [70, 75, 80, 85, 90, 95]">
                      <button @click="setStudioQuality(q)"
                        :class="currentStudioQuality === q ? 'bg-cyan-600 text-white font-bold shadow' : 'bg-slate-800 text-slate-400 hover:text-slate-200'"
                        class="px-2 py-1 rounded transition"
                        v-text="q + '%'"></button>
                    </template>
                  </div>
                </div>

                <!-- Section 4: Geometry (Retina 2x) -->
                <div class="space-y-3 pt-2 border-t border-slate-800">
                  <div class="flex items-center justify-between">
                    <span class="text-[11px] font-bold text-slate-400 uppercase tracking-wider">Геометрія та Масштаб</span>
                    <span class="text-[10px] font-mono"
                      :class="currentStudioRetina ? 'text-amber-400 font-bold' : 'text-emerald-400 font-bold'"
                      v-text="currentStudioRetina ? 'Downscaled' : '100% Оригінал'"></span>
                  </div>

                  <label class="flex items-start space-x-2.5 cursor-pointer select-none p-2.5 rounded-xl bg-slate-950/60 border border-slate-800/80 hover:border-slate-700 transition">
                    <input type="checkbox" :checked="currentStudioRetina" @change="toggleStudioRetina()"
                      class="mt-0.5 rounded bg-slate-800 border-slate-700 text-cyan-500 focus:ring-0 focus:ring-offset-0 w-4 h-4 cursor-pointer">
                    <div class="text-slate-300 text-xs">
                      <div class="font-medium">Зменшити до Retina 2x</div>
                      <div class="text-[10px] mt-0.5 leading-snug">
                        <span v-show="!currentStudioRetina" class="text-emerald-400">
                          ✓ 100% оригінальна якість (<span v-text="(currentStudioImage?.naturalWidth || currentStudioImage?.width || 0) + '×' + (currentStudioImage?.naturalHeight || currentStudioImage?.height || 0) + ' px'"></span>)
                        </span>
                        <span v-show="currentStudioRetina" class="text-amber-300 flex items-center space-x-1">
                          <svg class="w-3 h-3 text-amber-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
                          </svg>
                          <span>Зменшується до: <span class="font-bold font-mono" v-text="(currentStudioImage?.recommendedRetinaWidth || (currentStudioImage?.maxRenderedWidth ? currentStudioImage?.maxRenderedWidth * 2 : 0) || 200) + ' px'"></span></span>
                        </span>
                      </div>
                    </div>
                  </label>
                </div>

                <!-- Section 5: Gradient Enhancements -->
                <div class="space-y-3 pt-2 border-t border-slate-800">
                  <div class="text-[11px] font-bold text-slate-400 uppercase tracking-wider">Градієнти та Деталі</div>

                  <label class="flex items-start space-x-2.5 cursor-pointer select-none p-2.5 rounded-xl bg-slate-950/60 border border-slate-800/80 hover:border-slate-700 transition">
                    <input type="checkbox" :checked="currentStudioDither" @change="toggleStudioDither()"
                      class="mt-0.5 rounded bg-slate-800 border-slate-700 text-cyan-500 focus:ring-0 focus:ring-offset-0 w-4 h-4 cursor-pointer">
                    <div class="text-slate-300 text-xs">
                      <div class="font-medium">Anti-Banding Dither</div>
                      <div class="text-[10px] text-slate-500">Прибирає смуги та кільця на плавних градієнтах</div>
                    </div>
                  </label>
                </div>
              </div>
            </template>

          </div>

          <!-- Bottom Sidebar Actions -->
          <div class="space-y-2 pt-4 border-t border-slate-800 shrink-0">
            <template v-if="studioOverridesCount > 0">
              <button @click="resetAllStudioOverrides()"
                class="w-full py-1.5 text-slate-400 hover:text-rose-300 text-[11px] text-center transition">
                Скинути всі кастомні оверрайди (<span v-text="studioOverridesCount"></span>)
              </button>
            </template>

            <template v-if="packageContext?.active">
              <button @click="saveCurrentToPackage()"
                class="w-full py-2.5 bg-fuchsia-600 hover:bg-fuchsia-500 text-white font-bold rounded-xl transition text-center shadow flex items-center justify-center space-x-2">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4"/>
                </svg>
                <span>Оновити в пакеті (⌘S)</span>
              </button>
              <button @click="openPackageCompareHTML()"
                class="w-full py-2 bg-slate-800 hover:bg-slate-700 text-cyan-300 font-medium rounded-xl transition text-center border border-slate-700 text-xs flex items-center justify-center space-x-2">
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/>
                </svg>
                <span>Відкрити compare.html</span>
              </button>
            </template>
            <template v-else>
              <button @click="downloadCurrentStudioWebP()"
                class="w-full py-2.5 bg-emerald-600 hover:bg-emerald-500 text-white font-bold rounded-xl transition text-center shadow flex items-center justify-center space-x-2">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/>
                </svg>
                <span v-text="(currentStudioImage?.format === 'svg' && imageStudio.currentResult?.isSkipped) ? 'Завантажити цей SVG' : 'Завантажити цей WebP'"></span>
              </button>
            </template>
            <button @click="downloadOriginalImage(currentStudioImage?.url)"
              class="w-full py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 font-medium rounded-xl transition text-center border border-slate-700 text-xs flex items-center justify-center space-x-2">
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/>
              </svg>
              <span>Зберегти оригінал на диск</span>
            </button>
          </div>

        </div>

      </div>

      <!-- BOTTOM FILMSTRIP (HORIZONTAL THUMBNAILS CAROUSEL) -->
      <div class="h-24 border-t border-slate-800 bg-slate-950 px-4 flex items-center space-x-2 overflow-x-auto shrink-0 shadow-inner">
        <template v-for="(img, idx) in imageStudioImages" :key="img.url">
          <div @click="studioSelectImage(idx)"
            :class="idx === imageStudio.currentIndex ? 'ring-2 ring-cyan-400 bg-slate-800/90 scale-105' : 'opacity-60 hover:opacity-100 bg-slate-900 border border-slate-800'"
            class="h-18 w-24 shrink-0 rounded-xl p-1.5 cursor-pointer transition flex flex-col justify-between overflow-hidden relative">
            
            <!-- Thumbnail preview -->
            <div class="h-10 w-full flex items-center justify-center overflow-hidden rounded bg-slate-950/60">
              <img :src="img.url" class="max-h-full max-w-full object-cover" loading="lazy" alt="thumb">
            </div>

            <!-- Meta badge -->
            <div class="flex items-center justify-between text-[9px] font-mono mt-1">
              <span class="truncate text-slate-300 font-bold" v-text="img.basename"></span>
              
              <!-- Status indicator -->
              <template v-if="img.isModified || packageContext?.modifiedIds?.includes(img.id)">
                <span class="w-2 h-2 rounded-full bg-emerald-400 inline-block shrink-0 ring-1 ring-emerald-300" title="Оновлено в пакеті"></span>
              </template>
              <template v-else-if="imageStudio.overrides[img.url]?.skip">
                <span class="w-2 h-2 rounded-full bg-rose-500 inline-block shrink-0" title="Пропущено"></span>
              </template>
              <template v-else-if="!imageStudio.overrides[img.url]?.skip && imageStudio.overrides[img.url]">
                <span class="w-2 h-2 rounded-full bg-purple-400 inline-block shrink-0" title="Кастомні налаштування"></span>
              </template>
            </div>
          </div>
        </template>
      </div>

    </div>
</template>

<script>
import { useApp } from '@/store/app';

export default {
  name: 'ImageStudio',
  setup() {
    return useApp();
  }
};
</script>
