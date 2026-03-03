## Security hardening checklist

### mTLS (recommended for kiosks)

Terminate TLS and **mTLS** at your API gateway / load balancer (NGINX / Envoy / API Gateway).

- **Server verifies kiosk client certificates** (per-device certs issued by your internal CA).
- Gateway forwards only verified requests to the Go API over a private network.

Sample NGINX include file is provided at `deploy/nginx/mtls.conf`.

### HMAC tuning (kiosk routes)

Kiosk routes are protected by HMAC (server reads `kiosks.hmac_secret`).

- **Timestamp window**: configurable with `HMAC_MAX_SKEW_SECONDS` (default `300`).
  - Suggested production value: `120` (2 minutes) if kiosk clocks are reasonably synced.
- **Signature**: HMAC-SHA256 over:
  \[
  message = body + timestamp + kiosk_code
  \]
  where `timestamp` is **Unix seconds**.

### Offline encryption (server-side decryption)

Offline payloads are encrypted on the kiosk device using an envelope scheme:

- **AES-256-GCM** encrypts the plaintext payload.
- The AES key is encrypted with **RSA-OAEP(SHA-256)** using the **public key** in the frontend.
- The Go backend decrypts using the **RSA private key**.

Required environment variables:

- **Frontend**: `NEXT_PUBLIC_ENCRYPTION_PUBLIC_KEY` (PEM, `BEGIN PUBLIC KEY`)
- **Backend**: `OFFLINE_PRIVATE_KEY_PEM` (PEM, PKCS#8 or PKCS#1)

Kiosk offline sync endpoint:

- `POST /api/v1/kiosk/offline/sync` (HMAC-authenticated)
  - Body: `{ "encrypted_payload": "<envelope-json-string>" }`

