import { createI18n } from 'vue-i18n'
import en from './locales/en.js'
import zh from './locales/zh.js'
import zhTw from './locales/zh-tw.js'
import ja from './locales/ja.js'

const STORAGE_KEY = 'talkrealm_ui_locale'
const SUPPORTED_LOCALES = ['zh', 'zh-tw', 'ja', 'en']

function normalizeLocale(input) {
    if (!input) return null
    const lower = input.toLowerCase()
    if (SUPPORTED_LOCALES.includes(lower)) return lower
    if (lower === 'zh-hant' || lower === 'zh_tw' || lower === 'zh-tw') return 'zh-tw'
    if (lower.startsWith('zh')) return 'zh'
    if (lower.startsWith('ja')) return 'ja'
    if (lower.startsWith('en')) return 'en'
    return null
}

function detectInitialLocale() {
    const stored = normalizeLocale(localStorage.getItem(STORAGE_KEY))
    if (stored) return stored

    const browser = normalizeLocale(navigator.language)
    if (browser) return browser

    return 'zh'
}

export const i18n = createI18n({
    legacy: false,
    locale: detectInitialLocale(),
    fallbackLocale: 'zh',
    messages: {
        en,
        zh,
        'zh-tw': zhTw,
        ja
    }
})

export function setLocale(locale) {
    const normalized = normalizeLocale(locale) || 'zh'
    i18n.global.locale.value = normalized
    localStorage.setItem(STORAGE_KEY, normalized)
}

export function getLocale() {
    return i18n.global.locale.value
}

export function getLocaleStorageKey() {
    return STORAGE_KEY
}

export function translate(key, params = {}) {
    return i18n.global.t(key, params)
}
