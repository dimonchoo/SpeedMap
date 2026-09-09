export function createConfigModule() {
  return {
    config: {
      sitemapUrl: '',
      concurrency: 1,
      heavyImageThresholdKB: 100,
      webpQuality: 80,
      minWebPQuality: 75,
      skipIfNoWebPSavings: true,
      adaptiveQuality: true,
      resizeToRetina: false,
      autoPruneHistory: true,
      historyRetentionRuns: 20,
      historyRetentionDays: 30,
      filterTrackingBeacons: true,
      excludedImagePatterns: [
        'googleadservices.com',
        'doubleclick.net',
        'facebook.com/tr',
        'bat.bing.com',
        'clarity.ms',
        'px.ads.linkedin.com',
        'analytics.google.com',
        'google-analytics.com',
        't.co/1/i/adsct',
        'stats.wp.com',
        '/pagead/'
      ],
      pngWebPRatio: 30,
      jpgWebPRatio: 60,
      gifWebPRatio: 50,
      authUser: '',
      authPass: '',
      userAgent: '',
      headers: [],
      isMobile: true,
      autoScroll: false,
      soundEnabled: true,
      systemNotifications: true,
      timeoutSec: 30
    },

    defaultDesktopUA: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 SpeedMap/1.0',
    defaultMobileUA: 'Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36 SpeedMap/1.0',

    getActiveUserAgent() {
      if (this.config.userAgent && this.config.userAgent.trim()) {
        return this.config.userAgent.trim();
      }
      return this.config.isMobile ? this.defaultMobileUA : this.defaultDesktopUA;
    },

    resetUserAgent() {
      this.config.userAgent = '';
    },

    get excludedPatternsText() {
      if (!this.config.excludedImagePatterns || !Array.isArray(this.config.excludedImagePatterns)) {
        return '';
      }
      return this.config.excludedImagePatterns.join('\n');
    },

    set excludedPatternsText(val) {
      if (typeof val !== 'string') return;
      this.config.excludedImagePatterns = val.split('\n').map(s => s.trim()).filter(Boolean);
    },

    loadSavedConfig() {
      try {
        const saved = localStorage.getItem('speedmap_config_v1');
        if (saved) {
          const parsed = JSON.parse(saved);
          this.config = { ...this.config, ...parsed };
          if (this.config.adaptiveQuality !== false) {
            this.config.adaptiveQuality = true;
          }
          if (this.config.resizeToRetina === true) {
            this.config.resizeToRetina = true;
          } else {
            this.config.resizeToRetina = false;
          }
          if (this.config.autoPruneHistory !== false) {
            this.config.autoPruneHistory = true;
          }
          if (this.config.historyRetentionRuns === undefined || this.config.historyRetentionRuns === null) {
            this.config.historyRetentionRuns = 20;
          }
          if (this.config.historyRetentionDays === undefined || this.config.historyRetentionDays === null) {
            this.config.historyRetentionDays = 30;
          }
          if (this.config.systemNotifications !== false) {
            this.config.systemNotifications = true;
          }
          if (this.config.sitemapUrl && !this.sitemapInput) {
            this.sitemapInput = this.config.sitemapUrl;
          }
          if (Array.isArray(this.config.headers)) {
            this.config.headers = this.config.headers.filter(h => !(h.key === 'X-SpeedMap-Scanner' && h.value === '1.0'));
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
        if (this.gdriveClientID && this.gdriveClientSecret && window.go?.main?.App?.SaveGDriveCredentials) {
          window.go.main.App.SaveGDriveCredentials(this.gdriveClientID.trim(), this.gdriveClientSecret.trim());
        }
        if (this.activeProfile) {
          this.saveCurrentConfigToActiveProfile();
        }
      } catch (err) {
        console.error("Failed to save config:", err);
      }
    },

    toggleDeviceMode() {
      this.config.isMobile = !this.config.isMobile;
      this.saveConfig();
      if (this.config.isMobile) {
        this.showToast('info', 'Режим змінено 📱', 'Mobile (375×812, емуляція 4G)');
        this.addLog('info', '📱 Увімкнено режим Mobile (375×812, 4G Slowdown, 4x CPU throttling).');
      } else {
        this.showToast('info', 'Режим змінено 🖥️', 'Desktop (1920×1080, повна швидкість)');
        this.addLog('info', '🖥️ Увімкнено режим Desktop (1920×1080, Full Bandwidth).');
      }
    },

    addHeader() {
      this.config.headers.push({ key: '', value: '' });
    },

    removeHeader(index) {
      this.config.headers.splice(index, 1);
    }
  };
}
