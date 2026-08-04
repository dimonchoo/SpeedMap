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
    showSettings: false,
    sitemapInput: 'https://example.com/sitemap.xml',
    discoveredUrls: [],
    selectedUrls: [],
    urlFilter: '',
    isParsing: false,
    parseError: '',

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

    initApp() {
      // Listen to Wails scan progress events
      if (window.runtime && window.runtime.EventsOn) {
        window.runtime.EventsOn("scan:progress", (progress) => {
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
          }

          if (progress.isFinished) {
            this.isScanning = false;
          }
        });

        window.runtime.EventsOn("scan:canceled", () => {
          this.isScanning = false;
        });
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
      } catch (err) {
        this.parseError = err.message || 'Помилка при парсингу sitemap';
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
      this.processedCount = 0;
      this.totalToScan = this.selectedUrls.length;
      this.currentScanningUrl = 'Запуск процесів Chrome...';

      try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.StartScan) {
          await window.go.main.App.StartScan(this.config, this.selectedUrls);
        }
      } catch (err) {
        alert("Не вдалося запустити сканування: " + err.message);
        this.isScanning = false;
      }
    },

    // Priority Single Rescan URL
    async rescanSingle(item) {
      if (!item || !item.url) return;
      this.rescanLoadingMap[item.id] = true;

      try {
        let updatedResult = null;
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.RescanSingleURL) {
          updatedResult = await window.go.main.App.RescanSingleURL(this.config, item.url, item.id);
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
        }
      } catch (err) {
        alert("Помилка при повторному скануванні: " + err.message);
      } finally {
        this.rescanLoadingMap[item.id] = false;
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
