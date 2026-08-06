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
      soundEnabled: true, // Audio notifications toggle
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
    // WebKit/Wails: Audio.play() from scan:progress is not a user gesture → autoplay blocked.
    // Unlock once on click (Start Scan / first interaction), then reuse elements.
    audioUnlocked: false,
    audioCtx: null,
    pikaPageEl: null,
    pikaFullEl: null,

    // Site Analytics & Comparison State
    siteAnalytics: null,
    runComparison: null,
    isComputingAnalytics: false,

    // Font Inspector Hub State
    fontSearchQuery: '',
    fontFilterTab: 'all',
    fontSortKey: 'pages',

    // Image Optimization & Comparison Hub State (SEOAEO-235)
    imageSearchQuery: '',
    imageFilterTab: 'all', // 'all' | 'heavy' | 'non-webp' | 'missing-lazy' | 'png' | 'jpg'
    imageSortKey: 'size', // 'size' | 'savings' | 'duration' | 'pages'
    imagePage: 1,
    imagePerPage: 50,
    filteredImages: [],
    isExportingReport: false,


    selectedImageComparison: null, // { url, conversionResult, isConverting, error }
	isBatchDownloadingZIP: false,
    isExportingWPApply: false,
    showWPPathModal: false,
    wpPathInput: '/var/www/site',


    // Google Drive Cloud Integration State
    gdriveStatus: { connected: false, email: '' },
    isConnectingGDrive: false,
    isUploadingGDrive: false,

    // Multi-Site Profiles State
    siteProfiles: [],
    activeProfileId: '',
    profileSearchQuery: '',
    showProfileDropdown: false,
    showProfileModal: false,
    editingProfile: {
      id: '',
      name: '',
      sitemapUrl: '',
      config: null
    },

    get activeProfile() {
      if (!this.siteProfiles || this.siteProfiles.length === 0) return null;
      return this.siteProfiles.find(p => p.id === this.activeProfileId) || this.siteProfiles[0];
    },

    get filteredSiteProfiles() {
      if (!this.siteProfiles) return [];
      if (!this.profileSearchQuery.trim()) return this.siteProfiles;
      const q = this.profileSearchQuery.toLowerCase().trim();
      return this.siteProfiles.filter(p => 
        (p.name && p.name.toLowerCase().includes(q)) || 
        (p.sitemapUrl && p.sitemapUrl.toLowerCase().includes(q))
      );
    },

    async checkGDriveStatus() {
      try {
        if (window.go?.main?.App?.GetGDriveStatus) {
          const res = await window.go.main.App.GetGDriveStatus();
          if (res) {
            this.gdriveStatus = res;
          }
        }
      } catch (e) {
        console.error("GDrive status check error:", e);
      }
    },

    async connectGDrive() {
      this.isConnectingGDrive = true;
      this.showToast('info', 'Google Drive', 'Відкриваємо браузер для авторизації Google...');
      try {
        if (window.go?.main?.App?.StartGDriveAuth) {
          const email = await window.go.main.App.StartGDriveAuth();
          if (email) {
            this.gdriveStatus = { connected: true, email: email };
            this.showToast('success', 'Успішно підключено 🟢', email);
            this.addLog('success', `Google Drive авторизовано: ${email}`);
          }
        }
      } catch (err) {
        console.error("GDrive Auth Error:", err);
        this.showToast('error', 'Помилка авторизації', err.message || err);
      } finally {
        this.isConnectingGDrive = false;
      }
    },

    async disconnectGDrive() {
      try {
        if (window.go?.main?.App?.DisconnectGDrive) {
          await window.go.main.App.DisconnectGDrive();
          this.gdriveStatus = { connected: false, email: '' };
          this.showToast('info', 'Відключено', 'Google Drive відключено.');
        }
      } catch (e) {
        console.error("GDrive disconnect error:", e);
      }
    },

    async uploadFontsToGDrive() {
      if (!this.siteAnalytics || !this.siteAnalytics.fontUsage || this.siteAnalytics.fontUsage.length === 0) {
        this.showToast('warning', 'Відсутні дані', 'Спочатку виконайте сканування сторінок.');
        return;
      }
      if (!this.gdriveStatus.connected) {
        this.showToast('warning', 'Не підключено', 'Спочатку підключіть Google Drive у Налаштуваннях.');
        this.showSettings = true;
        return;
      }

      this.isUploadingGDrive = true;
      this.showToast('info', 'Вивантаження ☁️', 'Надсилаємо звіт у Google Drive...');
      try {
        let csv = "Font Family,Format,Occurrences,Page Coverage %,Avg Load Duration (ms),Formatted Size,Direct Asset URL,Page URLs List\n";
        this.siteAnalytics.fontUsage.forEach(f => {
          const family = `"${(f.family || '').replace(/"/g, '""')}"`;
          const type = `"${(f.type || '').replace(/"/g, '""')}"`;
          const fontUrl = `"${(f.url || '').replace(/"/g, '""')}"`;
          const pageUrlsList = `"${(f.pageUrls || []).join('; ').replace(/"/g, '""')}"`;
          csv += `${family},${type},${f.occurrences || 0},${f.percentage || 0},${f.avgDurationMs || 0},"${f.formattedSize || ''}",${fontUrl},${pageUrlsList}\n`;
        });

        // Save temporary local CSV file first
        const tempPath = await window.go.main.App.ExportFontsCSV(csv);
        if (tempPath && window.go.main.App.UploadFileToGDrive) {
          const folderName = (this.activeProfile?.name || 'Site') + ' Reports';
          const res = await window.go.main.App.UploadFileToGDrive(tempPath, folderName);
          if (res && res.webViewLink) {
            navigator.clipboard.writeText(res.webViewLink);
            this.showToast('success', 'Вивантажено ☁️ (Link copied 📋)', res.webViewLink);
            this.addLog('success', `Файл з звітом шрифтів вивантажено в Google Drive: ${res.webViewLink}`);
          }
        }
      } catch (err) {
        console.error("GDrive upload error:", err);
        this.showToast('error', 'Помилка вивантаження', err.message || err);
      } finally {
        this.isUploadingGDrive = false;
      }
    },

    async loadSiteProfiles() {
      try {
        let list = [];
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.ListSiteProfiles) {
          list = await window.go.main.App.ListSiteProfiles();
        }

        if (!list || list.length === 0) {
          const defaultUrl = this.config.sitemapUrl || this.sitemapInput || '';
          const defaultName = defaultUrl ? defaultUrl.replace(/^https?:\/\//, '').split('/')[0] : 'Основний сайт';
          
          const defaultProfile = {
            id: '',
            name: defaultName,
            sitemapUrl: defaultUrl,
            config: { ...this.config }
          };
          
          if (window.go && window.go.main && window.go.main.App && window.go.main.App.SaveSiteProfile) {
            const saved = await window.go.main.App.SaveSiteProfile(defaultProfile);
            if (saved) {
              list = [saved];
            }
          } else {
            list = [{ ...defaultProfile, id: 'site_default' }];
          }
        }

        this.siteProfiles = list || [];
        
        const lastActiveId = localStorage.getItem('speedmap_active_profile_id');
        const found = this.siteProfiles.find(p => p.id === lastActiveId);
        if (found) {
          this.applyProfile(found);
        } else if (this.siteProfiles.length > 0) {
          this.applyProfile(this.siteProfiles[0]);
        }
        console.log("[JS LOG] Loaded site profiles:", this.siteProfiles.length, "Active ID:", this.activeProfileId);
      } catch (err) {
        console.error("Failed to load site profiles:", err);
      }
    },

    applyProfile(profile) {
      if (!profile) return;
      this.activeProfileId = profile.id;
      localStorage.setItem('speedmap_active_profile_id', profile.id);
      
      if (profile.sitemapUrl) {
        this.sitemapInput = profile.sitemapUrl;
        this.config.sitemapUrl = profile.sitemapUrl;
      }
      if (profile.config) {
        this.config = { ...this.config, ...profile.config };
      }
      this.saveConfig();
    },

    async selectProfile(profile) {
      if (!profile) return;
      this.applyProfile(profile);
      this.showProfileDropdown = false;
      this.addLog('info', `🌐 Переключено на сайт: ${profile.name} (${profile.sitemapUrl || 'Без sitemap'})`);
      this.showToast('info', 'Сайт змінено 🌐', profile.name);

      if (profile.sitemapUrl) {
        await this.parseSitemap();
      }
    },

    openCreateProfileModal() {
      this.showProfileDropdown = false;
      this.editingProfile = {
        id: '',
        name: '',
        sitemapUrl: '',
        config: JSON.parse(JSON.stringify(this.config))
      };
      this.showProfileModal = true;
    },

    openEditProfileModal(profile) {
      this.showProfileDropdown = false;
      this.editingProfile = {
        id: profile.id,
        name: profile.name,
        sitemapUrl: profile.sitemapUrl,
        config: profile.config ? JSON.parse(JSON.stringify(profile.config)) : JSON.parse(JSON.stringify(this.config))
      };
      this.showProfileModal = true;
    },

    addHeaderToEditingProfile() {
      if (!this.editingProfile.config) return;
      if (!this.editingProfile.config.headers) this.editingProfile.config.headers = [];
      this.editingProfile.config.headers.push({ key: '', value: '' });
    },

    removeHeaderFromEditingProfile(idx) {
      if (!this.editingProfile.config || !this.editingProfile.config.headers) return;
      this.editingProfile.config.headers.splice(idx, 1);
    },


    async saveProfileFromModal() {
      if (!this.editingProfile.name.trim() && !this.editingProfile.sitemapUrl.trim()) {
        this.showToast('warning', 'Заповніть дані', 'Вкажіть назву сайту або URL sitemap');
        return;
      }

      try {
        if (!this.editingProfile.name.trim()) {
          this.editingProfile.name = this.editingProfile.sitemapUrl.replace(/^https?:\/\//, '').split('/')[0] || 'Новий сайт';
        }

        let saved = null;
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.SaveSiteProfile) {
          saved = await window.go.main.App.SaveSiteProfile(this.editingProfile);
        } else {
          saved = { ...this.editingProfile, id: this.editingProfile.id || ('site_' + Date.now()) };
        }

        this.showProfileModal = false;
        await this.loadSiteProfiles();
      this.checkGDriveStatus();
        if (saved) {
          const fresh = this.siteProfiles.find(p => p.id === saved.id) || saved;
          this.applyProfile(fresh);
        }
        this.showToast('success', 'Профіль збережено 🟢', this.editingProfile.name);
        this.addLog('success', `Успішно збережено профіль сайту: ${this.editingProfile.name}`);
      } catch (err) {
        console.error("Failed to save profile:", err);
        this.showToast('error', 'Помилка збереження', err.message);
      }
    },

    async deleteProfile(profileId) {
      const p = this.siteProfiles.find(item => item.id === profileId);
      const name = p ? p.name : 'сайту';
      if (!confirm(`Ви дійсно бажаєте видалити профіль "${name}"?`)) return;

      try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.DeleteSiteProfile) {
          await window.go.main.App.DeleteSiteProfile(profileId);
        }
        
        this.showToast('info', 'Профіль видалено', name);
        this.addLog('info', `Видалено профіль сайту: ${name}`);
        await this.loadSiteProfiles();
      this.checkGDriveStatus();
      } catch (err) {
        console.error('Failed to delete profile:', err);
        this.showToast('error', 'Помилка видалення', err.message);
      }
    },

    async saveCurrentConfigToActiveProfile() {
      if (!this.activeProfile) return;
      const updated = {
        ...this.activeProfile,
        sitemapUrl: this.sitemapInput || this.config.sitemapUrl,
        config: { ...this.config }
      };
      
      try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.SaveSiteProfile) {
          await window.go.main.App.SaveSiteProfile(updated);
        }
        // Update local active profile reference without triggering IPC reload loop
        const idx = this.siteProfiles.findIndex(p => p.id === updated.id);
        if (idx >= 0) {
          this.siteProfiles[idx] = updated;
        }
      } catch (err) {
        console.error("Failed to save current config to active profile:", err);
      }
    },

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
        if (this.activeProfile) {
          this.saveCurrentConfigToActiveProfile();
        }
      } catch (err) {
        console.error("Failed to save config:", err);
      }
    },

    initApp() {
      console.log("[JS LOG] speedMapApp initialized.");
      this.loadSavedConfig();
      this.loadSiteProfiles();
      this.checkGDriveStatus();
      this.addLog('info', 'SpeedMap додаток готовий до роботи.');

      // Unlock WebAudio on first real click (needed for later scan-complete sounds)
      const unlockOnce = () => {
        this.unlockAudio();
        window.removeEventListener('pointerdown', unlockOnce, true);
        window.removeEventListener('keydown', unlockOnce, true);
      };
      window.addEventListener('pointerdown', unlockOnce, true);
      window.addEventListener('keydown', unlockOnce, true);

      if (this.$watch) {
        this.$watch('imageSearchQuery', () => { this.imagePage = 1; this.updateFilteredImages(); });
        this.$watch('imageFilterTab', () => { this.imagePage = 1; this.updateFilteredImages(); });
        this.$watch('imageSortKey', () => { this.imagePage = 1; this.updateFilteredImages(); });
        this.$watch('activeTab', (tab) => {
          if (tab === 'images') {
            this.imagePage = 1;
            this.updateFilteredImages();
          }
        });
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
            this.showToast('success', 'Сканування завершено ⚡', `Всього перевірено ${this.scanResults.length} сторінок.`);
            this.addLog('success', `Сканування завершено! Усього оброблено ${this.scanResults.length} сторінок.`);

            if (this.totalToScan <= 3) {
              this.playPikaPageSound();
            } else {
              this.playPikaFullSound();
            }
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
            this.updateFilteredImages();
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

      // Must unlock during the click that starts the scan — finish event is not a user gesture
      await this.unlockAudio();

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
      await this.unlockAudio();
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
            this.playPikaPageSound();
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

    updateFilteredImages() {
      if (!this.siteAnalytics || !this.siteAnalytics.allImages) {
        this.filteredImages = [];
        return;
      }

      let list = this.siteAnalytics.allImages;

      // Text search filter
      if (this.imageSearchQuery && this.imageSearchQuery.trim()) {
        const q = this.imageSearchQuery.toLowerCase().trim();
        list = list.filter(img => (img.url && img.url.toLowerCase().includes(q)) || (img.format && img.format.toLowerCase().includes(q)));
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

      // Sorting (shallow copy to prevent mutating raw analytics array)
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

    get filteredFonts() {
      if (!this.siteAnalytics || !this.siteAnalytics.fontUsage) return [];
      let list = [...this.siteAnalytics.fontUsage];

      if (this.fontSearchQuery && this.fontSearchQuery.trim()) {
        const q = this.fontSearchQuery.toLowerCase().trim();
        list = list.filter(f => (f.family && f.family.toLowerCase().includes(q)) || (f.type && f.type.toLowerCase().includes(q)) || (f.url && f.url.toLowerCase().includes(q)));
      }

      if (this.fontFilterTab !== 'all') {
        const target = this.fontFilterTab.toLowerCase();
        list = list.filter(f => f.type && f.type.toLowerCase().includes(target));
      }

      const sortKey = this.fontSortKey;
      list.sort((a, b) => {
        if (sortKey === 'pages') return (b.occurrences || 0) - (a.occurrences || 0);
        if (sortKey === 'duration') return (b.avgDurationMs || 0) - (a.avgDurationMs || 0);
        if (sortKey === 'size') return (b.transferSize || 0) - (a.transferSize || 0);
        return 0;
      });

      return list;
    },

    async exportFontsCSV() {
      if (!this.siteAnalytics || !this.siteAnalytics.fontUsage || this.siteAnalytics.fontUsage.length === 0) {
        this.showToast('warning', 'Відсутні дані', 'Немає даних про шрифти для експорту.');
        return;
      }
      let csv = "Font Family,Format,Occurrences,Page Coverage %,Avg Load Duration (ms),Formatted Size,Direct Asset URL,Page URLs List\n";
      this.siteAnalytics.fontUsage.forEach(f => {
        const family = `"${(f.family || '').replace(/"/g, '""')}"`;
        const type = `"${(f.type || '').replace(/"/g, '""')}"`;
        const fontUrl = `"${(f.url || '').replace(/"/g, '""')}"`;
        const pageUrlsList = `"${(f.pageUrls || []).join('; ').replace(/"/g, '""')}"`;
        csv += `${family},${type},${f.occurrences || 0},${f.percentage || 0},${f.avgDurationMs || 0},"${f.formattedSize || ''}",${fontUrl},${pageUrlsList}\n`;
      });

      try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.ExportFontsCSV) {
          const filePath = await window.go.main.App.ExportFontsCSV(csv);
          if (filePath) {
            this.showToast('success', 'CSV збережено 🟢', filePath);
            this.addLog('success', `Експортовано CSV звіт шрифтів з URL сторінок: ${filePath}`);
            return;
          }
        }
      } catch (e) {
        console.warn("Native CSV save dialog fallback:", e);
      }

      // Browser fallback (saves to ~/Downloads)
      const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
      const link = document.createElement('a');
      link.href = URL.createObjectURL(blob);
      link.setAttribute('download', `speedmap_fonts_report_${Date.now()}.csv`);
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      this.showToast('success', 'CSV завантажено 🟢', 'Збережено у папочку Завантаження (~/Downloads)');
    },


    async exportFontsJSON() {
      if (!this.siteAnalytics || !this.siteAnalytics.fontUsage || this.siteAnalytics.fontUsage.length === 0) {
        this.showToast('warning', 'Відсутні дані', 'Немає даних про шрифти для експорту.');
        return;
      }
      const jsonStr = JSON.stringify(this.siteAnalytics.fontUsage, null, 2);

      try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.ExportFontsJSON) {
          const filePath = await window.go.main.App.ExportFontsJSON(jsonStr);
          if (filePath) {
            this.showToast('success', 'JSON збережено 🟢', filePath);
            this.addLog('success', `Експортовано JSON звіт шрифтів: ${filePath}`);
            return;
          }
        }
      } catch (e) {
        console.warn("Native JSON save dialog fallback:", e);
      }

      // Browser fallback (saves to ~/Downloads)
      const dataStr = "data:text/json;charset=utf-8," + encodeURIComponent(jsonStr);
      const link = document.createElement('a');
      link.setAttribute('href', dataStr);
      link.setAttribute('download', `speedmap_fonts_report_${Date.now()}.json`);
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      this.showToast('success', 'JSON завантажено 🟢', 'Збережено у папочку Завантаження (~/Downloads)');
    },


    copyFontUrl(url) {
      if (!url) return;
      navigator.clipboard.writeText(url).then(() => {
        this.showToast('success', 'URL скопійовано 📋', url);
      }).catch(err => {
        console.error('Failed to copy font URL:', err);
      });
    },

    get paginatedImages() {
      const list = this.filteredImages || [];
      const start = (this.imagePage - 1) * (this.imagePerPage || 50);
      return list.slice(start, start + (this.imagePerPage || 50));
    },

    get totalImagePages() {
      const len = this.filteredImages ? this.filteredImages.length : 0;
      return Math.ceil(len / (this.imagePerPage || 50)) || 1;
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

    async pickWPFolderOnLocal() {
      try {
        if (window.go?.main?.App?.SelectDirectory) {
          const selected = await window.go.main.App.SelectDirectory("Виберіть папку сайту WordPress");
          if (selected) {
            this.wpPathInput = selected;
            this.showToast('success', 'Папку вибрано 📁', selected);
          }
        }
      } catch (e) {
        console.error("Directory picker error:", e);
      }
    },

    exportWordPressWebPApply() {
      if (!this.scanResults || this.scanResults.length === 0) {
        this.showToast('warning', 'Немає скану', 'Спочатку проскануйте сайт.');
        return;
      }
      this.showWPPathModal = true;
    },

    async confirmExportWPApply() {
      const wordpressPath = (this.wpPathInput || '').trim();
      if (!wordpressPath) {
        this.showToast('info', 'Скасовано', 'Потрібен шлях до WordPress.');
        return;
      }
      this.showWPPathModal = false;
      const domain = this.config.sitemapUrl || this.sitemapInput || 'site';
      this.isExportingWPApply = true;
      this.addLog('info', `🧩 Генерація WP-CLI WebP apply PHP (path=${wordpressPath})...`);
      try {
        if (window.go?.main?.App?.ExportWordPressWebPApplyPHP) {
          const savedPath = await window.go.main.App.ExportWordPressWebPApplyPHP(
            domain, this.config, this.scanResults, wordpressPath
          );
          this.addLog('success', `🟢 WP apply PHP збережено: ${savedPath}`);
          this.addLog('info', `На оточенні: wp eval-file <file>.php --path=${wordpressPath}`);
          this.showToast('success', 'WP apply PHP готовий', savedPath);
        } else {
          throw new Error('ExportWordPressWebPApplyPHP method not available');
        }
      } catch (err) {
        console.error('Failed to export WP apply PHP:', err);
        this.addLog('error', `Помилка WP apply PHP: ${err.message}`);
        this.showToast('error', 'Помилка WP apply PHP', err.message);
      } finally {
        this.isExportingWPApply = false;
      }
    },

    async unlockAudio() {
      if (this.config.soundEnabled === false) return false;
      try {
        const AC = window.AudioContext || window.webkitAudioContext;
        if (AC) {
          if (!this.audioCtx) this.audioCtx = new AC();
          if (this.audioCtx.state === 'suspended') {
            await this.audioCtx.resume();
          }
        }

        if (!this.pikaPageEl) {
          this.pikaPageEl = new Audio('/pika-page.mp3');
          this.pikaPageEl.preload = 'auto';
        }
        if (!this.pikaFullEl) {
          this.pikaFullEl = new Audio('/pika-full.mp3');
          this.pikaFullEl.preload = 'auto';
        }

        // Warm HTMLAudioElement during user gesture so later event-driven play() works in WebKit
        if (!this.audioUnlocked) {
          const warm = this.pikaPageEl;
          const prevVol = warm.volume;
          warm.volume = 0.001;
          try {
            await warm.play();
            warm.pause();
            warm.currentTime = 0;
          } catch (err) {
            console.warn("[JS LOG] Audio warm-up blocked:", err);
          }
          warm.volume = prevVol || 0.85;
          this.audioUnlocked = true;
          console.log("[JS LOG] Audio unlocked for scan notifications");
        }
        return true;
      } catch (err) {
        console.warn("[JS LOG] unlockAudio failed:", err);
        return false;
      }
    },

    async playPikaPageSound() {
      if (this.config.soundEnabled === false) return;
      // Prefer OS player: WebKitGTK often cannot decode MP3 (falls back to synth beep)
      try {
        if (window.go?.main?.App?.PlayNotificationSound) {
          await window.go.main.App.PlayNotificationSound('page');
          this.addLog('info', '⚡ Pikachu (page) через системний плеєр');
          return;
        }
      } catch (err) {
        console.warn("[JS LOG] native page sound failed:", err);
      }
      try {
        await this.unlockAudio();
        const el = this.pikaPageEl || new Audio('/pika-page.mp3');
        this.pikaPageEl = el;
        el.pause();
        el.currentTime = 0;
        el.volume = 0.85;
        await el.play();
        this.addLog('info', '⚡ Звук завершення (page) відтворено');
      } catch (err) {
        console.warn("[JS LOG] pika-page play failed:", err);
        await this.playSynthPikaPi();
      }
    },

    async playPikaFullSound() {
      if (this.config.soundEnabled === false) return;
      try {
        if (window.go?.main?.App?.PlayNotificationSound) {
          await window.go.main.App.PlayNotificationSound('full');
          this.addLog('info', '⚡ Pikachu (full) через системний плеєр');
          return;
        }
      } catch (err) {
        console.warn("[JS LOG] native full sound failed:", err);
      }
      try {
        await this.unlockAudio();
        const el = this.pikaFullEl || new Audio('/pika-full.mp3');
        this.pikaFullEl = el;
        el.pause();
        el.currentTime = 0;
        el.volume = 0.90;
        await el.play();
        this.addLog('info', '⚡ Звук завершення (full) відтворено');
      } catch (err) {
        console.warn("[JS LOG] pika-full play failed:", err);
        await this.playSynthPikaPi();
      }
    },

    async playSynthPikaPi() {
      try {
        const AudioContext = window.AudioContext || window.webkitAudioContext;
        if (!AudioContext) return;
        if (!this.audioCtx) this.audioCtx = new AudioContext();
        const ctx = this.audioCtx;
        if (ctx.state === 'suspended') {
          await ctx.resume();
        }

        const osc1 = ctx.createOscillator();
        const gain1 = ctx.createGain();
        osc1.type = 'sine';
        osc1.frequency.setValueAtTime(1046.50, ctx.currentTime);
        osc1.frequency.exponentialRampToValueAtTime(1318.51, ctx.currentTime + 0.12);

        gain1.gain.setValueAtTime(0.35, ctx.currentTime);
        gain1.gain.exponentialRampToValueAtTime(0.01, ctx.currentTime + 0.14);

        osc1.connect(gain1);
        gain1.connect(ctx.destination);

        osc1.start(ctx.currentTime);
        osc1.stop(ctx.currentTime + 0.15);

        const osc2 = ctx.createOscillator();
        const gain2 = ctx.createGain();
        osc2.type = 'triangle';
        osc2.frequency.setValueAtTime(1567.98, ctx.currentTime + 0.15);
        osc2.frequency.exponentialRampToValueAtTime(1879.47, ctx.currentTime + 0.28);

        gain2.gain.setValueAtTime(0.0, ctx.currentTime + 0.14);
        gain2.gain.setValueAtTime(0.4, ctx.currentTime + 0.15);
        gain2.gain.exponentialRampToValueAtTime(0.01, ctx.currentTime + 0.38);

        osc2.connect(gain2);
        gain2.connect(ctx.destination);

        osc2.start(ctx.currentTime + 0.15);
        osc2.stop(ctx.currentTime + 0.40);
        this.addLog('info', '⚡ Synth звук відтворено (fallback)');
      } catch (err) {
        console.warn("Could not play synth sound:", err);
        this.addLog('warning', `Звук не відтворено: ${err.message || err}`);
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