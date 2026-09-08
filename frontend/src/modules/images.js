export function createImagesModule() {
  return {
    imageSearchQuery: '',
    imageFilterTab: 'all',
    imageSortKey: 'size',
    imagePage: 1,
    imagePerPage: 50,
    filteredImages: [],
    isExportingReport: false,

    selectedImageComparison: null,
    imageSectionTab: 'catalog',
    isBatchDownloadingZIP: false,
    isExportingWPApply: false,
    exportProgressMinimized: false,
    exportProgress: null,
    showWPPathModal: false,
    wpPathInput: '/var/www/site',
    wpApplyConfirmed: false,

    // Image Studio State (SEOAEO-235 Live Fine-Tuning & Preview)
    get paginatedImages() {
      const list = this.filteredImages || [];
      const start = (this.imagePage - 1) * (this.imagePerPage || 50);
      return list.slice(start, start + (this.imagePerPage || 50));
    },

    get totalImagePagesCount() {
      const total = (this.filteredImages || []).length;
      return Math.max(1, Math.ceil(total / (this.imagePerPage || 50)));
    },

    setImageFilterTab(tab) {
      this.imageFilterTab = tab;
      this.imagePage = 1;
      this.updateFilteredImages();
    },

    updateFilteredImages() {
      if (!this.siteAnalytics || !this.siteAnalytics.allImages) {
        this.filteredImages = [];
        return;
      }

      let list = this.siteAnalytics.allImages;

      if (this.imageSearchQuery && this.imageSearchQuery.trim()) {
        const q = this.imageSearchQuery.toLowerCase().trim();
        list = list.filter(img => (img.url && img.url.toLowerCase().includes(q)) || (img.format && img.format.toLowerCase().includes(q)));
      }

      const tab = this.imageFilterTab;
      if (tab === 'heavy') {
        list = list.filter(img => img.isHeavy);
      } else if (tab === 'non-webp') {
        list = list.filter(img => {
          const fmt = (img.format || '').toLowerCase();
          return fmt !== 'webp' && fmt !== 'avif' && fmt !== 'svg';
        });
      } else if (tab === 'svg') {
        list = list.filter(img => {
          const fmt = (img.format || '').toLowerCase();
          const u = (img.url || '').toLowerCase().split('?')[0];
          return fmt === 'svg' || u.endsWith('.svg');
        });
      } else if (tab === 'missing-lazy') {
        list = list.filter(img => !img.isLazy && !img.isLCP);
      } else if (tab === 'png') {
        list = list.filter(img => (img.format || '').toLowerCase() === 'png');
      } else if (tab === 'jpg') {
        list = list.filter(img => {
          const fmt = (img.format || '').toLowerCase();
          return fmt === 'jpg' || fmt === 'jpeg';
        });
      }

      list = [...list];
      const sortKey = this.imageSortKey;
      list.sort((a, b) => {
        if (sortKey === 'size') return (b.maxTransferSize || 0) - (a.maxTransferSize || 0);
        if (sortKey === 'savings') return (b.estimatedSavingsBytes || 0) - (a.estimatedSavingsBytes || 0);
        if (sortKey === 'duration') return (b.avgDurationMs || 0) - (a.avgDurationMs || 0);
        if (sortKey === 'pages') return (b.pageCount || 0) - (a.pageCount || 0);
        return 0;
      });

      this.filteredImages = list;
    },

    async downloadOriginalImage(url) {
      const targetUrl = url || (this.currentStudioImage && this.currentStudioImage.url);
      if (!targetUrl) return;
      this.showToast?.('info', 'Збереження файлу', 'Завантаження оригінального зображення...');
      try {
        if (window.go?.main?.App?.DownloadOriginalImage) {
          const savedPath = await window.go.main.App.DownloadOriginalImage(targetUrl, this.config);
          this.addLog?.('success', `🟢 Оригінальний файл збережено: ${savedPath}`);
          this.showToast?.('success', 'Файл збережено 🟢', savedPath, savedPath);
        }
      } catch (err) {
        console.error('Failed to download original image:', err);
        this.showToast?.('error', 'Помилка збереження файлу', err.message);
      }
    },

    async openImageQualityModal(img) {
      if (!img || !img.url) return;
      this.selectedImageComparison = {
        url: img.url,
        conversionResult: null,
        isConverting: true,
        error: null
      };
      this.addLog?.('info', `🖼️ Конвертація зображення в WebP (якість ${this.config.webpQuality || 80}%): ${img.url}`);

      try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.ConvertImageToWebP) {
          const res = await window.go.main.App.ConvertImageToWebP(img.url, this.config);
          this.selectedImageComparison.conversionResult = res;
          this.addLog?.('success', `🟢 Конвертацію завершено: ${res.originalFormatted} ➔ ${res.optimizedFormatted} (-${res.savingsPercent.toFixed(1)}%)`);
        } else {
          throw new Error('ConvertImageToWebP method not available');
        }
      } catch (err) {
        console.error('Failed to convert image:', err);
        this.selectedImageComparison.error = err.message || 'Помилка завантаження/конвертації зображення';
        this.addLog?.('error', `Помилка конвертації WebP: ${err.message}`);
      } finally {
        if (this.selectedImageComparison) {
          this.selectedImageComparison.isConverting = false;
        }
      }
    },

    async downloadSingleWebP(url) {
      const targetUrl = url || (this.selectedImageComparison && this.selectedImageComparison.url);
      if (!targetUrl) return;
      const isSvg = targetUrl.toLowerCase().split('?')[0].endsWith('.svg');
      this.showToast?.('info', isSvg ? 'Збереження SVG' : 'Збереження WebP', 'Завантаження та обробка зображення...');
      try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.DownloadSingleWebPImage) {
          const savedPath = await window.go.main.App.DownloadSingleWebPImage(targetUrl, this.config);
          this.addLog?.('success', `🟢 Зображення збережено: ${savedPath}`);
          this.showToast?.('success', 'Файл збережено 🟢', savedPath, savedPath);
        }
      } catch (err) {
        console.error('Failed to download single image:', err);
        this.showToast?.('error', 'Помилка збереження', err.message);
      }
    },

    async downloadBatchWebPZIP() {
      const heavyImages = this.filteredImages.filter(img => img.isHeavy || img.format !== 'webp');
      if (heavyImages.length === 0) {
        this.showToast?.('warning', 'Відсутні важкі зображення', 'Не знайдено зображень для пакетного стиснення.');
        return;
      }
      this.isBatchDownloadingZIP = true;
      this.addLog?.('info', `📦 Конвертація ${heavyImages.length} зображень та упакування у ZIP (WebP якість ${this.config.webpQuality || 80}%)...`);
      this.showToast?.('info', 'Пакетна компресія WebP', `Оптимізація ${heavyImages.length} зображень...`);

      try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.DownloadOptimizedWebPZIP) {
          const urls = heavyImages.map(img => img.url);
          const zipPath = await window.go.main.App.DownloadOptimizedWebPZIP(urls, this.config);
          this.addLog?.('success', `🟢 ZIP архів успішно збережено: ${zipPath}`);
          this.showToast?.('success', 'ZIP Архів збережено 📦', zipPath, zipPath);
        }
      } catch (err) {
        console.error('Failed to download ZIP archive:', err);
        this.addLog?.('error', `Помилка створення ZIP архіву: ${err.message}`);
        this.showToast?.('error', 'Помилка пакетної компресії', err.message);
      } finally {
        this.isBatchDownloadingZIP = false;
      }
    },

    async pickWPFolderOnLocal() {
      try {
        if (window.go?.main?.App?.SelectDirectory) {
          const selected = await window.go.main.App.SelectDirectory("Виберіть папку сайту WordPress");
          if (selected) {
            this.wpPathInput = selected;
            this.showToast?.('success', 'Папку вибрано', selected);
          }
        }
      } catch (e) {
        console.error("Directory picker error:", e);
      }
    },

    exportWordPressWebPApply() {
      if (!this.scanResults || this.scanResults.length === 0) {
        this.showToast?.('warning', 'Немає скану', 'Спочатку проскануйте сайт.');
        return;
      }
      this.wpApplyConfirmed = false;
      this.showWPPathModal = true;
    },

    async confirmExportWPApply() {
      const wordpressPath = (this.wpPathInput || '').trim();
      if (!wordpressPath) {
        this.showToast?.('info', 'Скасовано', 'Потрібен шлях до WordPress.');
        return;
      }
      if (!this.wpApplyConfirmed) {
        this.showToast?.('info', 'Підтвердіть', 'Поставте галочку підтвердження перед записом.');
        return;
      }
      this.showWPPathModal = false;
      const domain = this.config.sitemapUrl || this.sitemapInput || 'site';
      this.isExportingWPApply = true;
      this.exportProgress = { current: 0, total: 0, percent: 0, filename: 'Підготовка файлів...' };
      this.addLog?.('info', `WP package: convert → folder → ${wordpressPath}...`);
      try {
        let res;
        if (window.go?.main?.App?.ExportWordPressWebPApplyPHPWithOverrides) {
          const overrides = this.imageStudio?.overrides || {};
          res = await window.go.main.App.ExportWordPressWebPApplyPHPWithOverrides(
            domain, this.config, this.scanResults, wordpressPath, overrides
          );
        } else if (window.go?.main?.App?.ExportWordPressWebPApplyPHP) {
          res = await window.go.main.App.ExportWordPressWebPApplyPHP(
            domain, this.config, this.scanResults, wordpressPath
          );
        }
        if (res) {
          const count = res.webpCount ?? res.WebPCount ?? 0;
          const applyPHP = res.applyPHP || res.ApplyPHP || '';
          const reviewZIP = res.reviewZIP || res.ReviewZIP || '';
          const rollbackPHP = res.rollbackPHP || res.RollbackPHP || '';
          const pkg = res.packageDir || res.PackageDir || '';
          this.addLog?.('success', `Package: ${count} WebP → ${pkg}`);
          this.addLog?.('info', `Apply PHP: ${applyPHP}`);
          this.addLog?.('info', `ZIP: ${reviewZIP}`);
          this.addLog?.('info', `Rollback: ${rollbackPHP}`);
          this.addLog?.('info', `На Stage: розпакуй ZIP/залий папку, потім wp eval-file …/apply.php --path=/path/to/wp`);
          this.addLog?.('info', `PHP скопіює images/*/optimized.webp → uploads/{webpRel} і оновить attachment.`);
          const targetOpen = pkg || reviewZIP || applyPHP;
          this.showToast?.('success', `${count} WebP package`, targetOpen, targetOpen);

          if (this.config.systemNotifications !== false && window.go?.main?.App?.SendSystemNotification) {
            window.go.main.App.SendSystemNotification(
              "SpeedMap",
              "Експорт WebP готовий 📦",
              `Оптимізовано ${count} зображень!`
            );
          }
        } else {
          throw new Error('ExportWordPressWebPApplyPHP method not available');
        }
      } catch (err) {
        console.error('Failed to export WP apply PHP:', err);
        this.addLog?.('error', `Помилка WP apply PHP: ${err.message}`);
        this.showToast?.('error', 'Помилка WP apply PHP', err.message);
      } finally {
        this.isExportingWPApply = false;
        this.exportProgress = null;
        this.exportProgressMinimized = false;
      }
    },

  };
}
