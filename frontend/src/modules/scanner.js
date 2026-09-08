export function createScannerModule() {
  return {
    sitemapInput: '',
    discoveredUrls: [],
    selectedUrls: [],
    urlFilter: '',
    pageSearchQuery: '',
    isParsing: false,
    parseError: '',

    // Domain & IP resolution state
    domainInfo: {
      domain: '',
      ip: '',
      ips: [],
      isCloudflare: false,
      provider: '',
      serverHeader: '',
      isLoading: false
    },

    // Scan state
    isScanning: false,
    scanProgress: null,
    processedCount: 0,
    totalToScan: 0,
    currentScanningUrl: '',
    scanResults: [],
    statusFilter: 'all',
    pagesPage: 1,
    pagesPerPage: 50,
    selectedDetail: null,
    rescanLoadingMap: {},
    w3cLoadingMap: {},

    get filteredDiscoveredUrls() {
      if (!this.urlFilter.trim()) {
        return this.discoveredUrls.map(u => ({ url: u }));
      }
      const query = this.urlFilter.toLowerCase();
      return this.discoveredUrls
        .filter(u => u.toLowerCase().includes(query))
        .map(u => ({ url: u }));
    },

    get progressPercentage() {
      if (this.totalToScan === 0) return 0;
      return Math.round((this.processedCount / this.totalToScan) * 100);
    },

    get statusCounts() {
      const counts = { good: 0, 'needs-improvement': 0, poor: 0, error: 0, critical: 0 };
      const list = this.scanResults || [];
      for (let i = 0; i < list.length; i++) {
        const st = list[i].overallStatus;
        if (st in counts) counts[st]++;
        if (st === 'poor' || st === 'error') counts.critical++;
      }
      return counts;
    },

    countByStatus(status) {
      if (status === 'critical') return this.statusCounts.critical;
      return this.statusCounts[status] || 0;
    },

    get filteredResults() {
      let results = this.scanResults || [];
      if (this.statusFilter === 'critical') {
        results = results.filter(r => r.overallStatus === 'poor' || r.overallStatus === 'error');
      } else if (this.statusFilter !== 'all') {
        results = results.filter(r => r.overallStatus === this.statusFilter);
      }

      if (this.pageSearchQuery && this.pageSearchQuery.trim()) {
        const q = this.pageSearchQuery.toLowerCase().trim();
        results = results.filter(r => 
          (r.url && r.url.toLowerCase().includes(q)) || 
          (r.id && String(r.id).includes(q)) ||
          (r.pluginCacheStatus && r.pluginCacheStatus.toLowerCase().includes(q)) ||
          (r.cloudflareCacheStatus && r.cloudflareCacheStatus.toLowerCase().includes(q))
        );
      }

      return results;
    },

    get totalPagesCount() {
      return Math.max(1, Math.ceil(this.filteredResults.length / this.pagesPerPage));
    },

    get paginatedResults() {
      const results = this.filteredResults;
      const start = (this.pagesPage - 1) * this.pagesPerPage;
      return results.slice(start, start + this.pagesPerPage);
    },

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
    },

    selectAllUrls() {
      this.selectedUrls = [...this.discoveredUrls];
    },

    deselectAllUrls() {
      this.selectedUrls = [];
    },

    openUrlInBrowser(url) {
      if (!url) return;
      const cleanUrl = String(url).trim();
      if (!cleanUrl) return;
      if (window.go && window.go.main && window.go.main.App && window.go.main.App.OpenURL) {
        window.go.main.App.OpenURL(cleanUrl);
      } else {
        window.open(cleanUrl, '_blank');
      }
    },

    async resolveCurrentDomain(url) {
      const target = url || this.sitemapInput || this.config.sitemapUrl;
      if (!target) {
        this.domainInfo = { domain: '', ip: '', ips: [], isCloudflare: false, provider: '', serverHeader: '', isLoading: false };
        return;
      }
      this.domainInfo.isLoading = true;
      try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.ResolveDomain) {
          const res = await window.go.main.App.ResolveDomain(target);
          if (res && res.ip) {
            this.domainInfo = {
              domain: res.domain || '',
              ip: res.ip || '',
              ips: res.ips || [],
              isCloudflare: !!res.isCloudflare,
              provider: res.provider || (res.isCloudflare ? 'Cloudflare Proxy' : 'Direct Server'),
              serverHeader: res.serverHeader || '',
              isLoading: false
            };
            return;
          }
        }
        this.domainInfo.isLoading = false;
      } catch (err) {
        console.warn("[JS LOG] Domain IP resolution error:", err);
        this.domainInfo.isLoading = false;
      }
    },

    async parseSitemap() {
      if (!this.sitemapInput.trim()) {
        this.parseError = 'Будь ласка, вкажіть URL sitemap';
        return;
      }

      this.isParsing = true;
      this.parseError = '';
      this.config.sitemapUrl = this.sitemapInput.trim();
      this.saveConfig?.();
      this.addLog?.('info', `Запит на парсинг sitemap: ${this.sitemapInput.trim()}`);
      this.showToast?.('info', 'Парсинг sitemap', `Завантаження та парсинг ${this.sitemapInput.trim()}...`);

      try {
        let urls = [];
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.ParseSitemap) {
          urls = await window.go.main.App.ParseSitemap(this.sitemapInput.trim(), this.config);
        } else {
          urls = [
            this.sitemapInput.replace('/sitemap.xml', '') + '/page-1',
            this.sitemapInput.replace('/sitemap.xml', '') + '/page-2',
            this.sitemapInput.replace('/sitemap.xml', '') + '/blog/post-1',
          ];
        }

        this.discoveredUrls = urls;
        this.selectedUrls = [...urls];
        this.addLog?.('success', `Успішно знайдено ${urls.length} сторінок у sitemap.`);
        this.showToast?.('success', 'Sitemap розпарсено', `Знайдено ${urls.length} URL для перевірки.`);
      } catch (err) {
        this.parseError = err.message || 'Помилка при парсингу sitemap';
        this.addLog?.('error', `Помилка парсингу sitemap: ${this.parseError}`);
        this.showToast?.('error', 'Помилка Sitemap', this.parseError);
      } finally {
        this.isParsing = false;
      }
    },

    async startScan() {
      if (this.selectedUrls.length === 0) return;

      await this.unlockAudio?.();

      this.isScanning = true;
      this.scanResults = [];
      this.siteAnalytics = null;
      this.runComparison = null;
      this.processedCount = 0;
      this.totalToScan = this.selectedUrls.length;
      this.currentScanningUrl = 'Запуск процесів Chrome...';

      this.addLog?.('info', `Запуск повного сканування (${this.selectedUrls.length} сторінок, ${this.config.concurrency} Chrome потоків)...`);
      this.showToast?.('info', 'Сканування запущено', `Сканується ${this.selectedUrls.length} сторінок...`);

      try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.StartScan) {
          await window.go.main.App.StartScan(this.config, this.selectedUrls);
        }
      } catch (err) {
        alert("Не вдалося запустити сканування: " + err.message);
        this.addLog?.('error', `Не вдалося запустити сканування: ${err.message}`);
        this.showToast?.('error', 'Помилка старту', err.message);
        this.isScanning = false;
      }
    },

    async cancelScan() {
      if (window.go && window.go.main && window.go.main.App && window.go.main.App.CancelScan) {
        await window.go.main.App.CancelScan();
      }
      this.isScanning = false;
    },

    async rescanSingle(item) {
      console.log("[JS LOG] rescanSingle clicked:", item);
      const target = item || this.selectedDetail;
      if (!target || !target.url) {
        console.warn("[JS LOG] rescanSingle missing item or url:", target);
        this.addLog?.('warning', 'Пересканування: немає URL сторінки.');
        this.showToast?.('warning', 'Немає URL', 'Відкрийте деталі сторінки і спробуйте знову.');
        return;
      }
      const pageId = target.id;
      await this.unlockAudio?.();
      this.rescanLoadingMap = { ...this.rescanLoadingMap, [pageId]: true };

      this.addLog?.('info', `🔄 Пересканування однієї сторінки [${pageId}]: ${target.url}`);
      this.showToast?.('info', 'Пересканування сторінки', `Лише ця URL (не весь сайт): ${target.url}`);

      try {
        let updatedResult = null;
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.RescanSingleURL) {
          updatedResult = await window.go.main.App.RescanSingleURL(this.config, target.url, pageId);
        } else {
          throw new Error("Wails backend RescanSingleURL method is unavailable — перезапустіть додаток після збірки");
        }

        if (updatedResult) {
          const idx = this.scanResults.findIndex(r => r.id === pageId);
          if (idx >= 0) {
            this.scanResults[idx] = updatedResult;
            this.scanResults = [...this.scanResults];
          }
          if (this.selectedDetail && this.selectedDetail.id === pageId) {
            this.selectedDetail = updatedResult;
          }

          if (updatedResult.error) {
            this.addLog?.('error', `❌ Помилка перевірки [${pageId}]: ${updatedResult.error}`);
            this.showToast?.('error', 'Помилка сканування', updatedResult.error);
          } else {
            this.addLog?.('success', `✅ Сторінку оновлено [${pageId}]: Status ${updatedResult.statusCode}, LCP ${updatedResult.grades?.lcp?.formatted || '-'}`);
            this.showToast?.('success', 'Сторінку оновлено', `Нові метрики для ${target.url}`);
            this.playPikaPageSound?.();
          }

          await this.updateAnalytics?.();
        }
      } catch (err) {
        console.error("[JS LOG] Error in rescanSingle:", err);
        this.addLog?.('error', `Помилка при повторному скануванні: ${err.message || err}`);
        this.showToast?.('error', 'Помилка повторного сканування', err.message || String(err));
      } finally {
        this.rescanLoadingMap = { ...this.rescanLoadingMap, [pageId]: false };
      }
    },

    async validateW3C(item) {
      console.log("[JS LOG] validateW3C clicked:", item);
      const target = item || this.selectedDetail;
      if (!target || !target.url) {
        console.warn("[JS LOG] validateW3C missing item or url:", target);
        this.addLog?.('warning', 'W3C: немає URL сторінки для валідації.');
        this.showToast?.('warning', 'Немає URL', 'Відкрийте деталі сторінки і натисніть W3C знову.');
        return;
      }
      const pageId = target.id;
      this.w3cLoadingMap = { ...this.w3cLoadingMap, [pageId]: true };

      this.addLog?.('info', `🌐 Запит на W3C HTML5 валідацію: ${target.url}`);
      this.showToast?.('info', 'W3C Валідація', `Надсилання HTML до W3C Nu Validator API...`);

      try {
        let report = null;
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.ValidateW3C) {
          report = await window.go.main.App.ValidateW3C(target.url);
        } else {
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
            this.addLog?.('error', `❌ Помилка W3C Валідації для ${target.url}: ${report.error}`);
            this.showToast?.('error', 'Помилка W3C API', report.error);
          } else if (report.isValid) {
            this.addLog?.('success', `🟢 W3C Валідацію пройдено ідеально! 0 помилок.`);
            this.showToast?.('success', 'W3C Валідно 🟢', 'Сторінка відповідає стандартам W3C.');
          } else {
            this.addLog?.('warning', `⚠️ W3C Звіт для ${target.url}: ${report.errorCount} помилок, ${report.warningCount} варнінгів.`);
            this.showToast?.('warning', 'Зауваження W3C', `Знайдено ${report.errorCount} помилок та ${report.warningCount} варнінгів у розмітці.`);
          }
        }
      } catch (err) {
        console.error("[JS LOG] Error in validateW3C:", err);
        this.addLog?.('error', `Помилка W3C Валідації: ${err.message || err}`);
        this.showToast?.('error', 'W3C Помилка', err.message || String(err));
      } finally {
        this.w3cLoadingMap = { ...this.w3cLoadingMap, [pageId]: false };
      }
    },

    initScannerEvents() {
      if (window.runtime && window.runtime.EventsOn) {
        window.runtime.EventsOn("scan:progress", async (progress) => {
          console.log("[JS LOG] scan:progress event:", progress);
          this.scanProgress = progress;
          this.processedCount = progress.totalProcessed;
          this.totalToScan = progress.totalUrls;
          this.currentScanningUrl = progress.currentUrl;

          if (progress.latestResult) {
            const idx = this.scanResults.findIndex(r => r.id === progress.latestResult.id);
            if (idx >= 0) {
              this.scanResults[idx] = progress.latestResult;
              this.scanResults = [...this.scanResults];
            } else {
              this.scanResults.push(progress.latestResult);
              this.scanResults = [...this.scanResults];
            }

            this.addLog?.('info', `Оброблено сторінку [${progress.latestResult.id}]: ${progress.latestResult.url} (${progress.latestResult.overallStatus})`);
          }

          if (progress.isFinished) {
            this.isScanning = false;
            this.showToast?.('success', 'Сканування завершено ⚡', `Всього перевірено ${this.scanResults.length} сторінок.`);
            this.addLog?.('success', `Сканування завершено! Усього оброблено ${this.scanResults.length} сторінок.`);

            if (this.config.systemNotifications !== false && window.go?.main?.App?.SendSystemNotification) {
              const pagesCount = this.scanResults.length;
              window.go.main.App.SendSystemNotification(
                "SpeedMap",
                "Сканування завершено ⚡",
                `Оброблено ${pagesCount} сторінок. Звіт готовий до перегляду!`
              );
            }

            if (this.totalToScan <= 3) {
              this.playPikaPageSound?.();
            } else {
              this.playPikaFullSound?.();
            }
            await this.updateAnalytics?.();
          }
        });

        window.runtime.EventsOn("scan:canceled", () => {
          this.isScanning = false;
          this.showToast?.('warning', 'Сканування скасовано', 'Сканування зупинено за запитом.');
          this.addLog?.('warning', 'Сканування скасовано користувачем.');
        });

        window.runtime.EventsOn("export:progress", (data) => {
          this.exportProgress = data;
        });
      }
    }
  };
}
