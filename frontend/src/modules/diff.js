export function createDiffModule() {
  return {
    historyRunsList: [],
    selectedBaseRunId: '',
    selectedCurrentRunId: '',
    runsDiffResult: null,
    exportRecordsList: [],
    selectedBaseExportPath: '',
    selectedCurrentExportPath: '',
    exportDiffReport: null,
    diffViewMode: 'packages',
    runsDiffFilter: 'all',
    runsDiffSearch: '',
    isLoadingDiff: false,
    diffCurrentPage: 1,
    diffPageSize: 50,
    diffClientCache: {},

    get filteredDiffFiles() {
      if (!this.runsDiffResult || !this.runsDiffResult.files) return [];
      const q = (this.runsDiffSearch || '').toLowerCase().trim();
      const f = this.runsDiffFilter;
      const files = this.runsDiffResult.files;
      if (!q && f === 'all') return files;
      return files.filter(item => {
        if (f === 'degraded' && item.status !== 'degraded') return false;
        if (f === 'improved' && item.status !== 'improved') return false;
        if (f === 'same' && item.status !== 'same') return false;
        if (f === 'new' && item.status !== 'new' && item.status !== 'removed') return false;
        if (!q) return true;
        return item._search ? item._search.includes(q) : ((item.basename || '') + ' ' + (item.url || '')).toLowerCase().includes(q);
      });
    },

    get filteredExportDiffFiles() {
      if (!this.exportDiffReport || !this.exportDiffReport.files) return [];
      const q = (this.runsDiffSearch || '').toLowerCase().trim();
      const f = this.runsDiffFilter;
      const files = this.exportDiffReport.files;
      if (!q && f === 'all') return files;
      return files.filter(item => {
        if (f === 'degraded' && item.status !== 'degraded') return false;
        if (f === 'improved' && item.status !== 'improved') return false;
        if (f === 'same' && item.status !== 'same') return false;
        if (f === 'new' && item.status !== 'new' && item.status !== 'removed') return false;
        if (!q) return true;
        return item._search ? item._search.includes(q) : ((item.basename || '') + ' ' + (item.sourceUrl || '')).toLowerCase().includes(q);
      });
    },

    get activeDiffFiles() {
      return this.diffViewMode === 'packages' ? this.filteredExportDiffFiles : this.filteredDiffFiles;
    },

    get diffTotalItems() {
      return this.activeDiffFiles.length;
    },

    get diffTotalPages() {
      return Math.max(1, Math.ceil(this.diffTotalItems / this.diffPageSize));
    },

    get paginatedDiffFiles() {
      const start = (this.diffCurrentPage - 1) * this.diffPageSize;
      return this.activeDiffFiles.slice(start, start + this.diffPageSize);
    },

    nextDiffPage() {
      if (this.diffCurrentPage < this.diffTotalPages) {
        this.diffCurrentPage++;
      }
    },

    prevDiffPage() {
      if (this.diffCurrentPage > 1) {
        this.diffCurrentPage--;
      }
    },

    setDiffPage(page) {
      if (page >= 1 && page <= this.diffTotalPages) {
        this.diffCurrentPage = page;
      }
    },

    async loadHistoryRunsForDiff() {
      try {
        const domain = this.config.sitemapUrl || this.sitemapInput || '';
        if (window.go?.main?.App?.GetAllHistoryRuns) {
          const list = await window.go.main.App.GetAllHistoryRuns(domain);
          this.historyRunsList = list || [];
          if (this.historyRunsList.length >= 2 && !this.selectedBaseRunId) {
            this.selectedCurrentRunId = this.historyRunsList[0].id;
            this.selectedBaseRunId = this.historyRunsList[1].id;
            await this.runComparisonDiff();
          } else if (this.historyRunsList.length === 1 && !this.selectedCurrentRunId) {
            this.selectedCurrentRunId = this.historyRunsList[0].id;
            this.selectedBaseRunId = this.historyRunsList[0].id;
          }
        }
      } catch (e) {
        console.error('Failed to load history runs for diff:', e);
      }
    },

    async runComparisonDiff() {
      if (!this.selectedBaseRunId || !this.selectedCurrentRunId) return;
      const cacheKey = `runs:${this.selectedBaseRunId}:${this.selectedCurrentRunId}`;
      if (this.diffClientCache[cacheKey]) {
        this.runsDiffResult = this.diffClientCache[cacheKey];
        this.diffCurrentPage = 1;
        return;
      }
      this.isLoadingDiff = true;
      try {
        if (window.go?.main?.App?.CompareHistoryRuns) {
          const res = await window.go.main.App.CompareHistoryRuns(this.selectedBaseRunId, this.selectedCurrentRunId);
          if (res?.files) {
            for (let i = 0; i < res.files.length; i++) {
              const f = res.files[i];
              f._search = ((f.basename || '') + ' ' + (f.url || '')).toLowerCase();
            }
          }
          this.diffClientCache[cacheKey] = res;
          this.runsDiffResult = res;
          this.diffCurrentPage = 1;
          this.addLog?.('info', `Порівняння прогонів: ${res.totalFiles} файлів (🔴 ${res.degradedCount} погіршилися, 🟢 ${res.improvedCount} покращилися)`);
        }
      } catch (e) {
        console.error('Failed to compare runs:', e);
        this.showToast?.('error', 'Помилка порівняння', e.message);
      } finally {
        this.isLoadingDiff = false;
      }
    },

    async loadExportHistoryForDiff() {
      try {
        const domain = this.config.sitemapUrl || this.sitemapInput || '';
        if (window.go?.main?.App?.GetExportHistory) {
          const list = await window.go.main.App.GetExportHistory(domain);
          this.exportRecordsList = list || [];
          if (this.exportRecordsList.length >= 2) {
            const hasCurr = this.exportRecordsList.some(r => r.manifestPath === this.selectedCurrentExportPath);
            const hasBase = this.exportRecordsList.some(r => r.manifestPath === this.selectedBaseExportPath);
            if (!this.selectedCurrentExportPath || !hasCurr) {
              this.selectedCurrentExportPath = this.exportRecordsList[0].manifestPath;
            }
            if (!this.selectedBaseExportPath || !hasBase) {
              this.selectedBaseExportPath = this.exportRecordsList[1].manifestPath;
            }
            await this.runExportPackageDiff();
          } else if (this.exportRecordsList.length === 1) {
            this.selectedCurrentExportPath = this.exportRecordsList[0].manifestPath;
            this.selectedBaseExportPath = this.exportRecordsList[0].manifestPath;
            await this.runExportPackageDiff();
          }
        }
      } catch (e) {
        console.error('Failed to load export history:', e);
      }
    },

    async browseBaseExportFolder() {
      try {
        if (window.go?.main?.App?.SelectDirectory) {
          const dir = await window.go.main.App.SelectDirectory("Виберіть базову папку експорту (speedmap-webp-*)");
          if (dir) {
            const manifestPath = dir.endsWith('manifest.json') ? dir : (dir.replace(/\/+$/, '') + '/manifest.json');
            this.selectedBaseExportPath = manifestPath;
            await this.runExportPackageDiff();
          }
        }
      } catch (e) {
        console.error('Error selecting base folder:', e);
      }
    },

    async browseCurrentExportFolder() {
      try {
        if (window.go?.main?.App?.SelectDirectory) {
          const dir = await window.go.main.App.SelectDirectory("Виберіть поточну папку експорту (speedmap-webp-*)");
          if (dir) {
            const manifestPath = dir.endsWith('manifest.json') ? dir : (dir.replace(/\/+$/, '') + '/manifest.json');
            this.selectedCurrentExportPath = manifestPath;
            await this.runExportPackageDiff();
          }
        }
      } catch (e) {
        console.error('Error selecting current folder:', e);
      }
    },

    async runExportPackageDiff() {
      if (!this.selectedBaseExportPath || !this.selectedCurrentExportPath) return;
      const cacheKey = `pkg:${this.selectedBaseExportPath}:${this.selectedCurrentExportPath}`;
      if (this.diffClientCache[cacheKey]) {
        this.exportDiffReport = this.diffClientCache[cacheKey];
        this.diffCurrentPage = 1;
        return;
      }
      this.isLoadingDiff = true;
      try {
        if (window.go?.main?.App?.CompareExportPackages) {
          const res = await window.go.main.App.CompareExportPackages(this.selectedBaseExportPath, this.selectedCurrentExportPath);
          if (res?.files) {
            for (let i = 0; i < res.files.length; i++) {
              const f = res.files[i];
              f._search = ((f.basename || '') + ' ' + (f.sourceUrl || '')).toLowerCase();
            }
          }
          this.diffClientCache[cacheKey] = res;
          this.exportDiffReport = res;
          this.diffCurrentPage = 1;
          this.addLog?.('info', `Порівняння пакетів експорту: ${res.totalFiles} файлів (🔴 ${res.degradedCount} погіршилися, 🟢 ${res.improvedCount} покращилися)`);
        }
      } catch (e) {
        console.error('Failed to compare export packages:', e);
        this.showToast?.('error', 'Помилка порівняння пакетів', e.message);
      } finally {
        this.isLoadingDiff = false;
      }
    }
  };
}
