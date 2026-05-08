import { createPinia } from 'pinia'
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate'

const pinia = createPinia()
pinia.use(piniaPluginPersistedstate)

export default pinia

export { useAuthStore } from './auth'
export { useLayoutStore } from './layout'
export { useFeedStore } from './feed'
export { useNotificationStore } from './notification'
export { useMessageStore } from './message'
export { useInteractionStore } from './interaction'