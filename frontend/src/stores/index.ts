import { createPinia } from 'pinia'

const pinia = createPinia()

export default pinia

export { useAuthStore } from './auth'
export { useLayoutStore } from './layout'
export { useFeedStore } from './feed'
export { useNotificationStore } from './notification'
export { useMessageStore } from './message'