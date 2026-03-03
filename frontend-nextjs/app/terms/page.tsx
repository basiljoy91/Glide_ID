'use client'

export default function TermsPage() {
  return (
    <div className="container mx-auto px-4 py-16">
      <div className="max-w-3xl space-y-6">
        <h1 className="text-4xl font-bold tracking-tight">Terms of Service</h1>
        <p className="text-muted-foreground">
          This is a template Terms of Service page for Glide ID. Replace with your finalized legal
          text before production launch.
        </p>

        <div className="space-y-4 text-sm text-muted-foreground">
          <div className="border rounded-lg p-5 bg-card">
            <div className="font-medium text-foreground mb-2">Service description</div>
            <p>
              Glide ID provides tenant-scoped attendance tracking, identity verification, and kiosk
              workflows. Organizations are responsible for user onboarding, policy configuration,
              and lawful basis for biometric processing.
            </p>
          </div>
          <div className="border rounded-lg p-5 bg-card">
            <div className="font-medium text-foreground mb-2">Acceptable use</div>
            <p>
              You agree not to misuse the service, bypass security controls, or attempt to access
              data belonging to other tenants. Suspicious activity may be logged and restricted.
            </p>
          </div>
          <div className="border rounded-lg p-5 bg-card">
            <div className="font-medium text-foreground mb-2">Disclaimer</div>
            <p>
              This template is provided for development and demo purposes only and does not
              constitute legal advice.
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}

