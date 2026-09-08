import { downloadFileBlob } from '../utils/download.js';

export function createFontsModule() {
  return {
    fontSearchQuery: '',
    fontFilterTab: 'all',
    fontSortKey: 'pages',
    fontViewMode: 'by-page',
    fontPage: 1,
    fontPerPage: 25,

    get pageFontsList() {
      if (!this.scanResults || this.scanResults.length === 0) return [];
      let list = this.scanResults.map(p => {
        const fonts = p.diagnostics?.fonts || [];
        return {
          id: p.id,
          url: p.url,
          fontCount: fonts.length,
          fonts: fonts
        };
      });

      if (this.fontSearchQuery && this.fontSearchQuery.trim()) {
        const q = this.fontSearchQuery.toLowerCase().trim();
        list = list.filter(item => {
          if (item.url && item.url.toLowerCase().includes(q)) return true;
          return item.fonts.some(f => (f.family && f.family.toLowerCase().includes(q)) || (f.type && f.type.toLowerCase().includes(q)));
        });
      }

      return list;
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

    get totalFontPagesCount() {
      return Math.max(1, Math.ceil(this.pageFontsList.length / this.fontPerPage));
    },

    get paginatedFontPages() {
      const start = (this.fontPage - 1) * this.fontPerPage;
      return this.pageFontsList.slice(start, start + this.fontPerPage);
    },

    copyFontUrl(url) {
      if (!url) return;
      navigator.clipboard.writeText(url);
      this.showToast?.('success', 'URL скопійовано 📋', url);
    },

    async exportFontsCSV() {
      if (!this.scanResults || this.scanResults.length === 0) {
        this.showToast?.('warning', 'Відсутні дані', 'Спочатку проскануйте сайт.');
        return;
      }
      let csv = "Page URL,Font Family,Format,Load Duration (ms),Direct Asset URL\n";
      this.scanResults.forEach(p => {
        const pageUrl = `"${(p.url || '').replace(/"/g, '""')}"`;
        const fonts = p.diagnostics?.fonts || [];
        if (fonts.length === 0) {
          csv += `${pageUrl},"System Font / None","none",0,""\n`;
        } else {
          fonts.forEach(f => {
            const family = `"${(f.family || '').replace(/"/g, '""')}"`;
            const type = `"${(f.type || '').replace(/"/g, '""')}"`;
            const fontUrl = `"${(f.url || '').replace(/"/g, '""')}"`;
            csv += `${pageUrl},${family},${type},${f.duration || 0},${fontUrl}\n`;
          });
        }
      });

      try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.ExportFontsCSV) {
          const filePath = await window.go.main.App.ExportFontsCSV(csv);
          if (filePath) {
            this.showToast?.('success', 'Посторінковий CSV збережено 🟢', filePath, filePath);
            this.addLog?.('success', `Експортовано посторінковий CSV звіт шрифтів: ${filePath}`);
            return;
          }
        }
      } catch (e) {
        console.warn("Native CSV save dialog fallback:", e);
      }

      downloadFileBlob(`speedmap_page_fonts_report_${Date.now()}.csv`, csv, 'text/csv;charset=utf-8;');
      this.showToast?.('success', 'CSV завантажено 🟢', 'Збережено у папочку Завантаження (~/Downloads)');
    },

    async exportFontsJSON() {
      if (!this.scanResults || this.scanResults.length === 0) {
        this.showToast?.('warning', 'Відсутні дані', 'Спочатку проскануйте сайт.');
        return;
      }
      const pageFontsData = this.scanResults.map(p => ({
        pageUrl: p.url,
        fontCount: (p.diagnostics?.fonts || []).length,
        fonts: p.diagnostics?.fonts || []
      }));

      const jsonStr = JSON.stringify(pageFontsData, null, 2);

      try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.ExportFontsJSON) {
          const filePath = await window.go.main.App.ExportFontsJSON(jsonStr);
          if (filePath) {
            this.showToast?.('success', 'Посторінковий JSON збережено 🟢', filePath, filePath);
            this.addLog?.('success', `Експортовано посторінковий JSON звіт шрифтів: ${filePath}`);
            return;
          }
        }
      } catch (e) {
        console.warn("Native JSON save dialog fallback:", e);
      }

      downloadFileBlob(`speedmap_page_fonts_report_${Date.now()}.json`, jsonStr, 'application/json;charset=utf-8;');
      this.showToast?.('success', 'JSON завантажено 🟢', 'Збережено у папочку Завантаження (~/Downloads)');
    }
  };
}
