<template>
<!-- WHO'S THAT POKÉMON? EASTER EGG MODAL -->
    <div v-show="showPokemonGame" class="fixed inset-0 z-[80] overflow-y-auto flex items-center justify-center p-4"
         @keydown.escape.window="showPokemonGame = false">
      <div class="fixed inset-0 bg-slate-950/85 backdrop-blur-md transition-opacity" @click="showPokemonGame = false"></div>

      <div class="relative max-w-md w-full anime-bg border-4 border-amber-400 rounded-3xl shadow-2xl overflow-hidden p-6 flex flex-col items-center text-center my-8 z-10 select-none animate-in fade-in zoom-in-95 duration-200">
        
        <!-- Close Button -->
        <button @click="showPokemonGame = false" class="absolute top-4 right-4 z-20 text-slate-400 hover:text-white bg-slate-900/80 hover:bg-slate-800 p-1.5 rounded-full border border-slate-700 transition" title="Закрити (Esc)">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
          </svg>
        </button>

        <!-- Rotating Sunburst Rays Background -->
        <div class="sunburst-rays pointer-events-none"></div>

        <!-- Title Header & Score Tracker -->
        <div class="relative z-10 space-y-1.5 mb-2">
          <div class="flex items-center justify-center space-x-2">
            <div class="inline-flex items-center space-x-1.5 bg-red-600 border border-amber-300 text-white font-black text-[10px] uppercase px-3 py-0.5 rounded-full shadow-lg tracking-wider">
              <span>⚡ SPEEDMAP EASTER EGG</span>
            </div>
            <div class="inline-flex items-center space-x-1 bg-slate-900/90 border border-amber-400/60 text-amber-300 font-bold text-[10px] px-2.5 py-0.5 rounded-full shadow">
              <span>🏆 <span v-text="pokemonGame.score"></span>/<span v-text="pokemonGame.total"></span></span>
              <template v-if="pokemonGame.streak > 1">
                <span class="text-rose-400 font-extrabold ml-1">🔥 <span v-text="pokemonGame.streak"></span></span>
              </template>
            </div>
          </div>
          <h2 class="text-2xl sm:text-3xl font-black text-amber-300 tracking-wider drop-shadow-[0_4px_8px_rgba(0,0,0,0.8)] font-serif uppercase">
            Хто це за покемон?
          </h2>
          <p class="text-[11px] text-blue-200 font-medium">Who's That Pokémon?</p>
        </div>

        <!-- Pokémon Display Circle Stage -->
        <div class="relative z-10 w-48 h-48 my-2 flex items-center justify-center bg-slate-900/90 border-4 border-blue-400/50 rounded-full shadow-inner p-5">
          <div class="absolute inset-0 rounded-full bg-radial from-blue-500/20 to-transparent pointer-events-none"></div>
          
          <template v-if="pokemonGame.current">
            <img :src="`pokemon/art/${pokemonGame.current.id}.webp`"
                 :class="pokemonGame.isRevealed ? 'poke-revealed' : 'poke-silhouette'"
                 class="max-h-32 max-w-32 w-auto h-auto object-contain transition-all duration-300 select-none pointer-events-none" />
          </template>
        </div>

        <!-- Reveal Result Banner -->
        <div class="relative z-10 min-h-12 flex items-center justify-center my-1.5">
          <template v-if="pokemonGame.isRevealed && pokemonGame.current">
            <div class="space-y-0.5 animate-bounce">
              <p class="text-lg sm:text-xl font-black drop-shadow" :class="pokemonGame.isCorrect ? 'text-emerald-300' : 'text-amber-300'">
                <span v-text="pokemonGame.isCorrect ? '🎉 ВГАДАЛИ! Це...' : '❌ Майже! Це був...'"></span>
                <span v-text="pokemonGame.current.name"></span>!
              </p>
              <p class="text-[11px] text-cyan-300 font-mono" v-text="`Спеціалізація: ${pokemonGame.current.perk}`"></p>
            </div>
          </template>
          <template v-if="!pokemonGame.isRevealed">
            <p class="text-xs font-semibold text-slate-300">Вгадайте, хто ховається у тіні:</p>
          </template>
        </div>

        <!-- Answer Options (4 buttons) -->
        <div class="relative z-10 w-full grid grid-cols-2 gap-2.5 my-2">
          <template v-for="opt in pokemonGame.options" :key="opt.id">
            <button @click="guessPokemon(opt)" :disabled="pokemonGame.isRevealed"
              class="font-bold py-2.5 px-3 rounded-xl border shadow-md transition transform text-xs cursor-pointer disabled:cursor-default"
              :class="!pokemonGame.isRevealed
                ? 'bg-slate-900/90 hover:bg-amber-500 hover:text-slate-950 text-slate-100 border-slate-700 hover:border-amber-300 active:scale-95'
                : (opt.id === pokemonGame.current?.id
                    ? 'bg-emerald-600/90 text-white border-emerald-300 ring-2 ring-emerald-400 scale-102 shadow-emerald-500/50'
                    : (opt.id === pokemonGame.selectedOpt?.id && !pokemonGame.isCorrect
                        ? 'bg-rose-700/80 text-rose-200 border-rose-500 line-through opacity-80'
                        : 'bg-slate-900/50 text-slate-500 border-slate-800 opacity-40'))">
              <span v-text="opt.name"></span>
            </button>
          </template>
        </div>

        <!-- Actions when Revealed -->
        <div class="relative z-10 w-full flex items-center justify-center space-x-2.5 pt-2" v-show="pokemonGame.isRevealed">
          <button @click="playPokemonCry()"
            class="bg-blue-600 hover:bg-blue-500 text-white font-bold py-2 px-4 rounded-xl shadow-lg border border-blue-400 transition flex items-center space-x-1.5 text-xs cursor-pointer active:scale-95">
            <span>🔊 Почути голос</span>
          </button>

          <button @click="startPokemonRound()"
            class="bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 text-white font-bold py-2 px-5 rounded-xl shadow-lg border border-emerald-400 transition flex items-center space-x-1.5 text-xs cursor-pointer active:scale-95">
            <span>🔄 Наступний покемон</span>
          </button>
        </div>

      </div>
    </div>
</template>

<script>
import { useApp } from '@/store/app';

export default {
  name: 'PokemonModal',
  setup() {
    return useApp();
  }
};
</script>
