'use client'

import { OnboardingData } from './OnboardingWizard'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Shield, Key } from 'lucide-react'

interface Step2Props {
  data: OnboardingData
  updateData: (updates: Partial<OnboardingData>) => void
}

const ssoProviders = [
  { value: 'okta', label: 'Okta' },
  { value: 'azure', label: 'Microsoft Azure AD' },
  { value: 'google', label: 'Google Workspace' },
  { value: 'saml', label: 'Generic SAML 2.0' },
  { value: 'oidc', label: 'Generic OIDC' },
]

export function Step2AccountCreation({ data, updateData }: Step2Props) {
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
            onChange={(e) => updateData({ adminEmail: e.target.value })}
            className="mt-1"
            required
          />
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
            Authentication Method *
          </Label>
          <RadioGroup
            value={data.authMethod}
            onValueChange={(value) =>
              updateData({ authMethod: value as 'sso' | 'password' })
            }
            className="space-y-4"
          >
            <div className="flex items-start space-x-3 p-4 border rounded-lg hover:bg-muted/50 cursor-pointer">
              <RadioGroupItem value="password" id="password" className="mt-1" />
              <label
                htmlFor="password"
                className="flex-1 cursor-pointer space-y-1"
              >
                <div className="flex items-center space-x-2">
                  <Key className="h-4 w-4" />
                  <span className="font-medium">Password Authentication</span>
                </div>
                <p className="text-sm text-muted-foreground">
                  Create a secure password for your account
                </p>
              </label>
            </div>

            <div className="flex items-start space-x-3 p-4 border rounded-lg hover:bg-muted/50 cursor-pointer">
              <RadioGroupItem value="sso" id="sso" className="mt-1" />
              <label htmlFor="sso" className="flex-1 cursor-pointer space-y-1">
                <div className="flex items-center space-x-2">
                  <Shield className="h-4 w-4" />
                  <span className="font-medium">Enterprise SSO (SAML/OIDC)</span>
                </div>
                <p className="text-sm text-muted-foreground">
                  Connect your corporate identity provider for passwordless login
                </p>
              </label>
            </div>
          </RadioGroup>
        </div>

        {data.authMethod === 'password' && (
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
        )}

        {data.authMethod === 'sso' && (
          <div className="space-y-4">
            <div>
              <Label htmlFor="ssoEmail">Corporate Email for SSO *</Label>
              <Input
                id="ssoEmail"
                type="email"
                placeholder="admin@company.com"
                value={data.ssoEmail || ''}
                onChange={(e) => updateData({ ssoEmail: e.target.value })}
                className="mt-1"
                required
              />
              <p className="text-sm text-muted-foreground mt-1">
                You&apos;ll be redirected to your identity provider to complete setup
              </p>
            </div>
            <div>
              <Label htmlFor="ssoProvider">SSO Provider</Label>
              <select
                id="ssoProvider"
                value={data.ssoProvider || ''}
                onChange={(e) => updateData({ ssoProvider: e.target.value })}
                className="mt-1 flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              >
                <option value="">Select provider (optional)</option>
                {ssoProviders.map((provider) => (
                  <option key={provider.value} value={provider.value}>
                    {provider.label}
                  </option>
                ))}
              </select>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
