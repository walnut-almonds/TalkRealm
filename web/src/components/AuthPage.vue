<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/useAppStore.js'

const store = useAppStore()
const { t } = useI18n()
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
    store.showNotification(t('auth.loginSuccess'), 'success')
    await store.loadUserData()
  } catch (e) {
    store.showNotification(e.message || t('auth.loginFailed'), 'error')
  } finally {
    store.setLoading(false)
  }
}

async function onRegister() {
  try {
    store.setLoading(true)
    const { api } = await import('@/api/index.js')
    await api.register(regUsername.value, regEmail.value, regPassword.value, regNickname.value)
    store.showNotification(t('auth.registerSuccess'), 'success')
    const res = await api.login(regEmail.value, regPassword.value)
    store.user = res.user
    await store.loadUserData()
  } catch (e) {
    store.showNotification(e.message || t('auth.registerFailed'), 'error')
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
        <h2>{{ t('auth.welcomeBack') }}</h2>
        <p class="auth-subtitle">{{ t('auth.welcomeSubtitle') }}</p>
        <form @submit.prevent="onLogin">
          <div class="form-group">
            <label>{{ t('auth.email') }}</label>
            <input v-model="loginEmail" type="email" placeholder="name@example.com" required />
          </div>
          <div class="form-group">
            <label>{{ t('auth.password') }}</label>
            <input v-model="loginPassword" type="password" :placeholder="t('auth.passwordPlaceholder')" required />
          </div>
          <button type="submit" class="btn-primary">{{ t('auth.login') }}</button>
          <div class="auth-divider"><span>{{ t('auth.or') }}</span></div>
          <button type="button" class="btn-oauth btn-google" @click="loginWithGoogle">
            <GoogleIcon />
            {{ t('auth.loginWithGoogle') }}
          </button>
          <div class="auth-switch">
            {{ t('auth.noAccount') }} <a href="#" @click.prevent="showLogin = false">{{ t('auth.signUpNow') }}</a>
          </div>
        </form>
      </div>

      <!-- Register -->
      <div :class="['auth-form', { active: !showLogin }]">
        <h2>{{ t('auth.createAccount') }}</h2>
        <p class="auth-subtitle">{{ t('auth.joinSubtitle') }}</p>
        <form @submit.prevent="onRegister">
          <div class="form-group">
            <label>{{ t('auth.email') }}</label>
            <input v-model="regEmail" type="email" placeholder="name@example.com" required />
          </div>
          <div class="form-group">
            <label>{{ t('auth.username') }}</label>
            <input v-model="regUsername" type="text" :placeholder="t('auth.usernamePlaceholder')" required minlength="3" maxlength="32" />
          </div>
          <div class="form-group">
            <label>{{ t('auth.nicknameOptional') }}</label>
            <input v-model="regNickname" type="text" :placeholder="t('auth.nicknamePlaceholder')" maxlength="64" />
          </div>
          <div class="form-group">
            <label>{{ t('auth.password') }}</label>
            <input v-model="regPassword" type="password" :placeholder="t('auth.passwordPlaceholder')" required minlength="6" />
          </div>
          <button type="submit" class="btn-primary">{{ t('auth.register') }}</button>
          <div class="auth-divider"><span>{{ t('auth.or') }}</span></div>
          <button type="button" class="btn-oauth btn-google" @click="loginWithGoogle">
            <GoogleIcon />
            {{ t('auth.continueWithGoogle') }}
          </button>
          <div class="auth-switch">
            {{ t('auth.haveAccount') }} <a href="#" @click.prevent="showLogin = true">{{ t('auth.loginNow') }}</a>
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
