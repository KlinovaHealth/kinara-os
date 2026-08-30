# Kinara OS — Device Security Policy

**Version:** 1.0  
**Classification:** Gates Foundation Deliverable  
**Scope:** All offline-capable clinic devices running the Kinara mobile client

---

## 1. Cache Scope and TTL

Devices cache a bounded snapshot of clinic data to enable offline-first operation in low-connectivity environments.

| Parameter | Value |
|---|---|
| Record cap | 200 records per device |
| Lookback window | Last 72 hours of patient visits |
| Cache population trigger | `POST /sync/pull` at device startup or re-auth |
| `expires_at` field | Included on every record (now + 72h) |
| Server enforcement | The pull endpoint queries only `visits WHERE clinic_id = ? AND visit_date >= now() - 72h LIMIT 200` |

Devices **may not** request a wider window or override the cap. The server enforces both constraints independently of client behavior.

---

## 2. Encryption at Rest

### Key Derivation

The device encryption key is never persisted. It exists in memory only for the duration of an active session.

```
key = PBKDF2-SHA256(
    password = staff_PIN (6 digits),
    salt     = device_secret (32 bytes, issued once at enrollment, stored by staff),
    iterations = 600_000,
    keylen = 32
)
```

The `device_secret` is returned **once** at enrollment via `POST /devices/enroll`. The server stores only the argon2id hash. Staff must store the secret in a secure credential store (e.g., password manager or printed and locked).

### Memory Lifecycle

- Key is derived at PIN entry and held in a zeroed memory buffer.
- Derived key is wiped (overwritten with zeros) on: session idle timeout (5 min), logout, device lock, or power-off.
- After key wipe, only ciphertext remains on disk — data is inaccessible without re-deriving the key.

### Cipher

All locally cached PHI is encrypted with **AES-256-GCM**. Each record has a unique 12-byte random nonce prepended to the ciphertext. Authentication tag validates both ciphertext integrity and nonce.

---

## 3. PIN and Session Policy

| Parameter | Value |
|---|---|
| PIN format | 6 digits |
| Session idle timeout | 5 minutes (inactivity wipe of in-memory key) |
| Failed PIN attempts before wipe | 10 |
| Wipe action | Local cache cleared; app remains installed; re-sync restores queue after re-auth |
| Re-sync after wipe | Device must call `POST /sync/pull` with a fresh device JWT after re-auth |

Failed PIN counter is stored locally and survives reboots. After wipe the counter resets.

---

## 4. Staleness Wipe

Devices enforce their own staleness check **without requiring cluster contact**. This protects against cases where a device is offline for an extended period and its cached data becomes dangerously stale.

| Parameter | Value |
|---|---|
| Staleness threshold | 7 days since last successful sync |
| Enforcement location | Both: client-enforced on startup AND server-enforced on `POST /sync/pull` and `GET /sync/status` |
| Wipe trigger | Local cached data cleared; app prompts for re-auth and re-sync |
| Server response when stale | `401 {"wipe": true, "reason": "stale_7_days"}` |

Client staleness check: On every app startup, compare `last_synced_at` (stored locally) against `now()`. If `now() - last_synced_at > 7 days`, wipe cache before displaying any data.

---

## 5. Clinic Scope Enforcement

All scope enforcement is **server-side in the JWT claim**. Client queries are never trusted for scoping.

### Token Shape (Device Session JWT, RS256, 5-minute TTL)

```json
{
  "uid": "<staff_uuid>",
  "role": "nurse",
  "device_id": "<device_uuid>",
  "clinic_id": "<clinic_uuid>",
  "scope": "clinic:<clinic_uuid>",
  "scopes": ["clinic:<clinic_uuid>"],
  "iat": 1234567890,
  "exp": 1234568190
}
```

### Server Enforcement Chain

1. `JWT` middleware validates RS256 signature and expiry.
2. `RequireClinicScope` middleware (all `/sync/*` routes): rejects any token where `scope` does not start with `"clinic:"` — returns **403**.
3. Pull handler: queries `WHERE clinic_id = <clinic_id from JWT claim>` — the claim value is never derived from the client request body.
4. Push handler: validates `patient_id` against `visits WHERE clinic_id = <claim>` before accepting any write. Out-of-scope patient → **rejected** with `"patient_not_in_clinic_scope"`.

A device credential scoped to clinic X will **never** receive or accept data for clinic Y, regardless of what the client sends.

---

## 6. Revocation

### Device Registry

Every enrolled device is recorded with:
- `id`, `device_name`, `clinic_id`, `assigned_staff_id`
- `device_secret_hash` (argon2id, never plaintext)
- `enrolled_at`, `last_seen_at`, `revoked_at`, `revoked_reason`

### Revocation Flow

1. Admin calls `POST /devices/{id}/revoke` with a reason.
2. `revoked_at` and `revoked_reason` set in database.
3. Next sync call (pull, push, or heartbeat) returns `401 {"wipe": true, "reason": "device_revoked"}`.
4. Client clears local cache immediately on receiving any wipe directive.
5. All events are written to the immutable `device_audit_log` table.

Revocation takes effect on the next server contact. Device JWT TTL is 5 minutes — maximum window before a revoked device is denied is 5 minutes plus the next sync attempt.

### Heartbeat

Devices call `POST /devices/{id}/heartbeat` periodically (recommended: every 15 minutes when online).

Response:
```json
{"wipe": false, "last_seen_at": "...", "cache_expires": "..."}
```
or:
```json
{"wipe": true, "reason": "device_revoked|stale_7_days"}
```

---

## 7. Sync and Idempotency

### Push Idempotency

Each write from the device carries a unique `idempotency_key` (`device_id + local_sequence_number`). The `sync_queue` table has a `UNIQUE(device_id, idempotency_key)` constraint. Duplicate pushes return `{"status": "duplicate"}` without reprocessing.

### Conflict Resolution

| Payload Type | Resolution |
|---|---|
| `consultation` | Last-write-wins (newer `received_at` applied) |
| `vital_signs` | Last-write-wins |
| `prescription` | Append-only — new record created, previous never modified |
| `referral` | Append-only — new record created, previous never modified |

The `sync_queue` table has a PostgreSQL `RULE DO INSTEAD NOTHING` on `UPDATE` and `DELETE`, making all records immutable after insertion. Conflict resolution is handled at application layer before insert.

---

## 8. Stolen Device Threat Model

| Threat | Mitigation |
|---|---|
| Device stolen while screen locked / idle | Key wiped after 5-min idle; attacker sees only AES-256-GCM ciphertext |
| Device stolen while active session | 10-PIN-attempt limit; failed attempts persist across reboots |
| Device stolen after 7+ days offline | Staleness wipe fires on next startup; no current PHI accessible |
| Attacker attempts server sync with stolen device | Admin revokes via dashboard; next server contact returns wipe directive |
| Attacker clones device storage | Ciphertext only; key derivation requires both staff PIN and device secret (2-factor) |
| Attacker intercepts network traffic | mTLS (TLS 1.3, mutual certificate auth) on all service-to-service and device-to-server connections |
| Compromised JWT used for wrong clinic | `RequireClinicScope` server middleware enforces clinic_id from claim; no client-supplied clinic_id accepted |

---

## 9. Audit Log

All device lifecycle events are written to `device_audit_log` with:
- `device_id`, `event` (enrolled|heartbeat|wipe_triggered|revoked), `actor_id`, `ip_address`, `created_at`

The table has PostgreSQL RULEs that silently ignore `UPDATE` and `DELETE` — all entries are permanent and tamper-resistant at the database layer.

---

## 10. Security Standards Summary

| Control | Implementation |
|---|---|
| Encryption in transit | mTLS (TLS 1.3), all services |
| Encryption at rest | AES-256-GCM, memory-only key |
| Key derivation | PBKDF2-SHA256 (600k iterations) |
| Device secret storage | argon2id hash only |
| Authentication | JWT RS256, 5-minute TTL for device sessions |
| Authorization | Clinic-scoped JWT claim, server-enforced |
| Audit | Immutable PostgreSQL audit log |
| PHI in list views | `national_id_enc` never returned in list endpoints |
| Session management | 5-min idle wipe, 10-attempt PIN lockout |
| Revocation | Admin dashboard → database → next-sync enforcement |
