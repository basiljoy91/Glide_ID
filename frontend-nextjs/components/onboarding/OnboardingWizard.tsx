'use client'

import { useState } from 'react'
import { CheckCircle2, Circle } from 'lucide-react'
import { Step1OrganizationDetails } from './Step1OrganizationDetails'
import { Step2AccountCreation } from './Step2AccountCreation'
import { Step3TeamSetup } from './Step3TeamSetup'
import { Step4Provisioning } from './Step4Provisioning'
import toast from 'react-hot-toast'

export interface OnboardingData {
  // Step 1: Organization Details
  companyName: string
  industry: string
  estimatedEmployees: number

  // Step 2: Account Creation
  adminEmail: string
  adminFirstName: string
  adminLastName: string
  adminPhone: string
  authMethod: 'sso' | 'password'
  password?: string
  ssoProvider?: string
  ssoEmail?: string

  // Step 3: Team Setup
  teamMembers: Array<{
    email: string
    role: 'org_admin' | 'hr' | 'dept_manager'
  }>

  // Step 4: Provisioning (from backend)
  tenantId?: string
  kioskCode?: string
  adminUserId?: string
}

export function OnboardingWizard() {
  const [currentStep, setCurrentStep] = useState(1)
  const [data, setData] = useState<OnboardingData>({
    companyName: '',
    industry: '',
    estimatedEmployees: 50,
    adminEmail: '',
    adminFirstName: '',
    adminLastName: '',
    adminPhone: '',
    authMethod: 'password',
    teamMembers: [],
  })

  const steps = [
    { number: 1, title: 'Organization Details' },
    { number: 2, title: 'Account Creation' },
    { number: 3, title: 'Team Setup' },
    { number: 4, title: 'Provisioning' },
  ]

  const updateData = (updates: Partial<OnboardingData>) => {
    setData((prev) => ({ ...prev, ...updates }))
  }

  const handleNext = async () => {
    // Validate current step before proceeding
    if (currentStep === 1) {
      if (!data.companyName || !data.industry) {
        toast.error('Please fill in all required fields')
        return
      }
    } else if (currentStep === 2) {
      if (!data.adminEmail || !data.adminFirstName || !data.adminLastName) {
        toast.error('Please fill in all required fields')
        return
      }
      if (data.authMethod === 'password' && !data.password) {
        toast.error('Please enter a password')
        return
      }
      if (data.authMethod === 'sso' && !data.ssoEmail) {
        toast.error('Please enter your corporate email for SSO')
        return
      }
    }

    // If moving to step 4, trigger provisioning
    if (currentStep === 3) {
      try {
        toast.loading('Provisioning your organization...', { id: 'provisioning' })
        const result = await provisionOrganization(data)
        updateData({
          tenantId: result.tenantId,
          kioskCode: result.kioskCode,
          adminUserId: result.adminUserId,
        })
        toast.success('Organization provisioned successfully!', { id: 'provisioning' })
        setCurrentStep(4)
      } catch (error: any) {
        toast.error(error.message || 'Failed to provision organization', { id: 'provisioning' })
      }
    } else {
      setCurrentStep((prev) => Math.min(prev + 1, 4))
    }
  }

  const handleBack = () => {
    setCurrentStep((prev) => Math.max(prev - 1, 1))
  }

  return (
    <div className="space-y-8">
      {/* Progress Steps */}
      <div className="flex items-center justify-between">
        {steps.map((step, index) => (
          <div key={step.number} className="flex items-center flex-1">
            <div className="flex flex-col items-center">
              <div
                className={`w-10 h-10 rounded-full flex items-center justify-center border-2 ${
                  currentStep > step.number
                    ? 'bg-primary border-primary text-primary-foreground'
                    : currentStep === step.number
                    ? 'border-primary text-primary'
                    : 'border-muted text-muted-foreground'
                }`}
              >
                {currentStep > step.number ? (
                  <CheckCircle2 className="w-6 h-6" />
                ) : (
                  <Circle className="w-6 h-6" />
                )}
              </div>
              <span
                className={`mt-2 text-sm ${
                  currentStep >= step.number ? 'text-foreground' : 'text-muted-foreground'
                }`}
              >
                {step.title}
              </span>
            </div>
            {index < steps.length - 1 && (
              <div
                className={`flex-1 h-0.5 mx-4 ${
                  currentStep > step.number ? 'bg-primary' : 'bg-muted'
                }`}
              />
            )}
          </div>
        ))}
      </div>

      {/* Step Content */}
      <div className="border rounded-lg p-8 bg-card">
        {currentStep === 1 && (
          <Step1OrganizationDetails data={data} updateData={updateData} />
        )}
        {currentStep === 2 && (
          <Step2AccountCreation data={data} updateData={updateData} />
        )}
        {currentStep === 3 && (
          <Step3TeamSetup data={data} updateData={updateData} />
        )}
        {currentStep === 4 && <Step4Provisioning data={data} />}
      </div>

      {/* Navigation Buttons */}
      {currentStep < 4 && (
        <div className="flex justify-between">
          <button
            onClick={handleBack}
            disabled={currentStep === 1}
            className="px-6 py-2 border rounded-md disabled:opacity-50 disabled:cursor-not-allowed hover:bg-muted"
          >
            Back
          </button>
          <button
            onClick={handleNext}
            className="px-6 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
          >
            {currentStep === 3 ? 'Complete Setup' : 'Next'}
          </button>
        </div>
      )}
    </div>
  )
}

// API call to provision organization
async function provisionOrganization(data: OnboardingData) {
  const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/public/onboarding/provision`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      organization: {
        name: data.companyName,
        industry: data.industry,
        estimated_employees: data.estimatedEmployees,
      },
      admin: {
        email: data.adminEmail,
        first_name: data.adminFirstName,
        last_name: data.adminLastName,
        phone: data.adminPhone,
        auth_method: data.authMethod,
        password: data.password,
        sso_email: data.ssoEmail,
        sso_provider: data.ssoProvider,
      },
      team_members: data.teamMembers,
    }),
  })

  if (!response.ok) {
    const error = await response.json()
    throw new Error(error.error || 'Failed to provision organization')
  }

  return response.json()
}

