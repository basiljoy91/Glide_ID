'use client'

import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import { FaceCamera } from '@/components/camera/FaceCamera'
import { useOfflineQueue } from '@/hooks/useOfflineQueue'
import { useKioskStore } from '@/store/useStore'
import toast from 'react-hot-toast'
import { config } from '@/lib/config'
import { hmacSha256Hex } from '@/lib/crypto'
import { parseAndMapBiometricError } from '@/lib/biometric-errors'

export default function KioskPage() {
  const params = useParams()
  const kioskCode = params.code as string
  const { addToQueue, isOnline } = useOfflineQueue()
  const { setKioskCode, kioskHmacSecret, kioskName, organizationName } = useKioskStore()
  const [monotonicStart] = useState(Date.now())
  const [networkStartTime] = useState(Date.now())
  const [hasConsented, setHasConsented] = useState(false)
  const [challengeType, setChallengeType] = useState<'turn_left' | 'turn_right' | 'blink' | 'move_closer'>('turn_left')
  const [successFlash, setSuccessFlash] = useState<{
    name?: string
    employeeId?: string
    status?: string
    message?: string
  } | null>(null)

  const challengeInstruction: Record<typeof challengeType, string> = {
    turn_left: 'Turn your head slightly to the left, then capture',
    turn_right: 'Turn your head slightly to the right, then capture',
    blink: 'Blink once naturally, then capture',
    move_closer: 'Move slightly closer to the camera, then capture',
  }

  const pickNextChallenge = () => {
    const challenges: Array<typeof challengeType> = ['turn_left', 'turn_right', 'blink', 'move_closer']
    const next = challenges[Math.floor(Math.random() * challenges.length)]
    setChallengeType(next)
  }

  useEffect(() => {
    setKioskCode(kioskCode)
  }, [kioskCode, setKioskCode])

  useEffect(() => {
    pickNextChallenge()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const handleCapture = async (
    imageData: string,
    metadata?: { framesBase64?: string[]; livenessType: 'active' | 'passive' }
  ) => {
    try {
      let directFailureMessage: string | null = null
      if (!hasConsented) {
        toast.error('Please accept the data privacy notice before continuing.')
        return
      }
      // Calculate monotonic offset (time since page load)
      const monotonicOffset = Date.now() - monotonicStart
      const networkOffset = navigator.onLine ? 0 : Date.now() - networkStartTime

      if (isOnline && navigator.onLine) {
        // Try to send directly to backend
        try {
          if (!kioskHmacSecret) {
            throw new Error('Kiosk secret not configured on this device')
          }
          const timestamp = Math.floor(Date.now() / 1000).toString()
          const body = JSON.stringify({
            image_base64: imageData.split(',')[1],
            kiosk_code: kioskCode,
            local_time: new Date().toISOString(),
            monotonic_offset_ms: monotonicOffset,
            verification_method: 'biometric',
            liveness_type: metadata?.livenessType || 'active',
            challenge_type: challengeType,
            frames_base64: metadata?.framesBase64 || [],
            has_consented: true,
          })
          const signature = await hmacSha256Hex(kioskHmacSecret, `${body}${timestamp}${kioskCode}`)
          const response = await fetch(`${config.apiUrl}/api/v1/kiosk/check-in`, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'X-Kiosk-Code': kioskCode,
              'X-Timestamp': timestamp,
              'X-HMAC-Signature': signature,
            },
            body,
          })

          if (response.ok) {
            const data = await response.json()
            if (data?.success === true) {
              setSuccessFlash({
                name: data.user_name,
                employeeId: data.employee_id,
                status: data.status,
                message: data.message || 'Check-in successful',
              })
              setTimeout(() => setSuccessFlash(null), 2200)
              pickNextChallenge()
              return
            }

            const logicalFailure = parseAndMapBiometricError(data, 'Live check-in failed')
            toast.error(logicalFailure)
            pickNextChallenge()
            return
          }
          const err = await response.json().catch(() => ({}))
          directFailureMessage = parseAndMapBiometricError(err, 'Live check-in failed')
          // Do not enqueue biometric/auth rejections offline; queue only retryable transport/server failures.
          if (response.status >= 400 && response.status < 500 && response.status !== 408 && response.status !== 429) {
            toast.error(directFailureMessage)
            pickNextChallenge()
            return
          }
          throw new Error(directFailureMessage)
        } catch (error) {
          console.error('Direct send failed, adding to queue:', error)
          directFailureMessage = parseAndMapBiometricError(
            error instanceof Error ? error.message : error,
            'Live check-in failed'
          )
        }
      }

      // Add to offline queue
      const canUseOfflineEncryption = Boolean(config.publicKey && globalThis.crypto?.subtle)
      if (!canUseOfflineEncryption) {
        throw new Error('Offline mode is not configured on this kiosk device')
      }

      await addToQueue('check_in', imageData, monotonicOffset + networkOffset)
      if (directFailureMessage) {
        toast.success(`${directFailureMessage} Saved offline and will sync when connection is restored.`)
      } else {
        toast.success('Saved offline. Will sync when connection is restored.')
      }
    } catch (error: any) {
      console.error('Check-in error:', error)
      toast.error(parseAndMapBiometricError(error?.message, 'Failed to process check-in'))
      pickNextChallenge()
    }
  }

  return (
    <div className="min-h-screen bg-background flex flex-col items-center justify-center p-4">
      {successFlash && (
        <div className="fixed inset-0 z-40 bg-green-500/35 backdrop-blur-[1px] flex items-center justify-center px-4">
          <div className="w-full max-w-lg rounded-xl border border-green-300 bg-green-50 dark:bg-green-950/80 p-6 text-center shadow-2xl">
            <div className="text-3xl font-bold text-green-800 dark:text-green-200">Access Granted</div>
            <div className="mt-2 text-green-700 dark:text-green-300">
              {successFlash.message || (successFlash.status === 'check_out' ? 'Checked out successfully' : 'Checked in successfully')}
            </div>
            <div className="mt-4 text-xl font-semibold text-foreground">{successFlash.name || 'Employee'}</div>
            <div className="text-sm text-muted-foreground">
              Employee ID: <span className="font-medium text-foreground">{successFlash.employeeId || 'N/A'}</span>
            </div>
          </div>
        </div>
      )}
      <div className="w-full max-w-4xl">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold mb-2">Check-In Portal</h1>
          <p className="text-muted-foreground">
            Position your face within the frame
          </p>
          <div className="mt-2 text-xs text-primary font-medium">
            Liveness challenge: {challengeInstruction[challengeType]}
          </div>
          {(organizationName || kioskName) && (
            <div className="mt-3 inline-flex flex-col items-start rounded-md border border-border bg-card px-3 py-2 text-left text-xs">
              <span className="text-muted-foreground">
                Connected to: <span className="text-foreground font-medium">{organizationName || 'Linked organization'}</span>
              </span>
              <span className="text-muted-foreground">
                Kiosk: <span className="text-foreground font-medium">{kioskName || kioskCode}</span>
              </span>
            </div>
          )}
          {!isOnline && (
            <div className="mt-4 px-4 py-2 bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200 rounded-md inline-block">
              ⚠️ Offline Mode - Your check-in will be synced when connection is restored
            </div>
          )}
        </div>

        {!hasConsented ? (
          <div className="border border-border rounded-lg bg-card p-6 space-y-4 text-left">
            <h2 className="text-xl font-semibold">Data Privacy & Biometric Consent</h2>
            <p className="text-sm text-muted-foreground">
              This kiosk uses facial recognition to verify your identity for attendance and access
              control. Your facial data is processed securely and may be stored as encrypted
              templates for future verification, in accordance with your organization&apos;s
              privacy policy and applicable laws.
            </p>
            <ul className="list-disc list-inside text-sm text-muted-foreground space-y-1">
              <li>Your face image is captured and converted into an encrypted biometric template.</li>
              <li>
                Templates are used only for attendance, access control, anomaly detection, and
                security audits.
              </li>
              <li>
                You can request access or deletion of your biometric data through your HR or
                administrator.
              </li>
            </ul>
            <div className="flex items-center justify-between gap-4 mt-4">
              <button
                onClick={() => {
                  toast.error('You must provide consent to use the biometric kiosk.')
                }}
                className="text-sm text-muted-foreground underline"
              >
                I do not consent
              </button>
              <button
                onClick={() => setHasConsented(true)}
                className="px-6 py-2 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm font-medium"
              >
                I understand and consent
              </button>
            </div>
          </div>
        ) : (
          <FaceCamera
            onCapture={handleCapture}
            livenessType="active"
            instruction={challengeInstruction[challengeType]}
            showFlashlight={false}
          />
        )}

        <div className="mt-8 text-center">
          <button
            onClick={() => {
              // PIN fallback option
              toast('PIN fallback not implemented in this demo')
            }}
            className="text-sm text-muted-foreground hover:text-foreground underline"
          >
            Use PIN instead
          </button>
        </div>
      </div>
    </div>
  )
}
