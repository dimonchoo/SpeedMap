import { createApp } from 'vue';
import App from './App.vue';
import './style.css';

const app = createApp(App);

// Click outside directive for dropdowns & modals
app.directive('click-outside', {
  mounted(el, binding) {
    el._clickOutsideHandler = (event) => {
      if (!(el === event.target || el.contains(event.target))) {
        if (typeof binding.value === 'function') {
          binding.value(event);
        }
      }
    };
    setTimeout(() => {
      document.addEventListener('click', el._clickOutsideHandler);
    }, 0);
  },
  unmounted(el) {
    if (el._clickOutsideHandler) {
      document.removeEventListener('click', el._clickOutsideHandler);
    }
  }
});

app.mount('#app');
