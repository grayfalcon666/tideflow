import { createApp } from 'vue'
import { Quasar, Notify } from 'quasar'
import App from './App.vue'
import router from './router'
import pinia from './stores'

// 引入 Quasar 样式
import 'quasar/src/css/index.sass'
import './styles/global.css'

const app = createApp(App)

app.use(Quasar, {
  plugins: { Notify },  // ← 注册 Notify
})
app.use(router)
app.use(pinia)

app.mount('#app')