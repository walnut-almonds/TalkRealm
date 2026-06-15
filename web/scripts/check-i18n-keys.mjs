import { readdirSync, readFileSync, statSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const webRoot = path.resolve(__dirname, '..')
const srcRoot = path.join(webRoot, 'src')

const localeFiles = {
    en: path.join(srcRoot, 'i18n/locales/en.js'),
    zh: path.join(srcRoot, 'i18n/locales/zh.js'),
    'zh-tw': path.join(srcRoot, 'i18n/locales/zh-tw.js'),
    ja: path.join(srcRoot, 'i18n/locales/ja.js'),
}

function walkFiles(dir, exts, out = []) {
    for (const name of readdirSync(dir)) {
        const full = path.join(dir, name)
        const st = statSync(full)
        if (st.isDirectory()) {
            if (name === 'i18n' || name === 'dist' || name === 'node_modules') continue
            walkFiles(full, exts, out)
            continue
        }
        if (exts.includes(path.extname(name))) out.push(full)
    }
    return out
}

function extractI18nKeys(content) {
    const keys = new Set()
    const re = /(?:\$t|\bt|translate)\(\s*['"`]([^'"`]+)['"`]/g
    let m
    while ((m = re.exec(content)) !== null) {
        keys.add(m[1])
    }
    return keys
}

function flattenKeys(obj, prefix = '', out = new Set()) {
    for (const [k, v] of Object.entries(obj || {})) {
        const next = prefix ? `${prefix}.${k}` : k
        if (v && typeof v === 'object' && !Array.isArray(v)) {
            flattenKeys(v, next, out)
        } else {
            out.add(next)
        }
    }
    return out
}

async function loadLocale(file) {
    const mod = await import(pathToFileURL(file).href)
    return mod.default || {}
}

async function main() {
    const sourceFiles = walkFiles(srcRoot, ['.vue', '.js'])
    const usedKeys = new Set()

    for (const file of sourceFiles) {
        const text = readFileSync(file, 'utf8')
        for (const k of extractI18nKeys(text)) usedKeys.add(k)
    }

    const locales = {}
    for (const [name, file] of Object.entries(localeFiles)) {
        locales[name] = flattenKeys(await loadLocale(file))
    }

    const missingInEn = [...usedKeys].filter((k) => !locales.en.has(k)).sort()
    const missingByLocale = {}
    for (const [name, keys] of Object.entries(locales)) {
        if (name === 'en') continue
        missingByLocale[name] = [...locales.en].filter((k) => !keys.has(k)).sort()
    }

    if (missingInEn.length > 0) {
        console.error('Missing i18n keys in en locale:')
        for (const k of missingInEn) console.error(`  - ${k}`)
        process.exit(1)
    }

    let hasLocaleGap = false
    for (const [name, list] of Object.entries(missingByLocale)) {
        if (list.length === 0) continue
        hasLocaleGap = true
        console.warn(`Missing keys in ${name} locale (${list.length}):`)
        for (const k of list) console.warn(`  - ${k}`)
    }

    if (hasLocaleGap) {
        console.warn('Locale gap detected. App can still run via fallbackLocale=en.')
    }

    console.log(`i18n key check passed. Used keys: ${usedKeys.size}. en keys: ${locales.en.size}.`)
}

main().catch((err) => {
    console.error(err)
    process.exit(1)
})
