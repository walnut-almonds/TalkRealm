// 應用程式狀態
const appState = {
    user: null,
    currentGuild: null,
    currentChannel: null,
    guilds: [],
    channels: [],
    members: [],
    messages: [],
    isLoading: false
};

// 初始化應用程式
document.addEventListener('DOMContentLoaded', () => {
    checkAuth();
    setupWebSocketHandlers();
});

// 檢查認證狀態
function checkAuth() {
    const token = localStorage.getItem(STORAGE_KEYS.TOKEN);
    
    if (token) {
        api.setToken(token);
        loadUserData();
    } else {
        showAuthPage();
    }
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
        
        // 後端返回 response 物件，包含 messages 陣列
        const messages = response.messages || [];
        
        if (before) {
            // 載入更多訊息（往前）
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
    
    if (!content || !appState.currentChannel) {
        return;
    }
    
    try {
        input.value = '';
        input.disabled = true;
        
        await api.sendMessage(appState.currentChannel.id, content);
        
        // 訊息會通過 WebSocket 接收，不需要手動添加
    } catch (error) {
        console.error('Failed to send message:', error);
        showNotification('發送訊息失敗', 'error');
        input.value = content; // 恢復輸入
    } finally {
        input.disabled = false;
        input.focus();
    }
}

// 處理訊息輸入鍵盤事件
function handleMessageKeyPress(event) {
    if (event.key === 'Enter' && !event.shiftKey) {
        event.preventDefault();
        sendMessage();
    }
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
        }
    });
}

// 處理新訊息
function handleNewMessage(message) {
    // 只處理當前頻道的訊息
    if (appState.currentChannel && message.channel_id === appState.currentChannel.id) {
        appState.messages.push(message);
        renderMessages();
        scrollToBottom();
    }
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

// 渲染成員列表
function renderMembers() {
    const container = document.getElementById('members-list');
    container.innerHTML = '';
    
    appState.members.forEach(member => {
        const memberElement = document.createElement('div');
        memberElement.className = 'member-item';
        
        const user = member.user || {};
        const status = user.status || 'offline';
        const nickname = user.nickname || user.username || 'Unknown';
        
        memberElement.innerHTML = `
            <div class="member-avatar">
                ${user.avatar ? `<img src="${user.avatar}" alt="${nickname}">` : '<i class="fas fa-user"></i>'}
                <span class="status-indicator ${status}"></span>
            </div>
            <div class="member-name">${nickname}</div>
        `;
        
        container.appendChild(memberElement);
    });
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
                         prevMessage.user_id === message.user_id &&
                         (new Date(message.created_at) - new Date(prevMessage.created_at)) < 300000; // 5分鐘內
        
        const messageElement = document.createElement('div');
        messageElement.className = 'message';
        messageElement.setAttribute('data-message-id', message.id);
        
        const user = message.user || {};
        const nickname = user.nickname || user.username || 'Unknown';
        const avatar = user.avatar;
        const timestamp = formatTimestamp(message.created_at);
        
        if (isGrouped) {
            messageElement.innerHTML = `
                <div class="message-avatar"></div>
                <div class="message-content">
                    <div class="message-text">${escapeHtml(message.content)}</div>
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
                    <div class="message-text">${escapeHtml(message.content)}</div>
                </div>
            `;
        }
        
        container.appendChild(messageElement);
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
        channelTopic.textContent = appState.currentChannel.description || '';
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
function handleLogout() {
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
    localStorage.removeItem(STORAGE_KEYS.USER);
    api.setToken(null);
    
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
        const channel = await api.createChannel(appState.currentGuild.id, name, type, description);
        
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
    // TODO: 實現社群設定
    showNotification('社群設定功能開發中...', 'info');
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
window.onclick = function(event) {
    if (event.target.classList.contains('modal')) {
        event.target.classList.remove('active');
    }
};
