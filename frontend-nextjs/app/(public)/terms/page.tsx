import type { Metadata } from 'next'

const siteUrl = process.env.NEXT_PUBLIC_SITE_URL || 'https://glide-id.example'

export const metadata: Metadata = {
  title: 'Terms of Service | Glide ID',
  description: 'Terms of service for Glide ID usage, acceptable use, and responsibilities.',
  metadataBase: new URL(siteUrl),
  alternates: { canonical: '/terms' },
}

export default function TermsPage() {
  return (
    <div className="container mx-auto px-4 py-16 space-y-10">
      <div className="max-w-3xl space-y-4">
        <div className="text-xs uppercase tracking-[0.3em] text-muted-foreground">
          Terms of service
        </div>
        <h1 className="text-4xl font-display font-semibold tracking-tight">Usage terms</h1>
        <p className="text-muted-foreground">
          These terms define how Glide ID may be used, responsibilities for administrators, and
          acceptable use of biometric services.
        </p>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        <div className="rounded-2xl border bg-background/80 p-6">
          <div className="font-semibold">Service description</div>
          <p className="mt-2 text-sm text-muted-foreground">
            Glide ID provides tenant-scoped attendance tracking, identity verification, and kiosk
            workflows.
          </p>
        </div>
        <div className="rounded-2xl border bg-background/80 p-6">
          <div className="font-semibold">Customer responsibilities</div>
          <p className="mt-2 text-sm text-muted-foreground">
            Organizations are responsible for lawful biometric processing, onboarding users, and
            configuring retention policies.
          </p>
        </div>
        <div className="rounded-2xl border bg-background/80 p-6">
          <div className="font-semibold">Acceptable use</div>
          <p className="mt-2 text-sm text-muted-foreground">
            You may not attempt to bypass security controls or access data across tenants.
          </p>
        </div>
        <div className="rounded-2xl border bg-background/80 p-6">
          <div className="font-semibold">Disclaimer</div>
          <p className="mt-2 text-sm text-muted-foreground">
            This content is a template for product demos and is not legal advice.
          </p>
        </div>
      </div>
    </div>
  )
}
