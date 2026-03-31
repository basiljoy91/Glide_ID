'use client'

import { useEffect, useState } from 'react'
import { OnboardingData } from './OnboardingWizard'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Shield, Key } from 'lucide-react'
import { Button } from '@/components/ui/button'
import toast from 'react-hot-toast'

interface Step2Props {
  data: OnboardingData
  updateData: (updates: Partial<OnboardingData>) => void
}

export function Step2AccountCreation({ data, updateData }: Step2Props) {
  const [code, setCode] = useState('')
  const [isSendingCode, setIsSendingCode] = useState(false)
  const [isVerifyingCode, setIsVerifyingCode] = useState(false)

  useEffect(() => {
    setCode('')
  }, [data.adminEmail])

  useEffect(() => {
    if (data.authMethod !== 'password' || data.ssoEmail || data.ssoProvider) {
      updateData({
        authMethod: 'password',
        ssoEmail: undefined,
        ssoProvider: undefined,
      })
    }
  }, [data.authMethod, data.ssoEmail, data.ssoProvider, updateData])

  const handleEmailChange = (email: string) => {
    updateData({
      adminEmail: email,
      emailVerificationChallengeId: undefined,
      emailVerified: false,
      emailVerifiedAt: undefined,
    })
  }

  const sendVerificationCode = async () => {
    if (!data.adminEmail.trim()) {
      toast.error('Enter the admin email first')
      return
    }
    try {
      setIsSendingCode(true)
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/public/onboarding/email-verification/start`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ email: data.adminEmail.trim().toLowerCase() }),
        }
      )
      if (!response.ok) {
        const error = await response.json().catch(() => ({}))
        throw new Error(error.error || 'Failed to send verification code')
      }
      const payload = await response.json()
      updateData({
        emailVerificationChallengeId: payload.challenge_id,
        emailVerified: false,
        emailVerifiedAt: undefined,
      })
      toast.success('Verification code sent')
    } catch (error: any) {
      toast.error(error.message || 'Failed to send verification code')
    } finally {
      setIsSendingCode(false)
    }
  }

  const verifyCode = async () => {
    if (!data.emailVerificationChallengeId) {
      toast.error('Send a verification code first')
      return
    }
    if (!code.trim()) {
      toast.error('Enter the verification code')
      return
    }
    try {
      setIsVerifyingCode(true)
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/public/onboarding/email-verification/verify`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            challenge_id: data.emailVerificationChallengeId,
            email: data.adminEmail.trim().toLowerCase(),
            code: code.trim(),
          }),
        }
      )
      if (!response.ok) {
        const error = await response.json().catch(() => ({}))
        throw new Error(error.error || 'Failed to verify code')
      }
      updateData({
        emailVerified: true,
        emailVerifiedAt: new Date().toISOString(),
      })
      toast.success('Email verified')
    } catch (error: any) {
      toast.error(error.message || 'Failed to verify code')
    } finally {
      setIsVerifyingCode(false)
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold mb-2">Account Creation</h2>
        <p className="text-muted-foreground">
          Set up your primary Organization Admin account
        </p>
      </div>

      <div className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <div>
            <Label htmlFor="adminFirstName">First Name *</Label>
            <Input
              id="adminFirstName"
              type="text"
              placeholder="John"
              value={data.adminFirstName}
              onChange={(e) => updateData({ adminFirstName: e.target.value })}
              className="mt-1"
              required
            />
          </div>
          <div>
            <Label htmlFor="adminLastName">Last Name *</Label>
            <Input
              id="adminLastName"
              type="text"
              placeholder="Doe"
              value={data.adminLastName}
              onChange={(e) => updateData({ adminLastName: e.target.value })}
              className="mt-1"
              required
            />
          </div>
        </div>

        <div>
          <Label htmlFor="adminEmail">Email Address *</Label>
          <Input
            id="adminEmail"
            type="email"
            placeholder="admin@company.com"
            value={data.adminEmail}
            onChange={(e) => handleEmailChange(e.target.value)}
            className="mt-1"
            required
          />
        </div>

        <div className="rounded-lg border border-border bg-muted/30 p-4 space-y-3">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <div className="font-medium">Email verification</div>
              <p className="text-sm text-muted-foreground">
                We&apos;ll send a one-time code to the admin email before provisioning your workspace.
              </p>
            </div>
            <Button type="button" variant="outline" onClick={() => void sendVerificationCode()} disabled={isSendingCode}>
              {isSendingCode ? 'Sending…' : data.emailVerificationChallengeId ? 'Resend code' : 'Send code'}
            </Button>
          </div>
          <div className="flex flex-col gap-2 sm:flex-row">
            <Input
              value={code}
              onChange={(e) => setCode(e.target.value)}
              placeholder="Enter 6-digit code"
              maxLength={6}
            />
            <Button type="button" onClick={() => void verifyCode()} disabled={isVerifyingCode || !data.emailVerificationChallengeId}>
              {isVerifyingCode ? 'Verifying…' : 'Verify email'}
            </Button>
          </div>
          <div className={`text-sm ${data.emailVerified ? 'text-green-700' : 'text-muted-foreground'}`}>
            {data.emailVerified ? 'Admin email verified. You can continue onboarding.' : 'Verification is required before you continue.'}
          </div>
        </div>

        <div>
          <Label htmlFor="adminPhone">Phone Number</Label>
          <Input
            id="adminPhone"
            type="tel"
            placeholder="+1 (555) 123-4567"
            value={data.adminPhone}
            onChange={(e) => updateData({ adminPhone: e.target.value })}
            className="mt-1"
          />
        </div>

        <div className="border-t pt-6">
          <Label className="text-base font-semibold mb-4 block">
            Authentication Method
          </Label>
          <div>
            <div className="rounded-lg border border-border bg-muted/30 p-4">
              <div className="flex items-center space-x-2">
                <Key className="h-4 w-4" />
                <span className="font-medium">Password Authentication</span>
              </div>
              <p className="mt-2 text-sm text-muted-foreground">
                Onboarding currently supports password-based admin accounts only.
                You can switch to SSO later when the full SSO sign-in flow is available.
              </p>
            </div>
          </div>
          <div className="rounded-lg border border-amber-300 bg-amber-50 p-4 text-sm text-amber-900">
            <div className="flex items-center gap-2 font-medium">
              <Shield className="h-4 w-4" />
              SSO onboarding is not available yet
            </div>
            <p className="mt-2">
              We&apos;re keeping onboarding on password authentication for now so every new admin account can sign in immediately after setup.
            </p>
          </div>
          <div>
            <Label htmlFor="password">Password *</Label>
            <Input
              id="password"
              type="password"
              placeholder="Enter a strong password"
              value={data.password || ''}
              onChange={(e) => updateData({ password: e.target.value })}
              className="mt-1"
              required
            />
            <p className="text-sm text-muted-foreground mt-1">
              Must be at least 8 characters with uppercase, lowercase, and numbers
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
