'use client'

import Link from 'next/link'
import { Button } from '@/components/ui/button'
import { KioskCodeLauncher } from '@/components/kiosk/KioskCodeLauncher'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useKioskStore } from '@/store/useStore'

export default function KioskStartPage() {
  const { kioskHmacSecret, setKioskHmacSecret } = useKioskStore()
  return (
    <div className="min-h-screen bg-background flex flex-col items-center justify-center p-6">
      <div className="w-full max-w-xl space-y-6">
        <div className="text-center space-y-2">
          <h1 className="text-3xl font-bold">Kiosk Check-In</h1>
          <p className="text-muted-foreground">
            Enter your kiosk code to open the camera check-in portal.
          </p>
        </div>

        <div className="bg-card border border-border rounded-lg p-4 shadow-sm space-y-2">
          <Label htmlFor="kiosk-secret">Kiosk HMAC Secret (device configuration)</Label>
          <Input
            id="kiosk-secret"
            type="password"
            value={kioskHmacSecret || ''}
            onChange={(e) => setKioskHmacSecret(e.target.value || null)}
            placeholder="Paste the kiosk secret once (stored locally on this device)"
          />
          <div className="text-xs text-muted-foreground">
            This is required for secure kiosk requests (HMAC). Keep it only on the kiosk device.
          </div>
        </div>

        <div className="bg-card border border-border rounded-lg p-4 shadow-sm">
          <KioskCodeLauncher variant="compact" />
        </div>

        <div className="text-center">
          <Link href="/landing">
            <Button variant="link">Back to landing</Button>
          </Link>
        </div>
      </div>
    </div>
  )
}

