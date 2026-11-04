import '@fontsource/jetbrains-mono/400.css'
import '@fontsource/jetbrains-mono/500.css'
import '@fontsource/jetbrains-mono/700.css'

import { createApp } from 'vue'

import App from './App.vue'
import router from './router'
import './styles/main.css'

createApp(App).use(router).mount('#app')
