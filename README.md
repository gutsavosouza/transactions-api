# Transactions API

> A type-safe financial transaction system built with **Go** and **PostgreSQL**. Designed for managing user accounts and secure inter-account money transfers.

<Callout type="info">
This is a modern REST API that handles user management, account creation, and secure transactions with real-time balance updates and comprehensive audit trails.
</Callout>

---

## Features

- **User Management:** Create and manage users, secure password hashing, and personal information endpoints
- **JWT Authentication:** Secure token-based authentication with configurable secret keys and middleware protection
- **Account Management:** Create multiple accounts per user, track balances with decimal precision, and manage account metadata
- **Transaction Processing:** Process secure inter-account transfers with status tracking (pending, completed, failed)

---

## Quick Start

### Prerequisites

- **Go** 1.25.5+
- **PostgreSQL** 16+
- **Docker & Docker Compose** (optional, for containerized database setup)

### Installation

#### Option 1: Using Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/gutsavosouza/transactions-api
cd transactions-api

# Start PostgreSQL with Docker Compose
docker-compose up -d

# Install dependencies
go mod download

# Apply migrations to database with goose
goose up

# Run the API server
go run ./cmd/*.go
```

#### Option 2: Local PostgreSQL

```bash
# Ensure PostgreSQL is running on localhost:5432
createdb transactions

# Set environment variables
export GOOSE_DBSTRING="host=localhost user=postgres password=postgres dbname=transactions sslmode=disable"
export GOOSE_DRIVER=postgres
GOOSE_MIGRATION_DIR=./internal/adapters/postgres/migrations
export JWT_SECRET="your-secret-key-here"

# Install dependencies
go mod download

# Apply migrations to database with goose
goose up

# Run application server
go run ./cmd
```

### Database Setup

Goose handles the migrations, and in order to apply them it is needed to run the proper goose command to up the migrations:

```bash
# Migrations are located in: internal/adapters/postgres/migrations/
# - 001_create_user.sql (Users table)
# - 002_create_account.sql (Accounts table)
# - 003_create_transactions.sql (Transactions table with status enum)
goose up
```

<Callout type="success">
The server will start on `http://localhost:8080` and you can verify it's running with:
```bash
curl http://localhost:8080/healthz
# Response: "OK"
```
</Callout>

---

## Project Structure

```
transctions-api/
├── cmd/
│   ├── main.go          # Application entry point, DB initialization
│   └── api.go           # Router setup, route definitions, server config
├── internal/
│   ├── accounts/        # Account management (handlers, types, business logic)
│   ├── adapters/
│   │   └── postgres/
│   │       ├── migrations/  # SQL migration files (Goose)
│   │       └── sqlc/        # Generated database code
│   ├── authentication/   # JWT & credential management
│   ├── token/           # JWT token generation and validation
│   ├── transactions/    # Transaction processing logic
│   ├── users/          # User management & registration
│   ├── utils/          # Shared utilities (JSON encoding, etc.)
│   └── env/            # Environment variable helpers
├── docker-compose.yaml  # PostgreSQL container configuration
├── go.mod              # Go module dependencies
└── sqlc.yaml           # SQLc code generation config
```

### Module Responsibilities

| Module                | Purpose                                               |
| --------------------- | ----------------------------------------------------- |
| **users**             | User registration, authentication, profile endpoints  |
| **authentication**    | Login logic, JWT middleware, token validation         |
| **accounts**          | CRUD operations for user accounts, balance management |
| **transactions**      | Money transfer processing, transaction history        |
| **adapters/postgres** | Database layer, migrations, generated SQL code (sqlc) |
| **token**             | JWT claims creation, token signing/verification       |
| **utils**             | Shared helpers (JSON response formatting, validation) |

---

## API Endpoints

All endpoints (except `/users/` and `/auth/login`) require JWT authentication via the `Authorization: Bearer <token>` header.

### Users

| Method | Endpoint       | Description              | Auth Required |
| ------ | -------------- | ------------------------ | ------------- |
| `POST` | `/v1/users`    | Register a new user      | ❌            |
| `GET`  | `/v1/users/me` | Get current user profile | ✅            |

### Authentication

| Method | Endpoint         | Description                    | Auth Required |
| ------ | ---------------- | ------------------------------ | ------------- |
| `POST` | `/v1/auth/login` | Authenticate and get JWT token | ❌            |

### Accounts

| Method  | Endpoint                       | Description            | Auth Required |
| ------- | ------------------------------ | ---------------------- | ------------- |
| `POST`  | `/v1/account`                  | Create new account     | ✅            |
| `GET`   | `/v1/account`                  | List all user accounts | ✅            |
| `GET`   | `/v1/account/{id}`             | Get account details    | ✅            |
| `PATCH` | `/v1/account/add-balance/{id}` | Add balance to account | ✅            |

### Transactions

| Method | Endpoint          | Description                      | Auth Required |
| ------ | ----------------- | -------------------------------- | ------------- |
| `POST` | `/v1/transaction` | Create transfer between accounts | ✅            |
| `GET`  | `/v1/transaction` | Get user's transaction history   | ✅            |

---

## 🗄️ Database Schema

The API uses PostgreSQL with three core tables:

### Users Table

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    cpf TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    password TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL
);
```

**Fields:**

- `id`: Unique identifier (UUID)
- `cpf`: Brazilian CPF number (unique, formatted as "XXX.XXX.XXX-XX")
- `name`: User's full name
- `password`: Bcrypt-hashed password
- `created_at` / `updated_at`: Timestamps for audit trail

### Accounts Table

```sql
CREATE TABLE accounts (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE NOT NULL,
    balance NUMERIC(19, 2) DEFAULT 0 NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL
);
```

**Fields:**

- `id`: Unique account identifier
- `user_id`: Foreign key to users table
- `balance`: Account balance with 2 decimal precision (max value: 99,999,999,999,999,999.99)

### Transactions Table

```sql
CREATE TYPE transaction_status AS ENUM ('pending', 'completed', 'failed');

CREATE TABLE transactions (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id) NOT NULL,
    from_account_id UUID REFERENCES accounts(id) NOT NULL,
    to_account_id UUID REFERENCES accounts(id) NOT NULL,
    amount NUMERIC(19, 2) NOT NULL,
    status transaction_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL
);
```

**Fields:**

- `id`: Transaction identifier
- `user_id`: User who initiated the transfer
- `from_account_id` / `to_account_id`: Source and destination accounts
- `amount`: Transfer amount with 2 decimal precision
- `status`: Current transaction state (pending → completed/failed)

### Entity Relationship Diagram

```
┌──────────────┐
│    USERS     │
│──────────────│
│ • id (PK)    │
│ • cpf (U)    │
│ • name       │
│ • password   │
└──────────────┘
       │
       │ 1:N
       ├─────────────────┐
       │                 │
       ▼                 ▼
┌──────────────┐  ┌──────────────────┐
│  ACCOUNTS    │  │  TRANSACTIONS    │
│──────────────│  │──────────────────│
│ • id (PK)    │  │ • id (PK)        │
│ • user_id(FK)│  │ • user_id (FK)   │
│ • balance    │  │ • from_acct (FK) │
│ • created_at │  │ • to_acct (FK)   │
│ • updated_at │  │ • amount         │
└──────────────┘  │ • status         │
       │          │ • created_at     │
       │          │ • updated_at     │
       └──────────┴──────────────────┘
        (referenced by transactions)
```

---

## 🛠️ Tech Stack

| Category                | Technology              | Purpose                                          |
| ----------------------- | ----------------------- | ------------------------------------------------ |
| **Language**            | Go 1.25.5               | High-performance, compiled backend               |
| **Web Framework**       | chi/v5                  | Lightweight, composable HTTP router              |
| **Database**            | PostgreSQL 16           | Reliable, ACID-compliant SQL database            |
| **Database Driver**     | pgx/v5                  | Native PostgreSQL driver with connection pooling |
| **Migrations**          | Goose                   | Database versioning and migrations               |
| **Code Generation**     | sqlc                    | Type-safe SQL from hand-written queries          |
| **Authentication**      | golang-jwt/v5           | Standard JWT token management                    |
| **Cryptography**        | golang.org/x/crypto     | Bcrypt password hashing                          |
| **Validation**          | go-playground/validator | Struct validation with tags                      |
| **Container**           | Docker & Compose        | Local development environment                    |
| **Document Validation** | brdoc                   | Brazilian document validation (CPF)              |

---

## 📝 Development

### Build from Source

```bash
# Download dependencies
go mod download

# Build executable
go build -o transactions-api ./cmd

# Run the built binary
./transactions-api
```

### Code Generation

The project uses **sqlc** to generate type-safe database code:

```bash
# Install sqlc
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Regenerate code after schema changes
sqlc generate
```

Configuration is defined in `sqlc.yaml`.

### Project Dependencies

```
github.com/go-chi/chi/v5               - HTTP routing
github.com/golang-jwt/jwt/v5           - JWT tokens
github.com/jackc/pgx/v5                - PostgreSQL driver
github.com/go-playground/validator/v10 - Input validation
github.com/google/uuid                 - UUID generation
golang.org/x/crypto                    - Security utilities
github.com/shopspring/decimal           - Decimal math
github.com/paemuri/brdoc                - Brazilian document validation
```
