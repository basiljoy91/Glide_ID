'use client'

import Link from 'next/link'
import { Button } from '@/components/ui/button'

const PLANS = [
  {
    name: 'Starter',
    price: '$49',
    period: '/mo',
    highlight: false,
    features: [
      '1 tenant workspace',
      'Up to 25 employees',
      '1 kiosk',
      'Org dashboard + basic reports',
      'HMAC-protected kiosk API',
    ],
  },
  {
    name: 'Professional',
    price: '$199',
    period: '/mo',
    highlight: true,
    features: [
      'Up to 250 employees',
      'Up to 10 kiosks',
      'Anomaly review workflow',
      'HRMS integration hub',
      'Offline encrypted queue sync',
    ],
  },
  {
    name: 'Enterprise',
    price: 'Custom',
    period: '',
    highlight: false,
    features: [
      'Unlimited employees & kiosks',
      'mTLS at gateway + device attestation options',
      'SSO (OIDC/SAML) rollout support',
      'Dedicated SLA + audit/compliance packages',
      'Custom integrations & exports',
    ],
  },
]

export default function PricingPage() {
  return (
    <div className="container mx-auto px-4 py-16 space-y-12">
      <div className="text-center max-w-3xl mx-auto">
        <h1 className="text-4xl font-bold tracking-tight">Pricing</h1>
        <p className="mt-4 text-muted-foreground text-lg">
          Simple tiers for teams, with enterprise-grade security built in.
        </p>
      </div>

      <div className="grid md:grid-cols-3 gap-6">
        {PLANS.map((p) => (
          <div
            key={p.name}
            className={
              p.highlight
                ? 'border rounded-lg p-6 bg-card shadow-sm ring-1 ring-primary'
                : 'border rounded-lg p-6 bg-card shadow-sm'
            }
          >
            <div className="flex items-baseline justify-between">
              <h2 className="text-xl font-semibold">{p.name}</h2>
              {p.highlight && (
                <span className="text-xs px-2 py-1 rounded bg-primary text-primary-foreground">
                  Most popular
                </span>
              )}
            </div>
            <div className="mt-4">
              <div className="text-4xl font-bold">
                {p.price}
                <span className="text-base font-normal text-muted-foreground">{p.period}</span>
              </div>
            </div>
            <ul className="mt-6 space-y-2 text-sm text-muted-foreground">
              {p.features.map((f) => (
                <li key={f}>- {f}</li>
              ))}
            </ul>
            <div className="mt-8">
              <Link href="/contact">
                <Button className="w-full" variant={p.highlight ? 'default' : 'outline'}>
                  {p.name === 'Enterprise' ? 'Talk to sales' : 'Contact us'}
                </Button>
              </Link>
            </div>
          </div>
        ))}
      </div>

      <div className="text-center">
        <Link href="/onboarding">
          <Button size="lg">Get started</Button>
        </Link>
      </div>
    </div>
  )
}

