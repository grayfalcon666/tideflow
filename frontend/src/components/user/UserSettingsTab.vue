<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useSettingsStore } from '../../stores/settings'
import { useAuthStore } from '../../stores/auth'
import * as userService from '../../services/user'
import TFIcon from '../common/TFIcon.vue'

const router = useRouter()
const settingsStore = useSettingsStore()
const authStore = useAuthStore()

const toggleLikesPublic = async (val: boolean) => {
  settingsStore.likesPublic = val
  try {
    await userService.updateMe({ likes_public: val })
  } catch {
    // revert on failure
    settingsStore.likesPublic = !val
  }
}

const handleLogout = () => {
  authStore.logout(router)
}
</script>

<template>
  <div class="settings-tab">
    <div class="settings-section">
      <div class="setting-item">
        <div class="setting-info">
          <span class="setting-label">记住上次刷到的位置</span>
          <span class="setting-desc">下次打开时自动恢复到上次离开时的视频</span>
        </div>
        <q-toggle
          :model-value="settingsStore.rememberPosition"
          @update:model-value="settingsStore.rememberPosition = $event"
          color="accent"
          size="md"
        />
      </div>
      <div class="setting-item">
        <div class="setting-info">
          <span class="setting-label">允许他人看到我赞过的视频</span>
          <span class="setting-desc">关闭后，其他用户无法在你的主页查看你赞过的视频</span>
        </div>
        <q-toggle
          :model-value="settingsStore.likesPublic"
          @update:model-value="toggleLikesPublic($event)"
          color="accent"
          size="md"
        />
      </div>
    </div>

    <div class="settings-footer">
      <button class="logout-btn" @click="handleLogout">
        <TFIcon name="logout" :size="18" color="var(--text-negative)" />
        <span>退出登录</span>
      </button>
    </div>
  </div>
</template>

<style scoped lang="scss">
.settings-tab {
  display: flex;
  flex-direction: column;
  min-height: 300px;
}

.settings-section {
  flex: 1;
}

.setting-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4);
  background: var(--bg-card, var(--bg-base));
  border-radius: 12px;
  gap: var(--space-4);

  & + & {
    margin-top: var(--space-3);
  }
}

.setting-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.setting-label {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
}

.setting-desc {
  font-size: 12px;
  color: var(--text-muted);
}

.settings-footer {
  padding-top: var(--space-6);
  margin-top: auto;
}

.logout-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  padding: 12px;
  background: var(--bg-card, var(--bg-base));
  border: 1px solid var(--border-color);
  border-radius: 12px;
  color: var(--text-negative);
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: background var(--transition-fast);

  &:hover {
    background: var(--bg-hover);
  }
}
</style>
