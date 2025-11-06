# TalkRealm API Documentation

## API Base URL
```
http://localhost:8080/api/v1
```

## Authentication
Most endpoints require JWT authentication. Include the token in the Authorization header:
```
Authorization: Bearer <your_jwt_token>
```

## Endpoints

### Health Check
- **GET** `/health` - Check service health
- **GET** `/ping` - Simple ping endpoint

### Authentication
- **POST** `/api/v1/auth/register` - Register new user
- **POST** `/api/v1/auth/login` - User login

### Users (Protected)
- **GET** `/api/v1/users/me` - Get current user info
- **PUT** `/api/v1/users/me` - Update current user info

### Guilds/Servers (Protected)
- **POST** `/api/v1/guilds` - Create new guild
- **GET** `/api/v1/guilds` - List user's guilds
- **GET** `/api/v1/guilds/:id` - Get guild details
- **PUT** `/api/v1/guilds/:id` - Update guild
- **DELETE** `/api/v1/guilds/:id` - Delete guild

### Channels (Protected)
- **POST** `/api/v1/channels` - Create new channel
- **GET** `/api/v1/channels/:id` - Get channel details
- **PUT** `/api/v1/channels/:id` - Update channel
- **DELETE** `/api/v1/channels/:id` - Delete channel

### WebSocket (Protected)
- **GET** `/api/v1/ws` - Establish WebSocket connection for real-time messaging

## Response Format
All responses follow this format:
```json
{
  "status": "success|error",
  "data": {},
  "message": "optional message"
}
```
