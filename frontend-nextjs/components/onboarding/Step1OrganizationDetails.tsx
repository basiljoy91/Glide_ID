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
            This helps us provision the right resources for your organization
          </p>
        </div>
      </div>
    </div>
  )
}

