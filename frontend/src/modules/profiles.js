export function createProfilesModule() {
  return {
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
        this.resolveCurrentDomain?.(profile.sitemapUrl);
      }
      if (profile.config) {
        this.config = { ...this.config, ...profile.config };
      }
      if (this.config.adaptiveQuality !== false) {
        this.config.adaptiveQuality = true;
      }
      if (this.config.resizeToRetina !== false) {
        this.config.resizeToRetina = true;
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
      this.saveConfig?.();
    },

    async selectProfile(profile) {
      if (!profile) return;
      this.applyProfile(profile);
      this.showProfileDropdown = false;
      this.addLog?.('info', `🌐 Переключено на сайт: ${profile.name} (${profile.sitemapUrl || 'Без sitemap'})`);
      this.showToast?.('info', 'Сайт змінено 🌐', profile.name);

      if (profile.sitemapUrl) {
        await this.parseSitemap?.();
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
        this.showToast?.('warning', 'Заповніть дані', 'Вкажіть назву сайту або URL sitemap');
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
        this.checkGDriveStatus?.();
        if (saved) {
          const fresh = this.siteProfiles.find(p => p.id === saved.id) || saved;
          this.applyProfile(fresh);
        }
        this.showToast?.('success', 'Профіль збережено 🟢', this.editingProfile.name);
        this.addLog?.('success', `Успішно збережено профіль сайту: ${this.editingProfile.name}`);
      } catch (err) {
        console.error("Failed to save profile:", err);
        this.showToast?.('error', 'Помилка збереження', err.message);
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
        
        this.showToast?.('info', 'Профіль видалено', name);
        this.addLog?.('info', `Видалено профіль сайту: ${name}`);
        await this.loadSiteProfiles();
        this.checkGDriveStatus?.();
      } catch (err) {
        console.error('Failed to delete profile:', err);
        this.showToast?.('error', 'Помилка видалення', err.message);
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
        const idx = this.siteProfiles.findIndex(p => p.id === updated.id);
        if (idx >= 0) {
          this.siteProfiles[idx] = updated;
        }
      } catch (err) {
        console.error("Failed to save current config to active profile:", err);
      }
    }
  };
}
