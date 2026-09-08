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
      zoomMode: 'fit',
      canvasBgMode: 'dark',
      canvasCustomColor: '#ffffff',
      isConverting: false,
      currentResult: null,
      error: null,
      _debounceTimer: null,
      _inFlight: false,
      overrides: {}
    },


    get imageStudioImages() {
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
      return this.config.webpQuality || 80;
    },

    get currentStudioLossless() {
      const override = this.currentStudioOverride;
      if (override.lossless !== undefined) {
        return override.lossless;
      }
      const img = this.currentStudioImage;
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
        return `width: ${Math.round(origW * 1.5)}px; height: ${Math.round(origH * 1.5)}px; max-width: none; max-height: none;`;
      }
      return `aspect-ratio: ${ratio}; width: 100%; max-width: 100%; max-height: calc(100vh - 200px); height: auto;`;
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

      if (targetImg) {
        const url = typeof targetImg === 'string' ? targetImg : targetImg.url;
        const foundIdx = list.findIndex(img => img.url === url);
        this.imageStudio.currentIndex = foundIdx >= 0 ? foundIdx : 0;
      } else {
        this.imageStudio.currentIndex = 0;
      }

      this.imageStudio.isOpen = true;
      this.imageStudio.toggleShowOriginal = false;
      this.imageStudio.zoomMode = 'fit';

      const curImg = this.currentStudioImage;
      if (curImg && curImg.format === 'svg') {
        this.imageStudio.viewMode = 'side';
      }

      this.loadStudioCurrent();
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
        this.imageStudio.currentIndex--;
        const cur = this.currentStudioImage;
        if (cur && cur.format === 'svg' && this.imageStudio.viewMode === 'split') {
          this.imageStudio.viewMode = 'side';
        }
        this.loadStudioCurrent();
      }
    },

    studioNextImage() {
      if (this.imageStudio.currentIndex < this.imageStudioImages.length - 1) {
        this.imageStudio.currentIndex++;
        const cur = this.currentStudioImage;
        if (cur && cur.format === 'svg' && this.imageStudio.viewMode === 'split') {
          this.imageStudio.viewMode = 'side';
        }
        this.loadStudioCurrent();
      }
    },

    studioSelectImage(idx) {
      if (idx >= 0 && idx < this.imageStudioImages.length) {
        this.imageStudio.currentIndex = idx;
        const cur = this.currentStudioImage;
        if (cur && cur.format === 'svg' && this.imageStudio.viewMode === 'split') {
          this.imageStudio.viewMode = 'side';
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
      }, 80);
    },

    async executeStudioConvert() {
      const img = this.currentStudioImage;
      if (!img || !img.url) {
        this.imageStudio.isConverting = false;
        return;
      }

      if (this.imageStudio._inFlight) return;
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
    }
  };
}
