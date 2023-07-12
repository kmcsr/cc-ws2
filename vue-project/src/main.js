import { createApp } from 'vue'
import { newRouter } from './router'
import App from './App.vue'

import './assets/main.css'

createApp(App).
	use(newRouter()).
	mount('#app')
