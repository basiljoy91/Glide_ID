'use client'

import { OnboardingData } from './OnboardingWizard'
import { Button } from '@/components/ui/button'
import { CheckCircle2, Copy, Shield } from 'lucide-react'
import Link from 'next/link'
import { useState } from 'react'
import toast from 'react-hot-toast'

interface Step4Props {
  data: OnboardingData
}

export function Step4Provisioning({ data }: Step4Props) {
  const [copied, setCopied] = useState(false)

  const copyKioskCode = () => {
    if (data.kioskCode) {
      navigator.clipboard.writeText(data.kioskCode)
      setCopied(true)
      toast.success('Kiosk code copied to clipboard!')
      setTimeout(() => setCopied(false), 2000)
    }
  }

  return (
    <div className="space-y-6 text-center">
      <div className="flex justify-center">
        <div className="rounded-full bg-primary/10 p-4">
          <CheckCircle2 className="h-12 w-12 text-primary" />
        </div>
      </div>

      <div>
        <h2 className="text-2xl font-semibold mb-2">Organization Provisioned Successfully!</h2>
        <p className="text-muted-foreground">
          Your workspace is ready. Save your kiosk code securely.
        </p>
      </div>

      {/* Kiosk Code Display */}
      <div className="border-2 border-primary rounded-lg p-8 bg-primary/5">
        <div className="flex items-center justify-center space-x-2 mb-4">
          <Shield className="h-5 w-5 text-primary" />
          <Label className="text-sm font-semibold text-muted-foreground">
            Your 10-Digit Kiosk Code
          </Label>
        </div>
        <div className="flex items-center justify-center space-x-4">
          <code className="text-4xl font-mono font-bold tracking-wider">
            {data.kioskCode || '0000000000'}
          </code>
          <button
            onClick={copyKioskCode}
            className="p-2 border rounded-md hover:bg-muted"
            title="Copy to clipboard"
          >
            <Copy className="h-5 w-5" />
          </button>
        </div>
        <p className="text-sm text-muted-foreground mt-4 max-w-md mx-auto">
          <strong>Important:</strong> Save this code securely. You&apos;ll need it to configure
          physical check-in kiosks. This code is permanently assigned to your organization.
        </p>
      </div>

      {/* Next Steps */}
      <div className="border rounded-lg p-6 text-left bg-muted/50">
        <h3 className="font-semibold mb-4">Next Steps</h3>
        <ol className="space-y-2 list-decimal list-inside text-sm text-muted-foreground">
          <li>Save your kiosk code in a secure location</li>
          <li>Check your email for account verification</li>
          <li>Log in to your admin dashboard</li>
          <li>Add departments and employees</li>
          <li>Configure your first kiosk device</li>
        </ol>
      </div>

      {/* Action Buttons */}
      <div className="flex flex-col sm:flex-row gap-4 justify-center">
        <Link href="/admin/login">
          <Button size="lg">Go to Admin Dashboard</Button>
        </Link>
        <Link href="/dashboard">
          <Button size="lg" variant="outline">
            View Getting Started Guide
          </Button>
        </Link>
      </div>
    </div>
  )
}

// Re-export Label for use here
function Label({ children, className }: { children: React.ReactNode; className?: string }) {
  return <label className={className}>{children}</label>
}
