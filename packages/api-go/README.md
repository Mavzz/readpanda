# ReadPanda Go API

This is the Go backend API for the ReadPanda application, converted from the original Node.js implementation.

## Features

- User authentication (email/password and Google OAuth)
- JWT-based session management
- PostgreSQL database integration
- Firebase Storage for file uploads
- Book publishing and management
- User preferences and notifications
- CORS and logging middleware

## Prerequisites

- Go 1.21 or later
- PostgreSQL database
- Firebase project with Storage enabled
- Google OAuth credentials (optional, for Google login)

## Installation

1. Navigate to the api-go directory:
```bash
cd packages/api-go
```

2. Install dependencies:
```bash
go mod download
```

## Configuration

Create a `.env.local` file in the `packages/api-go` directory with the following variables:

```env
# Server Configuration
PORT=3000
API_VERSION=/api/v1

# PostgreSQL Database
PG_USER=your_db_user
PG_HOST=localhost
PG_DB=your_db_name
PG_PASSWORD=your_db_password
PG_PORT=5432

# JWT
JWT_SECRET=your_super_secret_jwt_key
JWT_REFRESH_SECRET=your_refresh_secret_key

# Crypto (for frontend password encryption)
CRYPTO_SECRET=your_crypto_secret

# Google OAuth
GOOGLE_CLIENT_ID=your_google_client_id.apps.googleusercontent.com

# Firebase
FIREBASE_TYPE=service_account
FIREBASE_PROJECT_ID=your_project_id
FIREBASE_PRIVATE_KEY_ID=your_private_key_id
FIREBASE_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n"
FIREBASE_CLIENT_EMAIL=your_service_account@project.iam.gserviceaccount.com
FIREBASE_CLIENT_ID=your_client_id
FIREBASE_AUTH_URI=https://accounts.google.com/o/oauth2/auth
FIREBASE_TOKEN_URI=https://oauth2.googleapis.com/token
FIREBASE_AUTH_PROVIDER=https://www.googleapis.com/oauth2/v1/certs
FIREBASE_CERT_URL=https://www.googleapis.com/robot/v1/metadata/x509/your_service_account%40project.iam.gserviceaccount.com
FIREBASE_STORAGE_BUCKET=your_project.appspot.com
```

## TLS/HTTPS Setup (Optional)

If you want to run the server with HTTPS, place `cert.pem` and `key.pem` files in the `packages/api-go` directory. The server will automatically use them if found.

To generate self-signed certificates for development:
```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes
```

## Running the Server

### Development
```bash
go run cmd/server/main.go
```

### Production Build
```bash
go build -o api-go cmd/server/main.go
./api-go
```

## API Endpoints

### Authentication
- `POST /api/v1/signup` - Create a new user
- `POST /api/v1/auth/login` - Login with email/password
- `POST /api/v1/auth/google` - Login with Google OAuth
- `POST /api/v1/auth/logout` - Logout user
- `GET /api/v1/token/refresh` - Refresh access token

### Users
- `GET /api/v1/users` - Get all users

### Books
- `POST /api/v1/books/upload` - Upload a book with cover and manuscript
- `GET /api/v1/books` - Get books for authenticated user
- `GET /api/v1/books/all` - Get all books

### Preferences
- `GET /api/v1/user/preferences` - Get user preferences
- `POST /api/v1/user/preferences` - Update user preferences
- `GET /api/v1/genres` - Get all genres
- `GET /api/v1/subgenres` - Get all subgenres

### Notifications
- `GET /api/v1/notifications` - Get user notifications
- `GET /api/v1/notifications/unread/count` - Get unread notification count

## Project Structure

```
packages/api-go/
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go         # Configuration management
│   ├── database/
│   │   └── database.go       # Database connection
│   ├── handlers/
│   │   ├── users.go          # User authentication handlers
│   │   ├── books.go          # Book management handlers
│   │   ├── preferences.go    # User preferences handlers
│   │   └── notifications.go  # Notification handlers
│   ├── middleware/
│   │   └── middleware.go     # HTTP middleware (CORS, logging, auth)
│   ├── models/
│   │   └── models.go         # Data models
│   └── utils/
│       ├── utils.go          # Utility functions (JWT, crypto)
│       └── firebase.go       # Firebase storage integration
├── go.mod                    # Go module definition
└── README.md                 # This file
```

## Differences from Node.js Version

1. **Type Safety**: Go provides compile-time type checking
2. **Performance**: Generally faster execution and lower memory usage
3. **Concurrency**: Built-in goroutines for handling concurrent requests
4. **Error Handling**: Explicit error handling instead of try-catch
5. **Compilation**: Compiled binary instead of interpreted code

## Migration Notes

All major features from the Node.js API have been ported:
- ✅ User authentication (email/password)
- ✅ Google OAuth integration
- ✅ JWT token management
- ✅ PostgreSQL database operations
- ✅ Firebase Storage integration
- ✅ Book upload and management
- ✅ User preferences
- ✅ Notifications
- ✅ CORS middleware
- ✅ Request logging

## Testing

To test the API endpoints, you can use tools like:
- cURL
- Postman
- Thunder Client (VS Code extension)

Example:
```bash
curl -X POST http://localhost:3000/api/v1/signup \
  -H "Content-Type: application/json" \
  -d '{"username":"test","email":"test@example.com","password":"encrypted_password"}'
```

## License

MIT
