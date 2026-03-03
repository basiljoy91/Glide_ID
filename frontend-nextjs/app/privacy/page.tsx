'use client'

export default function PrivacyPage() {
  return (
    <div className="container mx-auto px-4 py-16">
      <div className="max-w-3xl space-y-6">
        <h1 className="text-4xl font-bold tracking-tight">Privacy Policy</h1>
        <p className="text-muted-foreground">
          This is a template privacy policy page for Glide ID. Replace this with your finalized
          legal text before production launch.
        </p>

        <div className="space-y-4 text-sm text-muted-foreground">
          <div className="border rounded-lg p-5 bg-card">
            <div className="font-medium text-foreground mb-2">Biometric data</div>
            <p>
              When enabled by an organization, Glide ID may collect facial images to generate
              encrypted biometric templates for identity verification. Templates are used for
              attendance, access control, anomaly detection, and security auditing.
            </p>
          </div>
          <div className="border rounded-lg p-5 bg-card">
            <div className="font-medium text-foreground mb-2">Tenant isolation</div>
            <p>
              Customer data is logically separated by tenant. Access is controlled by role-based
              permissions and database policies designed to prevent cross-tenant access.
            </p>
          </div>
          <div className="border rounded-lg p-5 bg-card">
            <div className="font-medium text-foreground mb-2">Retention & deletion</div>
            <p>
              Organizations control retention periods. Terminated employee biometric templates can
              be purged automatically after a retention window (e.g., 30 days) depending on
              configuration and legal requirements.
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}

