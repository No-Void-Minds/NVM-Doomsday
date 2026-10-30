import '@fontsource/archivo-narrow/700.css'
import '@fontsource-variable/archivo'
import '@fontsource/ibm-plex-mono/400.css'
import '@fontsource/ibm-plex-mono/500.css'

import { createApp } from 'vue'

import App from './App.vue'
import router from './router'
import './styles/main.css'

createApp(App).use(router).mount('#app')
