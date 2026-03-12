'use client'

import { useMemo, useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useKioskStore } from '@/store/useStore'
import { config } from '@/lib/config'
import { hmacSha256Hex } from '@/lib/crypto'
import toast from 'react-hot-toast'

type ConnectionState =
  | { status: 'idle' }
  | { status: 'success'; message: string; organizationName?: string; kioskName?: string }
  | { status: 'error'; message: string }

function mapKioskCredentialError(raw: string): string {
  const msg = raw.toLowerCase()
  if (msg.includes('invalid kiosk code')) {
    return 'Kiosk code is invalid, revoked, or not active.'
  }
  if (msg.includes('invalid hmac signature')) {
    return 'Kiosk secret is incorrect for this code. Verify the secret and retry.'
  }
  if (msg.includes('timestamp expired')) {
    return 'Device clock appears out of sync. Correct the system time and retry.'
  }
  if (msg.includes('kiosk code required')) {
    return 'Kiosk code is required.'
  }
  if (msg.includes('hmac signature required')) {
    return 'Kiosk secret is required to sign requests.'
  }
  return raw || 'Unable to verify kiosk credentials.'
}

export default function KioskStartPage() {
  const router = useRouter()
  const {
    kioskCode,
    kioskHmacSecret,
    kioskName,
    organizationName,
    credentialsVerifiedAt,
    setKioskCode,
    setKioskHmacSecret,
    setKioskConnectionMeta,
    clearKioskConnectionMeta,
  } = useKioskStore()
  const [codeInput, setCodeInput] = useState(kioskCode || '')
  const [isVerifying, setIsVerifying] = useState(false)
  const [connection, setConnection] = useState<ConnectionState>({ status: 'idle' })

  const cleanedCode = useMemo(() => codeInput.replace(/\s+/g, '').trim(), [codeInput])
  const secretReady = Boolean((kioskHmacSecret || '').trim())
  const isStoredVerified =
    Boolean(credentialsVerifiedAt) &&
    cleanedCode.length > 0 &&
    cleanedCode === kioskCode &&
    Boolean(kioskName) &&
    Boolean(organizationName)

  const verifyConnection = async () => {
    if (!cleanedCode) {
      setConnection({ status: 'error', message: 'Enter kiosk code before verifying.' })
      return
    }
    if (!secretReady || !kioskHmacSecret) {
      setConnection({ status: 'error', message: 'Enter kiosk secret before verifying.' })
      return
    }

    try {
      setIsVerifying(true)
      setConnection({ status: 'idle' })

      const timestamp = Math.floor(Date.now() / 1000).toString()
      const signature = await hmacSha256Hex(kioskHmacSecret.trim(), `${timestamp}${cleanedCode}`)

      const response = await fetch(`${config.apiUrl}/api/v1/kiosk/heartbeat`, {
        method: 'GET',
        headers: {
          'X-Kiosk-Code': cleanedCode,
          'X-Timestamp': timestamp,
          'X-HMAC-Signature': signature,
        },
      })

      const payload = await response.json().catch(() => ({}))
      if (!response.ok) {
        const message = mapKioskCredentialError(String(payload?.error || 'Credential verification failed'))
        clearKioskConnectionMeta()
        setConnection({ status: 'error', message })
        return
      }

      const orgName = typeof payload?.organization_name === 'string' ? payload.organization_name : undefined
      const resolvedKioskName = typeof payload?.kiosk_name === 'string' ? payload.kiosk_name : undefined
      setKioskCode(cleanedCode)
      setKioskConnectionMeta(
        resolvedKioskName || null,
        orgName || null,
        new Date().toISOString()
      )

      const message = orgName
        ? `Connected successfully to ${orgName}.`
        : 'Connected successfully.'
      setConnection({
        status: 'success',
        message,
        organizationName: orgName,
        kioskName: resolvedKioskName,
      })
      toast.success('Kiosk credentials verified')
    } catch (error: any) {
      const message = mapKioskCredentialError(String(error?.message || 'Credential verification failed'))
      clearKioskConnectionMeta()
      setConnection({ status: 'error', message })
    } finally {
      setIsVerifying(false)
    }
  }

  const canOpenPortal =
    connection.status === 'success' ||
    isStoredVerified

  return (
    <div className="min-h-screen bg-background flex flex-col items-center justify-center p-6">
      <div className="w-full max-w-xl space-y-6">
        <div className="text-center space-y-2">
          <h1 className="text-3xl font-bold">Kiosk Check-In</h1>
          <p className="text-muted-foreground">
            Verify kiosk credentials, then open the check-in portal.
          </p>
        </div>

        <div className="bg-card border border-border rounded-lg p-4 shadow-sm space-y-3">
          <div>
            <Label htmlFor="kiosk-code">Kiosk Code</Label>
            <Input
              id="kiosk-code"
              value={codeInput}
              onChange={(e) => {
                setCodeInput(e.target.value)
                setConnection({ status: 'idle' })
              }}
              placeholder="Enter kiosk code (e.g. 1234567890)"
              className="mt-1"
            />
          </div>
          <Label htmlFor="kiosk-secret">Kiosk HMAC Secret (device configuration)</Label>
          <Input
            id="kiosk-secret"
            type="password"
            value={kioskHmacSecret || ''}
            onChange={(e) => {
              setKioskHmacSecret(e.target.value || null)
              setConnection({ status: 'idle' })
            }}
            placeholder="Paste the kiosk secret once (stored locally on this device)"
          />
          <div className="text-xs text-muted-foreground">
            This is required for secure kiosk requests (HMAC). Keep it only on the kiosk device.
          </div>

          <div className="flex flex-col sm:flex-row gap-2 pt-1">
            <Button onClick={() => void verifyConnection()} disabled={isVerifying || !cleanedCode || !secretReady}>
              {isVerifying ? 'Verifying...' : 'Verify Connection'}
            </Button>
            <Button
              variant="outline"
              disabled={!canOpenPortal || !cleanedCode}
              onClick={() => {
                setKioskCode(cleanedCode)
                router.push(`/kiosk/${encodeURIComponent(cleanedCode)}`)
              }}
            >
              Open Check-In Portal
            </Button>
          </div>

          {connection.status === 'success' && (
            <div className="rounded-md border border-green-300/60 bg-green-50 dark:bg-green-950/40 px-3 py-2 text-sm">
              <div className="font-medium text-green-800 dark:text-green-200">{connection.message}</div>
              <div className="text-green-700/90 dark:text-green-300/90">
                Organization: {connection.organizationName || 'Linked'}
              </div>
              <div className="text-green-700/90 dark:text-green-300/90">
                Kiosk: {connection.kioskName || cleanedCode}
              </div>
            </div>
          )}

          {connection.status === 'error' && (
            <div className="rounded-md border border-red-300/60 bg-red-50 dark:bg-red-950/40 px-3 py-2 text-sm text-red-700 dark:text-red-300">
              {connection.message}
            </div>
          )}

          {connection.status === 'idle' && isStoredVerified && kioskName && organizationName && (
            <div className="rounded-md border border-border bg-muted/40 px-3 py-2 text-sm text-muted-foreground">
              Last verified for <span className="font-medium text-foreground">{organizationName}</span> ({kioskName}).
            </div>
          )}
        </div>

        <div className="text-center">
          <Link href="/">
            <Button variant="link">Back to home</Button>
          </Link>
        </div>
      </div>
    </div>
  )
}
