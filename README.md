# FantasticTelegram – Blog Platform with Microservices Architecture
FantasticTelegram is a production-ready blog platform with authentication, post management, and async notifications.  
It implements the **Outbox Pattern** for eventual consistency, gRPC inter-service communication, Redis caching, and Kafka-based event processing.  
Built with Go best practices – Clean Architecture, fully tested, and containerized.
---
## Killer Features
- **User Management** – Registration, JWT authentication, and profile management with Bcrypt password hashing.
- **Post CRUD** – Create, read, update, and delete posts with Redis caching (10 min TTL) and title/ID lookup.
- **Outbox Pattern** – Atomic post + event writes ensure no lost notifications. Kafka consumer processes async events.
- **Security & Isolation** – JWT authentication, rate limiting (Redis-based sliding window), and per-user post ownership validation.
- **Performance Optimized** – gRPC client for user validation, Redis cache layer, and structured Zerolog logging with trace IDs.
- **Quality First** – 100% passing unit tests (handler, service, domain), Docker Compose integration tests, and SOLID principles.
---
## Tech Stack
![Go](https://img.shields.io/badge/Go-1.26-blue?logo=go)
![Chi](https://img.shields.io/badge/Chi-v5-green?logo=go)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-17-blue?logo=postgresql)
![Redis](https://img.shields.io/badge/Redis-7-alpine?logo=redis)
![Kafka](https://img.shields.io/badge/Kafka-✓-brown?logo=apachekafka)
![gRPC](https://img.shields.io/badge/gRPC-✓-purple?logo=grpc)
![Docker](https://img.shields.io/badge/Docker-✓-blue?logo=docker)
![JWT](https://img.shields.io/badge/JWT-✓-lightgrey)
![Zerolog](https://img.shields.io/badge/Zerolog-✓-silver)
---
## Quick Start
### Prerequisites
- [Git](https://git-scm.com/)
- [Docker](https://docker.com/) & [Docker Compose](https://docs.docker.com/compose/)
### Steps
```bash
# 1. clone this repository:
git clone https://github.com/disdreamq/fantastic-telegram.git
cd fantastic-telegram
# 2. run the project
docker compose up -d
# 3. after all, delete created network and containers.
docker compose down
```
### After startup, the services will be available at:
**User Service API:** http://localhost:8080  
**Post Service API:** http://localhost:8081  
**Interactive Swagger docs:** http://localhost:8080/swagger/ and http://localhost:8081/swagger/
## API endpoints
### User Service (port 8080)
POST /register – create a user, does not require authentication.  
POST /login – authenticate and obtain a JWT token, does not require authentication.  
GET /users/{userID} – get user by ID, does not require authentication.  
GET /users/email/{email} – get user by email, does not require authentication.  
PUT /users/{userID} – update user, requires authentication (Bearer token).  
DELETE /users/{userID} – delete user, requires authentication (Bearer token).  
### Post Service (port 8081)
POST /posts/ – create a post, requires authentication (Bearer token).  
GET /posts/id/{postID} – get post by ID, does not require authentication.  
GET /posts/title/{title} – get post by title, does not require authentication.  
PUT /posts/{postID} – update a post (ownership validated), requires authentication.  
DELETE /posts/{postID} – delete a post (ownership validated), requires authentication.  
**All endpoints that require authentication expect a valid JWT token in the `Authorization: Bearer <token>` header.**
## Running Tests
Unit tests are written with the standard `testing` package and run locally:
```bash
# All unit tests
go test ./services/post/internal/... -count=1
go test ./services/user/internal/... -count=1
# With coverage
go test ./services/post/internal/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```
Integration tests (require Docker) use Testcontainers:
```bash
go test ./services/post/test/... -count=1
go test ./services/user/test/... -count=1
```
## Architecture Overview
```
  Telegram Bot ──▶ User Service ──▶ Post Service ──▶ Kafka Topic ──▶ Notification Service
                       │                    │                      │
                       ▼                    ▼                      ▼
                  PostgreSQL          Redis Cache            Outbox Table
```
- **Outbox Pattern** – Post service writes post + outbox event atomically in one DB transaction. A background producer publishes to Kafka.
- **gRPC** – Post service validates tokens via gRPC client calling User service.
- **Redis Cache** – Post data cached for 10 minutes; invalidation on update/delete.
- **Rate Limiting** – Redis-based sliding window (different limits for public vs. protected routes).
- **Clean Architecture** – Domain → Service → Port (interface) → Repository/Transport separation.
## Database Migrations
Migrations are managed with golang-migrate and stored in `migrations/`:
```bash
# Apply all migrations (handled automatically by docker-compose)
# Or manually:
migrate -path migrations -database "postgresql://..." up
```
Current migrations:
| # | File | Description |
|---|------|-------------|
| 1 | `000001_init_users_table` | Users table with email, username, password hash |
| 2 | `000002_init_posts_table` | Posts table with user_id, title, content |
| 3 | `000003_adding_index` | Indexes on `posts.user_id`, `posts.title`, `users.email` |
| 4 | `000004_init_outbox_table` | Outbox table for event-driven notifications |

