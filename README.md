## qwikkle-api

Backend API service for Qwikkle built with ElysiaJS and TypeScript.

### Prerequisites

- [Bun](https://bun.sh) 1.0.0 or newer installed
- PostgreSQL 12 or newer
- Node.js 25+ (optional, for development tools)

### Installation

1. Install dependencies:
```bash
bun install
```

2. Copy environment variables:
```bash
cp .env.example .env
```

3. Configure your database and JWT secrets in `.env`:
```bash
# Required
POSTGRES_DSN=postgres://user:password@localhost:5432/qwikkle?sslmode=disable
JWT_ACCESS_SECRET=your-secure-random-string
JWT_REFRESH_SECRET=another-secure-random-string

# Optional S3 configuration
S3_REGION=us-east-1
S3_BUCKET=your-bucket-name
S3_ACCESS_KEY_ID=your-access-key
S3_SECRET_ACCESS_KEY=your-secret-key
```

### Running locally

Development mode with hot reload:
```bash
bun run dev
```

Production mode:
```bash
bun run start
```

The service will start on `http://localhost:8080`. You can verify it with:

```bash
curl http://localhost:8080/health
```

### Available Scripts

- `bun run dev` - Start development server with hot reload
- `bun run start` - Start production server
- `bun run build` - Build for production
- `bun run lint` - Check code with Biome
- `bun run lint:fix` - Fix code formatting issues

### API Documentation

Swagger UI is available at:
- `http://localhost:8080/swagger`

### API Endpoints

#### Health
- `GET /health` - Basic health check
- `GET /health/readyz` - Detailed readiness check with database status

#### Authentication
- `POST /auth/signup` - User registration
- `POST /auth/login` - User login
- `GET /auth/me` - Get current user profile
- `PUT /auth/me/profile` - Update user profile
- `GET /auth/availability?qkId=<id>` - Check QKID availability

#### Uploads
- `POST /uploads/presigned-url` - Get presigned upload URL
- `POST /uploads/:uploadId/complete` - Mark upload as completed
- `GET /uploads/:uploadId` - Get upload info

#### Admin (Bearer token required)
- `GET /admin/users` - List users with pagination
- `GET /admin/users/:userId` - Get user details
- `PATCH /admin/users/:userId/status` - Update user status
- `GET /admin/uploads` - List uploads
- `GET /admin/stats` - Get admin statistics

### Database Setup

The application expects PostgreSQL with the following tables:
- `users` - User accounts
- `sessions` - User sessions
- `organizations` - Organization data
- `organization_members` - Organization memberships
- `organization_documents` - Organization documents
- `uploads` - File upload records
- `goose_db_version` - Migration tracking

### Development

The project uses:
- **ElysiaJS** - Web framework
- **TypeScript** - Type safety
- **Postgres.js** - Database client
- **Zod** - Schema validation
- **Biome** - Code formatting and linting
- **Swagger** - API documentation


