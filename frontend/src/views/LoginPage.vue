<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import TFIcon from '../components/common/TFIcon.vue'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const username = ref('')
const password = ref('')
const loading = ref(false)
const errorMsg = ref('')

const redirect = route.query.redirect as string || '/'

const login = async () => {
  if (!username.value || !password.value) {
    errorMsg.value = '请输入用户名和密码'
    return
  }
  errorMsg.value = ''
  loading.value = true
  try {
    await authStore.login(username.value, password.value)
    router.push(redirect)
  } catch (e: any) {
    errorMsg.value = e.message || '登录失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <div class="login-card">
      <div class="card-logo">
        <TFIcon name="waves" :size="48" color="var(--accent)" />
        <h1>TideFlow</h1>
      </div>

      <h2>登录</h2>

      <q-input
        v-model="username"
        label="用户名"
        outlined
        dark
        autocomplete="username"
        class="form-input"
      />
      <q-input
        v-model="password"
        label="密码"
        type="password"
        outlined
        dark
        autocomplete="current-password"
        class="form-input"
        @keyup.enter="login"
      />

      <p v-if="errorMsg" class="error-msg">{{ errorMsg }}</p>

      <q-btn
        class="login-btn"
        no-caps
        label="登录"
        color="primary"
        :loading="loading"
        @click="login"
      />

      <p class="switch-link">
        还没有账号？
        <router-link to="/register">去注册</router-link>
      </p>
    </div>
  </div>
</template>

<style scoped lang="scss">
.login-page {
  min-height: 100svh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-base);
  padding: var(--space-4);
}

.login-card {
  width: 100%;
  max-width: 380px;
  background: var(--bg-surface);
  border-radius: var(--radius-lg);
  padding: var(--space-8);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-4);
  box-shadow: var(--shadow-elevated);
}

.card-logo {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: var(--space-2);

  h1 {
    font-size: 24px;
    font-weight: 700;
    color: var(--text-base);
  }
}

h2 {
  font-size: 20px;
  font-weight: 700;
  color: var(--text-base);
  align-self: flex-start;
}

.form-input {
  width: 100%;
  :deep(.q-field__control) {
    background: var(--bg-elevated);
    border-radius: var(--radius-sm);
  }
}

.error-msg {
  color: var(--text-negative);
  font-size: 14px;
  text-align: center;
}

.login-btn {
  width: 100%;
  border-radius: var(--radius-pill);
  font-weight: 700;
  font-size: 16px;
  height: 48px;
}

.switch-link {
  font-size: 14px;
  color: var(--text-secondary);

  a {
    color: var(--accent);
    font-weight: 600;
  }
}
</style>
