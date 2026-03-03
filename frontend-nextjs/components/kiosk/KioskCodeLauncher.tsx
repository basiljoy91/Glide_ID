'use client'

import { useMemo, useState } from 'react'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

export function KioskCodeLauncher({
  variant = 'hero',
}: {
  variant?: 'hero' | 'compact'
}) {
  const router = useRouter()
  const [code, setCode] = useState('')

  const cleaned = useMemo(() => code.replace(/\s+/g, '').trim(), [code])

  const go = () => {
    if (!cleaned) return
    router.push(`/kiosk/${encodeURIComponent(cleaned)}`)
  }

  return (
    <div
      className={
        variant === 'compact'
          ? 'flex flex-col sm:flex-row gap-2 items-stretch sm:items-center'
          : 'mt-8 max-w-xl mx-auto'
      }
    >
      {variant !== 'compact' && (
        <div className="text-sm font-medium text-muted-foreground mb-2">
          Have a kiosk code? Start check-in now.
        </div>
      )}
      <div className="flex flex-col sm:flex-row gap-2">
        <Input
          value={code}
          onChange={(e) => setCode(e.target.value)}
          placeholder="Enter kiosk code (e.g. 1234567890)"
          className={variant === 'compact' ? 'sm:w-64' : ''}
        />
        <Button onClick={go} disabled={!cleaned}>
          Start Check-In
        </Button>
      </div>
    </div>
  )
}

