import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { createPricingApi, setPricingApi } from './services/provide'

import './design/tokens.css'
import './design/base.css'

// Composition root: выбор реализации PricingApi здесь (mock -> http одной заменой).
const api = createPricingApi()
setPricingApi(api)

const app = createApp(App)
app.provide('pricingApi', api)
app.use(createPinia())
app.use(router)
app.mount('#app')
