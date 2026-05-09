import { createApp } from 'vue'
import { Quasar, Notify, Dialog } from 'quasar'
import App from './App.vue'
import router from './router'
import pinia from './stores'
import { initApi } from './services/api'
import { useAuthStore } from './stores/auth'

// 引入 Quasar 样式
import 'quasar/src/css/index.sass'
import './styles/global.css'

const app = createApp(App)

app.use(Quasar, {
  plugins: { Notify, Dialog },
})
app.use(pinia)

// 注入 router 和 authStore 到 API 模块（必须在 initFromStorage 之前，因为 fetchMe 依赖拦截器注入 token）
initApi(router, () => useAuthStore())

// 阻塞式初始化：在路由守卫执行前完成 token 刷新
const authStore = useAuthStore()
await authStore.initFromStorage()

app.use(router)
app.mount('#app')