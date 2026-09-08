export function createCloudModule() {
  return {
    gdriveStatus: { connected: false, email: '' },
    isConnectingGDrive: false,
    isUploadingGDrive: false,
    gdriveClientID: '',
    gdriveClientSecret: '',

    async checkGDriveStatus() {
      try {
        if (window.go?.main?.App?.GetGDriveStatus) {
          const res = await window.go.main.App.GetGDriveStatus();
          if (res) {
            this.gdriveStatus = res;
          }
        }
        if (window.go?.main?.App?.GetGDriveCredentials) {
          const creds = await window.go.main.App.GetGDriveCredentials();
          if (creds) {
            if (creds.clientID) this.gdriveClientID = creds.clientID;
            if (creds.clientSecret) this.gdriveClientSecret = creds.clientSecret;
          }
        }
      } catch (e) {
        console.error("GDrive status check error:", e);
      }
    },

    async connectGDrive() {
      if (!this.gdriveClientID.trim() || !this.gdriveClientSecret.trim()) {
        this.showToast?.('warning', 'Введіть ключі', 'Введіть Google Client ID та Client Secret.');
        return;
      }
      this.isConnectingGDrive = true;
      this.showToast?.('info', 'Google Drive', 'Відкриваємо браузер для авторизації Google...');
      try {
        if (window.go?.main?.App?.SaveGDriveCredentials) {
          await window.go.main.App.SaveGDriveCredentials(
            this.gdriveClientID.trim(),
            this.gdriveClientSecret.trim()
          );
        }
        if (window.go?.main?.App?.StartGDriveAuth) {
          const email = await window.go.main.App.StartGDriveAuth(
            this.gdriveClientID.trim(),
            this.gdriveClientSecret.trim()
          );
          if (email) {
            this.gdriveStatus = { connected: true, email: email };
            this.showToast?.('success', 'Успішно підключено 🟢', email);
            this.addLog?.('success', `Google Drive авторизовано: ${email}`);
          }
        }
      } catch (err) {
        console.error("GDrive Auth Error:", err);
        this.showToast?.('error', 'Помилка авторизації', err.message || err);
      } finally {
        this.isConnectingGDrive = false;
      }
    },

    async disconnectGDrive() {
      try {
        if (window.go?.main?.App?.DisconnectGDrive) {
          await window.go.main.App.DisconnectGDrive();
          this.gdriveStatus = { connected: false, email: '' };
          this.showToast?.('info', 'Відключено', 'Google Drive відключено.');
        }
      } catch (e) {
        console.error("GDrive disconnect error:", e);
      }
    },

    async uploadFontsToGDrive() {
      if (!this.siteAnalytics || !this.siteAnalytics.fontUsage || this.siteAnalytics.fontUsage.length === 0) {
        this.showToast?.('warning', 'Відсутні дані', 'Спочатку виконайте сканування сторінок.');
        return;
      }
      if (!this.gdriveStatus.connected) {
        this.showToast?.('warning', 'Не підключено', 'Спочатку підключіть Google Drive у Налаштуваннях.');
        this.settingsTab = 'cloud';
        this.showSettings = true;
        return;
      }

      this.isUploadingGDrive = true;
      this.showToast?.('info', 'Вивантаження ☁️', 'Надсилаємо звіт у Google Drive...');
      try {
        let csv = "Font Family,Format,Occurrences,Page Coverage %,Avg Load Duration (ms),Formatted Size,Direct Asset URL,Page URLs List\n";
        this.siteAnalytics.fontUsage.forEach(f => {
          const family = `"${(f.family || '').replace(/"/g, '""')}"`;
          const type = `"${(f.type || '').replace(/"/g, '""')}"`;
          const fontUrl = `"${(f.url || '').replace(/"/g, '""')}"`;
          const pageUrlsList = `"${(f.pageUrls || []).join('; ').replace(/"/g, '""')}"`;
          csv += `${family},${type},${f.occurrences || 0},${f.percentage || 0},${f.avgDurationMs || 0},"${f.formattedSize || ''}",${fontUrl},${pageUrlsList}\n`;
        });

        const tempPath = await window.go.main.App.ExportFontsCSV(csv);
        if (tempPath && window.go.main.App.UploadFileToGDrive) {
          const folderName = (this.activeProfile?.name || 'Site') + ' Reports';
          const res = await window.go.main.App.UploadFileToGDrive(tempPath, folderName);
          if (res && res.webViewLink) {
            navigator.clipboard.writeText(res.webViewLink);
            this.showToast?.('success', 'Вивантажено ☁️ (Link copied 📋)', res.webViewLink);
            this.addLog?.('success', `Файл з звітом шрифтів вивантажено в Google Drive: ${res.webViewLink}`);
          }
        }
      } catch (err) {
        console.error("GDrive upload error:", err);
        this.showToast?.('error', 'Помилка вивантаження', err.message || err);
      } finally {
        this.isUploadingGDrive = false;
      }
    },

    async uploadFormsToGDrive() {
      if (!this.siteAnalytics || !this.siteAnalytics.forms || this.siteAnalytics.forms.length === 0) {
        this.showToast?.('warning', 'Відсутні дані', 'Спочатку виконайте сканування сторінок.');
        return;
      }
      if (!this.gdriveStatus.connected) {
        this.showToast?.('warning', 'Не підключено', 'Спочатку підключіть Google Drive у Налаштуваннях.');
        this.settingsTab = 'cloud';
        this.showSettings = true;
        return;
      }

      this.isUploadingGDrive = true;
      this.showToast?.('info', 'Вивантаження ☁️', 'Надсилаємо звіт форм у Google Drive...');
      try {
        let csv = "Form Title,Engine,Page Count,Method,Action,Field Count,Fields Summary,File Upload,Allowed File Types,Captcha Type,Captcha Active,Page URLs List\n";
        this.siteAnalytics.forms.forEach(f => {
          const title = `"${(f.title || '').replace(/"/g, '""')}"`;
          const engine = `"${(f.engine || '').replace(/"/g, '""')}"`;
          const method = `"${(f.method || 'POST').replace(/"/g, '""')}"`;
          const action = `"${(f.action || '').replace(/"/g, '""')}"`;
          const count = f.fieldCount || (f.fields || []).length;
          const fieldsSummary = `"${(f.fields || []).map(fld => fld.name + (fld.isRequired ? '*' : '') + ':' + fld.type).join('; ').replace(/"/g, '""')}"`;
          const fileUpload = f.hasFileUpload ? '"Yes"' : '"No"';
          const allowedFiles = `"${(f.allowedFileTypes || '').replace(/"/g, '""')}"`;
          const captchaType = `"${(f.captcha?.type || 'none').replace(/"/g, '""')}"`;
          const captchaActive = f.captcha?.isActive ? '"Yes"' : '"No"';
          const pageUrlsList = `"${(f.pages || []).join('; ').replace(/"/g, '""')}"`;
          csv += `${title},${engine},${f.pageCount || 0},${method},${action},${count},${fieldsSummary},${fileUpload},${allowedFiles},${captchaType},${captchaActive},${pageUrlsList}\n`;
        });

        const filename = `speedmap_forms_audit_${Date.now()}.csv`;
        const res = await window.go.main.App.UploadFileToGDrive(filename, csv, "text/csv");
        if (res && res.success) {
          this.showToast?.('success', 'Вивантажено в Google Drive ☁️', `Файл збережено: ${res.fileName}`);
          this.addLog?.('success', `Звіт по формах вивантажено на Google Drive: ${res.fileUrl}`);
        } else {
          throw new Error(res?.error || 'Помилка вивантаження');
        }
      } catch (err) {
        this.showToast?.('error', 'Помилка Google Drive', err.message || err);
      } finally {
        this.isUploadingGDrive = false;
      }
    },

    async uploadIframesToGDrive() {
      if (!this.siteAnalytics || !this.siteAnalytics.iframes || this.siteAnalytics.iframes.length === 0) {
        this.showToast?.('warning', 'Відсутні дані', 'Спочатку виконайте сканування сторінок.');
        return;
      }
      if (!this.gdriveStatus.connected) {
        this.showToast?.('warning', 'Не підключено', 'Спочатку підключіть Google Drive у Налаштуваннях.');
        this.settingsTab = 'cloud';
        this.showSettings = true;
        return;
      }

      this.isUploadingGDrive = true;
      this.showToast?.('info', 'Вивантаження ☁️', 'Надсилаємо звіт iframe у Google Drive...');
      try {
        let csv = "Iframe SRC,Title,Occurrences,Page Count,Loaded Count,Missed Count,Lazy Loading,Avg Duration (ms),Max Transfer Size,Dimensions,Page URLs List\n";
        this.siteAnalytics.iframes.forEach(f => {
          const src = `"${(f.src || '').replace(/"/g, '""')}"`;
          const title = `"${(f.title || '').replace(/"/g, '""')}"`;
          const isLazy = f.isLazy ? '"Yes"' : '"No"';
          const size = `"${f.formattedSize || (f.maxTransferSize ? (f.maxTransferSize + ' B') : '0 B')}"`;
          const dimensions = `"${(f.width || 0)}x${(f.height || 0)}"`;
          const pageUrlsList = `"${(f.pages || []).join('; ').replace(/"/g, '""')}"`;
          csv += `${src},${title},${f.occurrences || 0},${f.pageCount || 0},${f.loadedCount || 0},${f.missedCount || 0},${isLazy},${f.avgDurationMs || 0},${size},${dimensions},${pageUrlsList}\n`;
        });

        const tempPath = await window.go.main.App.ExportIframesCSV(csv);
        if (tempPath && window.go.main.App.UploadFileToGDrive) {
          const folderName = (this.activeProfile?.name || 'Site') + ' Reports';
          const res = await window.go.main.App.UploadFileToGDrive(tempPath, folderName);
          if (res && res.webViewLink) {
            navigator.clipboard.writeText(res.webViewLink);
            this.showToast?.('success', 'Вивантажено ☁️ (Link copied 📋)', res.webViewLink);
            this.addLog?.('success', `Файл зі звітом iframe вивантажено в Google Drive: ${res.webViewLink}`);
          }
        }
      } catch (err) {
        console.error("GDrive iframe upload error:", err);
        this.showToast?.('error', 'Помилка вивантаження', err.message || err);
      } finally {
        this.isUploadingGDrive = false;
      }
    }
  };
}
