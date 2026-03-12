import Link from 'next/link'
import { Button } from '@/components/ui/button'
import type { Metadata } from 'next'

const siteUrl = process.env.NEXT_PUBLIC_SITE_URL || 'https://glide-id.example'

export const metadata: Metadata = {
  title: 'Compliance | Glide ID',
  description:
    'Compliance workflows for consent, retention, auditing, and tenant isolation in Glide ID.',
  metadataBase: new URL(siteUrl),
  alternates: { canonical: '/compliance' },
}

export default function CompliancePage() {
  return (
    <div className="container mx-auto px-4 py-16 space-y-10">
      <div className="max-w-3xl">
        <div className="text-xs uppercase tracking-[0.3em] text-muted-foreground">
          Compliance
        </div>
        <h1 className="mt-3 text-4xl font-display font-semibold tracking-tight">
          Control, consent, and auditability
        </h1>
        <p className="mt-4 text-muted-foreground text-lg">
          Glide ID provides building blocks for compliance workflows (consent, retention, auditing,
          and purging). Your organization configures policies to match local regulations.
        </p>
      </div>

      <div className="grid md:grid-cols-2 gap-6">
        <div className="rounded-2xl border bg-background/80 p-6">
          <div className="font-semibold">Consent</div>
          <div className="mt-2 text-sm text-muted-foreground">
            Biometric consent is captured before kiosk biometric use, and recorded in the user
            profile as a consent flag and consent date.
          </div>
        </div>
        <div className="rounded-2xl border bg-background/80 p-6">
          <div className="font-semibold">Retention & purging</div>
          <div className="mt-2 text-sm text-muted-foreground">
            Automated purging can be scheduled in the database (e.g., daily) to remove biometric
            templates for terminated employees after a retention window.
          </div>
        </div>
        <div className="rounded-2xl border bg-background/80 p-6">
          <div className="font-semibold">Audit trails</div>
          <div className="mt-2 text-sm text-muted-foreground">
            Sensitive actions can be logged for traceability (admin actions, data exports, anomaly
            resolution notes).
          </div>
        </div>
        <div className="rounded-2xl border bg-background/80 p-6">
          <div className="font-semibold">Tenant isolation</div>
          <div className="mt-2 text-sm text-muted-foreground">
            Database Row-Level Security and tenant-scoped APIs help prevent accidental data
            exposure between organizations.
          </div>
        </div>
      </div>

      <div className="flex gap-3">
        <Link href="/privacy">
          <Button variant="outline">Privacy policy</Button>
        </Link>
        <Link href="/terms">
          <Button variant="outline">Terms</Button>
        </Link>
      </div>
    </div>
  )
}
