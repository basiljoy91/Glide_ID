'use client'

import Link from 'next/link'
import { Button } from '@/components/ui/button'

export default function AboutPage() {
  return (
    <div className="container mx-auto px-4 py-16 space-y-12">
      <div className="max-w-3xl">
        <h1 className="text-4xl font-bold tracking-tight">About Glide ID</h1>
        <p className="mt-4 text-muted-foreground text-lg">
          Glide ID is a multi-tenant, security-first attendance and identity platform designed for
          organizations that need biometric verification, offline-capable kiosks, and strict data
          isolation.
        </p>
      </div>

      <section className="grid lg:grid-cols-2 gap-10 items-start">
        <div className="space-y-4">
          <h2 className="text-2xl font-semibold">Multi-tenant process (how it works)</h2>
          <p className="text-muted-foreground">
            Glide ID is built as a true multi-tenant SaaS. Each organization is created as a
            <strong className="text-foreground"> tenant</strong>, and every user, kiosk, attendance
            log, and integration record is stored with a tenant identifier.
          </p>
          <div className="space-y-3">
            <div className="border rounded-lg p-4 bg-card">
              <div className="font-medium mb-1">1) Tenant provisioning</div>
              <div className="text-sm text-muted-foreground">
                When a customer completes onboarding, we create a tenant workspace with its own
                settings (SSO domain/provider, kiosk code, limits, etc.).
              </div>
            </div>
            <div className="border rounded-lg p-4 bg-card">
              <div className="font-medium mb-1">2) Tenant-scoped identity</div>
              <div className="text-sm text-muted-foreground">
                Admins and employees belong to exactly one tenant. RBAC controls what they can see
                and do within that tenant (Org Admin, HR, Dept Manager, Employee).
              </div>
            </div>
            <div className="border rounded-lg p-4 bg-card">
              <div className="font-medium mb-1">3) Database isolation (RLS)</div>
              <div className="text-sm text-muted-foreground">
                Row-Level Security policies ensure a tenant can only access its own rows. Even if a
                bug exists at the API layer, cross-tenant access is blocked at the database level.
              </div>
            </div>
            <div className="border rounded-lg p-4 bg-card">
              <div className="font-medium mb-1">4) Tenant-aware kiosks (HMAC + optional mTLS)</div>
              <div className="text-sm text-muted-foreground">
                Kiosk devices authenticate requests using per-kiosk secrets (HMAC). For higher
                security environments, mTLS is terminated at the gateway so only managed kiosks can
                connect.
              </div>
            </div>
          </div>
        </div>

        <div className="space-y-4">
          <h2 className="text-2xl font-semibold">Advantages</h2>
          <ul className="list-disc list-inside text-muted-foreground space-y-2">
            <li>
              <strong className="text-foreground">Hard tenant isolation</strong> via RLS + scoped
              APIs
            </li>
            <li>
              <strong className="text-foreground">Security-first</strong>: HMAC, offline encryption,
              audit logs, least-privilege RBAC
            </li>
            <li>
              <strong className="text-foreground">Offline-first kiosks</strong> with encrypted queue
              and reconciliation
            </li>
            <li>
              <strong className="text-foreground">Scalable operations</strong> with clear tenant
              boundaries for support and reporting
            </li>
            <li>
              <strong className="text-foreground">Compliance-ready</strong> consent, retention and
              automated purging patterns
            </li>
          </ul>

          <div className="border rounded-lg p-6 bg-card">
            <h3 className="text-xl font-semibold">Developer</h3>
            <div className="mt-4 grid sm:grid-cols-[160px_1fr] gap-6 items-start">
              <div className="border rounded-lg bg-muted/50 aspect-square w-full max-w-[180px] flex items-center justify-center text-sm text-muted-foreground">
                Photo
              </div>
              <div className="space-y-2">
                <div className="font-medium">Basil Joy</div>
                <div className="text-sm text-muted-foreground">
                  Full-stack developer. Building secure multi-tenant systems, kiosk workflows, and
                  identity-first product experiences.
                </div>
                <div className="text-sm text-muted-foreground">
                  If you want your actual photo here later, add an image file under
                  <code className="text-foreground"> public/</code> and we can swap this placeholder
                  to a real profile image.
                </div>
              </div>
            </div>
          </div>

          <div className="flex gap-3">
            <Link href="/onboarding">
              <Button>Start onboarding</Button>
            </Link>
            <Link href="/contact">
              <Button variant="outline">Contact</Button>
            </Link>
          </div>
        </div>
      </section>
    </div>
  )
}

