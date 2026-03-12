'use client'

import Link from 'next/link'
import { Button } from '@/components/ui/button'

const INTEGRATIONS = [
  { name: 'Workday', status: 'Supported (webhooks + sync scaffolding)' },
  { name: 'SAP', status: 'Supported (webhooks + sync scaffolding)' },
  { name: 'BambooHR', status: 'Supported (webhooks + config)' },
  { name: 'Custom', status: 'Supported (generic webhook processing)' },
]

export default function IntegrationsPage() {
  return (
    <div className="container mx-auto px-4 py-16 space-y-10">
      <div className="max-w-3xl">
        <h1 className="text-4xl font-bold tracking-tight">Integrations</h1>
        <p className="mt-4 text-muted-foreground text-lg">
          Connect HRMS and payroll providers per tenant. Configure integrations securely inside your
          Org Admin portal.
        </p>
      </div>

      <div className="border rounded-lg bg-card">
        <div className="border-b px-4 py-2 text-sm font-semibold text-muted-foreground">
          Available providers
        </div>
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b text-left">
              <th className="px-4 py-2">Provider</th>
              <th className="px-4 py-2">Status</th>
            </tr>
          </thead>
          <tbody>
            {INTEGRATIONS.map((i) => (
              <tr key={i.name} className="border-b last:border-b-0">
                <td className="px-4 py-2 font-medium">{i.name}</td>
                <td className="px-4 py-2 text-muted-foreground">{i.status}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="flex gap-3">
        <Link href="/admin/login">
          <Button variant="outline">Admin login</Button>
        </Link>
        <Link href="/onboarding">
          <Button>Provision a tenant</Button>
        </Link>
      </div>
    </div>
  )
}

