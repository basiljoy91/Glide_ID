'use client'

import { useEffect, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { FaceCamera } from '@/components/camera/FaceCamera'
import toast from 'react-hot-toast'
import { config } from '@/lib/config'
import { parseAndMapBiometricError } from '@/lib/biometric-errors'

interface EnrollUserInfo {
  id: string
  employee_id: string
  first_name: string
  last_name: string
  role: string
}

export default function EnrollPage() {
  const params = useParams()
  const router = useRouter()
  const token = params.token as string
  const [hasConsented, setHasConsented] = useState(false)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [info, setInfo] = useState<EnrollUserInfo | null>(null)
  const [infoError, setInfoError] = useState<string | null>(null)
  const [isLoadingInfo, setIsLoadingInfo] = useState(true)

  useEffect(() => {
    const loadInfo = async () => {
      try {
        setIsLoadingInfo(true)
        const resp = await fetch(
          `${config.apiUrl}/api/v1/public/enroll/info/${encodeURIComponent(token)}`
        )
        if (!resp.ok) {
          const err = await resp.json().catch(() => ({}))
          throw new Error(err.error || 'Invalid or expired enrollment link')
        }
        const data = await resp.json()
        setInfo(data)
      } catch (e: any) {
        setInfoError(e.message || 'Link is invalid or expired.')
      } finally {
        setIsLoadingInfo(false)
      }
    }
    void loadInfo()
  }, [token])

  const handleCapture = async (imageData: string) => {
    if (!hasConsented) {
      toast.error('Please accept the data privacy notice before continuing.')
      return
    }
    try {
      setIsSubmitting(true)
      const base64 = imageData.split(',')[1]
      const resp = await fetch(
        `${config.apiUrl}/api/v1/public/enroll/face/${encodeURIComponent(token)}`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ image_base64: base64 }),
        }
      )
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        const friendlyError = parseAndMapBiometricError(err, 'Enrollment failed')
        throw new Error(friendlyError)
      }
      toast.success('Face enrolled successfully')
      router.push('/landing')
    } catch (e: any) {
      const message = parseAndMapBiometricError(e?.message, 'Failed to enroll face')
      toast.error(message)
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="min-h-screen bg-background flex flex-col items-center justify-center p-4">
      <div className="w-full max-w-3xl space-y-6">
        <div className="text-center space-y-2">
          <h1 className="text-3xl font-bold">Employee Face Enrollment</h1>
          {isLoadingInfo ? (
            <p className="text-muted-foreground text-sm">Loading enrollment details…</p>
          ) : infoError ? (
            <p className="text-destructive text-sm">{infoError}</p>
          ) : info ? (
            <p className="text-muted-foreground">
              Enrolling face for{' '}
              <span className="font-semibold">
                {info.first_name} {info.last_name}
              </span>{' '}
              (Employee ID: {info.employee_id})
            </p>
          ) : null}
        </div>

        {!info || infoError ? (
          <div className="border border-border rounded-lg bg-card p-6 text-sm text-muted-foreground text-center">
            This enrollment link is not valid anymore. Please request a new enrollment email from
            your HR or administrator.
          </div>
        ) : !hasConsented ? (
          <div className="border border-border rounded-lg bg-card p-6 space-y-4 text-left">
            <h2 className="text-xl font-semibold">Data Privacy & Biometric Consent</h2>
            <p className="text-sm text-muted-foreground">
              Your organization is using Glide ID to verify attendance via facial recognition. By
              continuing, you agree that a biometric template of your face may be stored and used
              solely for attendance and access control, according to your organization&apos;s
              policies.
            </p>
            <div className="flex flex-col gap-2 text-sm text-muted-foreground">
              <span className="font-medium text-foreground">For best results:</span>
              <span>- Stand in good, even lighting.</span>
              <span>- Remove hats or anything covering your face.</span>
              <span>- Be ready to gently tilt your head and blink.</span>
            </div>
            <div className="flex items-center justify-between gap-4 mt-4">
              <button
                onClick={() => {
                  toast.error('You must provide consent to enroll your face.')
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
          <>
            <div className="border border-border rounded-lg bg-card p-4 space-y-3">
              <div className="text-sm font-medium text-foreground text-center">
                Follow these steps before capturing:
              </div>
              <ol className="list-decimal list-inside text-sm text-muted-foreground space-y-1">
                <li>Center your face in the oval guide.</li>
                <li>Gently tilt your head left and right.</li>
                <li>Blink once or twice to confirm liveness.</li>
              </ol>
            </div>
            <FaceCamera
              onCapture={handleCapture}
              livenessType="active"
              showFlashlight={false}
            />
            {isSubmitting && (
              <p className="text-xs text-muted-foreground text-center">
                Uploading and processing your face template…
              </p>
            )}
          </>
        )}
      </div>
    </div>
  )
}

