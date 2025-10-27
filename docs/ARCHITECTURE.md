# Vinylhound Backend Architecture

## Architecture Decision

**Status**: ✅ **RESOLVED - Use Monolith Architecture**

**Date**: 2025-10-24

### Context

The Vinylhound backend codebase currently has **two parallel implementations**:
1. **Monolithic Application** (`cmd/vinylhound/`) - Single unified service
2. **Microservices** (`services/*`) - Four separate services (user, catalog, rating, playlist)

This duplication creates:
- Maintenance overhead (changes needed in multiple places)
- Code duplication (user auth, rating logic exists in both)
- Confusion about which code path is actually used
- Deployment complexity without clear benefits

### Decision

**We are consolidating to a MONOLITHIC ARCHITECTURE** (`cmd/vinylhound/`)

### Rationale

1. **Current Scale**: The application is in early stages with limited traffic
2. **Team Size**: Small team benefits from simpler deployment and debugging
3. **Development Speed**: Faster iteration without inter-service communication complexity
4. **Operational Simplicity**: Single deployment, single database, simpler monitoring
5. **Future Path**: Can extract services later when clear boundaries emerge

### Implementation Plan

#### Phase 1: Consolidation (Complete by Week 3)
- [x] Document architecture decision
- [ ] Remove or deprecate microservices code (`services/*`)
- [ ] Ensure all functionality exists in monolith
- [ ] Update Docker configuration to deploy monolith only

#### Phase 2: Monolith Optimization (Weeks 4-6)
- [ ] Add comprehensive tests
- [ ] Implement structured logging
- [ ] Add metrics and monitoring
- [ ] Optimize database queries

### Future Considerations

**When to Consider Microservices:**
- User base > 100,000 active users
- Clear performance bottlenecks in specific components
- Need for independent scaling (e.g., rating service gets 10x traffic)
- Multiple teams working on different domains
- Different technology requirements (e.g., ML service needs Python)

**Extraction Candidates** (if/when needed):
1. **Rating/Review Service** - High read/write volume, could benefit from separate scaling
2. **Search/Catalog Service** - Could use Elasticsearch, different tech stack
3. **Recommendation Engine** - ML workload, Python/TensorFlow

---

## Current Architecture

### System Overview

```
┌─────────────────┐
│   Frontend      │
│  (Svelte/Vite)  │
└────────┬────────┘
         │ HTTP/REST
         ▼
┌─────────────────┐
│  API Gateway    │
│   (Go/Mux)      │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────────┐
│     Monolithic Backend              │
│      (cmd/vinylhound)               │
│                                     │
│  ┌──────────────────────────────┐  │
│  │   HTTP API Layer              │  │
│  │   /internal/httpapi          │  │
│  └──────────┬───────────────────┘  │
│             │                       │
│  ┌──────────▼───────────────────┐  │
│  │   Application Services        │  │
│  │   - UserService               │  │
│  │   - AlbumService              │  │
│  │   - RatingService             │  │
│  └──────────┬───────────────────┘  │
│             │                       │
│  ┌──────────▼───────────────────┐  │
│  │   Data Store Layer            │  │
│  │   /internal/store            │  │
│  │   - SQL queries               │  │
│  │   - Transaction handling      │  │
│  └──────────┬───────────────────┘  │
└─────────────┼───────────────────────┘
              │
              ▼
       ┌────────────┐
       │ PostgreSQL │
       └────────────┘
```

### Component Responsibilities

#### 1. HTTP API Layer (`/internal/httpapi`)
- **Purpose**: Handle HTTP requests/responses
- **Responsibilities**:
  - Request parsing and validation
  - Response serialization
  - HTTP status code mapping
  - CORS handling
  - Authentication middleware

**Key Files**:
- `server.go` - HTTP handlers and routing
- Handles endpoints: `/api/signup`, `/api/login`, `/api/me/*`, `/api/albums`

#### 2. Application Services (`/internal/app`)
- **Purpose**: Business logic and workflows
- **Responsibilities**:
  - Coordinate between different data stores
  - Enforce business rules
  - Transaction orchestration

**Services**:
- `UserService` - User management, authentication
- `AlbumService` - Album operations, search
- `RatingService` - Rating and preference management

#### 3. Data Store Layer (`/internal/store`)
- **Purpose**: Database access and queries
- **Responsibilities**:
  - SQL query execution
  - Data mapping (DB ↔ Go structs)
  - Transaction management
  - Database-level validation

**Key Files**:
- `store.go` - Main store interface
- `auth.go` - User authentication queries
- `albums.go` - Album data access
- `preferences.go` - User preferences

#### 4. Shared Libraries (`/shared/go`)
- **Purpose**: Reusable components
- **Components**:
  - `auth/` - Password hashing, token generation
  - `middleware/` - HTTP middleware (CORS, auth)
  - `models/` - Shared data structures
  - `database/` - Database connection utilities

### Data Flow

#### Example: User Login
```
1. POST /api/login {username, password}
   ↓
2. httpapi.Server.handleLogin()
   ↓
3. store.ValidateCredentials(username, password)
   ↓
4. SQL: SELECT * FROM users WHERE username = $1
   ↓
5. bcrypt.CompareHashAndPassword(...)
   ↓
6. store.CreateSession(token, userID)
   ↓
7. SQL: INSERT INTO sessions ...
   ↓
8. Return {token: "..."}
```

### Database Schema

See [migrations/](../migrations/) for complete schema.

**Key Tables**:
- `users` - User accounts
- `sessions` - Authentication sessions
- `user_content` - User's content preferences
- `albums` - Album catalog
- `user_album_preferences` - User ratings and favorites

### API Endpoints

#### Authentication
- `POST /api/signup` - Create new user account
- `POST /api/login` - Authenticate user

#### User Profile
- `GET /api/me/content` - Get user's content (requires auth)
- `PUT /api/me/content` - Update user's content (requires auth)

#### Albums
- `GET /api/albums` - Search/list albums (supports filters)
- `GET /api/album?id={id}` - Get single album

#### Preferences
- `GET /api/me/albums` - Get user's album preferences (requires auth)
- `GET /api/me/albums/preferences` - Get detailed preferences (requires auth)
- `PUT /api/me/albums/{id}/preference` - Update album preference (requires auth)
- `DELETE /api/me/albums/{id}/preference` - Remove preference (requires auth)

### Configuration

**Environment Variables**:
```bash
# Required
DATABASE_URL=postgresql://user:pass@host:port/dbname
JWT_SECRET=your-secret-key-here

# Optional
PORT=8080
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
```

See [.env.example](../.env.example) for full configuration.

### Deployment

**Development**:
```bash
# Start database
docker-compose up -d postgres

# Run migrations
go run cmd/migrate up

# Start server
DATABASE_URL=postgresql://... JWT_SECRET=... go run cmd/vinylhound
```

**Production**:
```bash
# Build binary
go build -o vinylhound cmd/vinylhound/main.go

# Run
./vinylhound
```

### Monitoring & Observability

**Current**:
- Basic logging to stdout
- HTTP status codes for error tracking

**Planned** (Week 2):
- Structured logging with zerolog
- Prometheus metrics
- Request tracing with correlation IDs

### Security

**Current Measures**:
- Password hashing with bcrypt
- Session-based authentication
- SQL parameterization (prevents injection)
- CORS configuration

**Planned Improvements**:
- Rate limiting on auth endpoints
- Password strength requirements
- Session expiry validation
- Security headers (CSP, X-Frame-Options, etc.)

---

## Migration from Microservices

### Status: Microservices Deprecated

The `services/*` directory contains deprecated microservice implementations that are **not currently used in production**.

**Deprecated Services**:
- `services/user-service/` - User management (superseded by monolith)
- `services/catalog-service/` - Album catalog (superseded by monolith)
- `services/rating-service/` - Ratings (superseded by monolith)
- `services/playlist-service/` - Playlists (superseded by monolith)

**Removal Plan**:
1. Verify all functionality exists in monolith
2. Archive microservices to separate branch
3. Remove from main codebase
4. Update Docker configuration

---

## Performance Considerations

### Current Bottlenecks

1. **N+1 Queries**: Album rating stats fetched individually
   - **Impact**: Slow album list endpoints
   - **Solution**: Use JOIN with GROUP BY

2. **No Caching**: All data fetched from DB on every request
   - **Impact**: Higher DB load
   - **Solution**: Add Redis for album lists

3. **Full Table Scans**: Text search uses ILIKE without indexes
   - **Impact**: Slow search
   - **Solution**: Add GIN indexes for full-text search

### Optimization Roadmap

**Week 3-4**:
- [ ] Fix N+1 queries
- [ ] Add database indexes
- [ ] Implement query result caching

**Week 5-6**:
- [ ] Add full-text search indexes
- [ ] Implement pagination
- [ ] Add request rate limiting

---

## Testing Strategy

**Current Coverage**: ~20% (HTTP layer only)

**Target Coverage**: 70%+

**Test Pyramid**:
```
        ┌─────────┐
        │   E2E   │  (10%)
        └─────────┘
      ┌─────────────┐
      │ Integration │  (30%)
      └─────────────┘
    ┌─────────────────┐
    │   Unit Tests    │  (60%)
    └─────────────────┘
```

**Testing Plan**:
1. Unit tests for store layer
2. Unit tests for application services
3. Integration tests for database
4. HTTP endpoint tests (existing)

---

## Questions or Suggestions?

File an issue or contact the team.

**Last Updated**: 2025-10-24
**Maintained By**: Development Team
