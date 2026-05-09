<script setup>
import { ref } from 'vue'
import { useAppStore } from '@/stores/useAppStore.js'

const store = useAppStore()
const showLogin = ref(true)

const loginEmail = ref('')
const loginPassword = ref('')
const regEmail = ref('')
const regUsername = ref('')
const regNickname = ref('')
const regPassword = ref('')

async function onLogin() {
  try {
    store.setLoading(true)
    await store.$patch({}) // no-op, just ensure store is reactive
    const { api } = await import('@/api/index.js')
    const res = await api.login(loginEmail.value, loginPassword.value)
    store.user = res.user
    store.showNotification('登入成功！', 'success')
    await store.loadUserData()
  } catch (e) {
    store.showNotification(e.message || '登入失敗', 'error')
  } finally {
    store.setLoading(false)
  }
}

async function onRegister() {
  try {
    store.setLoading(true)
    const { api } = await import('@/api/index.js')
    await api.register(regUsername.value, regEmail.value, regPassword.value, regNickname.value)
    store.showNotification('註冊成功！正在登入...', 'success')
    const res = await api.login(regEmail.value, regPassword.value)
    store.user = res.user
    await store.loadUserData()
  } catch (e) {
    store.showNotification(e.message || '註冊失敗', 'error')
  } finally {
    store.setLoading(false)
  }
}

function loginWithGoogle() {
  window.location.href = '/api/v1/auth/google'
}
</script>

<template>
  <div class="auth-container">
    <div class="auth-box">
      <div class="auth-logo">
        <i class="fas fa-comments"></i>
        <h1>TalkRealm</h1>
      </div>

      <!-- Login -->
      <div :class="['auth-form', { active: showLogin }]">
        <h2>歡迎回來！</h2>
        <p class="auth-subtitle">我們很高興再次見到您！</p>
        <form @submit.prevent="onLogin">
          <div class="form-group">
            <label>電子郵件</label>
            <input v-model="loginEmail" type="email" placeholder="name@example.com" required />
          </div>
          <div class="form-group">
            <label>密碼</label>
            <input v-model="loginPassword" type="password" placeholder="請輸入密碼" required />
          </div>
          <button type="submit" class="btn-primary">登入</button>
          <div class="auth-divider"><span>或</span></div>
          <button type="button" class="btn-oauth btn-google" @click="loginWithGoogle">
            <GoogleIcon />
            使用 Google 登入
          </button>
          <div class="auth-switch">
            還沒有帳號？ <a href="#" @click.prevent="showLogin = false">立即註冊</a>
          </div>
        </form>
      </div>

      <!-- Register -->
      <div :class="['auth-form', { active: !showLogin }]">
        <h2>建立帳號</h2>
        <p class="auth-subtitle">加入 TalkRealm 社群</p>
        <form @submit.prevent="onRegister">
          <div class="form-group">
            <label>電子郵件</label>
            <input v-model="regEmail" type="email" placeholder="name@example.com" required />
          </div>
          <div class="form-group">
            <label>使用者名稱</label>
            <input v-model="regUsername" type="text" placeholder="請輸入使用者名稱" required minlength="3" maxlength="32" />
          </div>
          <div class="form-group">
            <label>暱稱（選填）</label>
            <input v-model="regNickname" type="text" placeholder="顯示名稱" maxlength="64" />
          </div>
          <div class="form-group">
            <label>密碼</label>
            <input v-model="regPassword" type="password" placeholder="請輸入密碼" required minlength="6" />
          </div>
          <button type="submit" class="btn-primary">註冊</button>
          <div class="auth-divider"><span>或</span></div>
          <button type="button" class="btn-oauth btn-google" @click="loginWithGoogle">
            <GoogleIcon />
            使用 Google 繼續
          </button>
          <div class="auth-switch">
            已有帳號？ <a href="#" @click.prevent="showLogin = true">立即登入</a>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script>
// Inline Google SVG icon component
const GoogleIcon = {
  template: `<svg class="oauth-icon" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
    <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" fill="#4285F4"/>
    <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
    <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l3.66-2.84z" fill="#FBBC05"/>
    <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
  </svg>`
}
</script>
