import { ref } from 'vue'
import { api } from '@/api/index.js'

export function useFileUpload(appStore) {
    const pendingChips = ref([]) // [{ id, name, progress, done, fileId }]

    async function uploadFile(file) {
        if (!appStore.currentChannel) {
            appStore.showNotification('請先選擇一個頻道', 'error')
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
            appStore.pendingFileIds.push(file_id)
        } catch (e) {
            console.error('Upload failed', e)
            appStore.showNotification(`上傳失敗：${e.message}`, 'error')
            pendingChips.value = pendingChips.value.filter(c => c.id !== chipId)
        }
    }

    function removeChip(chipId, fileId) {
        appStore.pendingFileIds = appStore.pendingFileIds.filter(id => String(id) !== String(fileId))
        pendingChips.value = pendingChips.value.filter(c => c.id !== chipId)
    }

    function clearChips() {
        pendingChips.value = []
    }

    return { pendingChips, uploadFile, removeChip, clearChips }
}
