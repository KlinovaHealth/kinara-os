# Patient Service — Kinara OS

Manages encrypted patient records for the Kinara OS platform.
All PHI (Protected Health Information) is AES-256-GCM encrypted before touching PostgreSQL.
Every read and write produces an immutable audit log entry.

**Part of:** Kinara OS — "Where Africa Coordinates"  
**Owner:** Klinova LLC  
**Language:** Go 1.21  
**Auth:** JWT (external) + mTLS (inter-service)

---

## API Endpoints

| Method | Path | Role Required | Description |
|--------|------|---------------|-------------|
| `POST` | `/api/v1/patients` | admin, nurse, doctor, frontdesk | Create patient |
| `GET` | `/api/v1/patients` | admin, doctor, nurse, analyst, government | List patients (paginated) |
| `GET` | `/api/v1/patients/:id` | any authenticated | Get patient by ID |
| `PUT` | `/api/v1/patients/:id` | admin, nurse, doctor | Update patient |
| `DELETE` | `/api/v1/patients/:id` | admin | Soft-delete patient |
| `GET` | `/api/v1/patients/:id/audit` | admin, analyst | Get immutable audit log |
| `GET` | `/health` | — | Liveness probe |
| `GET` | `/ready` | — | Readiness probe |

---

## Request Examples

### Create Patient
```bash
curl -X POST https://patient-service.kinara.internal/api/v1/patients \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "national_id": "GH-1234567890",
    "full_name": "Akosua Asante",
    "date_of_birth": "1990-05-15",
    "gender": "female",
    "phone_number": "+233 20 000 1234",
    "country": "Ghana",
    "region": "Greater Accra",
    "blood_type": "O+",
    "allergies": ["penicillin"],
    "emergency_contact": {
      "name": "Kwame Asante",
      "phone": "+233 20 000 5678",
      "relationship": "spouse"
    }
  }'
```

### Response
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "national_id": "GH-1234567890",
    "full_name": "Akosua Asante",
    "date_of_birth": "1990-05-15T00:00:00Z",
    "gender": "female",
    "phone_number": "+233 20 000 1234",
    "country": "Ghana",
    "region": "Greater Accra",
    "blood_type": "O+",
    "allergies": ["penicillin"],
    "status": "active",
    "created_at": "2026-08-29T10:00:00Z"
  }
}
```

### List Patients (with filters)
```bash
curl "https://patient-service.kinara.internal/api/v1/patients?country=Ghana&status=active&page=1&limit=20" \
  -H "Authorization: Bearer $JWT_TOKEN"
```

---

## Database Schema

```sql
patients (
  id UUID PK,
  national_id_enc TEXT,       -- AES-256-GCM encrypted
  full_name_enc TEXT,         -- AES-256-GCM encrypted
  date_of_birth_enc TEXT,     -- AES-256-GCM encrypted
  gender TEXT,                -- NOT encrypted (indexed)
  phone_number_enc TEXT,      -- AES-256-GCM encrypted
  email_enc TEXT,             -- AES-256-GCM encrypted
  address_enc TEXT,           -- AES-256-GCM encrypted
  country TEXT,               -- NOT encrypted (indexed for analytics)
  region TEXT,                -- NOT encrypted (indexed for analytics)
  blood_type_enc TEXT,        -- AES-256-GCM encrypted
  allergies_enc TEXT,         -- AES-256-GCM encrypted JSON array
  emergency_contact_*_enc TEXT, -- AES-256-GCM encrypted
  status TEXT,                -- active | inactive | deceased | transferred
  created_by UUID,
  created_at TIMESTAMPTZ,
  updated_at TIMESTAMPTZ,
  deleted_at TIMESTAMPTZ      -- soft delete
)

patient_audit_log (
  id UUID PK,
  patient_id UUID FK,
  action TEXT,                -- create | read | update | delete | search
  accessor_id UUID,
  accessor_role TEXT,
  ip_address TEXT,
  request_id TEXT,
  changes JSONB,
  created_at TIMESTAMPTZ
  -- UPDATE and DELETE are blocked by PostgreSQL rules (immutable)
)
```

---

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `ENCRYPTION_KEY` | Yes | 64-char hex string (32 bytes, AES-256) |
| `JWT_PUBLIC_KEY_PATH` | Yes | Path to RSA public key PEM |
| `TLS_CERT_PATH` | Yes | Path to service TLS certificate |
| `TLS_KEY_PATH` | Yes | Path to service TLS private key |
| `CA_CERT_PATH` | Yes | Path to Kinara OS internal CA cert |
| `REDIS_ADDR` | No | Redis address (default: `localhost:6379`) |
| `REDIS_PASSWORD` | No | Redis password |
| `PORT` | No | HTTP port (default: `8080`) |

---

## Local Development

```bash
# Generate a 32-byte encryption key
openssl rand -hex 32

# Run PostgreSQL
docker run -d --name kinara-postgres \
  -e POSTGRES_DB=kinara_patient \
  -e POSTGRES_PASSWORD=devpassword \
  -p 5432:5432 postgres:16-alpine

# Apply migration
psql $DATABASE_URL -f db/migrations/001_create_patients.sql

# Run tests
go test ./... -v -cover

# Build
go build -o patient-service ./main.go
```

---

## Deployment (DigitalOcean / Kubernetes)

```bash
# Build and push Docker image
docker build -t registry.digitalocean.com/kinara-os/patient-service:latest .
docker push registry.digitalocean.com/kinara-os/patient-service:latest

# Create Kubernetes secrets
kubectl create secret generic patient-service-secrets \
  --from-literal=database-url="$DATABASE_URL" \
  --from-literal=encryption-key="$ENCRYPTION_KEY" \
  --from-literal=redis-addr="$REDIS_ADDR" \
  --from-literal=redis-password="$REDIS_PASSWORD" \
  -n kinara-staging

# Deploy
kubectl apply -f k8s/deployment.yaml
kubectl rollout status deployment/patient-service -n kinara-staging
```

---

## Security

- **Encryption:** AES-256-GCM, random nonce per field per write
- **Auth:** RS256 JWT (external callers) + mTLS (internal services)
- **Rate limiting:** 1,000 req/min per mTLS certificate CN
- **Audit:** Immutable insert-only audit log — every read and write logged
- **Soft delete:** Patient records are never physically destroyed
- **Zero trust:** No request is trusted without a valid JWT and client cert
- **No secrets in code:** All keys loaded from environment variables at startup
