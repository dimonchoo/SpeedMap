export function createNotificationsModule() {
  return {
    activityLogs: [],
    toast: {
      show: false,
      type: 'info',
      title: '',
      message: '',
      path: null
    },

    // Audio & Pokémon Game State
    audioUnlocked: false,
    audioCtx: null,
    pikaPageEl: null,
    pikaFullEl: null,

    showPokemonGame: false,
    pokemonGame: {
      current: null,
      options: [],
      isRevealed: false,
      selectedOpt: null,
      isCorrect: false,
      score: 0,
      streak: 0,
      total: 0
    },
    pokedex: [
      { id: "pikachu", name: "Пікачу (Pikachu)", perk: "Блискавичний TTFB & LCP" },
      { id: "bulbasaur", name: "Бульбазавр (Bulbasaur)", perk: "Зелена зона Core Web Vitals" },
      { id: "charmander", name: "Чармандер (Charmander)", perk: "Гаряча компресія WebP Q85" },
      { id: "squirtle", name: "Сквіртл (Squirtle)", perk: "Очищення кешу Cloudflare" },
      { id: "eevee", name: "Іві (Eevee)", perk: "Адаптивний рендер під Retina 2x" },
      { id: "gengar", name: "Генгар (Gengar)", perk: "Пошук важких DOM-вузлів" },
      { id: "snorlax", name: "Снорлакс (Snorlax)", perk: "Оптимізація мегабайтних банерів" },
      { id: "psyduck", name: "Псайдак (Psyduck)", perk: "Діагностика складних помилок LCP" },
      { id: "jigglypuff", name: "Джігліпаф (Jigglypuff)", perk: "Плавний CLS 0.000" }
    ],

    addLog(type, message) {
      this.activityLogs.unshift({
        time: new Date().toLocaleTimeString(),
        type,
        message
      });
      if (this.activityLogs.length > 100) this.activityLogs.pop();
    },

    showToast(type, title, message, actionPath = null) {
      this.toast = {
        show: true,
        type,
        title,
        message,
        path: actionPath
      };
      // Keep toast open longer if there's a direct action path (12s vs 4s)
      const timeoutMs = actionPath ? 12000 : 4000;
      setTimeout(() => {
        if (this.toast.message === message) {
          this.toast.show = false;
        }
      }, timeoutMs);
    },

    revealInFinder(targetPath) {
      if (!targetPath) return;
      if (window.go && window.go.main && window.go.main.App && window.go.main.App.RevealInFinder) {
        window.go.main.App.RevealInFinder(targetPath);
      } else if (window.go && window.go.main && window.go.main.App && window.go.main.App.OpenURL) {
        window.go.main.App.OpenURL('file://' + targetPath);
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

    openPokemonGame(targetId) {
      this.showPokemonGame = true;
      this.startPokemonRound(targetId);
    },

    startPokemonRound(targetId) {
      this.pokemonGame.isRevealed = false;
      this.pokemonGame.selectedOpt = null;
      this.pokemonGame.isCorrect = false;
      let target = null;
      if (targetId) {
        target = this.pokedex.find(p => p.id === targetId);
      }
      if (!target) {
        target = this.pokedex[Math.floor(Math.random() * this.pokedex.length)];
      }
      this.pokemonGame.current = target;

      const shuffled = [...this.pokedex].sort(() => 0.5 - Math.random());
      if (!shuffled.slice(0, 4).some(p => p.id === target.id)) {
        shuffled[0] = target;
      }
      this.pokemonGame.options = shuffled.slice(0, 4).sort(() => 0.5 - Math.random());
    },

    guessPokemon(opt) {
      if (this.pokemonGame.isRevealed) return;
      this.pokemonGame.selectedOpt = opt;
      this.pokemonGame.isCorrect = (opt.id === this.pokemonGame.current?.id);
      this.pokemonGame.total++;
      if (this.pokemonGame.isCorrect) {
        this.pokemonGame.score++;
        this.pokemonGame.streak++;
      } else {
        this.pokemonGame.streak = 0;
      }
      this.pokemonGame.isRevealed = true;
      this.playPokemonCry();
    },

    playPokemonCry() {
      if (!this.pokemonGame.current) return;
      try {
        const audio = new Audio(`pokemon/audio/${this.pokemonGame.current.id}.mp3`);
        audio.play().catch(() => {});
      } catch (e) {}
    }
  };
}
