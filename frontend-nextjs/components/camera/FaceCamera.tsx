'use client'

import { useRef, useEffect, useState, useCallback } from 'react'
import { useAmbientLight } from '@/hooks/useAmbientLight'
import toast from 'react-hot-toast'

interface FaceCameraProps {
  onCapture: (imageData: string) => void
  onError?: (error: string) => void
  livenessType?: 'active' | 'passive'
  showFlashlight?: boolean
}

export function FaceCamera({
  onCapture,
  onError,
  livenessType = 'passive',
  showFlashlight = true,
}: FaceCameraProps) {
  const videoRef = useRef<HTMLVideoElement>(null)
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [stream, setStream] = useState<MediaStream | null>(null)
  const [permissionStatus, setPermissionStatus] = useState<'granted' | 'denied' | 'prompt' | 'checking'>('checking')
  const [isCapturing, setIsCapturing] = useState(false)
  const [userFeedback, setUserFeedback] = useState<string>('')
  const { isDark } = useAmbientLight()
  const [showFlashlightOverlay, setShowFlashlightOverlay] = useState(false)

  useEffect(() => {
    checkCameraPermission()
    return () => {
      if (stream) {
        stream.getTracks().forEach(track => track.stop())
      }
    }
  }, [])

  useEffect(() => {
    if (isDark && showFlashlight) {
      setShowFlashlightOverlay(true)
    } else {
      setShowFlashlightOverlay(false)
    }
  }, [isDark, showFlashlight])

  const checkCameraPermission = async () => {
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
  }

  const startCamera = async () => {
    try {
      const mediaStream = await navigator.mediaDevices.getUserMedia({
        video: {
          facingMode: 'user',
          width: { ideal: 1280 },
          height: { ideal: 720 },
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
  }

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
      
      // Basic face detection feedback (simplified)
      setUserFeedback('Processing...')
      
      // Call onCapture callback
      onCapture(imageData)
      
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
  }, [onCapture, onError])

  const handleRetryPermission = () => {
    setPermissionStatus('prompt')
    startCamera()
  }

  if (permissionStatus === 'denied') {
    return (
      <div className="flex flex-col items-center justify-center p-8 bg-card rounded-lg border">
        <div className="text-center space-y-4">
          <div className="text-2xl">📷</div>
          <h3 className="text-lg font-semibold">Camera Permission Required</h3>
          <p className="text-muted-foreground">
            Please enable camera permissions in your browser settings to continue.
          </p>
          <div className="text-sm text-muted-foreground space-y-2 mt-4">
            <p><strong>Chrome/Edge:</strong> Click the lock icon in the address bar → Camera → Allow</p>
            <p><strong>Firefox:</strong> Click the shield icon → Permissions → Camera → Allow</p>
            <p><strong>Safari:</strong> Safari → Preferences → Websites → Camera → Allow</p>
          </div>
          <button
            onClick={handleRetryPermission}
            className="mt-4 px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
          >
            Retry
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
          {livenessType === 'active' 
            ? 'Please move your head slightly' 
            : 'Position your face within the frame'}
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

