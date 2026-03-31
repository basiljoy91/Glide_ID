'use client'

import { OnboardingData } from './OnboardingWizard'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

interface Step1Props {
  data: OnboardingData
  updateData: (updates: Partial<OnboardingData>) => void
}

const industries = [
  'Technology',
  'Healthcare',
  'Finance',
  'Manufacturing',
  'Retail',
  'Education',
  'Government',
  'Real Estate',
  'Hospitality',
  'Other',
]

const planOptions = [
  {
    value: 'starter',
    label: 'Starter',
    description: 'Best for smaller teams getting set up quickly.',
    capacity: 'Up to 25 employees, 1 kiosk',
  },
  {
    value: 'professional',
    label: 'Professional',
    description: 'For growing organizations that need more capacity.',
    capacity: 'Up to 250 employees, up to 10 kiosks',
  },
  {
    value: 'enterprise',
    label: 'Enterprise',
    description: 'For large rollouts and complex admin/security requirements.',
    capacity: 'Unlimited employees, enterprise-scale kiosk support',
  },
] as const

export function Step1OrganizationDetails({ data, updateData }: Step1Props) {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold mb-2">Organization Details</h2>
        <p className="text-muted-foreground">
          Tell us about your company to get started
        </p>
      </div>

      <div className="space-y-4">
        <div>
          <Label htmlFor="companyName">Company Name *</Label>
          <Input
            id="companyName"
            type="text"
            placeholder="Acme Corporation"
            value={data.companyName}
            onChange={(e) => updateData({ companyName: e.target.value })}
            className="mt-1"
            required
          />
        </div>

        <div>
          <Label htmlFor="industry">Industry *</Label>
          <Select
            value={data.industry}
            onValueChange={(value) => updateData({ industry: value })}
          >
            <SelectTrigger className="mt-1">
              <SelectValue placeholder="Select your industry" />
            </SelectTrigger>
            <SelectContent>
              {industries.map((industry) => (
                <SelectItem key={industry} value={industry}>
                  {industry}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div>
          <Label htmlFor="estimatedEmployees">
            Estimated Number of Employees *
          </Label>
          <Input
            id="estimatedEmployees"
            type="number"
            min="1"
            placeholder="50"
            value={data.estimatedEmployees || ''}
            onChange={(e) =>
              updateData({ estimatedEmployees: parseInt(e.target.value) || 0 })
            }
            className="mt-1"
            required
          />
          <p className="text-sm text-muted-foreground mt-1">
            We use this to size your initial seat count and validate the selected plan.
          </p>
        </div>

        <div className="border-t pt-6">
          <Label className="text-base font-semibold mb-4 block">Initial Plan *</Label>
          <div className="grid gap-3 md:grid-cols-3">
            {planOptions.map((plan) => {
              const active = data.planTier === plan.value
              return (
                <button
                  key={plan.value}
                  type="button"
                  onClick={() => updateData({ planTier: plan.value })}
                  className={`rounded-lg border p-4 text-left transition-colors ${
                    active
                      ? 'border-primary bg-primary/5'
                      : 'border-border hover:border-primary/50 hover:bg-muted/40'
                  }`}
                >
                  <div className="font-semibold">{plan.label}</div>
                  <div className="mt-1 text-sm text-muted-foreground">{plan.description}</div>
                  <div className="mt-3 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                    {plan.capacity}
                  </div>
                </button>
              )
            })}
          </div>
          <p className="mt-2 text-sm text-muted-foreground">
            Onboarding now provisions the selected plan and creates the initial billing subscription from it.
          </p>
        </div>
      </div>
    </div>
  )
}
