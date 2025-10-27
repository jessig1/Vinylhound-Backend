# Migration Guide - Week 1 & 2 Improvements

This guide helps you migrate from the old codebase to the new improved version with better security, logging, and architecture.

## 🎯 What Changed

### Week 1: Security & Architecture
1. ✅ Removed hardcoded secrets
2. ✅ Created missing database migrations
3. ✅ Documented architecture decision (monolith)

### Week 2: Foundation
4. ✅ Added structured logging
5. ✅ Created centralized config validation
6. ✅ Standardized API versioning to `/api/v1/`

## 🔄 Migration Steps

### Step 1: Update Environment Configuration

#### Before
```bash
# No environment validation
# Secrets hardcoded in code
```

#### After
```bash
# Copy example configuration
cp .env.example .env

# REQUIRED: Generate and set JWT_SECRET
openssl rand -base64 32
# Add to .env:
JWT_SECRET=<generated-secret>

# Set database URL
DATABASE_URL=postgresql://vinylhound:password@localhost:5432/vinylhound
```

**Action Items**:
- [ ] Create `.env` file from `.env.example`
- [ ] Generate secure `JWT_SECRET` (minimum 16 characters)
- [ ] Set `DATABASE_URL` with your database credentials
- [ ] **NEVER commit `.env` to git** (already in `.gitignore`)

---

### Step 2: Run New Database Migrations

New migrations were added for tables that were previously created ad-hoc.

```bash
# Run migrations to create missing tables
go run cmd/migrate up

# Verify migrations completed
psql $DATABASE_URL -c "\dt"
```

**New Tables**:
- `user_content` - User content preferences (with proper indexes)
- `sessions` - Authentication sessions (with expiry tracking)

**Updated Tables**:
- `users` - Added `updated_at` column for consistency

**Action Items**:
- [ ] Run migrations on development database
- [ ] Run migrations on staging database (when ready)
- [ ] Run migrations on production database (when ready)
- [ ] Verify all tables exist

---

### Step 3: Update Application Code

#### 3a. Remove Local Hardcoded Values

**services/user-service/internal/service/user_service.go**

❌ **Old** (hardcoded secret):
```go
func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{
		repo:     repo,
		tokenMgr: auth.NewTokenManager("your-secret-key"),
	}
}
```

✅ **New** (environment-driven):
```go
func NewUserService(repo repository.UserRepository) *UserService {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		panic("JWT_SECRET environment variable is required")
	}

	return &UserService{
		repo:     repo,
		tokenMgr: auth.NewTokenManager(jwtSecret),
	}
}
```

**Action Items**:
- [ ] Search for hardcoded `"your-secret-key"` and replace
- [ ] Search for hardcoded `"localhost:3000"` and replace with env var

---

#### 3b. Update CORS Configuration

**shared/go/middleware/cors.go**

❌ **Old** (hardcoded origins):
```go
AllowedOrigins: []string{"http://localhost:3000", "http://localhost:8080"}
```

✅ **New** (environment-driven):
```go
// Reads from CORS_ALLOWED_ORIGINS environment variable
// Falls back to default localhost origins for development
```

**Action Items**:
- [ ] Set `CORS_ALLOWED_ORIGINS` in production (comma-separated)
- [ ] Use defaults for local development
- [ ] Update for each environment (staging, production)

---

### Step 4: Adopt Centralized Configuration

#### Use New Config Package

**cmd/vinylhound/main.go** or any service main:

✅ **New Pattern**:
```go
import "vinylhound/shared/config"

func main() {
	// Load and validate configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// All config is validated at startup
	// Access via cfg.Database.URL, cfg.Server.Port, etc.
}
```

**Benefits**:
- ✅ All required variables validated at startup
- ✅ Clear error messages if config is missing
- ✅ Type-safe access to configuration
- ✅ Centralized defaults

**Action Items**:
- [ ] Update main.go to use centralized config
- [ ] Remove duplicate config loading code
- [ ] Add config validation to startup

---

### Step 5: Implement Structured Logging

#### Replace Standard Logging

❌ **Old**:
```go
import "log"

log.Println("Server started on port 8080")
log.Printf("User %s logged in", username)
```

✅ **New**:
```go
import "vinylhound/shared/logging"

// Simple logging
logging.Info("Server started")

// Structured logging with context
logger := logging.WithContext(ctx)
logger.Info().
	Str("username", username).
	Str("request_id", requestID).
	Msg("User logged in")
```

**Benefits**:
- ✅ JSON output for log aggregation
- ✅ Request IDs for tracing
- ✅ Structured fields for filtering
- ✅ Log levels (debug, info, warn, error)

**Action Items**:
- [ ] Initialize logger in main.go
- [ ] Replace `log.Println` with `logging.Info`
- [ ] Replace `log.Printf` with structured logging
- [ ] Add request ID to all HTTP handlers

---

#### Add Request Logging Middleware

**cmd/vinylhound/server.go**:

✅ **Add middleware**:
```go
import "vinylhound/shared/middleware"

// In your server setup
handler := middleware.Recovery()(
	middleware.RequestLogging()(
		middleware.CORS(corsConfig)(
			apiHandler,
		),
	),
)
```

**Benefits**:
- ✅ Automatic request/response logging
- ✅ Request IDs in all logs
- ✅ Duration tracking
- ✅ Panic recovery

**Action Items**:
- [ ] Add RequestLogging middleware
- [ ] Add Recovery middleware
- [ ] Verify logs include request_id
- [ ] Test panic recovery

---

### Step 6: Update API Endpoints

API endpoints now support both **v1** and **legacy** paths for backward compatibility.

#### Supported Endpoints

| Legacy Path | New v1 Path | Status |
|------------|-------------|--------|
| `/api/signup` | `/api/v1/auth/signup` | Both supported |
| `/api/login` | `/api/v1/auth/login` | Both supported |
| `/api/me/content` | `/api/v1/users/profile` | Both supported |
| `/api/me/albums` | `/api/v1/me/albums` | Both supported |
| `/api/albums` | `/api/v1/albums` | Both supported |
| `/api/album?id=X` | `/api/v1/albums/{id}` | Both supported |

**Migration Strategy**:
1. **Phase 1** (Now): Both paths work
2. **Phase 2** (After frontend migration): Deprecate legacy paths
3. **Phase 3** (Future): Remove legacy paths

**Action Items**:
- [ ] Update frontend to use `/api/v1/` paths
- [ ] Test all endpoints with new paths
- [ ] Plan deprecation timeline for legacy paths

---

### Step 7: Verify Installation

Run through this checklist to ensure everything is working:

```bash
# 1. Configuration
cat .env
# Verify JWT_SECRET is set and secure
# Verify DATABASE_URL is correct

# 2. Database
go run cmd/migrate up
psql $DATABASE_URL -c "SELECT tablename FROM pg_tables WHERE schemaname = 'public';"
# Should see: users, sessions, user_content, albums, user_album_preferences

# 3. Dependencies
go mod download
go mod tidy

# 4. Build
go build cmd/vinylhound/main.go
# Should compile without errors

# 5. Run
./vinylhound
# Should start without panics
# Check logs for "Server started"

# 6. Test API
curl http://localhost:8080/api/v1/albums
# Should return albums or empty array

# 7. Test authentication
curl -X POST http://localhost:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"testpass123"}'
# Should return success or "user exists"
```

**Checklist**:
- [ ] Configuration loads without errors
- [ ] Database migrations complete
- [ ] Server starts successfully
- [ ] Logs are structured JSON
- [ ] API endpoints respond
- [ ] Authentication works

---

## 🚨 Breaking Changes

### None (All Changes Are Backward Compatible)

The migration is designed to be **backward compatible**:
- ✅ Legacy API paths still work
- ✅ Existing database schema supported
- ✅ Gradual migration possible

**However**, you MUST:
1. Set `JWT_SECRET` environment variable (previously hardcoded)
2. Run new database migrations (adds missing tables)
3. Set `DATABASE_URL` (previously could use defaults)

---

## 🔍 Verification Tests

### Test 1: Configuration Validation

```bash
# Should fail with clear error
JWT_SECRET="" go run cmd/vinylhound/main.go
# Expected: "JWT_SECRET is required"

# Should fail with clear error
DATABASE_URL="" go run cmd/vinylhound/main.go
# Expected: "DATABASE_URL is required"

# Should succeed
JWT_SECRET=test-secret-123456 DATABASE_URL=postgresql://... go run cmd/vinylhound/main.go
```

### Test 2: Database Migrations

```bash
# Check migration version
psql $DATABASE_URL -c "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;"
# Expected: Latest version (0006 or higher)

# Verify tables exist
psql $DATABASE_URL -c "\dt"
# Expected: users, sessions, user_content, albums, user_album_preferences
```

### Test 3: Structured Logging

```bash
# Start server and check log format
go run cmd/vinylhound/main.go 2>&1 | head -5
# Expected: JSON formatted logs with timestamp, level, message
```

### Test 4: API Versioning

```bash
# Test v1 endpoint
curl http://localhost:8080/api/v1/albums

# Test legacy endpoint
curl http://localhost:8080/api/albums

# Both should return same response
```

---

## 📊 Rollback Plan

If you need to rollback:

### Rollback Migrations

```bash
# Rollback to specific version
go run cmd/migrate down --to 0003

# Or rollback one migration at a time
go run cmd/migrate down
```

### Rollback Code Changes

```bash
# Revert to previous commit
git revert HEAD

# Or checkout specific commit
git checkout <previous-commit>
```

### Restore Configuration

```bash
# Remove new config requirements (not recommended)
# Instead, set environment variables to maintain security
```

---

## 🎓 Best Practices Going Forward

### 1. Never Hardcode Secrets

❌ **Bad**:
```go
jwtSecret := "my-secret-key"
```

✅ **Good**:
```go
jwtSecret := os.Getenv("JWT_SECRET")
if jwtSecret == "" {
	log.Fatal("JWT_SECRET is required")
}
```

### 2. Always Use Structured Logging

❌ **Bad**:
```go
log.Printf("User %s logged in", username)
```

✅ **Good**:
```go
logging.WithContext(ctx).Info().
	Str("username", username).
	Msg("User logged in")
```

### 3. Validate Configuration at Startup

❌ **Bad**:
```go
port := os.Getenv("PORT")
// Might be empty or invalid
```

✅ **Good**:
```go
cfg, err := config.Load()
if err != nil {
	log.Fatal(err)
}
// All config validated
```

### 4. Use API Versioning

❌ **Bad**:
```go
mux.HandleFunc("/api/users", handler)
// Breaking changes affect all clients
```

✅ **Good**:
```go
mux.HandleFunc("/api/v1/users", handler)
mux.HandleFunc("/api/v2/users", handlerV2)
// Can support multiple versions
```

---

## 📞 Need Help?

- **Configuration Issues**: Check `.env.example` for required variables
- **Migration Errors**: See [docs/migrations.md](migrations.md)
- **Architecture Questions**: See [docs/ARCHITECTURE.md](ARCHITECTURE.md)
- **General Questions**: File an issue or contact the team

---

**Migration Completed**: 2025-10-24
**Estimated Time**: 30-60 minutes
**Difficulty**: Easy (backward compatible)
