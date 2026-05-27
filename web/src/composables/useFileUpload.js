import { ref } from 'vue'
import { api } from '@/api/index.js'

/**
 * useFileUpload
 * @param {object} context - Either appStore (guild chat) or a DM context object:
 *   {
 *     checkReady?: () => boolean,       // optional gate (e.g. channel selected)
 *     onNotReady?: () => void,          // called when gate fails
 *     addFileId: (id) => void,          // push confirmed fileId somewhere
 *     removeFileId: (id) => void,       // remove fileId on chip removal
 *     showNotification: (msg, type) => void,
 *   }
 *
 * Backward compat: if context has `pendingFileIds` array and `currentChannel`,
 * it's treated as the legacy appStore.
 */
export function useFileUpload(context) {
    const pendingChips = ref([]) // [{ id, name, progress, done, fileId }]

    function isLegacyStore(ctx) {
        return Array.isArray(ctx.pendingFileIds)
    }

    function resolveContext(ctx) {
        if (isLegacyStore(ctx)) {
            return {
                checkReady: () => !!ctx.currentChannel,
                onNotReady: () => ctx.showNotification('請先選擇一個頻道', 'error'),
                addFileId: (id) => ctx.pendingFileIds.push(id),
                removeFileId: (id) => { ctx.pendingFileIds = ctx.pendingFileIds.filter(x => String(x) !== String(id)) },
                showNotification: ctx.showNotification.bind(ctx),
            }
        }
        return {
            checkReady: ctx.checkReady ?? (() => true),
            onNotReady: ctx.onNotReady ?? (() => { }),
            addFileId: ctx.addFileId,
            removeFileId: ctx.removeFileId,
            showNotification: ctx.showNotification,
        }
    }

    async function uploadFile(file) {
        const ctx = resolveContext(context)
        if (!ctx.checkReady()) {
            ctx.onNotReady()
            return
        }
        const chipId = `upload-${Date.now()}-${Math.random()}`
        pendingChips.value.push({ id: chipId, name: file.name, progress: 0, done: false, fileId: null })

        try {
            const { file_id, upload_url } = await api.presignUpload(file.name, file.type || 'application/octet-stream', file.size)
            await api.uploadToMinio(upload_url, file, (pct) => {
                const chip = pendingChips.value.find(c => c.id === chipId)
                if (chip) chip.progress = pct
            })
            await api.confirmUpload(file_id)
            const chip = pendingChips.value.find(c => c.id === chipId)
            if (chip) { chip.done = true; chip.fileId = file_id }
            ctx.addFileId(file_id)
        } catch (e) {
            console.error('Upload failed', e)
            ctx.showNotification(`上傳失敗：${e.message}`, 'error')
            pendingChips.value = pendingChips.value.filter(c => c.id !== chipId)
        }
    }

    function removeChip(chipId, fileId) {
        const ctx = resolveContext(context)
        ctx.removeFileId(fileId)
        pendingChips.value = pendingChips.value.filter(c => c.id !== chipId)
    }

    function clearChips() {
        pendingChips.value = []
    }

    return { pendingChips, uploadFile, removeChip, clearChips }
}
