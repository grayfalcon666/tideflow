<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import TFIcon from '../components/common/TFIcon.vue'

const router = useRouter()
const authStore = useAuthStore()

const username = ref('')
const password = ref('')
const confirmPassword = ref('')
const loading = ref(false)
const errorMsg = ref('')

const register = async () => {
  if (!username.value || !password.value) {
    errorMsg.value = '请填写所有必填项'
    return
  }
  if (username.value.length < 3 || username.value.length > 30) {
    errorMsg.value = '用户名需 3~30 位'
    return
  }
  if (password.value.length < 6) {
    errorMsg.value = '密码至少 6 位'
    return
  }
  if (password.value !== confirmPassword.value) {
    errorMsg.value = '两次密码不一致'
    return
  }
  errorMsg.value = ''
  loading.value = true
  try {
    await authStore.register(username.value, password.value)
    router.push('/')
  } catch (e: any) {
    errorMsg.value = e.message || '注册失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="register-page">
    <div class="register-card">
      <div class="card-logo">
        <TFIcon name="waves" :size="48" color="var(--accent)" />
        <h1>TideFlow</h1>
      </div>

      <h2>注册</h2>

      <q-input
        v-model="username"
        label="用户名（3~30位）"
        outlined
        dark
        autocomplete="username"
        class="form-input"
      />
      <q-input
        v-model="password"
        label="密码（至少6位）"
        type="password"
        outlined
        dark
        autocomplete="new-password"
        class="form-input"
      />
      <q-input
        v-model="confirmPassword"
        label="确认密码"
        type="password"
        outlined
        dark
        autocomplete="new-password"
        class="form-input"
        @keyup.enter="register"
      />

      <p v-if="errorMsg" class="error-msg">{{ errorMsg }}</p>

      <q-btn
        class="register-btn"
        no-caps
        label="注册"
        color="primary"
        :loading="loading"
        @click="register"
      />

      <p class="switch-link">
        已有账号？
        <router-link to="/account">去登录</router-link>
      </p>
    </div>
  </div>
</template>

<style scoped lang="scss">
.register-page {
  min-height: 100svh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-base);
  padding: var(--space-4);
}

.register-card {
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

.register-btn {
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
