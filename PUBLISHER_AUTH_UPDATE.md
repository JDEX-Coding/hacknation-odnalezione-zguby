# Publisher Authentication Update - Summary

## Changes Made

Updated Service C (Publisher) to use **email/password login authentication** instead of API key authentication.

---

## Modified Files

### 1. **`service-c-publisher/internal/client/dane_gov_client.go`**

**Changes:**
- Added `Login()` method to authenticate with dane.gov.pl API
- Changed from `apiKey` to `email`, `password`, and `token` fields
- Updated `NewDaneGovClient()` to accept email/password instead of API key
- Token is obtained via login and stored for subsequent API calls
- All API requests now use the JWT token obtained from login

**Key Addition:**
```go
func (c *DaneGovClient) Login(ctx context.Context) error {
    // POST to /api/v1/auth/login with email/password
    // Receives JWT token in response
    // Stores token for subsequent API calls
}
```

### 2. **`service-c-publisher/main.go`**

**Changes:**
- Updated `Config` struct to use `DaneGovEmail` and `DaneGovPassword`
- Removed `DaneGovAPIKey` field
- Added login call during initialization
- Application now logs in before starting to consume messages

**Flow:**
```go
1. Initialize client with email/password
2. Call Login() to get JWT token
3. Check API health
4. Start consuming messages
```

### 3. **`docker-compose.yml`**

**Changes:**
- Replaced `DANE_GOV_API_KEY` with:
  - `DANE_GOV_EMAIL`
  - `DANE_GOV_PASSWORD`

**Before:**
```yaml
- DANE_GOV_API_KEY=${DANE_GOV_API_KEY:-}
```

**After:**
```yaml
- DANE_GOV_EMAIL=${DANE_GOV_EMAIL:-}
- DANE_GOV_PASSWORD=${DANE_GOV_PASSWORD:-}
```

### 4. **`.env.example`**

**Added:**
```env
# dane.gov.pl API Login (for Service C Publisher)
DANE_GOV_API_URL=http://localhost:8000
DANE_GOV_EMAIL=admin2@mcod.local
DANE_GOV_PASSWORD=Hacknation-2025
PUBLISHER_ID=org-001
```

### 5. **`service-c-publisher/mock-api.go`**

**Added login endpoint:**
```go
http.HandleFunc("/api/v1/auth/login", ...)
```

- Accepts POST requests with email/password
- Returns mock JWT token
- Simulates real API login flow

### 6. **Documentation Files**

Updated:
- `service-c-publisher/README.md` - Updated configuration table
- `service-c-publisher/TESTING.md` - Updated test credentials
- `QUICKSTART_PUBLISHER.md` - Updated quick start examples

---

## Authentication Flow

### Before (API Key)
```
Publisher → API Request (with API key in header) → dane.gov.pl
```

### After (Login)
```
1. Publisher → POST /api/v1/auth/login (email/password) → dane.gov.pl
2. Publisher ← JWT Token ← dane.gov.pl
3. Publisher → API Requests (with JWT token in header) → dane.gov.pl
```

---

## Configuration Changes

### Old Environment Variables
```env
DANE_GOV_API_URL=http://localhost:8000
DANE_GOV_API_KEY=your-api-key-here
```

### New Environment Variables
```env
DANE_GOV_API_URL=http://localhost:8000
DANE_GOV_EMAIL=admin@mcod.local
DANE_GOV_PASSWORD=your-password
```

---

## Testing

### Local Testing with Mock API

1. **Start mock API:**
```powershell
cd service-c-publisher
go run mock-api.go
```

2. **Set credentials:**
```powershell
$env:DANE_GOV_API_URL="http://localhost:8000"
$env:DANE_GOV_EMAIL="test@example.com"
$env:DANE_GOV_PASSWORD="test-password"
```

3. **Run publisher:**
```powershell
go run main.go
```

4. **Expected output:**
```
🚀 Starting Service C: Publisher
Logging in to dane.gov.pl...
✅ Successfully logged in to dane.gov.pl email=test@example.com
✅ dane.gov.pl API is healthy
✅ Publisher service initialized successfully
🎧 Listening for messages on RabbitMQ...
```

### Docker Testing

```powershell
# Build with new authentication
docker build -t odnalezione-publisher:latest ./service-c-publisher

# Run with credentials
docker run --rm \
  -e RABBITMQ_URL=amqp://admin:admin123@rabbitmq:5672/ \
  -e DANE_GOV_API_URL=http://host.docker.internal:8000 \
  -e DANE_GOV_EMAIL=admin@mcod.local \
  -e DANE_GOV_PASSWORD=test-password \
  --network odnalezione-network \
  odnalezione-publisher:latest
```

---

## Migration Guide

If you have an existing deployment:

1. **Update `.env` file:**
```bash
# Remove old variable
# DANE_GOV_API_KEY=...

# Add new variables
DANE_GOV_EMAIL=your-email@example.com
DANE_GOV_PASSWORD=your-password
```

2. **Update docker-compose:**
```bash
docker compose down
docker compose pull
docker compose up --build -d c-publisher
```

3. **Verify logs:**
```bash
docker compose logs c-publisher | grep "Successfully logged in"
```

---

## Security Considerations

1. **Credentials Storage:**
   - Store email/password in `.env` file (git-ignored)
   - Never commit credentials to version control
   - Use secrets management in production

2. **Token Handling:**
   - JWT token is stored in memory only
   - Token is refreshed on service restart
   - No token persistence to disk

3. **Production Recommendations:**
   - Use environment variable injection
   - Consider using Kubernetes secrets
   - Implement token refresh logic for long-running instances
   - Monitor login failures

---

## Troubleshooting

### Login fails
```
Error: login failed with status 401: Invalid credentials
```

**Solution:**
- Verify `DANE_GOV_EMAIL` and `DANE_GOV_PASSWORD` are correct
- Check if account exists in dane.gov.pl
- Ensure API URL is correct

### Token expired during operation
Currently, tokens don't expire in the mock API. For production:
- Implement token refresh logic
- Handle 401 responses by re-authenticating
- Add retry mechanism

### Mock API not responding
```
Error: failed to send login request: dial tcp: connection refused
```

**Solution:**
- Ensure mock API is running on port 8000
- Check `DANE_GOV_API_URL` matches mock API address
- Verify firewall/network settings

---

## Benefits of This Change

✅ **More Secure:** Username/password can be rotated independently  
✅ **Standard Auth Flow:** Matches typical API authentication patterns  
✅ **Better for Testing:** Can use different accounts for testing  
✅ **Aligns with Example:** Matches go-api-example authentication  
✅ **Session Management:** JWT tokens can have expiration/refresh  

---

## Files Not Changed

- RabbitMQ consumer logic (unchanged)
- DCAT formatter (unchanged)
- Message handling flow (unchanged)
- Event models (unchanged)

The authentication change is isolated to the API client layer only.
