function speedMapApp() {
  return {
    // Config state - DEFAULT TO MOBILE MODE 📱
    config: {
      sitemapUrl: '',
      concurrency: 3,
      authUser: '',
      authPass: '',
      headers: [
        { key: 'X-SpeedMap-Scanner', value: '1.0' }
      ],
      isMobile: true, // Default to Mobile Emulation as requested
      timeoutSec: 30
    },

    // UI state
    activeTab: 'pages', // 'pages' | 'analytics'
    showSettings: false,
    showLogDrawer: false,
    sitemapInput: 'https://example.com/sitemap.xml',
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

    initApp() {
      console.log("[JS LOG] speedMapApp initialized.");
      this.addLog('info', 'SpeedMap додаток готовий до роботи.');

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
          const res = await window.go.main.App.ComputeSiteAnalytics(this.config.sitemapUrl || this.sitemapInput, this.scanResults);
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

    // Priority Single Rescan URL
    async rescanSingle(item) {
      console.log("[JS LOG] rescanSingle clicked:", item);
      if (!item || !item.url) {
        console.warn("[JS LOG] rescanSingle missing item or url:", item);
        return;
      }
      this.rescanLoadingMap = { ...this.rescanLoadingMap, [item.id]: true };

      this.addLog('info', `🔄 Пріоритетна перевірка сторінки [${item.id}]: ${item.url}`);
      this.showToast('info', 'Повторна перевірка', `Запущено сканування сторінки: ${item.url}`);

      try {
        let updatedResult = null;
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.RescanSingleURL) {
          console.log("[JS LOG] Calling window.go.main.App.RescanSingleURL...");
          updatedResult = await window.go.main.App.RescanSingleURL(this.config, item.url, item.id);
          console.log("[JS LOG] RescanSingleURL returned:", updatedResult);
        } else {
          console.error("[JS LOG] window.go.main.App.RescanSingleURL is NOT available!");
          throw new Error("Wails backend RescanSingleURL method is unavailable");
        }

        if (updatedResult) {
          const idx = this.scanResults.findIndex(r => r.id === item.id);
          if (idx >= 0) {
            this.scanResults[idx] = updatedResult;
            this.scanResults = [...this.scanResults]; // Force reactive UI update
          }
          if (this.selectedDetail && this.selectedDetail.id === item.id) {
            this.selectedDetail = updatedResult;
          }

          if (updatedResult.error) {
            this.addLog('error', `❌ Помилка перевірки [${item.id}]: ${updatedResult.error}`);
            this.showToast('error', 'Помилка сканування', updatedResult.error);
          } else {
            this.addLog('success', `✅ Успішно перевірено [${item.id}]: Status ${updatedResult.statusCode}, LCP ${updatedResult.grades?.lcp?.formatted || '-'}`);
            this.showToast('success', 'Сторінку оновлено', `Отримано нові показники для ${item.url}`);
          }

          await this.updateAnalytics();
        }
      } catch (err) {
        console.error("[JS LOG] Error in rescanSingle:", err);
        this.addLog('error', `Помилка при повторному скануванні: ${err.message}`);
        this.showToast('error', 'Помилка повторного сканування', err.message);
      } finally {
        this.rescanLoadingMap = { ...this.rescanLoadingMap, [item.id]: false };
      }
    },

    // Official W3C Nu API Validation
    async validateW3C(item) {
      console.log("[JS LOG] validateW3C clicked:", item);
      if (!item || !item.url) {
        console.warn("[JS LOG] validateW3C missing item or url:", item);
        return;
      }
      this.w3cLoadingMap = { ...this.w3cLoadingMap, [item.id]: true };

      this.addLog('info', `🌐 Запит на W3C HTML5 валідацію: ${item.url}`);
      this.showToast('info', 'W3C Валідація', `Надсилання HTML до W3C Nu Validator API...`);

      try {
        let report = null;
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.ValidateW3C) {
          console.log("[JS LOG] Calling window.go.main.App.ValidateW3C...");
          report = await window.go.main.App.ValidateW3C(item.url);
          console.log("[JS LOG] ValidateW3C returned:", report);
        } else {
          console.error("[JS LOG] window.go.main.App.ValidateW3C is NOT available!");
          throw new Error("Wails backend ValidateW3C method is unavailable");
        }

        if (report) {
          if (!item.diagnostics) item.diagnostics = {};
          item.diagnostics.w3c = report;

          const idx = this.scanResults.findIndex(r => r.id === item.id);
          if (idx >= 0) {
            this.scanResults[idx] = item;
            this.scanResults = [...this.scanResults];
          }
          if (this.selectedDetail && this.selectedDetail.id === item.id) {
            this.selectedDetail = item;
          }

          if (report.error) {
            this.addLog('error', `❌ Помилка W3C Валідації для ${item.url}: ${report.error}`);
            this.showToast('error', 'Помилка W3C API', report.error);
          } else if (report.isValid) {
            this.addLog('success', `🟢 W3C Валідацію пройдено ідеально! 0 помилок.`);
            this.showToast('success', 'W3C Валідно 🟢', 'Сторінка відповідає стандартам W3C.');
          } else {
            this.addLog('warning', `⚠️ W3C Звіт для ${item.url}: ${report.errorCount} помилок, ${report.warningCount} варнінгів.`);
            this.showToast('warning', 'Зауваження W3C', `Знайдено ${report.errorCount} помилок та ${report.warningCount} варнінгів у розмітці.`);
          }
        }
      } catch (err) {
        console.error("[JS LOG] Error in validateW3C:", err);
        this.addLog('error', `Помилка W3C Валідації: ${err.message}`);
        this.showToast('error', 'W3C Помилка', err.message);
      } finally {
        this.w3cLoadingMap = { ...this.w3cLoadingMap, [item.id]: false };
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
