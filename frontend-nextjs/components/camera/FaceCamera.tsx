'use client'

import { useRef, useEffect, useState, useCallback } from 'react'
import { useAmbientLight } from '@/hooks/useAmbientLight'
import toast from 'react-hot-toast'

interface FaceCameraProps {
  onCapture: (
    imageData: string,
    metadata?: {
      framesBase64?: string[]
      capturedAt?: string
      livenessType: 'active' | 'passive'
    }
  ) => void | Promise<void>
  onError?: (error: string) => void
  livenessType?: 'active' | 'passive'
  showFlashlight?: boolean
  instruction?: string
}

export function FaceCamera({
  onCapture,
  onError,
  livenessType = 'passive',
  showFlashlight = true,
  instruction,
}: FaceCameraProps) {
  const videoRef = useRef<HTMLVideoElement>(null)
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [stream, setStream] = useState<MediaStream | null>(null)
  const [permissionStatus, setPermissionStatus] = useState<'granted' | 'denied' | 'prompt' | 'checking'>('checking')
  const [isCapturing, setIsCapturing] = useState(false)
  const [userFeedback, setUserFeedback] = useState<string>('')
  const { isDark } = useAmbientLight()
  const [showFlashlightOverlay, setShowFlashlightOverlay] = useState(false)

  const startCamera = useCallback(async () => {
    try {
      const mediaStream = await navigator.mediaDevices.getUserMedia({
        video: {
          facingMode: 'user',
          width: { ideal: 960 },
          height: { ideal: 540 },
        },
      })

      setStream(mediaStream)
      if (videoRef.current) {
        videoRef.current.srcObject = mediaStream
        setPermissionStatus('granted')
      }
    } catch (error: any) {
      console.error('Camera access error:', error)
      setPermissionStatus('denied')
      if (onError) {
        onError(error.message || 'Failed to access camera')
      }
      toast.error('Camera access denied. Please enable camera permissions.')
    }
  }, [onError])

  const checkCameraPermission = useCallback(async () => {
    try {
      const result = await navigator.permissions.query({ name: 'camera' as PermissionName })
      setPermissionStatus(result.state as 'granted' | 'denied' | 'prompt')
      
      if (result.state === 'granted') {
        startCamera()
      }
    } catch (error) {
      // Fallback for browsers that don't support Permissions API
      startCamera()
    }
  }, [startCamera])

  // Initialize camera on mount
  useEffect(() => {
    checkCameraPermission()
  }, [checkCameraPermission])

  // Cleanup: stop camera tracks when stream changes or component unmounts
  useEffect(() => {
    return () => {
      if (stream) {
        stream.getTracks().forEach(track => track.stop())
      }
    }
  }, [stream])

  useEffect(() => {
    if (isDark && showFlashlight) {
      setShowFlashlightOverlay(true)
    } else {
      setShowFlashlightOverlay(false)
    }
  }, [isDark, showFlashlight])

  const capturePhoto = useCallback(async () => {
    if (!videoRef.current || !canvasRef.current) return

    setIsCapturing(true)
    setUserFeedback('Capturing...')

    try {
      const video = videoRef.current
      const canvas = canvasRef.current
      const context = canvas.getContext('2d')

      if (!context) {
        throw new Error('Could not get canvas context')
      }

      // Set canvas dimensions to match video
      canvas.width = video.videoWidth
      canvas.height = video.videoHeight

      // Draw video frame to canvas
      context.drawImage(video, 0, 0, canvas.width, canvas.height)

      // Convert to base64
      const imageData = canvas.toDataURL('image/jpeg', 0.9)

      const framesBase64: string[] = []
      if (livenessType === 'active') {
        for (let i = 0; i < 3; i++) {
          await new Promise((resolve) => setTimeout(resolve, 120))
          context.drawImage(video, 0, 0, canvas.width, canvas.height)
          const burst = canvas.toDataURL('image/jpeg', 0.8)
          const payload = burst.split(',')[1]
          if (payload) framesBase64.push(payload)
        }
      }
      
      // Basic face detection feedback (simplified)
      setUserFeedback('Processing...')
      
      // Call onCapture callback
      await onCapture(imageData, {
        framesBase64,
        capturedAt: new Date().toISOString(),
        livenessType,
      })
      
      setUserFeedback('Success!')
      setTimeout(() => setUserFeedback(''), 2000)
    } catch (error: any) {
      console.error('Capture error:', error)
      setUserFeedback('Error capturing image')
      if (onError) {
        onError(error.message || 'Failed to capture image')
      }
    } finally {
      setIsCapturing(false)
    }
  }, [livenessType, onCapture, onError])

  const handleRetryPermission = () => {
    setPermissionStatus('prompt')
    startCamera()
  }

  if (permissionStatus === 'denied') {
    return (
      <div className="flex flex-col items-center justify-center p-10 bg-red-50 dark:bg-red-950/20 rounded-xl border-2 border-red-500/50 shadow-inner min-h-[400px]">
        <div className="text-center max-w-md space-y-6">
          <div className="mx-auto flex h-20 w-20 items-center justify-center rounded-full bg-red-100 dark:bg-red-900/40">
            <span className="text-4xl">📷❌</span>
          </div>
          <div>
            <h3 className="text-2xl font-bold text-red-700 dark:text-red-400">Camera Access Blocked</h3>
            <p className="text-red-600/90 dark:text-red-300/90 mt-2 font-medium">
              We cannot load the kiosk because camera permissions are denied or unavailable.
            </p>
          </div>
          
          <div className="bg-background rounded-lg p-5 text-left border shadow-sm space-y-3">
            <p className="font-semibold text-foreground text-sm">How to fix this:</p>
            <ol className="list-decimal list-inside text-sm text-muted-foreground space-y-2">
              <li>Look for the camera icon <span className="inline-flex bg-muted rounded px-1 px-1 text-xs">📹</span> in your browser&apos;s address bar.</li>
              <li>Click it and change the setting to <strong>&quot;Always allow&quot;</strong>.</li>
              <li>Or, check your operating system&apos;s Privacy &amp; Security settings.</li>
            </ol>
          </div>
          
          <button
            onClick={handleRetryPermission}
            className="w-full h-12 text-base font-semibold bg-red-600 text-white hover:bg-red-700 rounded-lg shadow-sm transition-colors"
          >
            I&apos;ve enabled it, retry now
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="relative w-full max-w-2xl mx-auto">
      {/* Flashlight overlay for dark environments */}
      {showFlashlightOverlay && (
        <div className="flashlight-overlay active" />
      )}

      {/* Video element */}
      <div className="relative bg-black rounded-lg overflow-hidden aspect-video">
        <video
          ref={videoRef}
          autoPlay
          playsInline
          muted
          className="w-full h-full object-cover"
        />
        
        {/* User feedback overlay */}
        {userFeedback && (
          <div className="absolute inset-0 flex items-center justify-center bg-black/50 backdrop-blur-sm">
            <div className="bg-card p-4 rounded-lg shadow-lg">
              <p className="text-foreground font-medium">{userFeedback}</p>
            </div>
          </div>
        )}

        {/* Face detection guide overlay */}
        <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
          <div className="border-2 border-primary rounded-full w-64 h-80 opacity-50" />
        </div>
      </div>

      {/* Hidden canvas for capture */}
      <canvas ref={canvasRef} className="hidden" />

      {/* Controls */}
      <div className="mt-4 flex flex-col items-center space-y-4">
        <div className="text-sm text-muted-foreground text-center">
          {instruction || (livenessType === 'active'
            ? 'Please move your head slightly'
            : 'Position your face within the frame')}
        </div>
        
        <button
          onClick={capturePhoto}
          disabled={isCapturing || permissionStatus !== 'granted'}
          className="px-6 py-3 bg-primary text-primary-foreground rounded-full hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed flex items-center space-x-2"
        >
          {isCapturing ? (
            <>
              <div className="w-4 h-4 border-2 border-primary-foreground border-t-transparent rounded-full animate-spin" />
              <span>Capturing...</span>
            </>
          ) : (
            <>
              <span>📸</span>
              <span>Capture</span>
            </>
          )}
        </button>
      </div>
    </div>
  )
}
