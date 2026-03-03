function pemToArrayBuffer(pem: string): ArrayBuffer {
  const clean = pem
    .replace(/-----BEGIN PUBLIC KEY-----/g, '')
    .replace(/-----END PUBLIC KEY-----/g, '')
    .replace(/\s+/g, '')
  const bin = atob(clean)
  const bytes = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i)
  return bytes.buffer
}

function bytesToBase64(bytes: Uint8Array): string {
  let binary = ''
  for (let i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i])
  return btoa(binary)
}

function base64ToBytes(b64: string): Uint8Array {
  const bin = atob(b64)
  const out = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i)
  return out
}

export async function importRsaOaepPublicKey(pem: string): Promise<CryptoKey> {
  const spki = pemToArrayBuffer(pem)
  return crypto.subtle.importKey(
    'spki',
    spki,
    { name: 'RSA-OAEP', hash: 'SHA-256' },
    false,
    ['encrypt']
  )
}

export async function rsaOaepEncryptBase64(publicKey: CryptoKey, data: Uint8Array): Promise<string> {
  const buf = await crypto.subtle.encrypt(
    { name: 'RSA-OAEP' },
    publicKey,
    data.buffer as ArrayBuffer
  )
  return bytesToBase64(new Uint8Array(buf))
}

export async function aesGcmEncryptBase64(plaintext: Uint8Array): Promise<{
  keyRaw: Uint8Array
  ivB64: string
  ctB64: string
}> {
  const key = await crypto.subtle.generateKey({ name: 'AES-GCM', length: 256 }, true, [
    'encrypt',
  ])
  const keyRaw = new Uint8Array(await crypto.subtle.exportKey('raw', key))
  const iv = crypto.getRandomValues(new Uint8Array(12))
  const pt = plaintext.buffer.slice(
    plaintext.byteOffset,
    plaintext.byteOffset + plaintext.byteLength
  ) as ArrayBuffer
  const ct = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, key, pt)
  return { keyRaw, ivB64: bytesToBase64(iv), ctB64: bytesToBase64(new Uint8Array(ct)) }
}

export async function hmacSha256Hex(secret: string, message: string): Promise<string> {
  const enc = new TextEncoder()
  const key = await crypto.subtle.importKey('raw', enc.encode(secret).buffer as ArrayBuffer, { name: 'HMAC', hash: 'SHA-256' }, false, [
    'sign',
  ])
  const sig = await crypto.subtle.sign('HMAC', key, enc.encode(message).buffer as ArrayBuffer)
  const bytes = new Uint8Array(sig)
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('')
}

export function decodeEncryptedEnvelope(envelopeJson: string): {
  alg: string
  ek: Uint8Array
  iv: Uint8Array
  ct: Uint8Array
} {
  const obj = JSON.parse(envelopeJson)
  return {
    alg: obj.alg,
    ek: base64ToBytes(obj.ek),
    iv: base64ToBytes(obj.iv),
    ct: base64ToBytes(obj.ct),
  }
}

