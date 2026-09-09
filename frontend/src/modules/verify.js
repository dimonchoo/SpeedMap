/**
 * verify.js - Manifest QA & Staging Verification Module
 * Verifies that all exported WebP assets return 200 OK on target site (e.g. UAT)
 * and confirms that page HTML has been updated to use WebP instead of old rasters.
 */
export function createVerifyModule() {
  return {
    verifyState: {
      packagePath: '/Users/dmytrobuhaiov/Downloads/speedmap-webp-20260904-135055',
      targetURL: 'https://uat.infuse.com',
      checkPages: true,
      concurrency: 8,
      isRunning: false,
      progress: {
        phase: '',
        current: 0,
        total: 0,
        currentItem: '',
        passedCount: 0,
        warnCount: 0,
        failCount: 0
      },
      result: null,
      filter: 'all', // 'all', 'fail', 'warn', 'pass'
      searchQuery: '',
      currentPage: 1,
      itemsPerPage: 25,
      error: null
    },

    initVerifyEvents() {
      if (window.runtime?.EventsOn) {
        window.runtime.EventsOn('manifest:verify_progress', (progress) => {
          if (this.verifyState) {
            this.verifyState.progress = progress;
          }
        });
      }
    },

    async selectManifestPackageForVerify() {
      try {
        if (window.go?.main?.App?.SelectManifestDialog) {
          const dir = await window.go.main.App.SelectManifestDialog();
          if (dir) {
            this.verifyState.packagePath = dir;
          }
        }
      } catch (e) {
        console.error('Error selecting manifest directory:', e);
      }
    },

    async runManifestVerification() {
      let pkg = (this.verifyState.packagePath || '').trim();
      if (!pkg && this.packageContext?.packageDir) {
        pkg = this.packageContext.packageDir;
        this.verifyState.packagePath = pkg;
      }
      if (!pkg) {
        this.addToast?.('warning', 'Вкажіть шлях до папки пакету експорту або manifest.json');
        return;
      }

      const target = (this.verifyState.targetURL || '').trim();
      if (!target) {
        this.addToast?.('warning', 'Вкажіть URL цільового сайту (наприклад, https://uat.infuse.com)');
        return;
      }

      this.verifyState.isRunning = true;
      this.verifyState.error = null;
      this.verifyState.progress = {
        phase: 'verifying',
        current: 0,
        total: 0,
        currentItem: 'Ініціалізація підключення...',
        passedCount: 0,
        warnCount: 0,
        failCount: 0
      };

      try {
        this.addToast?.('info', `Запуск звірки з маніфестом на ${target}...`);
        const resp = await window.go.main.App.VerifyManifest(
          pkg,
          target,
          this.verifyState.checkPages
        );
        this.verifyState.result = resp;
        this.verifyState.currentPage = 1;

        if (resp.summary?.allPassed) {
          this.addToast?.('success', `Звірка успішна! Всі ${resp.summary.totalImages} зображень валідні та віддаються як WebP.`);
        } else if (resp.summary?.failedImages > 0) {
          this.addToast?.('error', `Виявлено помилки: ${resp.summary.failedImages} зображень не пройшли перевірку.`);
        } else {
          this.addToast?.('warning', `Звірка завершена з попередженнями: ${resp.summary?.warnedImages || 0}.`);
        }
      } catch (err) {
        console.error('Manifest verification failed:', err);
        this.verifyState.error = err.message || String(err);
        this.addToast?.('error', `Помилка перевірки: ${this.verifyState.error}`);
      } finally {
        this.verifyState.isRunning = false;
      }
    },

    get filteredVerifyItems() {
      const res = this.verifyState?.result;
      if (!res || !res.items) return [];

      let list = res.items;
      const f = this.verifyState.filter;
      if (f === 'fail') {
        list = list.filter(it => it.status === 'fail');
      } else if (f === 'warn') {
        list = list.filter(it => it.status === 'warn');
      } else if (f === 'pass') {
        list = list.filter(it => it.status === 'pass');
      }

      const q = (this.verifyState.searchQuery || '').toLowerCase().trim();
      if (q) {
        list = list.filter(it =>
          (it.id && it.id.toLowerCase().includes(q)) ||
          (it.basename && it.basename.toLowerCase().includes(q)) ||
          (it.targetWebpUrl && it.targetWebpUrl.toLowerCase().includes(q)) ||
          (it.samplePageUrl && it.samplePageUrl.toLowerCase().includes(q))
        );
      }
      return list;
    },

    get paginatedVerifyItems() {
      const items = this.filteredVerifyItems;
      const page = this.verifyState.currentPage || 1;
      const perPage = this.verifyState.itemsPerPage || 25;
      const start = (page - 1) * perPage;
      return items.slice(start, start + perPage);
    },

    get totalVerifyPages() {
      const count = this.filteredVerifyItems.length;
      const perPage = this.verifyState.itemsPerPage || 25;
      return Math.ceil(count / perPage) || 1;
    }
  };
}
