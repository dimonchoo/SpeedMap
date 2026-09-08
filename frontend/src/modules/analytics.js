import { downloadFileBlob } from '../utils/download.js';

export function createAnalyticsModule() {
  return {
    siteAnalytics: null,
    runComparison: null,
    isComputingAnalytics: false,
    analyticsDirty: true,
    domVirtTab: 'css',
    isPreviewingReport: false,
    isExportingReport: false,

    async updateAnalytics(force = false) {
      if (this.scanResults.length === 0) return;
      if (!force && !this.analyticsDirty && this.siteAnalytics) return;
      this.isComputingAnalytics = true;
      try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.ComputeSiteAnalytics) {
          const res = await window.go.main.App.ComputeSiteAnalytics(this.config.sitemapUrl || this.sitemapInput, this.config, this.scanResults);
          if (res) {
            this.siteAnalytics = res.analytics;
            this.runComparison = res.comparison;
            this.updateFilteredImages?.();
            this.analyticsDirty = false;
            this.addLog?.('info', `Оновлено загальну статистику сайту: Health Score = ${this.siteAnalytics.healthScore}%`);
          }
        }
      } catch (err) {
        console.error("Failed to compute analytics:", err);
      } finally {
        this.isComputingAnalytics = false;
      }
    },

    async openImageComparisonHTML() {
      if (!this.scanResults || this.scanResults.length === 0) {
        this.showToast?.('warning', 'Відсутні дані', 'Спочатку виконайте сканування сторінок.');
        return;
      }
      this.isPreviewingReport = true;
      this.addLog?.('info', '🌐 Запуск локального веб-сервера для перегляду звіту (SEOAEO-235)...');
      try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.PreviewImageComparisonHTML) {
          const domain = this.config.sitemapUrl || this.sitemapInput || 'site';
          const reportUrl = await window.go.main.App.PreviewImageComparisonHTML(domain, this.config, this.scanResults);
          this.addLog?.('success', `🟢 Звіт відкрито у браузері: ${reportUrl}`);
          this.showToast?.('success', 'Звіт відкрито 🟢', `Локальний веб-сервер запущено: ${reportUrl}`);
        } else {
          throw new Error('PreviewImageComparisonHTML method not available');
        }
      } catch (err) {
        console.error('Failed to preview HTML report:', err);
        this.addLog?.('error', `Помилка відкриття звіту: ${err.message}`);
        this.showToast?.('error', 'Помилка перегляду', err.message);
      } finally {
        this.isPreviewingReport = false;
      }
    },

    async exportImageReport() {
      if (!this.scanResults || this.scanResults.length === 0) {
        this.showToast?.('warning', 'Відсутні дані', 'Спочатку виконайте сканування сторінок.');
        return;
      }
      this.isExportingReport = true;
      this.addLog?.('info', '📄 Генерація HTML звіту порівняння зображень (SEOAEO-235)...');
      try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.ExportImageComparisonHTML) {
          const domain = this.config.sitemapUrl || this.sitemapInput || 'site';
          const savedPath = await window.go.main.App.ExportImageComparisonHTML(domain, this.config, this.scanResults);
          this.addLog?.('success', `🟢 Звіт успішно збережено у файл: ${savedPath}`);
          this.showToast?.('success', 'Звіт збережено 🟢', `Файл порівняння зображень створено: ${savedPath}`, savedPath);
        } else {
          throw new Error('ExportImageComparisonHTML method not available');
        }
      } catch (err) {
        console.error('Failed to export HTML report:', err);
        this.addLog?.('error', `Помилка збереження звіту: ${err.message}`);
        this.showToast?.('error', 'Помилка експорту', err.message);
      } finally {
        this.isExportingReport = false;
      }
    },

    async exportDOMVirtualizationCSS(cssContent, filename) {
      if (!cssContent) {
        this.showToast?.('warning', 'Немає даних', 'Селектори для оптимізації не знайдено.');
        return;
      }
      try {
        if (window.go?.main?.App?.ExportDOMVirtualizationCSS) {
          const filePath = await window.go.main.App.ExportDOMVirtualizationCSS(cssContent, filename || 'speedmap-dom-virtualization.css');
          if (filePath) {
            this.showToast?.('success', 'CSS збережено 🟢', filePath, filePath);
            this.addLog?.('success', `Експортовано CSS віртуалізації DOM: ${filePath}`);
            return;
          }
        }
      } catch (e) {
        console.warn("Native CSS save fallback:", e);
      }
      downloadFileBlob(filename || 'speedmap-dom-virtualization.css', cssContent, 'text/css;charset=utf-8;');
      this.showToast?.('success', 'CSS завантажено 🟢', 'Збережено на диск');
    },

    async exportDOMVirtualizationPHP(phpContent, filename) {
      if (!phpContent) {
        this.showToast?.('warning', 'Немає даних', 'Селектори для оптимізації не знайдено.');
        return;
      }
      try {
        if (window.go?.main?.App?.ExportDOMVirtualizationPHP) {
          const filePath = await window.go.main.App.ExportDOMVirtualizationPHP(phpContent, filename || 'speedmap-dom-virtualization.php');
          if (filePath) {
            this.showToast?.('success', 'PHP хук збережено 🟢', filePath, filePath);
            this.addLog?.('success', `Експортовано PHP хук (wp_head): ${filePath}`);
            return;
          }
        }
      } catch (e) {
        console.warn("Native PHP save fallback:", e);
      }
      downloadFileBlob(filename || 'speedmap-dom-virtualization.php', phpContent, 'application/x-php;charset=utf-8;');
      this.showToast?.('success', 'PHP завантажено 🟢', 'Збережено на диск');
    },

    async exportPagesCSV() {
      if (!this.scanResults || this.scanResults.length === 0) {
        this.showToast?.('warning', 'Відсутні дані', 'Спочатку проскануйте сайт.');
        return;
      }
      let csv = "URL,Title,Status Code,Response Time (ms),DOM Size (KB),Grade,LCP (s),CLS,FID (ms)\n";
      this.scanResults.forEach(p => {
        const url = `"${(p.url || '').replace(/"/g, '""')}"`;
        const title = `"${(p.title || '').replace(/"/g, '""')}"`;
        const status = p.statusCode || 0;
        const respTime = p.responseTimeMs || 0;
        const domSize = p.contentLength ? (p.contentLength / 1024).toFixed(1) : '0';
        const grade = p.webVitals?.grade || 'N/A';
        const lcp = p.webVitals?.lcp ? (p.webVitals.lcp / 1000).toFixed(2) : '0';
        const cls = p.webVitals?.cls ? p.webVitals.cls.toFixed(3) : '0';
        const fid = p.webVitals?.fid ? p.webVitals.fid.toFixed(0) : '0';
        csv += `${url},${title},${status},${respTime},${domSize},${grade},${lcp},${cls},${fid}\n`;
      });

      downloadFileBlob(`speedmap_pages_report_${Date.now()}.csv`, csv, 'text/csv;charset=utf-8;');
      this.showToast?.('success', 'CSV сторінок завантажено 🟢', 'Збережено у Завантаження');
    }
  };
}
