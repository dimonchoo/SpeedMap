import { downloadFileBlob } from '../utils/download.js';

export function createFormsModule() {
  return {
    formSearchQuery: '',
    formEngineFilter: 'all',
    formViewMode: 'by-page',
    selectedFormDetail: null,
    showFormModal: false,
    formPage: 1,
    formPerPage: 25,

    get pageFormsList() {
      if (!this.scanResults || this.scanResults.length === 0) return [];
      let list = this.scanResults.map(p => {
        const forms = p.diagnostics?.forms || [];
        return {
          id: p.id,
          url: p.url,
          formCount: forms.length,
          forms: forms,
          hasCaptcha: forms.some(f => f.captcha?.isActive),
          hasFileUpload: forms.some(f => f.hasFileUpload),
          engines: [...new Set(forms.map(f => f.engine))]
        };
      }).filter(item => item.formCount > 0);

      if (this.formSearchQuery && this.formSearchQuery.trim()) {
        const q = this.formSearchQuery.toLowerCase().trim();
        list = list.filter(item => {
          if (item.url && item.url.toLowerCase().includes(q)) return true;
          return item.forms.some(f =>
            (f.title && f.title.toLowerCase().includes(q)) ||
            (f.id && f.id.toLowerCase().includes(q)) ||
            (f.engine && f.engine.toLowerCase().includes(q)) ||
            (f.fields && f.fields.some(fld => (fld.name && fld.name.toLowerCase().includes(q)) || (fld.label && fld.label.toLowerCase().includes(q))))
          );
        });
      }

      if (this.formEngineFilter !== 'all') {
        if (this.formEngineFilter === 'has-captcha') {
          list = list.filter(item => item.forms.some(f => f.captcha?.isActive));
        } else if (this.formEngineFilter === 'no-captcha') {
          list = list.filter(item => item.forms.some(f => !f.captcha?.isActive));
        } else if (this.formEngineFilter === 'file-upload') {
          list = list.filter(item => item.forms.some(f => f.hasFileUpload));
        } else {
          const target = this.formEngineFilter.toLowerCase();
          list = list.filter(item => item.forms.some(f => f.engine && f.engine.toLowerCase().includes(target)));
        }
      }

      return list;
    },

    get filteredForms() {
      if (!this.siteAnalytics || !this.siteAnalytics.forms) return [];
      let list = [...this.siteAnalytics.forms];

      if (this.formSearchQuery && this.formSearchQuery.trim()) {
        const q = this.formSearchQuery.toLowerCase().trim();
        list = list.filter(f =>
          (f.title && f.title.toLowerCase().includes(q)) ||
          (f.id && f.id.toLowerCase().includes(q)) ||
          (f.engine && f.engine.toLowerCase().includes(q)) ||
          (f.pages && f.pages.some(p => p.toLowerCase().includes(q))) ||
          (f.fields && f.fields.some(fld => (fld.name && fld.name.toLowerCase().includes(q)) || (fld.label && fld.label.toLowerCase().includes(q))))
        );
      }

      if (this.formEngineFilter !== 'all') {
        if (this.formEngineFilter === 'has-captcha') {
          list = list.filter(f => f.captcha?.isActive);
        } else if (this.formEngineFilter === 'no-captcha') {
          list = list.filter(f => !f.captcha?.isActive);
        } else if (this.formEngineFilter === 'file-upload') {
          list = list.filter(f => f.hasFileUpload);
        } else {
          const target = this.formEngineFilter.toLowerCase();
          list = list.filter(f => f.engine && f.engine.toLowerCase().includes(target));
        }
      }
      list.sort((a, b) => (b.pageCount || 0) - (a.pageCount || 0));
      return list;
    },

    get totalFormPagesCount() {
      return Math.max(1, Math.ceil(this.pageFormsList.length / this.formPerPage));
    },

    get paginatedFormPages() {
      const start = (this.formPage - 1) * this.formPerPage;
      return this.pageFormsList.slice(start, start + this.formPerPage);
    },

    openFormDetailModal(form) {
      this.selectedFormDetail = form;
      this.showFormModal = true;
    },

    closeFormModal() {
      this.showFormModal = false;
      this.selectedFormDetail = null;
    },

    async exportFormsCSV() {
      if (!this.scanResults || this.scanResults.length === 0) {
        this.showToast?.('warning', 'Відсутні дані', 'Спочатку проскануйте сайт.');
        return;
      }
      let csv = "Page URL,Form ID,Form Title,Engine,Method,Action,Field Count,Fields List,File Upload,Allowed File Types,Captcha Type,Captcha Active,Hidden UTM Tokens\n";
      this.scanResults.forEach(p => {
        const pageUrl = `"${(p.url || '').replace(/"/g, '""')}"`;
        const forms = p.diagnostics?.forms || [];
        if (forms.length === 0) {
          csv += `${pageUrl},"No forms","none","none","none","none",0,"","false","","none","false",""\n`;
        } else {
          forms.forEach(f => {
            const formId = `"${(f.id || '').replace(/"/g, '""')}"`;
            const title = `"${(f.title || '').replace(/"/g, '""')}"`;
            const engine = `"${(f.engine || '').replace(/"/g, '""')}"`;
            const method = `"${(f.method || 'POST').replace(/"/g, '""')}"`;
            const action = `"${(f.action || '').replace(/"/g, '""')}"`;
            const count = f.fieldCount || (f.fields || []).length;
            const fieldsList = `"${(f.fields || []).map(fld => fld.name + (fld.isRequired ? '*' : '') + ':' + fld.type).join('; ').replace(/"/g, '""')}"`;
            const fileUpload = f.hasFileUpload ? '"Yes"' : '"No"';
            const allowedFiles = `"${(f.allowedFileTypes || '').replace(/"/g, '""')}"`;
            const captchaType = `"${(f.captcha?.type || 'none').replace(/"/g, '""')}"`;
            const captchaActive = f.captcha?.isActive ? '"Yes"' : '"No"';
            const hiddenTokens = `"${Object.entries(f.hiddenTokens || {}).map(([k, v]) => k + '=' + v).join('&').replace(/"/g, '""')}"`;
            csv += `${pageUrl},${formId},${title},${engine},${method},${action},${count},${fieldsList},${fileUpload},${allowedFiles},${captchaType},${captchaActive},${hiddenTokens}\n`;
          });
        }
      });

      try {
        if (window.go?.main?.App?.ExportFormsCSV) {
          const filePath = await window.go.main.App.ExportFormsCSV(csv);
          if (filePath) {
            this.showToast?.('success', 'Посторінковий CSV збережено 🟢', filePath, filePath);
            this.addLog?.('success', `Експортовано посторінковий CSV звіт форм: ${filePath}`);
            return;
          }
        }
      } catch (e) {
        console.warn("Native CSV save dialog fallback:", e);
      }

      downloadFileBlob(`speedmap_forms_report_${Date.now()}.csv`, csv, 'text/csv;charset=utf-8;');
      this.showToast?.('success', 'CSV завантажено 🟢', 'Збережено у папочку Завантаження (~/Downloads)');
    },

    async exportFormsJSON() {
      if (!this.scanResults || this.scanResults.length === 0) {
        this.showToast?.('warning', 'Відсутні дані', 'Спочатку проскануйте сайт.');
        return;
      }
      const pageFormsData = {
        summary: {
          totalUniqueForms: this.siteAnalytics?.totalFormCount || (this.siteAnalytics?.forms || []).length,
          pagesWithForms: this.pageFormsList.filter(p => p.formCount > 0).length,
          exportedAt: new Date().toISOString()
        },
        aggregatedForms: this.siteAnalytics?.forms || [],
        pages: this.scanResults.map(p => ({
          pageUrl: p.url,
          formCount: (p.diagnostics?.forms || []).length,
          forms: p.diagnostics?.forms || []
        }))
      };

      const jsonStr = JSON.stringify(pageFormsData, null, 2);

      try {
        if (window.go?.main?.App?.ExportFormsJSON) {
          const filePath = await window.go.main.App.ExportFormsJSON(jsonStr);
          if (filePath) {
            this.showToast?.('success', 'JSON звіт форм збережено 🟢', filePath, filePath);
            this.addLog?.('success', `Експортовано JSON звіт форм: ${filePath}`);
            return;
          }
        }
      } catch (e) {
        console.warn("Native JSON save dialog fallback:", e);
      }

      downloadFileBlob(`speedmap_forms_report_${Date.now()}.json`, jsonStr, 'application/json;charset=utf-8;');
      this.showToast?.('success', 'JSON завантажено 🟢', 'Збережено у папочку Завантаження (~/Downloads)');
    }
  };
}
