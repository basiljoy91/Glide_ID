'use client'

import Link from 'next/link'
import { Button } from '@/components/ui/button'

const FEATURES = [
  {
    title: 'Multi-tenant SaaS + RLS',
    desc: 'Tenant-scoped data model with database Row-Level Security for hard isolation.',
  },
  {
    title: 'Offline-first kiosks',
    desc: 'Encrypted offline queue with reconciliation and secure sync when connectivity returns.',
  },
  {
    title: 'HMAC + optional mTLS',
    desc: 'HMAC request signing for kiosks; mTLS termination at gateway for managed devices.',
  },
  {
    title: 'Org metrics + reports',
    desc: 'Tenant-scoped metrics, attendance reports, and anomaly review workflow.',
  },
  {
    title: 'HRMS integration hub',
    desc: 'Provider integrations stored per tenant with webhook processing and exports.',
  },
  {
    title: 'Compliance patterns',
    desc: 'Consent capture and automated purging scheduling for terminated employees.',
  },
]

export default function FeaturesPage() {
  return (
    <div className="container mx-auto px-4 py-16 space-y-10">
      <div className="max-w-3xl">
        <h1 className="text-4xl font-bold tracking-tight">Features</h1>
        <p className="mt-4 text-muted-foreground text-lg">
          Enterprise-grade features designed for security, scale, and compliance.
        </p>
      </div>

      <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
        {FEATURES.map((f) => (
          <div key={f.title} className="border rounded-lg p-6 bg-card">
            <div className="font-semibold">{f.title}</div>
            <div className="mt-2 text-sm text-muted-foreground">{f.desc}</div>
          </div>
        ))}
      </div>

      <div className="flex gap-3">
        <Link href="/onboarding">
          <Button>Get started</Button>
        </Link>
        <Link href="/contact">
          <Button variant="outline">Talk to us</Button>
        </Link>
      </div>
    </div>
  )
}

