// 應用程式狀態
const appState = {
    user: null,
    currentGuild: null,
    currentChannel: null,
    guilds: [],
    channels: [],
    members: [],
    messages: [],
    isLoading: false,
    // 待附加的已確認檔案 ID（上傳成功後儲存，隨下次訊息一起送出）
    pendingFileIds: []
};

// 初始化應用程式
document.addEventListener('DOMContentLoaded', async () => {
    await checkAuth();
    setupWebSocketHandlers();
});

// 檢查認證狀態
async function checkAuth() {
    // OAuth callback：後端將 token 帶在 query string 或 fragment 中
    const rawSearch = window.location.search || window.location.hash.replace(/^#/, '?');
    const params = new URLSearchParams(rawSearch);
    const oauthAccessToken = params.get('access_token');
    const oauthRefreshToken = params.get('refresh_token');

    if (oauthAccessToken) {
        api.setToken(oauthAccessToken);
        localStorage.setItem(STORAGE_KEYS.TOKEN, oauthAccessToken);
        if (oauthRefreshToken) {
            api.setRefreshToken(oauthRefreshToken);
            localStorage.setItem(STORAGE_KEYS.REFRESH_TOKEN, oauthRefreshToken);
        }
        // 清除 URL 中的 token 參數
        window.history.replaceState({}, document.title, window.location.pathname);
        try {
            await loadUserData();
            return;
        } catch (error) {
            console.error('OAuth login callback failed:', error);
            showAuthPage();
            return;
        }
    }

    const token = localStorage.getItem(STORAGE_KEYS.TOKEN);

    if (token) {
        api.setToken(token);
        try {
            await loadUserData();
        } catch (error) {
            console.error('Stored token failed:', error);
            showAuthPage();
        }
    } else {
        showAuthPage();
    }
}

// Google OAuth 登入
function loginWithGoogle() {
    window.location.href = '/api/v1/auth/google';
}

// 載入使用者資料
async function loadUserData() {
    try {
        showLoading(true);
        const response = await api.getCurrentUser();
        appState.user = response.user;

        // 連接 WebSocket
        wsManager.connect(api.token);

        // 載入社群
        await loadGuilds();

        showAppPage();
        updateUserPanel();

        // 嘗試恢復上次的社群和頻道
        const lastGuildId = localStorage.getItem(STORAGE_KEYS.LAST_GUILD);
        if (lastGuildId) {
            const guild = appState.guilds.find(g => g.id === parseInt(lastGuildId));
            if (guild) {
                await selectGuild(guild.id);

                const lastChannelId = localStorage.getItem(STORAGE_KEYS.LAST_CHANNEL);
                if (lastChannelId) {
                    const channel = appState.channels.find(c => c.id === parseInt(lastChannelId));
                    if (channel) {
                        selectChannel(channel.id);
                    }
                }
            }
        }
    } catch (error) {
        console.error('Failed to load user data:', error);
        showNotification('載入使用者資料失敗', 'error');
        handleLogout();
    } finally {
        showLoading(false);
    }
}

// 載入社群列表
async function loadGuilds() {
    try {
        const response = await api.getMyGuilds();
        // 後端直接返回陣列，不是包裹在物件中
        appState.guilds = Array.isArray(response) ? response : (response.guilds || []);
        renderGuilds();
    } catch (error) {
        console.error('Failed to load guilds:', error);
        showNotification('載入社群列表失敗', 'error');
    }
}

// 選擇社群
async function selectGuild(guildId) {
    try {
        showLoading(true);

        // 獲取社群詳情
        const guild = await api.getGuild(guildId);
        appState.currentGuild = guild;

        // 獲取頻道列表
        const channels = await api.getGuildChannels(guildId);
        appState.channels = Array.isArray(channels) ? channels : (channels.channels || []);

        // 獲取成員列表
        const members = await api.getGuildMembers(guildId);
        appState.members = Array.isArray(members) ? members : (members.members || []);

        // 預先填入成員用戶資料到快取
        userCache.clear();
        appState.members.forEach(m => { if (m.user) cacheUser(m.user); });
        if (appState.user) cacheUser(appState.user);

        // 更新 UI
        updateGuildHeader();
        renderChannels();
        renderMembers();

        // 儲存到本地
        localStorage.setItem(STORAGE_KEYS.LAST_GUILD, guildId);

        // 更新社群按鈕狀態
        document.querySelectorAll('.guild-item').forEach(item => {
            item.classList.remove('active');
        });
        const guildElement = document.querySelector(`[data-guild-id="${guildId}"]`);
        if (guildElement) {
            guildElement.classList.add('active');
        }
    } catch (error) {
        console.error('Failed to select guild:', error);
        showNotification('載入社群失敗', 'error');
    } finally {
        showLoading(false);
    }
}

// 選擇頻道
async function selectChannel(channelId) {
    try {
        showLoading(true);

        // 取消訂閱舊頻道
        if (appState.currentChannel) {
            wsManager.unsubscribeFromChannel(appState.currentChannel.id);
        }

        // 獲取頻道詳情
        const channel = await api.getChannel(channelId);
        appState.currentChannel = channel;

        // 訂閱新頻道
        wsManager.subscribeToChannel(channelId);

        // 載入訊息
        await loadMessages(channelId);

        // 更新 UI
        updateChannelHeader();
        renderMessages();
        scrollToBottom();

        // 儲存到本地
        localStorage.setItem(STORAGE_KEYS.LAST_CHANNEL, channelId);

        // 更新頻道按鈕狀態
        document.querySelectorAll('.channel-item').forEach(item => {
            item.classList.remove('active');
        });
        const channelElement = document.querySelector(`[data-channel-id="${channelId}"]`);
        if (channelElement) {
            channelElement.classList.add('active');
        }

        // 聚焦輸入框
        document.getElementById('message-input').focus();
    } catch (error) {
        console.error('Failed to select channel:', error);
        showNotification('載入頻道失敗', 'error');
    } finally {
        showLoading(false);
    }
}

// 載入訊息
async function loadMessages(channelId, before = null) {
    try {
        const response = await api.getChannelMessages(channelId, 50, before);

        // 後端返回 response 物件，包含 messages 陣列（已按 ID 升序排列）
        const messages = response.messages || [];

        if (before) {
            // 載入更多訊息（往前）：舊訊息在前，現有訊息在後
            appState.messages = [...messages, ...appState.messages];
        } else {
            // 首次載入
            appState.messages = messages;
        }

        return messages;
    } catch (error) {
        console.error('Failed to load messages:', error);
        showNotification('載入訊息失敗', 'error');
        return [];
    }
}

// 發送訊息
async function sendMessage() {
    const input = document.getElementById('message-input');
    const content = input.value.trim();
    const fileIds = [...appState.pendingFileIds];

    if (!content && fileIds.length === 0) return;
    if (!appState.currentChannel) return;

    // 產生冪等 nonce（UUID v4）
    // crypto.randomUUID() 僅在 secure context（HTTPS/localhost）可用，提供 fallback
    const nonce = (typeof crypto.randomUUID === 'function')
        ? crypto.randomUUID()
        : ([1e7] + -1e3 + -4e3 + -8e3 + -1e11).replace(/[018]/g, c =>
            (c ^ (crypto.getRandomValues(new Uint8Array(1))[0] & (15 >> (c / 4)))).toString(16)
        );

    // Optimistic UI：先在畫面顯示「發送中」訊息
    const optimisticMsg = {
        id: null,
        nonce,
        channel_id: appState.currentChannel.id,
        user_id: appState.user.id,
        user: appState.user,
        content: content || '',
        type: 'text',
        is_edited: false,
        attachments: [],
        created_at: new Date().toISOString(),
        _pending: true  // 標記為待確認
    };
    appState.messages.push(optimisticMsg);
    renderMessages();
    scrollToBottom();

    input.value = '';
    autoResizeTextarea(input);
    input.focus();

    // 清除已附加的檔案
    appState.pendingFileIds = [];
    clearFilePreview();

    try {
        await api.sendMessage(appState.currentChannel.id, content || '', 'text', nonce, fileIds);
        // 成功後 server 會透過 WS 廣播 message_create（含 nonce），
        // handleNewMessage 會用 nonce 把 optimistic message 替換掉
    } catch (error) {
        console.error('Failed to send message:', error);
        showNotification('發送訊息失敗', 'error');
        // 移除 optimistic message
        const idx = appState.messages.findIndex(m => m.nonce === nonce && m._pending);
        if (idx !== -1) appState.messages.splice(idx, 1);
        renderMessages();
        input.value = content; // 恢復輸入
        autoResizeTextarea(input);
        // 恢復待附加檔案
        appState.pendingFileIds = fileIds;
        renderFilePreview();
    }
}

// 自動調整 textarea 高度
function autoResizeTextarea(el) {
    el.style.height = 'auto';
    el.style.height = Math.min(el.scrollHeight, 200) + 'px';
}

// 處理訊息輸入鍵盤事件
function handleMessageKeyPress(event) {
    if (event.key === 'Enter' && !event.shiftKey) {
        event.preventDefault();
        sendMessage();
    }
}

// ===================== 檔案上傳相關 =====================

// 當使用者選取檔案後觸發
async function handleFileSelected(input) {
    const file = input.files[0];
    if (!file) return;
    // 重置 input 以便同一個檔案可以再次選取
    input.value = '';

    await uploadFile(file);
}

// 執行檔案上傳流程：presign → PUT → confirm
async function uploadFile(file) {
    if (!appState.currentChannel) {
        showNotification('請先選擇一個頻道', 'error');
        return;
    }

    // 先在預覽區顯示進度
    const previewId = `upload-${Date.now()}`;
    addFilePreviewChip(previewId, file.name, 0);

    try {
        // Step 1: 取得 pre-signed URL
        const presignResp = await api.presignUpload(file.name, file.type || 'application/octet-stream', file.size);
        const { file_id, upload_url } = presignResp;

        // Step 2: 直接 PUT 至 Minio，更新進度
        await api.uploadToMinio(upload_url, file, (pct) => {
            updateFilePreviewProgress(previewId, pct);
        });

        // Step 3: 通知 server 確認上傳完成
        await api.confirmUpload(file_id);

        // 標記為完成並記錄 file_id
        markFilePreviewDone(previewId, file.name, file_id);
        appState.pendingFileIds.push(file_id);

    } catch (err) {
        console.error('File upload failed:', err);
        showNotification(`上傳失敗：${err.message}`, 'error');
        removeFilePreviewChip(previewId);
    }
}

// ── 預覽 Chip 操作 ──

function getPreviewArea() {
    return document.getElementById('file-preview-area');
}

function addFilePreviewChip(id, filename, progress) {
    const area = getPreviewArea();
    area.style.display = 'flex';
    const chip = document.createElement('div');
    chip.className = 'file-preview-chip';
    chip.id = id;
    chip.innerHTML = `
        <i class="fas fa-spinner fa-spin"></i>
        <span class="chip-name">${escapeHtml(filename)}</span>
        <span class="chip-progress">${progress}%</span>
    `;
    area.appendChild(chip);
}

function updateFilePreviewProgress(id, pct) {
    const chip = document.getElementById(id);
    if (!chip) return;
    const prog = chip.querySelector('.chip-progress');
    if (prog) prog.textContent = `${pct}%`;
}

function markFilePreviewDone(id, filename, fileId) {
    const chip = document.getElementById(id);
    if (!chip) return;
    chip.dataset.fileId = fileId;
    chip.innerHTML = `
        <i class="fas fa-paperclip"></i>
        <span class="chip-name">${escapeHtml(filename)}</span>
        <button class="chip-remove" onclick="removeUploadedFile('${id}','${fileId}')" title="移除">
            <i class="fas fa-times"></i>
        </button>
    `;
    chip.classList.add('file-preview-chip--done');
}

function removeFilePreviewChip(id) {
    const chip = document.getElementById(id);
    if (chip) chip.remove();
    const area = getPreviewArea();
    if (area && area.children.length === 0) area.style.display = 'none';
}

// 使用者點擊移除按鈕
function removeUploadedFile(chipId, fileId) {
    appState.pendingFileIds = appState.pendingFileIds.filter(id => String(id) !== String(fileId));
    removeFilePreviewChip(chipId);
}

function clearFilePreview() {
    const area = getPreviewArea();
    if (!area) return;
    area.innerHTML = '';
    area.style.display = 'none';
}

function renderFilePreview() {
    // 若有需要可重繪，目前預覽已在上傳成功時即時建立，此處保留為 hook
}

// ── 訊息附件渲染 ──

const IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/gif', 'image/webp', 'image/svg+xml'];

function renderAttachments(attachments) {
    if (!attachments || attachments.length === 0) return '';

    const html = attachments.map(att => {
        if (!att || !att.file) return '';
        const file = att.file;
        const isImage = IMAGE_TYPES.includes(file.content_type);
        const safeName = escapeHtml(file.filename || 'attachment');

        if (isImage) {
            return `
                <div class="message-attachment message-attachment--image"
                     onclick="openLightbox(${file.id}, this)">
                    <img src=""
                         alt="${safeName}"
                         data-file-id="${file.id}"
                         data-load-image="1"
                         title="${safeName}">
                    <div class="img-overlay"><i class="fas fa-expand"></i></div>
                </div>`;
        }

        // 非圖片：顯示下載連結
        const sizeStr = formatFileSize(file.size || 0);
        return `
            <div class="message-attachment message-attachment--file">
                <i class="fas fa-file"></i>
                <div class="attachment-info">
                    <span class="attachment-name">${safeName}</span>
                    <span class="attachment-size">${sizeStr}</span>
                </div>
                <button class="btn-icon attachment-download" onclick="openAttachment(${file.id})" title="下載">
                    <i class="fas fa-download"></i>
                </button>
            </div>`;
    }).join('');

    return html ? `<div class="message-attachments">${html}</div>` : '';
}

// 點擊附件時開啟下載 URL
async function openAttachment(fileId) {
    try {
        const resp = await api.getFileDownloadUrl(fileId);
        window.open(resp.url, '_blank');
    } catch (err) {
        showNotification('無法取得檔案連結', 'error');
    }
}

// 開啟 Lightbox
async function openLightbox(fileId, boxEl) {
    const img = boxEl?.querySelector('img');
    const overlay = document.getElementById('lightbox-overlay');
    const lbImg = document.getElementById('lightbox-img');
    if (!overlay || !lbImg) return;

    // 如果小圖已載入則直接用其 src，否則重新取得
    if (img && img.src && !img.src.endsWith('/')) {
        lbImg.src = img.src;
    } else {
        try {
            const resp = await api.getFileDownloadUrl(fileId);
            lbImg.src = resp.url;
        } catch (_) {
            showNotification('無法載入圖片', 'error');
            return;
        }
    }

    overlay.classList.add('active');
    document.addEventListener('keydown', _lightboxKeyHandler);
}

function closeLightbox() {
    const overlay = document.getElementById('lightbox-overlay');
    const lbImg = document.getElementById('lightbox-img');
    if (overlay) overlay.classList.remove('active');
    if (lbImg) lbImg.src = '';
    document.removeEventListener('keydown', _lightboxKeyHandler);
}

function _lightboxKeyHandler(e) {
    if (e.key === 'Escape') closeLightbox();
}

// 載入圖片附件的 src（非同步取得簽名 URL）
async function loadAttachmentImage(fileId) {
    try {
        const resp = await api.getFileDownloadUrl(fileId);
        document.querySelectorAll(`img[data-file-id="${fileId}"]`).forEach(img => {
            img.onload = () => {
                img.style.opacity = '1';
                img.closest('.message-attachment--image')?.classList.add('loaded');
            };
            img.src = resp.url;
        });
    } catch (err) {
        console.warn('Failed to load image attachment:', fileId, err);
    }
}

// 格式化檔案大小
function formatFileSize(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
}

// ===================== END 檔案上傳相關 =====================

// 簡易 Markdown 渲染（支援: ``` 程式碼區塊, ` 行內程式碼, - 列舉, 換行）
function renderMarkdown(rawText) {
    // 以 ``` 分割，奇數索引為程式碼區塊
    const parts = rawText.split(/(```[\s\S]*?```)/g);

    return parts.map((part, i) => {
        if (i % 2 === 1) {
            // 程式碼區塊
            let inner = part.slice(3, -3);
            // 移除可能的語言標記首行（e.g. ```js\n...）
            const firstNewline = inner.indexOf('\n');
            if (firstNewline > 0 && /^\w+$/.test(inner.slice(0, firstNewline).trim())) {
                inner = inner.slice(firstNewline + 1);
            }
            return `<pre class="md-code-block"><code>${escapeHtml(inner)}</code></pre>`;
        }

        let html = escapeHtml(part);

        // 行內程式碼
        html = html.replace(/`([^`\n]+)`/g, '<code class="md-inline-code">$1</code>');

        // 逐行處理：列舉 & 換行
        const lines = html.split('\n');
        let inList = false;
        let out = '';

        for (let j = 0; j < lines.length; j++) {
            const line = lines[j];
            const isLast = j === lines.length - 1;

            if (/^- /.test(line)) {
                if (!inList) { out += '<ul class="md-list">'; inList = true; }
                out += `<li>${line.slice(2)}</li>`;
            } else {
                if (inList) { out += '</ul>'; inList = false; }
                out += line;
                if (!isLast) out += '<br>';
            }
        }
        if (inList) out += '</ul>';

        return out;
    }).join('');
}

// 設定 WebSocket 處理器
function setupWebSocketHandlers() {
    wsManager.onMessage((type, data) => {
        switch (type) {
            case 'message':
                handleNewMessage(data);
                break;
            case 'message_update':
                handleMessageUpdate(data);
                break;
            case 'message_delete':
                handleMessageDelete(data);
                break;
            case 'typing':
                handleTyping(data);
                break;
            case 'user_status':
                handleUserStatus(data);
                break;
            case 'channel_create':
                handleChannelCreate(data);
                break;
            case 'channel_update':
                handleChannelUpdate(data);
                break;
            case 'channel_delete':
                handleChannelDelete(data);
                break;
            case 'guild_update':
                handleGuildUpdate(data);
                break;
            case 'guild_delete':
                handleGuildDelete(data);
                break;
            case 'guild_member_add':
                handleGuildMemberAdd(data);
                break;
            case 'guild_member_remove':
                handleGuildMemberRemove(data);
                break;
            case 'guild_member_update':
                handleGuildMemberUpdate(data);
                break;
        }
    });
}

// 處理新訊息
function handleNewMessage(message) {
    // 只處理當前頻道的訊息
    if (!appState.currentChannel || message.channel_id !== appState.currentChannel.id) {
        return;
    }

    // 用 nonce 替換 optimistic message（避免重複顯示）
    if (message.nonce) {
        const idx = appState.messages.findIndex(m => m.nonce === message.nonce && m._pending);
        if (idx !== -1) {
            appState.messages[idx] = message;
            renderMessages();
            return;
        }
        // 防止重複：若已存在相同 nonce 的確認訊息，直接忽略
        if (appState.messages.some(m => m.nonce === message.nonce && !m._pending)) {
            return;
        }
    }

    appState.messages.push(message);
    renderMessages();
    scrollToBottom();
}

// 處理訊息更新
function handleMessageUpdate(message) {
    const index = appState.messages.findIndex(m => m.id === message.id);
    if (index !== -1) {
        appState.messages[index] = message;
        renderMessages();
    }
}

// 處理訊息刪除
function handleMessageDelete(data) {
    const index = appState.messages.findIndex(m => m.id === data.message_id);
    if (index !== -1) {
        appState.messages.splice(index, 1);
        renderMessages();
    }
}

// 處理正在輸入
function handleTyping(data) {
    // TODO: 顯示正在輸入指示器
    console.log(`${data.username} is typing...`);
}

// 處理使用者狀態更新
function handleUserStatus(data) {
    // 更新成員列表中的使用者狀態
    const member = appState.members.find(m => m.user_id === data.user_id);
    if (member && member.user) {
        member.user.status = data.status;
        renderMembers();
    }
}

// 處理頻道建立
function handleChannelCreate(channel) {
    if (appState.currentGuild && channel.guild_id === appState.currentGuild.id) {
        appState.channels.push(channel);
        renderChannels();
    }
}

// 處理頻道更新
function handleChannelUpdate(channel) {
    const index = appState.channels.findIndex(c => c.id === channel.id);
    if (index !== -1) {
        appState.channels[index] = channel;
        renderChannels();

        if (appState.currentChannel && appState.currentChannel.id === channel.id) {
            appState.currentChannel = channel;
            updateChannelHeader();
        }
    }
}

// 處理頻道刪除
function handleChannelDelete(data) {
    const index = appState.channels.findIndex(c => c.id === data.channel_id);
    if (index !== -1) {
        appState.channels.splice(index, 1);
        renderChannels();

        if (appState.currentChannel && appState.currentChannel.id === data.channel_id) {
            appState.currentChannel = null;
            appState.messages = [];
            updateChannelHeader();
            renderMessages();
        }
    }
}

// 處理社群更新
function handleGuildUpdate(guild) {
    const index = appState.guilds.findIndex(g => g.id === guild.id);
    if (index !== -1) {
        appState.guilds[index] = guild;
        renderGuilds();
        if (appState.currentGuild && appState.currentGuild.id === guild.id) {
            appState.currentGuild = guild;
            updateGuildHeader();
        }
    }
}

// 處理社群刪除
function handleGuildDelete(data) {
    const guildId = data.guild_id;
    appState.guilds = appState.guilds.filter(g => g.id !== guildId);
    renderGuilds();
    if (appState.currentGuild && appState.currentGuild.id === guildId) {
        showHomeView();
        showNotification('所在社群已被刪除', 'info');
    }
}

// 處理成員加入
function handleGuildMemberAdd(data) {
    if (!appState.currentGuild || data.guild_id !== appState.currentGuild.id) return;
    const exists = appState.members.some(m => m.user_id === data.user_id);
    if (!exists) {
        appState.members.push(data);
        renderMembers();
    }
}

// 處理成員離開
function handleGuildMemberRemove(data) {
    if (!appState.currentGuild || data.guild_id !== appState.currentGuild.id) return;
    appState.members = appState.members.filter(m => m.user_id !== data.user_id);
    renderMembers();
    // 若被踢出的是自己
    if (appState.user && data.user_id === appState.user.id) {
        appState.guilds = appState.guilds.filter(g => g.id !== data.guild_id);
        renderGuilds();
        showHomeView();
        showNotification('您已被移出該社群', 'info');
    }
}

// 處理成員更新
function handleGuildMemberUpdate(data) {
    if (!appState.currentGuild || data.guild_id !== appState.currentGuild.id) return;
    const index = appState.members.findIndex(m => m.user_id === data.user_id);
    if (index !== -1) {
        appState.members[index] = { ...appState.members[index], ...data };
        renderMembers();
    }
}

// 渲染社群列表
function renderGuilds() {
    const container = document.getElementById('guilds-list');
    container.innerHTML = '';

    appState.guilds.forEach(guild => {
        const guildElement = document.createElement('div');
        guildElement.className = 'guild-item';
        guildElement.setAttribute('data-guild-id', guild.id);
        guildElement.title = guild.name;
        guildElement.onclick = () => selectGuild(guild.id);

        if (guild.icon) {
            guildElement.innerHTML = `<img src="${guild.icon}" alt="${guild.name}">`;
        } else {
            // 使用社群名稱的首字母
            guildElement.textContent = guild.name.charAt(0).toUpperCase();
        }

        container.appendChild(guildElement);
    });
}

// 渲染頻道列表
function renderChannels() {
    const textChannels = appState.channels.filter(c => c.type === 'text');
    const voiceChannels = appState.channels.filter(c => c.type === 'voice');

    renderChannelList('text-channels-list', textChannels, 'hashtag');
    renderChannelList('voice-channels-list', voiceChannels, 'volume-up');
}

// 渲染頻道列表（輔助函數）
function renderChannelList(containerId, channels, iconClass) {
    const container = document.getElementById(containerId);
    container.innerHTML = '';

    channels.forEach(channel => {
        const channelElement = document.createElement('div');
        channelElement.className = 'channel-item';
        channelElement.setAttribute('data-channel-id', channel.id);
        channelElement.onclick = () => selectChannel(channel.id);

        channelElement.innerHTML = `
            <i class="fas fa-${iconClass}"></i>
            <span>${channel.name}</span>
        `;

        container.appendChild(channelElement);
    });
}

// 取得目前使用者在社群中的角色
function getCurrentUserRole() {
    if (!appState.user || !appState.members.length) return null;
    const me = appState.members.find(m => m.user_id === appState.user.id);
    return me ? me.role : null;
}

// 角色階層
const ROLE_LEVEL = { owner: 4, admin: 3, moderator: 2, member: 1 };

// 渲染成員列表
function renderMembers() {
    const container = document.getElementById('members-list');
    container.innerHTML = '';

    const myRole = getCurrentUserRole();
    const myLevel = ROLE_LEVEL[myRole] || 0;

    appState.members.forEach(member => {
        const memberElement = document.createElement('div');
        memberElement.className = 'member-item';

        const user = member.user || {};
        const status = user.status || 'offline';
        const nickname = user.nickname || user.username || 'Unknown';
        const role = member.role || 'member';
        const memberLevel = ROLE_LEVEL[role] || 0;

        const roleLabel = { owner: '擁有者', admin: '管理員', moderator: '版主', member: '' }[role] || '';
        const roleBadge = roleLabel ? `<span class="role-badge role-${role}">${roleLabel}</span>` : '';

        // admin/owner 可管理低階成員
        const canManage = myLevel >= ROLE_LEVEL.admin && memberLevel < myLevel && appState.user && member.user_id !== appState.user.id;
        const adminActions = canManage ? `
            <div class="member-actions">
                <button class="btn-icon-sm" title="更改角色" onclick="handleUpdateMemberRole(${appState.currentGuild.id}, ${member.user_id}, '${escapeHtml(nickname)}', '${role}')">
                    <i class="fas fa-user-shield"></i>
                </button>
                <button class="btn-icon-sm danger" title="移出社群" onclick="handleKickMember(${appState.currentGuild.id}, ${member.user_id}, '${escapeHtml(nickname)}')">
                    <i class="fas fa-user-times"></i>
                </button>
            </div>` : '';

        memberElement.innerHTML = `
            <div class="member-avatar">
                ${user.avatar ? `<img src="${user.avatar}" alt="${nickname}">` : '<i class="fas fa-user"></i>'}
                <span class="status-indicator ${status}"></span>
            </div>
            <div class="member-info">
                <div class="member-name">${escapeHtml(nickname)} ${roleBadge}</div>
            </div>
            ${adminActions}
        `;

        container.appendChild(memberElement);
    });
}

// 本地用戶快取（補充 API 沒有攜帶 user 的邊界情況，如歷史遺留資料）
const userCache = new Map();

// 將用戶資料存入快取
function cacheUser(user) {
    if (user && user.id) userCache.set(user.id, user);
}

// 解析訊息的發送者用戶資料
// 優先使用 message.user（後端 Preload），其次成員列表，最後快取 / API fallback
function resolveMessageUser(message) {
    // 1. 訊息本身攜帶 user（正常路徑）
    if (message.user && message.user.id) return message.user;
    const userId = message.user_id;
    // 2. 本地快取（含先前 API 取回的離群用戶）
    if (userCache.has(userId)) return userCache.get(userId);
    // 3. 成員列表
    const member = appState.members.find(m => m.user_id === userId);
    if (member && member.user) { cacheUser(member.user); return member.user; }
    // 4. 自己
    if (appState.user && appState.user.id === userId) { cacheUser(appState.user); return appState.user; }
    // 5. 非同步取回離群用戶，取回後重渲染
    fetchAndCacheUser(userId);
    return null;
}

// 非同步取回離群用戶資料，完成後重渲染訊息
async function fetchAndCacheUser(userId) {
    if (userCache.has(`pending:${userId}`)) return;
    userCache.set(`pending:${userId}`, true);
    try {
        const user = await api.getPublicUser(userId);
        cacheUser(user);
        renderMessages();
    } catch (_) {
        cacheUser({ id: userId, username: 'Unknown', nickname: 'Unknown', avatar: null });
    } finally {
        userCache.delete(`pending:${userId}`);
    }
}

// 渲染訊息列表
function renderMessages() {
    const container = document.getElementById('messages-container');

    if (!appState.currentChannel) {
        container.innerHTML = `
            <div class="welcome-message">
                <h1>歡迎來到 TalkRealm！</h1>
                <p>選擇一個頻道開始聊天，或建立一個新的社群。</p>
            </div>
        `;
        return;
    }

    if (appState.messages.length === 0) {
        container.innerHTML = `
            <div class="welcome-message">
                <h1>歡迎來到 #${appState.currentChannel.name}</h1>
                <p>這是 #${appState.currentChannel.name} 頻道的開始。</p>
            </div>
        `;
        return;
    }

    container.innerHTML = '';

    appState.messages.forEach((message, index) => {
        const prevMessage = index > 0 ? appState.messages[index - 1] : null;
        const isGrouped = prevMessage &&
            prevMessage.user_id === message.user_id;

        const messageElement = document.createElement('div');
        messageElement.className = `message${message._pending ? ' message--pending' : ''}`;
        messageElement.setAttribute('data-message-id', message.id || '');
        if (message.nonce) messageElement.setAttribute('data-nonce', message.nonce);

        const user = resolveMessageUser(message) || { username: 'Unknown', nickname: 'Unknown', avatar: null };
        const nickname = user.nickname || user.username || 'Unknown';
        const avatar = user.avatar;
        const timestamp = formatTimestamp(message.created_at);

        if (isGrouped) {
            messageElement.innerHTML = `
                <div class="message-avatar-spacer" aria-hidden="true"></div>
                <div class="message-content">
                    <div class="message-text">${renderMarkdown(message.content)}</div>
                    ${renderAttachments(message.attachments)}
                </div>
            `;
        } else {
            messageElement.innerHTML = `
                <div class="message-avatar">
                    ${avatar ? `<img src="${avatar}" alt="${nickname}">` : '<i class="fas fa-user"></i>'}
                </div>
                <div class="message-content">
                    <div class="message-header">
                        <span class="message-author">${escapeHtml(nickname)}</span>
                        <span class="message-timestamp">${timestamp}</span>
                    </div>
                    <div class="message-text">${renderMarkdown(message.content)}</div>
                    ${renderAttachments(message.attachments)}
                </div>
            `;
        }

        container.appendChild(messageElement);
    });

    // 觸發所有圖片附件載入（innerHTML 裡的 <script> 不會執行，改在此集中呼叫）
    container.querySelectorAll('img[data-load-image]').forEach(img => {
        const fileId = parseInt(img.getAttribute('data-file-id'), 10);
        if (fileId) loadAttachmentImage(fileId);
    });
}

// 更新社群標題
function updateGuildHeader() {
    const guildName = document.getElementById('guild-name');

    if (appState.currentGuild) {
        guildName.textContent = appState.currentGuild.name;
    } else {
        guildName.textContent = '選擇一個社群';
    }
}

// 更新頻道標題
function updateChannelHeader() {
    const channelIcon = document.getElementById('channel-icon');
    const channelName = document.getElementById('channel-name');
    const channelTopic = document.getElementById('channel-topic');

    if (appState.currentChannel) {
        channelIcon.className = appState.currentChannel.type === 'voice' ? 'fas fa-volume-up' : 'fas fa-hashtag';
        channelName.textContent = appState.currentChannel.name;
        channelTopic.textContent = appState.currentChannel.topic || '';
    } else {
        channelIcon.className = 'fas fa-hashtag';
        channelName.textContent = '歡迎';
        channelTopic.textContent = '';
    }
}

// 更新使用者面板
function updateUserPanel() {
    if (!appState.user) return;

    const userName = document.getElementById('user-name');
    const userStatus = document.getElementById('user-status');
    const userAvatar = document.getElementById('user-avatar');

    userName.textContent = appState.user.nickname || appState.user.username;

    const status = appState.user.status || 'online';
    userStatus.innerHTML = `
        <span class="status-indicator ${status}"></span>
        <span>${getStatusText(status)}</span>
    `;

    if (appState.user.avatar) {
        userAvatar.innerHTML = `<img src="${appState.user.avatar}" alt="${appState.user.username}">`;
    }
}

// 登入處理
async function handleLogin(event) {
    event.preventDefault();

    const email = document.getElementById('login-email').value;
    const password = document.getElementById('login-password').value;

    try {
        showLoading(true);
        const response = await api.login(email, password);

        appState.user = response.user;
        showNotification('登入成功！', 'success');

        // 延遲一下再載入，讓通知顯示
        setTimeout(() => {
            loadUserData();
        }, 500);
    } catch (error) {
        console.error('Login failed:', error);
        showNotification(error.message || '登入失敗', 'error');
    } finally {
        showLoading(false);
    }
}

// 註冊處理
async function handleRegister(event) {
    event.preventDefault();

    const username = document.getElementById('register-username').value;
    const email = document.getElementById('register-email').value;
    const password = document.getElementById('register-password').value;
    const nickname = document.getElementById('register-nickname').value;

    try {
        showLoading(true);
        await api.register(username, email, password, nickname);

        showNotification('註冊成功！正在登入...', 'success');

        // 自動登入
        setTimeout(async () => {
            try {
                const response = await api.login(email, password);
                appState.user = response.user;
                loadUserData();
            } catch (error) {
                showNotification('請手動登入', 'info');
                switchToLogin();
            }
        }, 1000);
    } catch (error) {
        console.error('Registration failed:', error);
        showNotification(error.message || '註冊失敗', 'error');
    } finally {
        showLoading(false);
    }
}

// 登出處理
async function handleLogout() {
    // 呼叫後端撤銷 refresh token
    await api.logout().catch(() => { });

    // 斷開 WebSocket
    wsManager.disconnect();

    // 清除狀態
    appState.user = null;
    appState.currentGuild = null;
    appState.currentChannel = null;
    appState.guilds = [];
    appState.channels = [];
    appState.members = [];
    appState.messages = [];

    // 清除本地儲存
    localStorage.removeItem(STORAGE_KEYS.TOKEN);
    localStorage.removeItem(STORAGE_KEYS.REFRESH_TOKEN);
    localStorage.removeItem(STORAGE_KEYS.USER);

    showAuthPage();
    showNotification('已登出', 'info');
}

// 建立社群
async function handleCreateGuild(event) {
    event.preventDefault();

    const name = document.getElementById('guild-name-input').value;
    const description = document.getElementById('guild-description-input').value;

    try {
        showLoading(true);
        const guild = await api.createGuild(name, description);

        showNotification('社群建立成功！', 'success');
        closeModal('create-guild-modal');

        // 後端直接返回 guild 物件，手動添加到列表並渲染
        if (guild && guild.id) {
            appState.guilds.push(guild);
            renderGuilds();

            // 自動選擇新建立的社群
            selectGuild(guild.id);
        }
    } catch (error) {
        console.error('Failed to create guild:', error);
        showNotification(error.message || '建立社群失敗', 'error');
    } finally {
        showLoading(false);
    }
}

// 更新社群
async function handleUpdateGuild(event) {
    event.preventDefault();
    if (!appState.currentGuild) return;

    const name = document.getElementById('guild-edit-name').value;
    const description = document.getElementById('guild-edit-description').value;

    try {
        showLoading(true);
        const guild = await api.updateGuild(appState.currentGuild.id, { name, description });
        appState.currentGuild = guild;
        // 更新社群列表中的名稱
        const idx = appState.guilds.findIndex(g => g.id === guild.id);
        if (idx !== -1) appState.guilds[idx] = guild;
        renderGuilds();
        updateGuildHeader();
        showNotification('社群資訊已更新', 'success');
    } catch (error) {
        showNotification(error.message || '更新失敗', 'error');
    } finally {
        showLoading(false);
    }
}

// 刪除社群
async function handleDeleteGuild() {
    if (!appState.currentGuild) return;
    if (!confirm(`確定要刪除「${appState.currentGuild.name}」？此操作無法復原。`)) return;

    try {
        showLoading(true);
        await api.deleteGuild(appState.currentGuild.id);
        closeModal('guild-settings-modal');
        appState.guilds = appState.guilds.filter(g => g.id !== appState.currentGuild.id);
        showNotification('社群已刪除', 'success');
        showHomeView();
        renderGuilds();
    } catch (error) {
        showNotification(error.message || '刪除失敗', 'error');
    } finally {
        showLoading(false);
    }
}

// 建立邀請碼
async function handleCreateInvite() {
    if (!appState.currentGuild) return;
    try {
        showLoading(true);
        const invite = await api.createInvite(appState.currentGuild.id);
        const code = invite.code || invite.invite_code || invite;
        document.getElementById('invite-code-value').textContent = code;
        document.getElementById('invite-code-display').classList.remove('hidden');
        showNotification('邀請碼已建立', 'success');
    } catch (error) {
        showNotification(error.message || '建立邀請碼失敗', 'error');
    } finally {
        showLoading(false);
    }
}

// 複製邀請碼
function copyInviteCode() {
    const code = document.getElementById('invite-code-value').textContent;
    if (!code) return;
    navigator.clipboard.writeText(code).then(() => {
        showNotification('邀請碼已複製', 'success');
    }).catch(() => {
        // 降級方案
        const el = document.getElementById('invite-code-value');
        const range = document.createRange();
        range.selectNodeContents(el);
        window.getSelection().removeAllRanges();
        window.getSelection().addRange(range);
        showNotification('請手動複製選取的文字', 'info');
    });
}

// 透過邀請碼加入社群
async function handleJoinByInvite(event) {
    event.preventDefault();
    const code = document.getElementById('invite-code-input').value.trim();
    if (!code) return;

    try {
        showLoading(true);
        await api.joinByInvite(code);
        closeModal('join-invite-modal');
        showNotification('加入社群成功！', 'success');
        // 重新載入社群列表
        await loadGuilds();
    } catch (error) {
        showNotification(error.message || '加入失敗，請檢查邀請碼', 'error');
    } finally {
        showLoading(false);
    }
}

// 踢除成員
async function handleKickMember(guildId, userId, username) {
    if (!confirm(`確定要將「${username}」移出社群？`)) return;
    try {
        showLoading(true);
        await api.kickMember(guildId, userId);
        appState.members = appState.members.filter(m => m.user_id !== userId);
        renderMembers();
        showNotification(`已移除成員 ${username}`, 'success');
    } catch (error) {
        showNotification(error.message || '移除失敗', 'error');
    } finally {
        showLoading(false);
    }
}

// 更新成員角色
async function handleUpdateMemberRole(guildId, userId, username, currentRole) {
    const roleOptions = currentRole === 'member'
        ? { '角色模版人': 'moderator' }
        : { '一般成員': 'member' };
    // 簡單切換：moderator <-> member
    const newRole = currentRole === 'member' ? 'moderator' : 'member';
    const label = newRole === 'moderator' ? '角色模版人' : '一般成員';
    if (!confirm(`將「${username}」的角色變更為 ${label}？`)) return;

    try {
        showLoading(true);
        await api.updateMemberRole(guildId, userId, newRole);
        const member = appState.members.find(m => m.user_id === userId);
        if (member) member.role = newRole;
        renderMembers();
        showNotification(`已更新 ${username} 的角色`, 'success');
    } catch (error) {
        showNotification(error.message || '更新角色失敗', 'error');
    } finally {
        showLoading(false);
    }
}

// 建立頻道
async function handleCreateChannel(event) {
    event.preventDefault();

    if (!appState.currentGuild) {
        showNotification('請先選擇一個社群', 'error');
        return;
    }

    const name = document.getElementById('channel-name-input').value;
    const type = document.getElementById('channel-type-input').value;
    const description = document.getElementById('channel-description-input').value;

    try {
        showLoading(true);
        const channel = await api.createChannel(appState.currentGuild.id, name, type, description); // description 傳給後端 topic 欄位（api.js 內部已對應）

        showNotification('頻道建立成功！', 'success');
        closeModal('create-channel-modal');

        // 後端直接返回 channel 物件，手動添加到列表
        if (channel && channel.id) {
            appState.channels.push(channel);
            renderChannels();

            // 自動選擇新建立的頻道
            selectChannel(channel.id);
        }
    } catch (error) {
        console.error('Failed to create channel:', error);
        showNotification(error.message || '建立頻道失敗', 'error');
    } finally {
        showLoading(false);
    }
}

// 更新使用者資訊
async function handleUpdateUser(event) {
    event.preventDefault();

    const nickname = document.getElementById('user-nickname-input').value;
    const avatar = document.getElementById('user-avatar-input').value;
    const status = document.getElementById('user-status-input').value;

    const updates = {};
    if (nickname) updates.nickname = nickname;
    if (avatar) updates.avatar = avatar;
    if (status) updates.status = status;

    try {
        showLoading(true);
        const response = await api.updateCurrentUser(updates);

        appState.user = response.user;
        updateUserPanel();

        showNotification('使用者資訊更新成功！', 'success');
        closeModal('user-settings-modal');
    } catch (error) {
        console.error('Failed to update user:', error);
        showNotification(error.message || '更新失敗', 'error');
    } finally {
        showLoading(false);
    }
}

// UI 輔助函數
function showAuthPage() {
    document.getElementById('auth-page').classList.remove('hidden');
    document.getElementById('app-page').classList.add('hidden');
}

function showAppPage() {
    document.getElementById('auth-page').classList.add('hidden');
    document.getElementById('app-page').classList.remove('hidden');
}

function switchToLogin() {
    document.getElementById('login-form').classList.add('active');
    document.getElementById('register-form').classList.remove('active');
}

function switchToRegister() {
    document.getElementById('login-form').classList.remove('active');
    document.getElementById('register-form').classList.add('active');
}

function showCreateGuildModal() {
    document.getElementById('create-guild-modal').classList.add('active');
    document.getElementById('guild-name-input').value = '';
    document.getElementById('guild-description-input').value = '';
}

function showCreateChannelModal(type) {
    document.getElementById('create-channel-modal').classList.add('active');
    document.getElementById('channel-type-input').value = type;
    document.getElementById('channel-name-input').value = '';
    document.getElementById('channel-description-input').value = '';
}

function showUserSettings() {
    if (!appState.user) return;

    document.getElementById('user-settings-modal').classList.add('active');
    document.getElementById('user-nickname-input').value = appState.user.nickname || '';
    document.getElementById('user-avatar-input').value = appState.user.avatar || '';
    document.getElementById('user-status-input').value = appState.user.status || 'online';
}

function showGuildSettings() {
    if (!appState.currentGuild) return;

    document.getElementById('guild-settings-modal').classList.add('active');
    document.getElementById('guild-edit-name').value = appState.currentGuild.name || '';
    document.getElementById('guild-edit-description').value = appState.currentGuild.description || '';
    // 清除上次的邀請碼
    document.getElementById('invite-code-display').classList.add('hidden');
    document.getElementById('invite-code-value').textContent = '';
}

function showJoinByInviteModal() {
    document.getElementById('join-invite-modal').classList.add('active');
    document.getElementById('invite-code-input').value = '';
}

function showHomeView() {
    appState.currentGuild = null;
    appState.currentChannel = null;
    appState.channels = [];
    appState.members = [];
    appState.messages = [];

    updateGuildHeader();
    updateChannelHeader();
    renderChannels();
    renderMembers();
    renderMessages();

    document.querySelectorAll('.guild-item').forEach(item => {
        item.classList.remove('active');
    });
}

function toggleMembersList() {
    const sidebar = document.getElementById('members-sidebar');
    sidebar.classList.toggle('hidden');
}

function closeModal(modalId) {
    document.getElementById(modalId).classList.remove('active');
}

function showLoading(show) {
    appState.isLoading = show;
    const spinner = document.getElementById('loading-spinner');
    if (show) {
        spinner.classList.remove('hidden');
    } else {
        spinner.classList.add('hidden');
    }
}

function showNotification(message, type = 'info') {
    const notification = document.getElementById('notification');
    notification.textContent = message;
    notification.className = `notification ${type} show`;

    setTimeout(() => {
        notification.classList.remove('show');
    }, 3000);
}

function scrollToBottom() {
    const container = document.getElementById('messages-container');
    container.scrollTop = container.scrollHeight;
}

// 工具函數
function formatTimestamp(timestamp) {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now - date;
    const diffMins = Math.floor(diffMs / 60000);

    if (diffMins < 1) return '剛剛';
    if (diffMins < 60) return `${diffMins} 分鐘前`;

    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `${diffHours} 小時前`;

    const diffDays = Math.floor(diffHours / 24);
    if (diffDays < 7) return `${diffDays} 天前`;

    return date.toLocaleDateString('zh-TW');
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function getStatusText(status) {
    const statusMap = {
        online: '線上',
        offline: '離線',
        away: '離開',
        busy: '忙碌'
    };
    return statusMap[status] || '離線';
}

// 關閉模態視窗（點擊外部）
window.onclick = function (event) {
    if (event.target.classList.contains('modal')) {
        event.target.classList.remove('active');
    }
};
