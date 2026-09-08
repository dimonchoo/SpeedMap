/**
 * studio.js - Image Studio fine-tuning, live preview, comparison, and overrides module.
 * SEOAEO-235 Live Fine-Tuning & Preview Workspace.
 */
export function createStudioModule() {
  return {
    // Image Studio State
    imageStudio: {
      isOpen: false,
      showSettingsDrawer: false,
      currentIndex: 0,
      viewMode: 'split',
      splitPos: 50,
      isDraggingSplit: false,
      toggleShowOriginal: false,
      zoomMode: 'fit', // 'fit', '100', '200', '400'
      canvasBgMode: 'dark',
      canvasCustomColor: '#ffffff',
      isConverting: false,
      currentResult: null,
      error: null,
      _debounceTimer: null,
      _inFlight: false,
      _pendingReq: false,
      _cache: {},
      overrides: {}
    },

    // Package Tuning Context (Post-export tuning mode)
    packageContext: {
      active: false,
      packageDir: '',
      manifestPath: '',
      domain: '',
      images: [],
      modifiedIds: [],
      searchQuery: '',
      isLoading: false
    },

    getStudioCacheKey(url) {
      if (!url) return '';
      const q = this.currentStudioQuality;
      const l = !!this.currentStudioLossless;
      const d = !!this.currentStudioDither;
      const r = !!this.currentStudioRetina;
      const o = this.currentStudioOverride;
      const maxW = r ? (o.maxW || 0) : 0;
      const maxH = r ? (o.maxH || 0) : 0;
      return `${url}|q:${q}|l:${l}|d:${d}|r:${r}|w:${maxW}|h:${maxH}`;
    },

    get imageStudioImages() {
      if (this.packageContext?.active) {
        let list = this.packageContext.images || [];
        const q = (this.packageContext.searchQuery || '').toLowerCase().trim();
        if (q) {
          list = list.filter(img =>
            (img.basename && img.basename.toLowerCase().includes(q)) ||
            (img.url && img.url.toLowerCase().includes(q)) ||
            (img.id && img.id.toLowerCase().includes(q))
          );
        }
        return list;
      }
      if (this.filteredImages && this.filteredImages.length > 0) {
        return this.filteredImages;
      }
      return this.siteAnalytics?.allImages || [];
    },

    get currentStudioImage() {
      const list = this.imageStudioImages;
      if (!list || list.length === 0) return null;
      const idx = Math.max(0, Math.min(this.imageStudio.currentIndex, list.length - 1));
      return list[idx] || null;
    },

    get currentStudioOverride() {
      const img = this.currentStudioImage;
      if (!img || !img.url) return {};
      return this.imageStudio.overrides[img.url] || {};
    },

    get currentStudioQuality() {
      const override = this.currentStudioOverride;
      if (override.quality !== undefined && override.quality > 0) {
        return override.quality;
      }
      const img = this.currentStudioImage;
      if (this.packageContext?.active && img?.quality && img.quality > 0) {
        return img.quality;
      }
      return this.config.webpQuality || 80;
    },

    get currentStudioLossless() {
      const override = this.currentStudioOverride;
      if (override.lossless !== undefined) {
        return override.lossless;
      }
      const img = this.currentStudioImage;
      if (this.packageContext?.active && img?.isLossless !== undefined) {
        return !!img.isLossless;
      }
      return img ? (img.format === 'png' && !img.isHeavy) : false;
    },

    get currentStudioDither() {
      const override = this.currentStudioOverride;
      if (override.dither !== undefined) {
        return override.dither;
      }
      return true;
    },

    get currentStudioRetina() {
      const override = this.currentStudioOverride;
      if (override.retina !== undefined) {
        return !!override.retina;
      }
      return false;
    },

    get currentStudioAspectRatio() {
      const res = this.imageStudio?.currentResult;
      if (res && res.originalWidth > 0 && res.originalHeight > 0) {
        return (res.originalWidth / res.originalHeight).toFixed(4);
      }
      const img = this.currentStudioImage;
      if (img && img.naturalWidth > 0 && img.naturalHeight > 0) {
        return (img.naturalWidth / img.naturalHeight).toFixed(4);
      }
      return '1.7778';
    },

    get studioContainerStyle() {
      const zoom = this.imageStudio.zoomMode || 'fit';
      const ratio = this.currentStudioAspectRatio;
      const res = this.imageStudio.currentResult;
      const origW = res?.originalWidth || 800;
      const origH = res?.originalHeight || 600;

      if (zoom === '100') {
        return `width: ${origW}px; height: ${origH}px; max-width: none; max-height: none;`;
      } else if (zoom === '200') {
        return `width: ${origW * 2}px; height: ${origH * 2}px; max-width: none; max-height: none;`;
      } else if (zoom === '400') {
        return `width: ${origW * 4}px; height: ${origH * 4}px; max-width: none; max-height: none;`;
      }
      // Fit mode: fit proportionally within screen while centering
      return `aspect-ratio: ${ratio}; max-width: calc(100vw - 160px); max-height: calc(100vh - 220px); width: min(100%, ${Math.max(origW, 360)}px); height: auto;`;
    },

    get studioSideImageStyle() {
      const zoom = this.imageStudio.zoomMode || 'fit';
      const res = this.imageStudio.currentResult;
      const origW = res?.originalWidth || 800;
      const origH = res?.originalHeight || 600;

      if (zoom === '100') {
        return `width: ${origW}px; height: ${origH}px; max-width: none; max-height: none; object-fit: contain;`;
      } else if (zoom === '200') {
        return `width: ${origW * 2}px; height: ${origH * 2}px; max-width: none; max-height: none; object-fit: contain;`;
      } else if (zoom === '400') {
        return `width: ${origW * 4}px; height: ${origH * 4}px; max-width: none; max-height: none; object-fit: contain;`;
      }
      return `max-height: 100%; max-width: 100%; width: auto; height: auto; object-fit: contain;`;
    },

    get studioCanvasBgStyle() {
      const mode = this.imageStudio?.canvasBgMode || 'dark';
      if (mode === 'light') {
        return 'background-color: #ffffff; background-image: none;';
      } else if (mode === 'checker') {
        return 'background-color: #f8fafc; background-image: repeating-conic-gradient(#cbd5e1 0% 25%, #ffffff 0% 50%); background-size: 20px 20px;';
      } else if (mode === 'custom') {
        const col = this.imageStudio?.canvasCustomColor || '#ffffff';
        return `background-color: ${col}; background-image: none;`;
      }
      return 'background-color: #020617; background-image: radial-gradient(#1e293b 1.5px, transparent 1.5px); background-size: 16px 16px;';
    },

    get currentStudioSkip() {
      const override = this.currentStudioOverride;
      return !!override.skip;
    },

    get isCurrentStudioOverridden() {
      const img = this.currentStudioImage;
      if (!img || !img.url) return false;
      return !!this.imageStudio.overrides[img.url];
    },

    get studioOverridesCount() {
      return Object.keys(this.imageStudio.overrides || {}).length;
    },

    get studioSkippedCount() {
      return Object.values(this.imageStudio.overrides || {}).filter(o => o.skip).length;
    },

    openImageStudio(targetImg) {
      const list = this.imageStudioImages;
      if (!list || list.length === 0) {
        this.showToast?.('info', 'Немає зображень', 'Спочатку запустіть скан сайту.');
        return;
      }

      let targetIdx = 0;
      if (targetImg) {
        const url = typeof targetImg === 'string' ? targetImg : targetImg.url;
        const foundIdx = list.findIndex(img => img.url === url);
        targetIdx = foundIdx >= 0 ? foundIdx : 0;
      }

      this.imageStudio.isOpen = true;
      this.imageStudio.toggleShowOriginal = false;
      this.imageStudio.zoomMode = 'fit';

      this.studioSelectImage(targetIdx);
    },

    closeImageStudio() {
      this.imageStudio.isOpen = false;
      this.imageStudio.showSettingsDrawer = false;
      this.imageStudio.toggleShowOriginal = false;
      this.imageStudio.isDraggingSplit = false;
    },

    toggleStudioSettingsDrawer() {
      this.imageStudio.showSettingsDrawer = !this.imageStudio.showSettingsDrawer;
    },

    studioPrevImage() {
      if (this.imageStudio.currentIndex > 0) {
        this.studioSelectImage(this.imageStudio.currentIndex - 1);
      }
    },

    studioNextImage() {
      if (this.imageStudio.currentIndex < this.imageStudioImages.length - 1) {
        this.studioSelectImage(this.imageStudio.currentIndex + 1);
      }
    },

    studioSelectImage(idx) {
      if (idx >= 0 && idx < this.imageStudioImages.length) {
        this.imageStudio.currentIndex = idx;
        const cur = this.currentStudioImage;
        if (!cur) return;

        // Instant preview if previously cached
        const key = this.getStudioCacheKey(cur.url);
        if (this.imageStudio._cache && this.imageStudio._cache[key]) {
          this.imageStudio.currentResult = this.imageStudio._cache[key];
          this.imageStudio.error = null;
          this.imageStudio.isConverting = false;
        } else {
          // Immediately reset currentResult so old image preview is not displayed on new image!
          this.imageStudio.currentResult = null;
          this.imageStudio.error = null;
          this.imageStudio.isConverting = true;
        }
        this.loadStudioCurrent();
      }
    },

    loadStudioCurrent() {
      const img = this.currentStudioImage;
      if (!img || !img.url) return;
      this.triggerStudioConvert();
    },

    triggerStudioConvert() {
      clearTimeout(this.imageStudio._debounceTimer);
      this.imageStudio.isConverting = true;
      this.imageStudio._debounceTimer = setTimeout(() => {
        this.executeStudioConvert();
      }, 40);
    },

    async executeStudioConvert() {
      const img = this.currentStudioImage;
      if (!img || !img.url) {
        this.imageStudio.isConverting = false;
        return;
      }

      // If already in flight, mark pending so it immediately re-runs when current completes
      if (this.imageStudio._inFlight) {
        this.imageStudio._pendingReq = true;
        return;
      }

      // Cache hit check
      const cacheKey = this.getStudioCacheKey(img.url);
      if (this.imageStudio._cache && this.imageStudio._cache[cacheKey]) {
        this.imageStudio.currentResult = this.imageStudio._cache[cacheKey];
        this.imageStudio.error = null;
        this.imageStudio.isConverting = false;
        return;
      }

      this.imageStudio._inFlight = true;

      const quality = this.currentStudioQuality;
      const lossless = this.currentStudioLossless;
      const dither = this.currentStudioDither;
      const useRetina = this.currentStudioRetina;
      const override = this.currentStudioOverride;
      let maxW = 0;
      let maxH = 0;
      if (useRetina) {
        maxW = (override.maxW !== undefined && override.maxW > 0) ? override.maxW : (img.recommendedRetinaWidth || (img.maxRenderedWidth ? img.maxRenderedWidth * 2 : 0) || 0);
        maxH = (override.maxH !== undefined && override.maxH > 0) ? override.maxH : (img.recommendedRetinaHeight || (img.maxRenderedHeight ? img.maxRenderedHeight * 2 : 0) || 0);
      }

      const tuneOpts = {
        quality: parseFloat(quality),
        lossless: !!lossless,
        exact: false,
        maxW: parseInt(maxW) || 0,
        maxH: parseInt(maxH) || 0,
        dither: !!dither
      };

      try {
        if (window.go?.main?.App?.TuneImagePreview) {
          const res = await window.go.main.App.TuneImagePreview(img.url, tuneOpts, this.config);
          if (!this.imageStudio._cache) this.imageStudio._cache = {};
          this.imageStudio._cache[cacheKey] = res;

          // Only update UI if user is still viewing this image
          if (this.currentStudioImage?.url === img.url) {
            this.imageStudio.currentResult = res;
            this.imageStudio.error = null;
          }
        } else {
          throw new Error('TuneImagePreview IPC method not available');
        }
      } catch (err) {
        console.error('Studio preview error:', err);
        if (this.currentStudioImage?.url === img.url) {
          this.imageStudio.error = err.message || 'Помилка генерації прев\'ю';
        }
      } finally {
        this.imageStudio._inFlight = false;
        this.imageStudio.isConverting = false;

        // If another request arrived while this was in flight, immediately run it!
        if (this.imageStudio._pendingReq) {
          this.imageStudio._pendingReq = false;
          this.executeStudioConvert();
        }
      }
    },

    setStudioQuality(val) {
      const img = this.currentStudioImage;
      if (!img || !img.url) return;
      const cur = this.imageStudio.overrides[img.url] || {};
      this.imageStudio.overrides = {
        ...this.imageStudio.overrides,
        [img.url]: {
          ...cur,
          quality: parseFloat(val),
          lossless: false
        }
      };
      this.triggerStudioConvert();
    },

    setStudioLossless(val) {
      const img = this.currentStudioImage;
      if (!img || !img.url) return;
      const cur = this.imageStudio.overrides[img.url] || {};
      this.imageStudio.overrides = {
        ...this.imageStudio.overrides,
        [img.url]: {
          ...cur,
          lossless: !!val
        }
      };
      this.triggerStudioConvert();
    },

    toggleStudioLossless() {
      this.setStudioLossless(!this.currentStudioLossless);
    },

    toggleStudioDither() {
      const img = this.currentStudioImage;
      if (!img || !img.url) return;
      const cur = this.imageStudio.overrides[img.url] || {};
      this.imageStudio.overrides = {
        ...this.imageStudio.overrides,
        [img.url]: {
          ...cur,
          dither: !this.currentStudioDither
        }
      };
      this.triggerStudioConvert();
    },

    toggleStudioRetina() {
      const img = this.currentStudioImage;
      if (!img || !img.url) return;
      const cur = this.imageStudio.overrides[img.url] || {};
      const willEnable = !this.currentStudioRetina;
      const recW = img.recommendedRetinaWidth || (img.maxRenderedWidth ? img.maxRenderedWidth * 2 : 0) || img.naturalWidth || 0;
      const recH = img.recommendedRetinaHeight || (img.maxRenderedHeight ? img.maxRenderedHeight * 2 : 0) || img.naturalHeight || 0;
      this.imageStudio.overrides = {
        ...this.imageStudio.overrides,
        [img.url]: {
          ...cur,
          retina: willEnable,
          maxW: willEnable ? recW : 0,
          maxH: willEnable ? recH : 0
        }
      };
      this.triggerStudioConvert();
    },

    toggleStudioSkip() {
      const img = this.currentStudioImage;
      if (!img || !img.url) return;
      const cur = this.imageStudio.overrides[img.url] || {};
      const newSkip = !this.currentStudioSkip;
      this.imageStudio.overrides = {
        ...this.imageStudio.overrides,
        [img.url]: {
          ...cur,
          skip: newSkip
        }
      };
      if (newSkip) {
        this.showToast?.('info', 'Файл виключено', 'Зображення не потрапить у фінальний експорт.');
      } else {
        this.showToast?.('success', 'Файл включено', 'Зображення буде оптимізовано та експортовано.');
      }
    },

    resetStudioCurrent() {
      const img = this.currentStudioImage;
      if (!img || !img.url) return;
      const newOverrides = { ...this.imageStudio.overrides };
      delete newOverrides[img.url];
      this.imageStudio.overrides = newOverrides;
      this.loadStudioCurrent();
      this.showToast?.('info', 'Скинуто до дефолту', 'Застосовано глобальні налаштування.');
    },

    resetAllStudioOverrides() {
      this.imageStudio.overrides = {};
      this.loadStudioCurrent();
      this.showToast?.('info', 'Всі налаштування скинуто', 'Усі зображення використовують глобальні дефолти.');
    },

    async downloadCurrentStudioWebP() {
      const img = this.currentStudioImage;
      if (!img || !img.url) return;
      const quality = this.currentStudioQuality;
      const lossless = this.currentStudioLossless;
      const dither = this.currentStudioDither;
      const useRetina = this.currentStudioRetina;
      const override = this.currentStudioOverride;
      const maxW = useRetina ? (override.maxW || img.recommendedRetinaWidth || 0) : 0;
      const maxH = useRetina ? (override.maxH || img.recommendedRetinaHeight || 0) : 0;

      const tuneOpts = {
        quality: parseFloat(quality),
        lossless: !!lossless,
        exact: false,
        maxW: parseInt(maxW) || 0,
        maxH: parseInt(maxH) || 0,
        dither: !!dither
      };

      try {
        if (window.go?.main?.App?.DownloadSingleWebPTuned) {
          const savedPath = await window.go.main.App.DownloadSingleWebPTuned(img.url, tuneOpts, this.config);
          const isSvgSaved = (img.format === 'svg' && this.imageStudio.currentResult?.isSkipped);
          this.showToast?.('success', isSvgSaved ? 'SVG збережено 🟢' : 'WebP збережено 🟢', savedPath, savedPath);
        }
      } catch (err) {
        console.error('Download tuned webp error:', err);
        this.showToast?.('error', 'Помилка збереження', err.message);
      }
    },

    startSplitDrag(e) {
      this.imageStudio.isDraggingSplit = true;
      this.onSplitMouseMove(e);
    },

    onSplitMouseMove(e) {
      if (!this.imageStudio.isDraggingSplit || !this.imageStudio.isOpen) return;
      const container = document.getElementById('studio-split-container');
      if (!container) return;
      const rect = container.getBoundingClientRect();
      if (!rect || rect.width <= 0) return;
      const clientX = (e.touches && e.touches.length > 0) ? e.touches[0].clientX : e.clientX;
      if (clientX === undefined) return;
      const x = Math.max(0, Math.min(clientX - rect.left, rect.width));
      this.imageStudio.splitPos = Math.round((x / rect.width) * 100);
    },

    stopSplitDrag() {
      this.imageStudio.isDraggingSplit = false;
    },

    handleStudioKeydown(e) {
      if (!this.imageStudio.isOpen) return;
      const tag = e.target.tagName ? e.target.tagName.toLowerCase() : '';
      if (tag === 'input' || tag === 'textarea' || tag === 'select') return;

      if ((e.metaKey || e.ctrlKey) && (e.key === 's' || e.key === 'S')) {
        e.preventDefault();
        if (this.packageContext?.active) {
          this.saveCurrentToPackage();
        } else {
          this.downloadCurrentStudioWebP();
        }
        return;
      }

      if (e.key === 'ArrowLeft') {
        e.preventDefault();
        this.studioPrevImage();
      } else if (e.key === 'ArrowRight') {
        e.preventDefault();
        this.studioNextImage();
      } else if (e.key === 'Escape') {
        e.preventDefault();
        if (this.imageStudio.showSettingsDrawer) {
          this.imageStudio.showSettingsDrawer = false;
        } else {
          this.closeImageStudio();
        }
      } else if (e.key === 'o' || e.key === 'O') {
        e.preventDefault();
        this.toggleStudioSettingsDrawer();
      } else if (e.key === 't' || e.key === 'T' || e.code === 'Space') {
        e.preventDefault();
        this.imageStudio.toggleShowOriginal = true;
      } else if (e.key === 's' || e.key === 'S') {
        e.preventDefault();
        this.toggleStudioSkip();
      }
    },

    handleStudioKeyup(e) {
      if (!this.imageStudio.isOpen) return;
      if (e.key === 't' || e.key === 'T' || e.code === 'Space') {
        this.imageStudio.toggleShowOriginal = false;
      }
    },

    onOriginalImgError(e) {
      if (this.currentStudioImage?.url && e?.target && e.target.src !== this.currentStudioImage.url) {
        e.target.src = this.currentStudioImage.url;
      }
    },

    async openExportPackageInStudio(targetDir) {
      try {
        let dir = targetDir;
        if (!dir) {
          if (window.go?.main?.App?.SelectDirectory) {
            dir = await window.go.main.App.SelectDirectory("Виберіть папку експорту SpeedMap (speedmap-webp-...)");
          }
        }
        if (!dir) return;

        this.packageContext.isLoading = true;
        this.showToast?.('info', 'Завантаження пакету', 'Зчитуємо manifest.json...');

        if (window.go?.main?.App?.LoadExportPackageForStudio) {
          const pkg = await window.go.main.App.LoadExportPackageForStudio(dir);
          if (!pkg || !pkg.images || pkg.images.length === 0) {
            throw new Error('У вказаній папці не знайдено зображень або некоректний manifest.json');
          }

          this.packageContext.active = true;
          this.packageContext.packageDir = pkg.packageDir;
          this.packageContext.manifestPath = pkg.manifestPath;
          this.packageContext.domain = pkg.domain;
          this.packageContext.images = pkg.images;
          this.packageContext.modifiedIds = [];
          this.packageContext.searchQuery = '';

          this.imageStudio.isOpen = true;
          this.imageStudio.currentIndex = 0;
          this.imageStudio._cache = {};
          this.studioSelectImage(0);

          const pkgName = pkg.packageDir.split('/').pop() || pkg.packageDir;
          this.showToast?.('success', 'Пакет відкрито 📦', `${pkg.count} зображень завантажено (${pkgName})`);
          this.addLog?.('success', `📦 Відкрито пакет для коригування: ${pkg.packageDir} (${pkg.count} зображень)`);
        } else {
          throw new Error('LoadExportPackageForStudio IPC method not available');
        }
      } catch (err) {
        console.error('Failed to open export package:', err);
        this.showToast?.('error', 'Помилка відкриття пакету', err.message);
        this.addLog?.('error', `Помилка відкриття пакету: ${err.message}`);
      } finally {
        this.packageContext.isLoading = false;
      }
    },

    closePackageStudioContext() {
      this.packageContext.active = false;
      this.packageContext.packageDir = '';
      this.packageContext.images = [];
      this.packageContext.searchQuery = '';
      this.imageStudio._cache = {};
      this.studioSelectImage(0);
      this.showToast?.('info', 'Режим пакету закрито', 'Повернуто до перегляду зображень поточного скану');
    },

    async saveCurrentToPackage() {
      const img = this.currentStudioImage;
      if (!this.packageContext.active || !img || !img.id) {
        return;
      }

      const q = this.currentStudioQuality;
      const l = this.currentStudioLossless;
      const d = this.currentStudioDither;
      const r = this.currentStudioRetina;
      const o = this.currentStudioOverride;
      const maxW = r ? (o.maxW || img.recommendedRetinaWidth || (img.maxRenderedWidth ? img.maxRenderedWidth * 2 : 0) || 0) : 0;
      const maxH = r ? (o.maxH || img.recommendedRetinaHeight || (img.maxRenderedHeight ? img.maxRenderedHeight * 2 : 0) || 0) : 0;

      const tuneOpts = {
        quality: parseFloat(q),
        lossless: !!l,
        exact: false,
        maxW: parseInt(maxW) || 0,
        maxH: parseInt(maxH) || 0,
        dither: !!d
      };

      try {
        this.showToast?.('info', 'Збереження в пакет', `Оновлюємо images/${img.id}/optimized.webp...`);
        if (window.go?.main?.App?.SaveTunedImageToPackage) {
          const res = await window.go.main.App.SaveTunedImageToPackage(
            this.packageContext.packageDir,
            img.id,
            img.url,
            tuneOpts,
            this.config
          );

          // Update local image object
          img.optimizedBytes = res.optimizedBytes;
          img.optimizedFormatted = res.optimizedFormatted;
          img.savingsPercent = res.savingsPercent;
          img.optimizedWidth = res.optimizedWidth;
          img.optimizedHeight = res.optimizedHeight;
          img.quality = tuneOpts.quality;
          img.isLossless = tuneOpts.lossless;
          img.isModified = true;

          if (!this.packageContext.modifiedIds.includes(img.id)) {
            this.packageContext.modifiedIds.push(img.id);
          }

          this.showToast?.('success', `Файл #${img.id} збережено 🟢`, `${img.basename} оновлено в пакеті (${res.optimizedFormatted}, ${res.savingsPercent.toFixed(1)}% економія)`);
          this.addLog?.('success', `🟢 Оновлено в пакеті: images/${img.id}/optimized.webp (${img.basename}) - ${res.optimizedFormatted}`);
        } else {
          throw new Error('SaveTunedImageToPackage IPC method not available');
        }
      } catch (err) {
        console.error('Failed to save to package:', err);
        this.showToast?.('error', 'Помилка збереження в пакет', err.message);
        this.addLog?.('error', `Помилка збереження в пакет: ${err.message}`);
      }
    },

    async openPackageCompareHTML() {
      if (!this.packageContext.active || !this.packageContext.packageDir) return;
      try {
        if (window.go?.main?.App?.OpenPackageCompareHTML) {
          await window.go.main.App.OpenPackageCompareHTML(this.packageContext.packageDir);
        }
      } catch (err) {
        console.error('Failed to open compare.html:', err);
        this.showToast?.('error', 'Помилка відкриття звіту', err.message);
      }
    }
  };
}


