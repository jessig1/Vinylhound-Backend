# Vinylhound Backend

A music discovery and rating platform backend built with Go and PostgreSQL.

## 🏗️ Architecture

Vinylhound uses a **monolithic architecture** for simplicity and development speed. See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed architecture documentation.

```
Frontend (Svelte) → API Gateway (Go) → Monolithic Backend (Go) → PostgreSQL
```

## 🚀 Quick Start

### Prerequisites

- Go 1.23 or later
- PostgreSQL 14+
- Docker (optional, for running PostgreSQL)

### 1. Clone and Setup

```bash
# Clone the repository
git clone <repository-url>
cd Vinylhound-Backend

# Copy environment configuration
cp .env.example .env

# Edit .env and set your values (especially JWT_SECRET)
# Generate a secure JWT_SECRET with:
# openssl rand -base64 32
```

### 2. Start Database

```bash
# Using Docker
docker run --name vinylhound-db \
  -e POSTGRES_PASSWORD=localpassword \
  -e POSTGRES_USER=vinylhound \
  -e POSTGRES_DB=vinylhound \
  -p 5432:5432 \
  -d postgres:16

# Or use existing PostgreSQL instance and update DATABASE_URL in .env
```

### 3. Run Migrations

```bash
go run cmd/migrate up
```

### 4. Start Server

```bash
# Development
go run cmd/vinylhound/main.go

# Production (build first)
go build -o vinylhound cmd/vinylhound/main.go
./vinylhound
```

The server will start on `http://localhost:8080` (or the port specified in your `.env`).

## 📁 Project Structure

```
Vinylhound-Backend/
├── cmd/
│   ├── vinylhound/       # Main application entry point (monolith)
│   ├── gateway/          # API gateway
│   └── migrate/          # Database migration tool
├── internal/
│   ├── httpapi/          # HTTP handlers and routing
│   ├── app/              # Business logic / application services
│   └── store/            # Data access layer
├── services/             # [DEPRECATED] Microservices (not used)
├── shared/go/
│   ├── auth/             # Authentication utilities
│   ├── config/           # Configuration management
│   ├── database/         # Database connection utilities
│   ├── logging/          # Structured logging
│   ├── middleware/       # HTTP middleware (CORS, auth, logging)
│   └── models/           # Shared data models
├── migrations/           # Database migrations
├── docs/                 # Documentation
├── .env.example          # Example environment configuration
└── README.md            # This file
```

## 🔧 Configuration

All configuration is done via environment variables. See [.env.example](.env.example) for all available options.

### Required Variables

```bash
# Database
DATABASE_URL=postgresql://user:password@localhost:5432/vinylhound

# Security
JWT_SECRET=your-secret-key-min-16-chars
```

### Optional Variables

```bash
# Server
PORT=8080
HOST=0.0.0.0

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173

# Logging
LOG_LEVEL=info        # debug, info, warn, error
LOG_FORMAT=json       # json, text

# Environment
ENV=development       # development, staging, production
```

## 🌐 API Endpoints

### Authentication
- `POST /api/v1/auth/signup` - Create new user account
- `POST /api/v1/auth/login` - Authenticate user and get token

### User Profile
- `GET /api/v1/users/profile` - Get user profile (requires auth)
- `PUT /api/v1/users/profile` - Update user profile (requires auth)

### Albums
- `GET /api/v1/albums` - List/search albums
  - Query params: `?artist=Beatles&genre=Rock&year=1969&rating=5`
- `GET /api/v1/albums/{id}` - Get single album

### User Album Preferences
- `GET /api/v1/me/albums` - Get user's albums (requires auth)
- `GET /api/v1/me/albums/preferences` - Get user's preferences (requires auth)
- `PUT /api/v1/me/albums/{id}/preference` - Update album preference (requires auth)
- `DELETE /api/v1/me/albums/{id}/preference` - Remove preference (requires auth)

**Authentication**: Include token in header: `Authorization: Bearer <token>`

See [docs/API.md](docs/API.md) for detailed API documentation (coming soon).

## 🗄️ Database

### Migrations

```bash
# Run all migrations
go run cmd/migrate up

# Rollback last migration
go run cmd/migrate down

# Rollback all migrations
go run cmd/migrate down --all

# Create new migration
cd migrations
touch 000X_description.up.sql
touch 000X_description.down.sql
```

### Schema

See [migrations/](migrations/) for current schema.

**Tables**:
- `users` - User accounts
- `sessions` - Authentication sessions
- `user_content` - User content preferences
- `albums` - Album catalog
- `user_album_preferences` - User ratings and favorites

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...

# Run specific package tests
go test ./internal/httpapi/...
```

## 📝 Development

### Adding a New Feature

1. **Define the API** - Add route in `internal/httpapi/server.go`
2. **Create Handler** - Implement handler function
3. **Add Business Logic** - Create/update service in `internal/app/`
4. **Data Access** - Add methods to store in `internal/store/`
5. **Write Tests** - Add tests for all layers
6. **Update Docs** - Document new endpoints

### Code Style

- Follow Go best practices and idioms
- Use `gofmt` for formatting
- Write descriptive commit messages
- Add comments for exported functions
- Keep functions small and focused

### Logging

Use structured logging throughout:

```go
import "vinylhound/shared/logging"

// Simple logging
logging.Info("Server started")
logging.Error(err, "Failed to connect to database")

// With context
logger := logging.WithContext(ctx)
logger.Info().Str("user_id", userID).Msg("User logged in")
```

## 🏭 Production Deployment

### Build

```bash
# Build optimized binary
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
  -ldflags '-w -s' \
  -o vinylhound cmd/vinylhound/main.go
```

### Docker

```bash
# Build image
docker build -t vinylhound-backend .

# Run container
docker run -d \
  --name vinylhound \
  -p 8080:8080 \
  -e DATABASE_URL=postgresql://... \
  -e JWT_SECRET=... \
  vinylhound-backend
```

### Environment Setup

1. **Use environment-specific configuration**
   - Development: `.env` file
   - Production: Environment variables from secrets manager

2. **Secure JWT_SECRET**
   - Never commit secrets to git
   - Use AWS Secrets Manager, HashiCorp Vault, or Kubernetes Secrets
   - Rotate secrets regularly

3. **Database**
   - Use connection pooling
   - Enable SSL (`sslmode=require`)
   - Regular backups
   - Monitor query performance

4. **Monitoring**
   - Centralized logging (ELK, Loki)
   - Metrics (Prometheus)
   - Error tracking (Sentry)
   - Uptime monitoring

## 🔒 Security

### Current Measures
- ✅ Password hashing with bcrypt
- ✅ Session-based authentication
- ✅ SQL injection prevention (parameterized queries)
- ✅ CORS configuration
- ✅ Environment-based secrets

### Recommendations
- [ ] Add rate limiting on auth endpoints
- [ ] Implement password strength requirements
- [ ] Add session expiry validation
- [ ] Enable security headers (CSP, X-Frame-Options)
- [ ] Regular security audits

## 🐛 Troubleshooting

### Database Connection Fails

```bash
# Check PostgreSQL is running
docker ps | grep postgres

# Test connection
psql postgresql://vinylhound:localpassword@localhost:5432/vinylhound

# Check DATABASE_URL format
echo $DATABASE_URL
```

### Migrations Fail

```bash
# Check migration status
go run cmd/migrate status

# Force version (use with caution)
go run cmd/migrate force <version>

# Manual SQL (last resort)
psql $DATABASE_URL < migrations/000X_fix.up.sql
```

### JWT_SECRET Error

```bash
# Verify JWT_SECRET is set
echo $JWT_SECRET

# Generate new secret
openssl rand -base64 32

# Update .env file
```

## 📚 Documentation

- [Architecture](docs/ARCHITECTURE.md) - System architecture and design decisions
- [Migrations](docs/migrations.md) - Database migration guide
- API Documentation - Coming soon
- Contributing Guide - Coming soon

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

[Your License Here]

## 👥 Team

[Your Team Information]

---

**Last Updated**: 2025-10-24
**Go Version**: 1.23+
**PostgreSQL Version**: 14+
