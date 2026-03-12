import type { Metadata } from 'next'

const siteUrl = process.env.NEXT_PUBLIC_SITE_URL || 'https://glide-id.example'

export const metadata: Metadata = {
  title: 'Privacy Policy | Glide ID',
  description:
    'Privacy policy for Glide ID. Understand how biometric data is processed, protected, and retained.',
  metadataBase: new URL(siteUrl),
  alternates: { canonical: '/privacy' },
}

export default function PrivacyPage() {
  return (
    <div className="container mx-auto px-4 py-16 space-y-10">
      <div className="max-w-3xl space-y-4">
        <div className="text-xs uppercase tracking-[0.3em] text-muted-foreground">
          Privacy policy
        </div>
        <h1 className="text-4xl font-display font-semibold tracking-tight">Privacy by design</h1>
        <p className="text-muted-foreground">
          Glide ID is built to minimize biometric exposure. We encrypt vectors, isolate tenants,
          and enforce retention windows to meet GDPR/CCPA expectations.
        </p>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        <div className="rounded-2xl border bg-background/80 p-6">
          <div className="font-semibold">Biometric processing</div>
          <p className="mt-2 text-sm text-muted-foreground">
            Facial images are vectorized in the AI service and raw images are not stored.
            Encrypted vectors are used for attendance and access control only.
          </p>
        </div>
        <div className="rounded-2xl border bg-background/80 p-6">
          <div className="font-semibold">Tenant isolation</div>
          <p className="mt-2 text-sm text-muted-foreground">
            Tenant-scoped access controls and row-level security prevent cross-organization data
            exposure.
          </p>
        </div>
        <div className="rounded-2xl border bg-background/80 p-6">
          <div className="font-semibold">Retention & deletion</div>
          <p className="mt-2 text-sm text-muted-foreground">
            Terminated employee biometric vectors are purged automatically after policy-defined
            windows (for example, 30 days).
          </p>
        </div>
        <div className="rounded-2xl border bg-background/80 p-6">
          <div className="font-semibold">Audit visibility</div>
          <p className="mt-2 text-sm text-muted-foreground">
            All administrative actions are logged for compliance review and anomaly investigations.
          </p>
        </div>
      </div>
    </div>
  )
}
