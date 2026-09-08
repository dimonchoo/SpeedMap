export function createShortcutsModule() {
  return {
    activeTab: 'pages',
    settingsTab: 'general',
    showSettings: false,
    showLogDrawer: false,

    initShortcuts() {
      // Global link interceptor: guarantees ANY link clicked anywhere in app opens in external browser
      document.addEventListener('click', (e) => {
        const link = e.target.closest('a');
        if (link && link.href) {
          const href = link.getAttribute('href') || '';
          if (href.startsWith('http://') || href.startsWith('https://') || link.target === '_blank') {
            e.preventDefault();
            e.stopPropagation();
            this.openUrlInBrowser?.(link.href);
          }
        }
      }, true);

      // Unlock WebAudio on first real click/tap/keydown
      const unlockOnce = () => {
        this.unlockAudio?.();
        window.removeEventListener('pointerdown', unlockOnce, true);
        window.removeEventListener('keydown', unlockOnce, true);
      };
      window.addEventListener('pointerdown', unlockOnce, true);
      window.addEventListener('keydown', unlockOnce, true);

      // Global keyboard navigation
      window.addEventListener('keydown', (e) => {
        if (this.imageStudio?.isOpen) {
          this.handleStudioKeydown?.(e);
          return;
        }

        const tag = e.target.tagName ? e.target.tagName.toLowerCase() : '';
        if (tag === 'input' || tag === 'textarea' || tag === 'select') return;

        // Cmd/Ctrl + Enter = Start / Cancel scan
        if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
          e.preventDefault();
          if (this.isScanning) {
            this.cancelScan?.();
          } else {
            this.startScan?.();
          }
        }
      });

      window.addEventListener('keyup', (e) => {
        if (this.imageStudio?.isOpen) {
          this.handleStudioKeyup?.(e);
        }
      });
    }
  };
}
