import { downloadFileBlob } from '../utils/download.js';

export function createIframesModule() {
  return {
    iframeSearchQuery: '',
    iframeViewMode: 'by-page',
    iframePage: 1,
    iframePerPage: 25,

    get pageIframesList() {
      if (!this.scanResults || this.scanResults.length === 0) return [];
      let list = this.scanResults.map(p => {
        const iframes = p.diagnostics?.iframes || [];
        return {
          id: p.id,
          url: p.url,
          iframeCount: iframes.length,
          loadedCount: iframes.filter(f => f.loadedDuringScan).length,
          missedCount: iframes.filter(f => !f.loadedDuringScan).length,
          iframes
        };
      }).filter(item => item.iframeCount > 0);

      if (this.iframeSearchQuery && this.iframeSearchQuery.trim()) {
        const q = this.iframeSearchQuery.toLowerCase().trim();
        list = list.filter(item => {
          if (item.url && item.url.toLowerCase().includes(q)) return true;
          return item.iframes.some(f =>
            (f.src && f.src.toLowerCase().includes(q)) ||
            (f.title && f.title.toLowerCase().includes(q))
          );
        });
      }

      return list;
    },

    get filteredIframes() {
      if (!this.siteAnalytics || !this.siteAnalytics.iframes) return [];
      let list = [...this.siteAnalytics.iframes];

      if (this.iframeSearchQuery && this.iframeSearchQuery.trim()) {
        const q = this.iframeSearchQuery.toLowerCase().trim();
        list = list.filter(f =>
          (f.src && f.src.toLowerCase().includes(q)) ||
          (f.title && f.title.toLowerCase().includes(q)) ||
          (f.pages && f.pages.some(p => p.toLowerCase().includes(q)))
        );
      }

      list.sort((a, b) => {
        if ((b.missedCount || 0) !== (a.missedCount || 0)) return (b.missedCount || 0) - (a.missedCount || 0);
        return (b.pageCount || 0) - (a.pageCount || 0);
      });

      return list;
    },

    get totalIframePagesCount() {
      return Math.max(1, Math.ceil(this.pageIframesList.length / this.iframePerPage));
    },

    get paginatedIframePages() {
      const start = (this.iframePage - 1) * this.iframePerPage;
      return this.pageIframesList.slice(start, start + this.iframePerPage);
    },

    copyIframeSrc(src) {
      if (!src) return;
      navigator.clipboard.writeText(src).then(() => {
        this.showToast?.('success', 'URL iframe скопійовано 📋', src);
      }).catch(err => {
        console.error('Failed to copy iframe src:', err);
      });
    },

    async exportIframesCSV() {
      if (!this.scanResults || this.scanResults.length === 0) {
        this.showToast?.('warning', 'Відсутні дані', 'Спочатку проскануйте сайт.');
        return;
      }
      let csv = "Page URL,Iframe SRC,Title,Status,Lazy Loading,In Viewport,Sandbox,Load Duration (ms),Transfer Size,Dimensions\n";
      this.scanResults.forEach(p => {
        const pageUrl = `"${(p.url || '').replace(/"/g, '""')}"`;
        const iframes = p.diagnostics?.iframes || [];
        if (iframes.length === 0) {
          csv += `${pageUrl},"No iframes","none","none","false","false","false",0,"0 B","0x0"\n`;
        } else {
          iframes.forEach(f => {
            const src = `"${(f.src || '').replace(/"/g, '""')}"`;
            const title = `"${(f.title || '').replace(/"/g, '""')}"`;
            const status = f.loadedDuringScan ? '"Loaded"' : '"Missed (Lazy/Deferred)"';
            const lazy = f.isLazy ? '"Yes"' : '"No"';
            const inViewport = f.inViewport ? '"Yes"' : '"No"';
            const sandbox = f.sandbox ? '"Yes"' : '"No"';
            const duration = f.duration || 0;
            const size = `"${f.formattedSize || (f.transferSize ? (f.transferSize + ' B') : '0 B')}"`;
            const dimensions = `"${(f.width || 0)}x${(f.height || 0)}"`;
            csv += `${pageUrl},${src},${title},${status},${lazy},${inViewport},${sandbox},${duration},${size},${dimensions}\n`;
          });
        }
      });

      try {
        if (window.go?.main?.App?.ExportIframesCSV) {
          const filePath = await window.go.main.App.ExportIframesCSV(csv);
          if (filePath) {
            this.showToast?.('success', 'Посторінковий CSV збережено 🟢', filePath, filePath);
            this.addLog?.('success', `Експортовано посторінковий CSV звіт iframe: ${filePath}`);
            return;
          }
        }
      } catch (e) {
        console.warn("Native CSV save dialog fallback:", e);
      }

      downloadFileBlob(`speedmap_iframes_report_${Date.now()}.csv`, csv, 'text/csv;charset=utf-8;');
      this.showToast?.('success', 'CSV завантажено 🟢', 'Збережено у папочку Завантаження (~/Downloads)');
    },

    async exportIframesJSON() {
      if (!this.scanResults || this.scanResults.length === 0) {
        this.showToast?.('warning', 'Відсутні дані', 'Спочатку проскануйте сайт.');
        return;
      }
      const pageIframesData = {
        summary: {
          totalUniqueIframes: this.siteAnalytics?.totalIframeCount || (this.siteAnalytics?.iframes || []).length,
          loadedCount: this.siteAnalytics?.loadedIframeCount || 0,
          missedCount: this.siteAnalytics?.missedIframeCount || 0,
          pagesWithIframes: this.pageIframesList.filter(p => p.iframeCount > 0).length,
          exportedAt: new Date().toISOString()
        },
        aggregatedIframes: this.siteAnalytics?.iframes || [],
        pages: this.scanResults.map(p => ({
          pageUrl: p.url,
          iframeCount: (p.diagnostics?.iframes || []).length,
          loadedCount: (p.diagnostics?.iframes || []).filter(f => f.loadedDuringScan).length,
          missedCount: (p.diagnostics?.iframes || []).filter(f => !f.loadedDuringScan).length,
          iframes: p.diagnostics?.iframes || []
        }))
      };

      const jsonStr = JSON.stringify(pageIframesData, null, 2);

      try {
        if (window.go?.main?.App?.ExportIframesJSON) {
          const filePath = await window.go.main.App.ExportIframesJSON(jsonStr);
          if (filePath) {
            this.showToast?.('success', 'JSON звіт iframe збережено 🟢', filePath, filePath);
            this.addLog?.('success', `Експортовано JSON звіт iframe: ${filePath}`);
            return;
          }
        }
      } catch (e) {
        console.warn("Native JSON save dialog fallback:", e);
      }

      downloadFileBlob(`speedmap_iframes_report_${Date.now()}.json`, jsonStr, 'application/json;charset=utf-8;');
      this.showToast?.('success', 'JSON завантажено 🟢', 'Збережено у папочку Завантаження (~/Downloads)');
    }
  };
}
