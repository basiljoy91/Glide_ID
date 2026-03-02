'use client'

import { useState } from 'react'
import { OnboardingWizard } from '@/components/onboarding/OnboardingWizard'

export default function OnboardingPage() {
  return (
    <div className="container mx-auto px-4 py-12 max-w-4xl">
      <div className="mb-8 text-center">
        <h1 className="text-3xl font-bold mb-2">Get Started with Glide ID</h1>
        <p className="text-muted-foreground">
          Set up your organization in just a few steps
        </p>
      </div>
      <OnboardingWizard />
    </div>
  )
}

