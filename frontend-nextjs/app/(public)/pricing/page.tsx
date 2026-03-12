'use client'

import { useState } from 'react'
import Link from 'next/link'
import { Button } from '@/components/ui/button'
import { Check, Minus, HelpCircle } from 'lucide-react'

type Currency = 'USD' | 'EUR' | 'GBP' | 'INR'
type BillingCycle = 'monthly' | 'annual'

const CURRENCIES: Record<Currency, { symbol: string; rate: number }> = {
  USD: { symbol: '$', rate: 1 },
  EUR: { symbol: '€', rate: 0.92 },
  GBP: { symbol: '£', rate: 0.79 },
  INR: { symbol: '₹', rate: 92 },
}

const PLANS = [
  {
    name: 'Starter',
    basePrice: 49,
    highlight: false,
    cta: 'Start free trial',
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
    basePrice: 199,
    highlight: true,
    cta: 'Start free trial',
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
    basePrice: null, // Custom
    highlight: false,
    cta: 'Talk to sales',
    features: [
      'Unlimited employees & kiosks',
      'mTLS at gateway + device attestation options',
      'SSO (OIDC/SAML) rollout support',
      'Dedicated SLA + audit/compliance packages',
      'Custom integrations & exports',
    ],
  },
]

const COMPARISON_FEATURES = [
  {
    category: 'Core Specs',
    items: [
      { name: 'Employees', starter: 'Up to 25', pro: 'Up to 250', enterprise: 'Unlimited' },
      { name: 'Kiosks', starter: '1', pro: 'Up to 10', enterprise: 'Unlimited' },
      { name: 'Tenant Workspaces', starter: '1', pro: '1', enterprise: 'Multiple' },
    ],
  },
  {
    category: 'Security & Access',
    items: [
      { name: 'HMAC Kiosk API', starter: true, pro: true, enterprise: true },
      { name: 'Offline Queue Sync', starter: false, pro: true, enterprise: true },
      { name: 'SSO (SAML/OIDC)', starter: false, pro: false, enterprise: true },
      { name: 'mTLS & Device Attestation', starter: false, pro: false, enterprise: true },
      { name: 'Role-Based Access (RBAC)', starter: false, pro: true, enterprise: true },
    ],
  },
  {
    category: 'Features & Support',
    items: [
      { name: 'Basic Reports', starter: true, pro: true, enterprise: true },
      { name: 'Anomaly Review', starter: false, pro: true, enterprise: true },
      { name: 'HRMS Integrations', starter: false, pro: true, enterprise: true },
      { name: 'API Docs & Webhooks', starter: false, pro: true, enterprise: true },
      { name: 'Dedicated SLA', starter: false, pro: false, enterprise: true },
      { name: 'Support Level', starter: 'Community', pro: 'Email / Chat', enterprise: '24/7 Phone + Dedicated Rep' },
    ],
  },
]

const FAQS = [
  {
    q: 'How does the free trial work?',
    a: 'You get full access to the Professional plan for 14 days. No credit card is required to start. At the end of the trial, you can choose the plan that best fits your needs.',
  },
  {
    q: 'Can I switch plans later?',
    a: 'Absolutely. You can upgrade or downgrade your plan at any time. Prorated charges or credits will automatically be applied to your account.',
  },
  {
    q: 'What happens if I go over my employee limit?',
    a: 'We will notify you when you approach your limit. We offer a grace period, after which you will need to upgrade to the next tier to add more employees.',
  },
  {
    q: 'Do you offer discounts for non-profits or educational institutions?',
    a: 'Yes! We offer a 30% discount for eligible non-profits and schools. Please contact our sales team to apply.',
  },
]

export default function PricingPage() {
  const [billingCycle, setBillingCycle] = useState<BillingCycle>('monthly')
  const [currency, setCurrency] = useState<Currency>('USD')

  const getPrice = (basePrice: number | null) => {
    if (basePrice === null) return 'Custom'
    
    let price = basePrice * CURRENCIES[currency].rate
    if (billingCycle === 'annual') {
      price = price * 0.8 // 20% discount
    }
    return `${CURRENCIES[currency].symbol}${Math.round(price)}`
  }

  const renderTableCell = (value: string | boolean) => {
    if (typeof value === 'boolean') {
      return value ? <Check className="h-5 w-5 text-primary mx-auto" /> : <Minus className="h-5 w-5 text-muted-foreground mx-auto" />
    }
    return <span className="text-sm font-medium">{value}</span>
  }

  return (
    <div className="container mx-auto px-4 py-16 space-y-20">
      
      {/* HEADER & TOGGLES */}
      <div className="text-center max-w-3xl mx-auto space-y-8">
        <div>
          <h1 className="text-4xl md:text-5xl font-bold tracking-tight">Simple, transparent pricing</h1>
          <p className="mt-4 text-muted-foreground text-lg">
            Start for free, then choose a plan that grows with your team. Enterprise-grade security built in.
          </p>
        </div>

        <div className="flex flex-col sm:flex-row items-center justify-center gap-6">
          {/* Billing Cycle Toggle */}
          <div className="flex items-center p-1 bg-muted rounded-full border">
            <button
              onClick={() => setBillingCycle('monthly')}
              className={`px-6 py-2 rounded-full text-sm font-medium transition-all ${
                billingCycle === 'monthly' ? 'bg-background shadow-sm text-foreground' : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              Monthly
            </button>
            <button
              onClick={() => setBillingCycle('annual')}
              className={`px-6 py-2 rounded-full text-sm font-medium transition-all flex items-center gap-2 ${
                billingCycle === 'annual' ? 'bg-background shadow-sm text-foreground' : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              Annual <span className="text-xs bg-primary/10 text-primary px-2 py-0.5 rounded-full">Save 20%</span>
            </button>
          </div>

          {/* Currency Selector */}
          <select
            value={currency}
            onChange={(e) => setCurrency(e.target.value as Currency)}
            className="bg-background border rounded-md px-3 py-2 text-sm font-medium focus:outline-none focus:ring-2 focus:ring-primary"
            aria-label="Select currency"
          >
            <option value="USD">USD ($)</option>
            <option value="EUR">EUR (€)</option>
            <option value="GBP">GBP (£)</option>
            <option value="INR">INR (₹)</option>
          </select>
        </div>
      </div>

      {/* PRICING CARDS */}
      <div className="grid md:grid-cols-3 gap-8 max-w-6xl mx-auto">
        {PLANS.map((p) => (
          <div
            key={p.name}
            className={`relative flex flex-col border rounded-2xl p-8 bg-card shadow-sm transition-all hover:shadow-md ${
              p.highlight ? 'ring-2 ring-primary scale-105 md:-translate-y-2' : ''
            }`}
          >
            {p.highlight && (
              <div className="absolute top-0 left-1/2 -translate-x-1/2 -translate-y-1/2">
                <span className="text-xs font-semibold px-3 py-1 rounded-full bg-primary text-primary-foreground shadow-sm">
                  Most popular
                </span>
              </div>
            )}
            
            <div className="mb-6">
              <h2 className="text-2xl font-bold">{p.name}</h2>
              <div className="mt-4 flex items-baseline gap-2">
                <span className="text-5xl font-extrabold tracking-tight">
                  {getPrice(p.basePrice)}
                </span>
                {p.basePrice !== null && (
                  <span className="text-muted-foreground font-medium">/{billingCycle === 'annual' ? 'mo' : 'mo'}</span>
                )}
              </div>
              {billingCycle === 'annual' && p.basePrice !== null && (
                <p className="mt-2 text-sm text-muted-foreground">Billed annually</p>
              )}
            </div>

            <ul className="space-y-4 mb-8 flex-1">
              {p.features.map((f) => (
                <li key={f} className="flex items-start gap-3">
                  <Check className="h-5 w-5 text-primary shrink-0" />
                  <span className="text-sm text-muted-foreground">{f}</span>
                </li>
              ))}
            </ul>

            <Link href={p.name === 'Enterprise' ? '/contact' : '/onboarding'} className="mt-auto">
              <Button className="w-full h-12 text-base font-semibold" variant={p.highlight ? 'default' : 'outline'}>
                {p.cta}
              </Button>
            </Link>
          </div>
        ))}
      </div>

      {/* COMPARISON TABLE */}
      <div className="max-w-5xl mx-auto hidden md:block pt-16 border-t">
        <h2 className="text-3xl font-bold text-center mb-10">Compare plans in detail</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr>
                <th className="w-1/4 p-4 text-sm font-semibold text-muted-foreground uppercase tracking-wider">Features</th>
                <th className="w-1/4 p-4 text-center font-bold text-lg">Starter</th>
                <th className="w-1/4 p-4 text-center font-bold text-lg text-primary">Professional</th>
                <th className="w-1/4 p-4 text-center font-bold text-lg">Enterprise</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {COMPARISON_FEATURES.map((category) => (
                <optgroup key={category.category} label={category.category} className="contents block appearance-none">
                  <tr>
                    <td colSpan={4} className="bg-muted/50 p-4 font-semibold text-sm">{category.category}</td>
                  </tr>
                  {category.items.map((item) => (
                    <tr key={item.name} className="hover:bg-muted/30 transition-colors">
                      <td className="p-4 text-sm font-medium">{item.name}</td>
                      <td className="p-4 text-center">{renderTableCell(item.starter)}</td>
                      <td className="p-4 text-center">{renderTableCell(item.pro)}</td>
                      <td className="p-4 text-center">{renderTableCell(item.enterprise)}</td>
                    </tr>
                  ))}
                </optgroup>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* FAQ SECTION */}
      <div className="max-w-3xl mx-auto pt-16 border-t">
        <div className="text-center mb-10">
          <HelpCircle className="h-10 w-10 text-primary mx-auto mb-4" />
          <h2 className="text-3xl font-bold">Frequently asked questions</h2>
        </div>
        <div className="grid gap-6">
          {FAQS.map((faq, i) => (
            <div key={i} className="border rounded-lg p-6 bg-card">
              <h3 className="text-lg font-semibold mb-2">{faq.q}</h3>
              <p className="text-muted-foreground leading-relaxed">{faq.a}</p>
            </div>
          ))}
        </div>
      </div>

      {/* BOTTOM CTA */}
      <div className="text-center bg-primary/5 rounded-2xl p-12 max-w-4xl mx-auto border transition-all hover:bg-primary/10">
        <h2 className="text-3xl font-bold mb-4">Ready to automate your attendance?</h2>
        <p className="text-lg text-muted-foreground mb-8">Join thousands of organizations using Glide ID today.</p>
        <Link href="/onboarding">
          <Button size="lg" className="h-12 px-8 text-lg font-semibold shadow-lg hover:shadow-xl transition-all hover:-translate-y-1">
            Start your free trial
          </Button>
        </Link>
      </div>
      
    </div>
  )
}

