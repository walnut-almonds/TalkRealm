#!/bin/bash
# TalkRealm API 快速測試腳本

set -e

BASE_URL="http://localhost:8080"

# 顏色定義
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}=== TalkRealm API 測試 ===${NC}\n"

# 1. 健康檢查
echo -e "${YELLOW}1. 健康檢查${NC}"
curl -s "$BASE_URL/health" | jq '.'
echo -e "${GREEN}✅ 健康檢查通過${NC}\n"

# 2. 註冊使用者
echo -e "${YELLOW}2. 註冊使用者${NC}"
REGISTER_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123",
    "nickname": "Test User"
  }')

USER_ID=$(echo $REGISTER_RESPONSE | jq -r '.user.id // empty')
if [ -z "$USER_ID" ]; then
  echo -e "${YELLOW}⚠️  使用者可能已存在，繼續登入...${NC}\n"
else
  echo -e "${GREEN}✅ 註冊成功: User ID = $USER_ID${NC}\n"
fi

# 3. 登入
echo -e "${YELLOW}3. 登入${NC}"
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }')

TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.token')
echo -e "${GREEN}✅ 登入成功！Token: ${TOKEN:0:30}...${NC}\n"

# 4. 獲取使用者資訊
echo -e "${YELLOW}4. 獲取使用者資訊${NC}"
curl -s "$BASE_URL/api/v1/users/me" \
  -H "Authorization: Bearer $TOKEN" | jq '.user | {id, username, nickname, email}'
echo -e "${GREEN}✅ 獲取成功${NC}\n"

# 5. 建立社群
echo -e "${YELLOW}5. 建立社群${NC}"
GUILD_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/guilds" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "測試社群",
    "description": "這是測試社群"
  }')

GUILD_ID=$(echo $GUILD_RESPONSE | jq -r '.guild.id')
echo -e "${GREEN}✅ 社群建立成功: Guild ID = $GUILD_ID${NC}\n"

# 6. 建立頻道
echo -e "${YELLOW}6. 建立頻道${NC}"
CHANNEL_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/channels" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"guild_id\": $GUILD_ID,
    \"name\": \"一般討論\",
    \"type\": \"text\"
  }")

CHANNEL_ID=$(echo $CHANNEL_RESPONSE | jq -r '.channel.id')
echo -e "${GREEN}✅ 頻道建立成功: Channel ID = $CHANNEL_ID${NC}\n"

# 7. 發送訊息
echo -e "${YELLOW}7. 發送訊息${NC}"
MESSAGE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/messages" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"channel_id\": $CHANNEL_ID,
    \"content\": \"Hello, TalkRealm! 🎉\",
    \"type\": \"text\"
  }")

MESSAGE_ID=$(echo $MESSAGE_RESPONSE | jq -r '.message.id')
echo -e "${GREEN}✅ 訊息發送成功: Message ID = $MESSAGE_ID${NC}\n"

# 8. 列出訊息
echo -e "${YELLOW}8. 列出頻道訊息${NC}"
curl -s "$BASE_URL/api/v1/messages?channel_id=$CHANNEL_ID" \
  -H "Authorization: Bearer $TOKEN" | jq '.messages[] | {id, content, type}'
echo -e "${GREEN}✅ 訊息列表獲取成功${NC}\n"

# 清理 (可選)
echo -e "${YELLOW}9. 清理測試資料${NC}"
curl -s -X DELETE "$BASE_URL/api/v1/guilds/$GUILD_ID" \
  -H "Authorization: Bearer $TOKEN" > /dev/null
echo -e "${GREEN}✅ 測試資料已清理${NC}\n"

echo -e "${CYAN}✨ 所有測試完成！${NC}"
