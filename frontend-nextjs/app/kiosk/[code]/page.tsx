'use client'

import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import { FaceCamera } from '@/components/camera/FaceCamera'
import { useOfflineQueue } from '@/hooks/useOfflineQueue'
import { useKioskStore } from '@/store/useStore'
import toast from 'react-hot-toast'
import { config } from '@/lib/config'
import { hmacSha256Hex } from '@/lib/crypto'

export default function KioskPage() {
  const params = useParams()
  const kioskCode = params.code as string
  const { addToQueue, isOnline } = useOfflineQueue()
  const { setKioskCode, kioskHmacSecret } = useKioskStore()
  const [monotonicStart] = useState(Date.now())
  const [networkStartTime] = useState(Date.now())
  const [hasConsented, setHasConsented] = useState(false)

  useEffect(() => {
    setKioskCode(kioskCode)
  }, [kioskCode, setKioskCode])

  const handleCapture = async (imageData: string) => {
    try {
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
            toast.success(data.message || 'Check-in successful!')
            return
          }
        } catch (error) {
          console.error('Direct send failed, adding to queue:', error)
        }
      }

      // Add to offline queue
      await addToQueue('check_in', imageData, monotonicOffset + networkOffset)
      toast.success('Saved offline. Will sync when connection is restored.')
    } catch (error: any) {
      console.error('Check-in error:', error)
      toast.error(error.message || 'Failed to process check-in')
    }
  }

  return (
    <div className="min-h-screen bg-background flex flex-col items-center justify-center p-4">
      <div className="w-full max-w-4xl">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold mb-2">Check-In Portal</h1>
          <p className="text-muted-foreground">
            Position your face within the frame
          </p>
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
            livenessType="passive"
            showFlashlight={true}
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

