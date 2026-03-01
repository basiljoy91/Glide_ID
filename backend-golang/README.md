# Enterprise Attendance API - Golang Backend

Core API backend for the Enterprise Facial Recognition Attendance & Identity System.

## Features

- **JWT Authentication**: Secure token-based authentication
- **SSO Support**: SAML 2.0/OIDC integration hooks
- **HMAC Verification**: Kiosk request signing and verification
- **Offline Time Reconciliation**: Monotonic clock support for offline kiosks
- **HRMS Integration**: Webhook endpoints for Workday, SAP, and custom HRMS
- **MQTT/WebSocket**: IoT door relay integration
- **RBAC**: Role-Based Access Control middleware
- **Audit Logging**: Comprehensive audit trail
- **Health Check**: Lightweight endpoint for frontend silent ping

## Architecture

```
backend-golang/
├── main.go                    # Application entry point
├── go.mod                     # Go module dependencies
├── internal/
│   ├── config/               # Configuration management
│   ├── database/             # PostgreSQL connection pool
│   ├── models/               # Data models
│   ├── middleware/           # Auth, HMAC, RBAC middleware
│   ├── services/             # Business logic
│   ├── handlers/             # HTTP request handlers
│   ├── router/               # Route setup
│   └── mqtt/                 # MQTT client for IoT
```

## Setup

### 1. Install Dependencies

```bash
go mod download
```

### 2. Configure Environment

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
```

Key variables:
- `DATABASE_URL`: Supabase PostgreSQL connection string
- `JWT_SECRET`: Secure random string for JWT signing
- `AI_SERVICE_URL`: Python AI microservice URL
- `AI_SERVICE_API_KEY`: API key for AI service

### 3. Run the Server

```bash
go run main.go
```

Or build and run:

```bash
go build -o bin/server main.go
./bin/server
```

## API Endpoints

### Health Check

```http
GET /health
```

Lightweight health check for frontend silent ping.

### Authentication

```http
POST /api/v1/public/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password"
}
```

### Kiosk Check-In (HMAC Authenticated)

```http
POST /api/v1/kiosk/check-in
X-Kiosk-Code: 1234567890
X-HMAC-Signature: <signature>
X-Timestamp: <unix_timestamp>
Content-Type: application/json

{
  "image_base64": "<base64_image>",
  "kiosk_code": "1234567890",
  "local_time": "2024-01-01T10:00:00Z",
  "monotonic_offset_ms": 5000,
  "verification_method": "biometric",
  "ip_address": "192.168.1.1"
}
```

### HRMS Webhook

```http
POST /webhooks/hrms/:provider
X-Tenant-ID: <tenant-uuid>
X-Webhook-Signature: <hmac_signature>
Content-Type: application/json

{
  "event": "user.created",
  "data": { ... }
}
```

## Security

### HMAC Signature Generation

Kiosks must sign requests with HMAC-SHA256:

```go
message := body + timestamp + kiosk_code
signature := HMAC-SHA256(message, hmac_secret)
```

### JWT Token

Tokens include:
- `user_id`: User UUID
- `tenant_id`: Tenant UUID
- `role`: User role (org_admin, hr, dept_manager, employee)
- `email`: User email

### Row-Level Security

All database queries respect RLS policies. The application sets:
- `app.current_tenant_id`
- `app.current_user_id`
- `app.is_ai_service` (for AI service operations)

## Offline Time Reconciliation

When a kiosk goes offline:
1. Kiosk records `local_time` and `monotonic_offset_ms`
2. Server calculates true punch time:
   ```go
   true_time = local_time + monotonic_offset_ms
   ```

## MQTT Integration

The API publishes door relay commands via MQTT:

```json
{
  "action": "open",
  "user_id": "<uuid>",
  "kiosk_id": "<uuid>",
  "timestamp": 1234567890
}
```

## HRMS Integration

### Supported Providers

- **Workday**: Webhook processing for user provisioning
- **SAP**: SAP SuccessFactors integration
- **Custom**: Generic webhook handler

### Webhook Payload Format

```json
{
  "event": "user.created",
  "user": {
    "employee_id": "EMP001",
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "department": "Engineering"
  }
}
```

## Deployment

### Render/Koyeb

1. Set environment variables in dashboard
2. Deploy from Git repository
3. Ensure `DATABASE_URL` is configured

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/server .
CMD ["./server"]
```

## Monitoring

- Health check: `/health`
- Logs: Check application logs for errors
- Database: Monitor connection pool usage

## Troubleshooting

### Database Connection Issues

- Verify `DATABASE_URL` format
- Check Supabase firewall rules
- Ensure RLS policies allow API access

### HMAC Verification Fails

- Verify kiosk HMAC secret matches database
- Check timestamp is within 5-minute window
- Ensure signature includes body + timestamp + kiosk_code

### MQTT Connection Fails

- MQTT is optional - service continues without it
- Verify `MQTT_BROKER_URL` is correct
- Check broker credentials

## License

Proprietary - Enterprise Attendance System

