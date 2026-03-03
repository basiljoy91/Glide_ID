# Kiosk Offline Mode Setup Guide (Step by Step)

This guide explains how to configure **offline kiosk mode** for Glide ID so kiosk check-ins can be queued securely when internet is unavailable and synced later.

---

## What offline mode requires

Offline mode in this project uses:

- **Frontend public key** to encrypt queued payloads in the browser (`NEXT_PUBLIC_ENCRYPTION_PUBLIC_KEY`)
- **Backend private key** to decrypt payloads during sync (`OFFLINE_PRIVATE_KEY_PEM`)
- Existing kiosk HMAC secret flow for request signing

If either key is missing, offline queueing will fail.

---

## 1) Generate RSA key pair

From project root (`Glide_ID`), run:

```bash
mkdir -p keys
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out keys/kiosk_offline_private.pem
openssl rsa -pubout -in keys/kiosk_offline_private.pem -out keys/kiosk_offline_public.pem
```

You should now have:

- `keys/kiosk_offline_private.pem`
- `keys/kiosk_offline_public.pem`

---

## 2) Configure frontend env (`.env.local`)

Create or edit:

- `frontend-nextjs/.env.local`

Add/update:

```env
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_AI_SERVICE_URL=http://localhost:8000
NEXT_PUBLIC_ENABLE_OFFLINE_MODE=true
NEXT_PUBLIC_ENCRYPTION_PUBLIC_KEY="-----BEGIN PUBLIC KEY-----\n...YOUR_PUBLIC_KEY_CONTENT...\n-----END PUBLIC KEY-----"
```

### How to fill `NEXT_PUBLIC_ENCRYPTION_PUBLIC_KEY`

Copy the full content of `keys/kiosk_offline_public.pem` and paste it into the env value using `\n` between lines.

Example helper command (macOS):

```bash
awk 'NF {sub(/\r/, ""); printf "%s\\n",$0;}' keys/kiosk_offline_public.pem
```

Copy output into `NEXT_PUBLIC_ENCRYPTION_PUBLIC_KEY="..."`.

---

## 3) Configure backend env (`.env`)

Create or edit:

- `backend-golang/.env`

Add/update:

```env
OFFLINE_PRIVATE_KEY_PEM="-----BEGIN PRIVATE KEY-----\n...YOUR_PRIVATE_KEY_CONTENT...\n-----END PRIVATE KEY-----"
```

### How to fill `OFFLINE_PRIVATE_KEY_PEM`

Copy full content of `keys/kiosk_offline_private.pem` and convert line breaks to `\n`.

Example helper command (macOS):

```bash
awk 'NF {sub(/\r/, ""); printf "%s\\n",$0;}' keys/kiosk_offline_private.pem
```

Copy output into `OFFLINE_PRIVATE_KEY_PEM="..."`.

---

## 4) Restart services (required)

After env changes, restart both services:

### Backend

```bash
cd backend-golang
go run main.go
```

### Frontend

```bash
cd frontend-nextjs
npm run dev
```

If either service was already running before env changes, restart it so new env values are loaded.

---

## 5) Verify offline mode works

1. Open kiosk page in browser.
2. Ensure kiosk HMAC secret is configured in kiosk UI.
3. Disconnect internet (or simulate offline in DevTools).
4. Capture a check-in image.
5. Expected: item is saved to offline queue successfully.
6. Reconnect internet.
7. Expected: queue sync posts to backend `/api/v1/kiosk/offline/sync` and marks records synced.

---

## 6) Common errors and fixes

### Error: `Offline mode is not configured on this kiosk device`

Cause:
- `NEXT_PUBLIC_ENCRYPTION_PUBLIC_KEY` is missing or invalid in frontend env.

Fix:
- Set a valid PEM public key in `frontend-nextjs/.env.local` and restart frontend.

---

### Error: `offline decryption not configured`

Cause:
- `OFFLINE_PRIVATE_KEY_PEM` is missing in backend env.

Fix:
- Set private key in `backend-golang/.env` and restart backend.

---

### Error: sync fails with decryption errors

Cause:
- Public key and private key do not belong to the same key pair.

Fix:
- Regenerate a key pair and update both env values from the same generated pair.

---

## 7) Security recommendations

- Never commit private keys to Git.
- Keep `keys/kiosk_offline_private.pem` restricted (server/admin only).
- Public key can be distributed to kiosk frontend.
- Rotate keys periodically and redeploy both frontend/backend when rotating.

---

## 8) Quick checklist

- [ ] Generated RSA key pair under `keys/`
- [ ] Set `NEXT_PUBLIC_ENCRYPTION_PUBLIC_KEY` in `frontend-nextjs/.env.local`
- [ ] Set `OFFLINE_PRIVATE_KEY_PEM` in `backend-golang/.env`
- [ ] Restarted frontend and backend
- [ ] Verified offline capture + online sync

---

If you want, the next improvement is adding a kiosk startup banner that immediately warns when offline encryption env is missing (before first capture).
