function speedMapApp() {
  return {
    // Config state - DEFAULT TO MOBILE MODE 📱
    config: {
      sitemapUrl: '',
      concurrency: 1, // Default 1 process as requested
      heavyImageThresholdKB: 100, // Default 100 KB heavy image threshold
      webpQuality: 80, // WebP compression quality (1 - 100)
      pngWebPRatio: 30, // Default 30% of original (70% savings)
      jpgWebPRatio: 60, // Default 60% of original (40% savings)
      gifWebPRatio: 50, // Default 50% of original (50% savings)
      authUser: '',
      authPass: '',
      headers: [
        { key: 'X-SpeedMap-Scanner', value: '1.0' }
      ],
      isMobile: true, // Default to Mobile Emulation as requested
      autoScroll: false, // Disabled by default as requested
      timeoutSec: 30
    },

    // UI state
    activeTab: 'pages', // 'pages' | 'analytics' | 'images'
    settingsTab: 'general', // 'general' | 'images' | 'scanner' | 'auth'
    showSettings: false,
    showLogDrawer: false,
    sitemapInput: '', // Empty initially, placeholder only
    discoveredUrls: [],
    selectedUrls: [],
    urlFilter: '',
    isParsing: false,
    parseError: '',

    // Activity Log & Toast Feed
    activityLogs: [],
    toast: {
      show: false,
      type: 'info', // 'info' | 'success' | 'warning' | 'error'
      title: '',
      message: ''
    },

    // Scan state
    isScanning: false,
    scanProgress: null,
    processedCount: 0,
    totalToScan: 0,
    currentScanningUrl: '',
    scanResults: [],
    statusFilter: 'all',
    selectedDetail: null,
    rescanLoadingMap: {},
    w3cLoadingMap: {},

    // Site Analytics & Comparison State
    siteAnalytics: null,
    runComparison: null,
    isComputingAnalytics: false,

    // Image Optimization & Comparison Hub State (SEOAEO-235)
    imageSearchQuery: '',
    imageFilterTab: 'all', // 'all' | 'heavy' | 'non-webp' | 'missing-lazy' | 'png' | 'jpg'
    imageSortKey: 'size', // 'size' | 'savings' | 'duration' | 'pages'
    isExportingReport: false,
    selectedImageComparison: null, // { url, conversionResult, isConverting, error }
    isBatchDownloadingZIP: false,

    loadSavedConfig() {
      try {
        const saved = localStorage.getItem('speedmap_config_v1');
        if (saved) {
          const parsed = JSON.parse(saved);
          this.config = { ...this.config, ...parsed };
          if (this.config.sitemapUrl && !this.sitemapInput) {
            this.sitemapInput = this.config.sitemapUrl;
          }
          console.log("[JS LOG] Loaded persistent config from localStorage:", this.config);
        }
      } catch (err) {
        console.error("Failed to load saved config:", err);
      }
    },

    saveConfig() {
      try {
        localStorage.setItem('speedmap_config_v1', JSON.stringify(this.config));
        console.log("[JS LOG] Saved config to localStorage.");
      } catch (err) {
        console.error("Failed to save config:", err);
      }
    },

    initApp() {
      console.log("[JS LOG] speedMapApp initialized.");
      this.loadSavedConfig();
      this.addLog('info', 'SpeedMap додаток готовий до роботи.');

      if (this.$watch) {
        this.$watch('config', () => {
          this.saveConfig();
        }, { deep: true });
      }

      // Listen to Wails scan progress events
      if (window.runtime && window.runtime.EventsOn) {
        window.runtime.EventsOn("scan:progress", async (progress) => {
          console.log("[JS LOG] scan:progress event:", progress);
          this.scanProgress = progress;
          this.processedCount = progress.totalProcessed;
          this.totalToScan = progress.totalUrls;
          this.currentScanningUrl = progress.currentUrl;

          if (progress.latestResult) {
            // Append or update result in scanResults
            const idx = this.scanResults.findIndex(r => r.id === progress.latestResult.id);
            if (idx >= 0) {
              this.scanResults[idx] = progress.latestResult;
              this.scanResults = [...this.scanResults]; // Force Alpine reactivity
            } else {
              this.scanResults.push(progress.latestResult);
              this.scanResults = [...this.scanResults];
            }

            this.addLog('info', `Оброблено сторінку [${progress.latestResult.id}]: ${progress.latestResult.url} (${progress.latestResult.overallStatus})`);
          }

          if (progress.isFinished) {
            this.isScanning = false;
            this.showToast('success', 'Сканування завершено', `Всього перевірено ${this.scanResults.length} сторінок.`);
            this.addLog('success', `Сканування завершено! Усього оброблено ${this.scanResults.length} сторінок.`);
            await this.updateAnalytics();
          }
        });

        window.runtime.EventsOn("scan:canceled", () => {
          this.isScanning = false;
          this.showToast('warning', 'Сканування скасовано', 'Сканування зупинено за запитом.');
          this.addLog('warning', 'Сканування скасовано користувачем.');
        });
      }
    },

    addLog(type, message) {
      const timeStr = new Date().toLocaleTimeString('uk-UA');
      this.activityLogs.unshift({ time: timeStr, type: type, message: message });
      if (this.activityLogs.length > 100) {
        this.activityLogs.pop();
      }
    },

    showToast(type, title, message) {
      this.toast = { show: true, type, title, message };
      setTimeout(() => {
        if (this.toast.title === title) {
          this.toast.show = false;
        }
      }, 5000);
    },

    async updateAnalytics() {
      if (this.scanResults.length === 0) return;
      this.isComputingAnalytics = true;
      try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.ComputeSiteAnalytics) {
          const res = await window.go.main.App.ComputeSiteAnalytics(this.config.sitemapUrl || this.sitemapInput, this.config, this.scanResults);
          if (res) {
            this.siteAnalytics = res.analytics;
            this.runComparison = res.comparison;
            this.addLog('info', `Оновлено загальну статистику сайту: Health Score = ${this.siteAnalytics.healthScore}%`);
          }
        }
      } catch (err) {
        console.error("Failed to compute analytics:", err);
      } finally {
        this.isComputingAnalytics = false;
      }
    },

    // Sitemap Discovery
    async parseSitemap() {
      if (!this.sitemapInput.trim()) {
        this.parseError = 'Будь ласка, вкажіть URL sitemap';
        return;
      }

      this.isParsing = true;
      this.parseError = '';
      this.config.sitemapUrl = this.sitemapInput.trim();
      this.saveConfig();
      this.addLog('info', `Запит на парсинг sitemap: ${this.sitemapInput.trim()}`);
      this.showToast('info', 'Парсинг sitemap', `Завантаження та парсинг ${this.sitemapInput.trim()}...`);

      try {
        let urls = [];
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.ParseSitemap) {
          urls = await window.go.main.App.ParseSitemap(this.sitemapInput.trim(), this.config);
        } else {
          // Fallback mock for browser preview
          urls = [
            this.sitemapInput.replace('/sitemap.xml', '') + '/page-1',
            this.sitemapInput.replace('/sitemap.xml', '') + '/page-2',
            this.sitemapInput.replace('/sitemap.xml', '') + '/blog/post-1',
          ];
        }

        this.discoveredUrls = urls;
        this.selectedUrls = [...urls]; // Select all by default
        this.addLog('success', `Успішно знайдено ${urls.length} сторінок у sitemap.`);
        this.showToast('success', 'Sitemap розпарсено', `Знайдено ${urls.length} URL для перевірки.`);
      } catch (err) {
        this.parseError = err.message || 'Помилка при парсингу sitemap';
        this.addLog('error', `Помилка парсингу sitemap: ${this.parseError}`);
        this.showToast('error', 'Помилка Sitemap', this.parseError);
      } finally {
        this.isParsing = false;
      }
    },

    get filteredDiscoveredUrls() {
      if (!this.urlFilter.trim()) {
        return this.discoveredUrls.map(u => ({ url: u }));
      }
      const query = this.urlFilter.toLowerCase();
      return this.discoveredUrls
        .filter(u => u.toLowerCase().includes(query))
        .map(u => ({ url: u }));
    },

    selectAllUrls() {
      this.selectedUrls = [...this.discoveredUrls];
    },

    deselectAllUrls() {
      this.selectedUrls = [];
    },

    addHeader() {
      this.config.headers.push({ key: '', value: '' });
    },

    removeHeader(index) {
      this.config.headers.splice(index, 1);
    },

    // Scan Actions
    async startScan() {
      if (this.selectedUrls.length === 0) return;

      this.isScanning = true;
      this.scanResults = [];
      this.siteAnalytics = null;
      this.runComparison = null;
      this.processedCount = 0;
      this.totalToScan = this.selectedUrls.length;
      this.currentScanningUrl = 'Запуск процесів Chrome...';

      this.addLog('info', `Запуск повного сканування (${this.selectedUrls.length} сторінок, ${this.config.concurrency} Chrome потоків)...`);
      this.showToast('info', 'Сканування запущено', `Сканується ${this.selectedUrls.length} сторінок...`);

      try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.StartScan) {
          await window.go.main.App.StartScan(this.config, this.selectedUrls);
        }
      } catch (err) {
        alert("Не вдалося запустити сканування: " + err.message);
        this.addLog('error', `Не вдалося запустити сканування: ${err.message}`);
        this.showToast('error', 'Помилка старту', err.message);
        this.isScanning = false;
      }
    },

    // Priority Single Rescan URL (one page only — does not start full site scan)
    async rescanSingle(item) {
      console.log("[JS LOG] rescanSingle clicked:", item);
      const target = item || this.selectedDetail;
      if (!target || !target.url) {
        console.warn("[JS LOG] rescanSingle missing item or url:", target);
        this.addLog('warning', 'Пересканування: немає URL сторінки.');
        this.showToast('warning', 'Немає URL', 'Відкрийте деталі сторінки і спробуйте знову.');
        return;
      }
      const pageId = target.id;
      this.rescanLoadingMap = { ...this.rescanLoadingMap, [pageId]: true };

      this.addLog('info', `🔄 Пересканування однієї сторінки [${pageId}]: ${target.url}`);
      this.showToast('info', 'Пересканування сторінки', `Лише ця URL (не весь сайт): ${target.url}`);

      try {
        let updatedResult = null;
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.RescanSingleURL) {
          console.log("[JS LOG] Calling window.go.main.App.RescanSingleURL...");
          updatedResult = await window.go.main.App.RescanSingleURL(this.config, target.url, pageId);
          console.log("[JS LOG] RescanSingleURL returned:", updatedResult);
        } else {
          console.error("[JS LOG] window.go.main.App.RescanSingleURL is NOT available!");
          throw new Error("Wails backend RescanSingleURL method is unavailable — перезапустіть додаток після збірки");
        }

        if (updatedResult) {
          const idx = this.scanResults.findIndex(r => r.id === pageId);
          if (idx >= 0) {
            this.scanResults[idx] = updatedResult;
            this.scanResults = [...this.scanResults]; // Force reactive UI update
          }
          if (this.selectedDetail && this.selectedDetail.id === pageId) {
            this.selectedDetail = updatedResult;
          }

          if (updatedResult.error) {
            this.addLog('error', `❌ Помилка перевірки [${pageId}]: ${updatedResult.error}`);
            this.showToast('error', 'Помилка сканування', updatedResult.error);
          } else {
            this.addLog('success', `✅ Сторінку оновлено [${pageId}]: Status ${updatedResult.statusCode}, LCP ${updatedResult.grades?.lcp?.formatted || '-'}`);
            this.showToast('success', 'Сторінку оновлено', `Нові метрики для ${target.url}`);
          }

          await this.updateAnalytics();
        }
      } catch (err) {
        console.error("[JS LOG] Error in rescanSingle:", err);
        this.addLog('error', `Помилка при повторному скануванні: ${err.message || err}`);
        this.showToast('error', 'Помилка повторного сканування', err.message || String(err));
      } finally {
        this.rescanLoadingMap = { ...this.rescanLoadingMap, [pageId]: false };
      }
    },

    // Official W3C Nu API Validation
    async validateW3C(item) {
      console.log("[JS LOG] validateW3C clicked:", item);
      const target = item || this.selectedDetail;
      if (!target || !target.url) {
        console.warn("[JS LOG] validateW3C missing item or url:", target);
        this.addLog('warning', 'W3C: немає URL сторінки для валідації.');
        this.showToast('warning', 'Немає URL', 'Відкрийте деталі сторінки і натисніть W3C знову.');
        return;
      }
      const pageId = target.id;
      this.w3cLoadingMap = { ...this.w3cLoadingMap, [pageId]: true };

      this.addLog('info', `🌐 Запит на W3C HTML5 валідацію: ${target.url}`);
      this.showToast('info', 'W3C Валідація', `Надсилання HTML до W3C Nu Validator API...`);

      try {
        let report = null;
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.ValidateW3C) {
          console.log("[JS LOG] Calling window.go.main.App.ValidateW3C...");
          report = await window.go.main.App.ValidateW3C(target.url);
          console.log("[JS LOG] ValidateW3C returned:", report);
        } else {
          console.error("[JS LOG] window.go.main.App.ValidateW3C is NOT available!");
          throw new Error("Wails backend ValidateW3C method is unavailable — перезапустіть додаток після збірки");
        }

        if (report) {
          const diagnostics = { ...(target.diagnostics || {}), w3c: report };
          const updated = { ...target, diagnostics };

          const idx = this.scanResults.findIndex(r => r.id === pageId);
          if (idx >= 0) {
            this.scanResults[idx] = updated;
            this.scanResults = [...this.scanResults];
          }
          if (this.selectedDetail && this.selectedDetail.id === pageId) {
            this.selectedDetail = updated;
          }

          if (report.error) {
            this.addLog('error', `❌ Помилка W3C Валідації для ${target.url}: ${report.error}`);
            this.showToast('error', 'Помилка W3C API', report.error);
          } else if (report.isValid) {
            this.addLog('success', `🟢 W3C Валідацію пройдено ідеально! 0 помилок.`);
            this.showToast('success', 'W3C Валідно 🟢', 'Сторінка відповідає стандартам W3C.');
          } else {
            this.addLog('warning', `⚠️ W3C Звіт для ${target.url}: ${report.errorCount} помилок, ${report.warningCount} варнінгів.`);
            this.showToast('warning', 'Зауваження W3C', `Знайдено ${report.errorCount} помилок та ${report.warningCount} варнінгів у розмітці.`);
          }
        }
      } catch (err) {
        console.error("[JS LOG] Error in validateW3C:", err);
        this.addLog('error', `Помилка W3C Валідації: ${err.message || err}`);
        this.showToast('error', 'W3C Помилка', err.message || String(err));
      } finally {
        this.w3cLoadingMap = { ...this.w3cLoadingMap, [pageId]: false };
      }
    },

    // Open URL in external default browser
    openUrlInBrowser(url) {
      if (!url) return;
      if (window.go && window.go.main && window.go.main.App && window.go.main.App.OpenURL) {
        window.go.main.App.OpenURL(url);
      } else {
        window.open(url, '_blank');
      }
    },

    async cancelScan() {
      if (window.go && window.go.main && window.go.main.App && window.go.main.App.CancelScan) {
        await window.go.main.App.CancelScan();
      }
      this.isScanning = false;
    },

    // Progress percentage
    get progressPercentage() {
      if (this.totalToScan === 0) return 0;
      return Math.round((this.processedCount / this.totalToScan) * 100);
    },

    // Filtering & Counts
    countByStatus(status) {
      return this.scanResults.filter(r => r.overallStatus === status).length;
    },

    get filteredResults() {
      if (this.statusFilter === 'all') return this.scanResults;
      if (this.statusFilter === 'critical') {
        return this.scanResults.filter(r => r.overallStatus === 'poor' || r.overallStatus === 'error');
      }
      return this.scanResults.filter(r => r.overallStatus === this.statusFilter);
    },

    get filteredImages() {
      if (!this.siteAnalytics || !this.siteAnalytics.allImages) return [];
      let list = [...this.siteAnalytics.allImages];

      // Text search filter
      if (this.imageSearchQuery.trim()) {
        const q = this.imageSearchQuery.toLowerCase().trim();
        list = list.filter(img => img.url.toLowerCase().includes(q) || (img.format && img.format.toLowerCase().includes(q)));
      }

      // Filter tabs
      if (this.imageFilterTab === 'heavy') {
        list = list.filter(img => img.isHeavy);
      } else if (this.imageFilterTab === 'non-webp') {
        list = list.filter(img => img.format !== 'webp' && img.format !== 'avif' && img.format !== 'svg');
      } else if (this.imageFilterTab === 'missing-lazy') {
        list = list.filter(img => !img.isLazy && !img.isLCP);
      } else if (this.imageFilterTab === 'png') {
        list = list.filter(img => img.format === 'png');
      } else if (this.imageFilterTab === 'jpg') {
        list = list.filter(img => img.format === 'jpg' || img.format === 'jpeg');
      }

      // Sorting
      list.sort((a, b) => {
        if (this.imageSortKey === 'size') return b.maxTransferSize - a.maxTransferSize;
        if (this.imageSortKey === 'savings') return b.estimatedSavingsBytes - a.estimatedSavingsBytes;
        if (this.imageSortKey === 'duration') return b.avgDurationMs - a.avgDurationMs;
        if (this.imageSortKey === 'pages') return b.pageCount - a.pageCount;
        return 0;
      });

      return list;
    },

    async previewImageReport() {
      if (!this.scanResults || this.scanResults.length === 0) {
        this.showToast('warning', 'Відсутні дані', 'Спочатку виконайте сканування сторінок.');
        return;
      }
      this.isPreviewingReport = true;
      this.addLog('info', '🌐 Запуск локального веб-сервера для перегляду звіту (SEOAEO-235)...');
      try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.PreviewImageComparisonHTML) {
          const domain = this.config.sitemapUrl || this.sitemapInput || 'site';
          const reportUrl = await window.go.main.App.PreviewImageComparisonHTML(domain, this.config, this.scanResults);
          this.addLog('success', `🟢 Звіт відкрито у браузері: ${reportUrl}`);
          this.showToast('success', 'Звіт відкрито 🟢', `Локальний веб-сервер запущено: ${reportUrl}`);
        } else {
          throw new Error('PreviewImageComparisonHTML method not available');
        }
      } catch (err) {
        console.error('Failed to preview HTML report:', err);
        this.addLog('error', `Помилка відкриття звіту: ${err.message}`);
        this.showToast('error', 'Помилка перегляду', err.message);
      } finally {
        this.isPreviewingReport = false;
      }
    },

    async exportImageReport() {
      if (!this.scanResults || this.scanResults.length === 0) {
        this.showToast('warning', 'Відсутні дані', 'Спочатку виконайте сканування сторінок.');
        return;
      }
      this.isExportingReport = true;
      this.addLog('info', '📄 Генерація HTML звіту порівняння зображень (SEOAEO-235)...');
      try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.ExportImageComparisonHTML) {
          const domain = this.config.sitemapUrl || this.sitemapInput || 'site';
          const savedPath = await window.go.main.App.ExportImageComparisonHTML(domain, this.config, this.scanResults);
          this.addLog('success', `🟢 Звіт успішно збережено у файл: ${savedPath}`);
          this.showToast('success', 'Звіт збережено 🟢', `Файл порівняння зображень створено: ${savedPath}`);
        } else {
          throw new Error('ExportImageComparisonHTML method not available');
        }
      } catch (err) {
        console.error('Failed to export HTML report:', err);
        this.addLog('error', `Помилка збереження звіту: ${err.message}`);
        this.showToast('error', 'Помилка експорту', err.message);
      } finally {
        this.isExportingReport = false;
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
      this.addLog('info', `🖼️ Конвертація зображення в WebP (якість ${this.config.webpQuality || 80}%): ${img.url}`);

      try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.ConvertImageToWebP) {
          const res = await window.go.main.App.ConvertImageToWebP(img.url, this.config);
          this.selectedImageComparison.conversionResult = res;
          this.addLog('success', `🟢 Конвертацію завершено: ${res.originalFormatted} ➔ ${res.optimizedFormatted} (-${res.savingsPercent.toFixed(1)}%)`);
        } else {
          throw new Error('ConvertImageToWebP method not available');
        }
      } catch (err) {
        console.error('Failed to convert image:', err);
        this.selectedImageComparison.error = err.message || 'Помилка завантаження/конвертації зображення';
        this.addLog('error', `Помилка конвертації WebP: ${err.message}`);
      } finally {
        if (this.selectedImageComparison) {
          this.selectedImageComparison.isConverting = false;
        }
      }
    },

    async downloadSingleWebP(url) {
      const targetUrl = url || (this.selectedImageComparison && this.selectedImageComparison.url);
      if (!targetUrl) return;
      this.showToast('info', 'Збереження WebP', 'Завантаження та конвертація зображення...');
      try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.DownloadSingleWebPImage) {
          const savedPath = await window.go.main.App.DownloadSingleWebPImage(targetUrl, this.config);
          this.addLog('success', `🟢 WebP зображення збережено: ${savedPath}`);
          this.showToast('success', 'Файл збережено 🟢', savedPath);
        }
      } catch (err) {
        console.error('Failed to download single WebP:', err);
        this.showToast('error', 'Помилка збереження WebP', err.message);
      }
    },

    async downloadBatchWebPZIP() {
      const heavyImages = this.filteredImages.filter(img => img.isHeavy || img.format !== 'webp');
      if (heavyImages.length === 0) {
        this.showToast('warning', 'Відсутні важкі зображення', 'Не знайдено зображень для пакетного стиснення.');
        return;
      }
      this.isBatchDownloadingZIP = true;
      this.addLog('info', `📦 Конвертація ${heavyImages.length} зображень та упакування у ZIP (WebP якість ${this.config.webpQuality || 80}%)...`);
      this.showToast('info', 'Пакетна компресія WebP', `Оптимізація ${heavyImages.length} зображень...`);

      try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.DownloadOptimizedWebPZIP) {
          const urls = heavyImages.map(img => img.url);
          const zipPath = await window.go.main.App.DownloadOptimizedWebPZIP(urls, this.config);
          this.addLog('success', `🟢 ZIP архів успішно збережено: ${zipPath}`);
          this.showToast('success', 'ZIP Архів збережено 📦', zipPath);
        }
      } catch (err) {
        console.error('Failed to download ZIP archive:', err);
        this.addLog('error', `Помилка створення ZIP архіву: ${err.message}`);
        this.showToast('error', 'Помилка пакетної компресії', err.message);
      } finally {
        this.isBatchDownloadingZIP = false;
      }
    },

    // Helpers
    badgeClass(status) {
      switch (status) {
        case 'good':
          return 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20';
        case 'needs-improvement':
          return 'bg-amber-500/10 text-amber-400 border border-amber-500/20';
        case 'poor':
          return 'bg-rose-500/10 text-rose-400 border border-rose-500/20';
        default:
          return 'bg-slate-800 text-slate-400';
      }
    }
  };
}
