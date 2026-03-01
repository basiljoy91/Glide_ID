'use client'

import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import { FaceCamera } from '@/components/camera/FaceCamera'
import { useOfflineQueue } from '@/hooks/useOfflineQueue'
import { useKioskStore } from '@/store/useStore'
import toast from 'react-hot-toast'
import { config } from '@/lib/config'

export default function KioskPage() {
  const params = useParams()
  const kioskCode = params.code as string
  const { addToQueue, isOnline } = useOfflineQueue()
  const { setKioskCode } = useKioskStore()
  const [monotonicStart] = useState(Date.now())
  const [networkStartTime] = useState(Date.now())

  useEffect(() => {
    setKioskCode(kioskCode)
  }, [kioskCode, setKioskCode])

  const handleCapture = async (imageData: string) => {
    try {
      // Calculate monotonic offset (time since page load)
      const monotonicOffset = Date.now() - monotonicStart
      const networkOffset = navigator.onLine ? 0 : Date.now() - networkStartTime

      if (isOnline && navigator.onLine) {
        // Try to send directly to backend
        try {
          const response = await fetch(`${config.apiUrl}/api/v1/kiosk/check-in`, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'X-Kiosk-Code': kioskCode,
              'X-Timestamp': Date.now().toString(),
              // HMAC signature would be calculated here
              'X-HMAC-Signature': '', // Placeholder
            },
            body: JSON.stringify({
              image_base64: imageData.split(',')[1],
              kiosk_code: kioskCode,
              local_time: new Date().toISOString(),
              monotonic_offset_ms: monotonicOffset,
              verification_method: 'biometric',
            }),
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

        <FaceCamera
          onCapture={handleCapture}
          livenessType="passive"
          showFlashlight={true}
        />

        <div className="mt-8 text-center">
          <button
            onClick={() => {
              // PIN fallback option
              toast.info('PIN fallback not implemented in this demo')
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

