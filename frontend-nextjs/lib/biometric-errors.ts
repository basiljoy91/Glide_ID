export function extractApiErrorMessage(payload: unknown, fallback = 'Request failed'): string {
  if (!payload) return fallback

  if (typeof payload === 'string') {
    return payload
  }

  if (typeof payload === 'object') {
    const record = payload as Record<string, unknown>
    const direct = record.error ?? record.detail ?? record.message
    if (typeof direct === 'string' && direct.trim()) {
      return direct
    }
  }

  return fallback
}

export function mapBiometricErrorMessage(rawMessage: string, fallback = 'Biometric verification failed. Please try again.'): string {
  if (!rawMessage) return fallback

  const normalized = rawMessage.toLowerCase()

  if (
    normalized.includes('invalid base64') ||
    normalized.includes('unsupported or invalid image format') ||
    normalized.includes('empty image payload')
  ) {
    return 'The captured image is invalid. Please retake your photo.'
  }

  if (normalized.includes('no face detected') || normalized.includes('face could not be detected')) {
    return 'No face was detected. Center your face in the frame and try again.'
  }

  if (normalized.includes('exceeds max size') || normalized.includes('image too large')) {
    return 'The captured image is too large. Please try again with a clearer frame.'
  }

  if (normalized.includes('camera') && normalized.includes('permission')) {
    return 'Camera permission is required. Please allow camera access and retry.'
  }

  if (normalized.includes('kiosk secret not configured')) {
    return 'This kiosk is missing its device secret. Please contact your administrator.'
  }

  if (
    normalized.includes('offline encryption public key not configured') ||
    normalized.includes('offline mode is not configured on this kiosk device')
  ) {
    return 'Offline mode is not configured on this kiosk device. Add the encryption public key in environment settings.'
  }

  if (normalized.includes('invalid or expired token')) {
    return 'This enrollment link is invalid or expired. Please request a new one.'
  }

  return rawMessage
}

export function parseAndMapBiometricError(payload: unknown, fallback: string): string {
  const raw = extractApiErrorMessage(payload, fallback)
  const trimmed = raw.trim()

  const jsonStart = trimmed.indexOf('{')
  const jsonEnd = trimmed.lastIndexOf('}')
  if (jsonStart >= 0 && jsonEnd > jsonStart) {
    const candidate = trimmed.slice(jsonStart, jsonEnd + 1)
    try {
      const parsed = JSON.parse(candidate) as Record<string, unknown>
      const nested = extractApiErrorMessage(parsed, raw)
      return mapBiometricErrorMessage(nested, fallback)
    } catch {
      return mapBiometricErrorMessage(raw, fallback)
    }
  }

  return mapBiometricErrorMessage(raw, fallback)
}
