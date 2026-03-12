import Link from 'next/link'
import { Button } from '@/components/ui/button'
import type { Metadata } from 'next'

const siteUrl = process.env.NEXT_PUBLIC_SITE_URL || 'https://glide-id.example'

export const metadata: Metadata = {
  title: 'About Glide ID | Security-First Attendance Platform',
  description:
    'Learn about Glide ID’s mission, leadership, and the security-first architecture behind our biometric attendance platform.',
  keywords: [
    'Glide ID',
    'biometric attendance',
    'security-first SaaS',
    'row-level security',
    'compliance automation',
  ],
  metadataBase: new URL(siteUrl),
  alternates: { canonical: '/about' },
  openGraph: {
    title: 'About Glide ID',
    description:
      'Learn about Glide ID’s mission, leadership, and the security-first architecture behind our biometric attendance platform.',
    url: '/about',
    siteName: 'Glide ID',
    images: [{ url: '/hero-visual.svg', width: 1200, height: 630, alt: 'Glide ID mission' }],
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'About Glide ID',
    description:
      'Learn about Glide ID’s mission, leadership, and the security-first architecture behind our biometric attendance platform.',
    images: ['/hero-visual.svg'],
  },
}

export default function AboutPage() {
  return (
    <div className="container mx-auto px-4 py-16 space-y-16">
      <section className="grid gap-12 lg:grid-cols-[1.1fr_0.9fr] lg:items-center">
        <div className="space-y-6">
          <div className="inline-flex items-center gap-2 rounded-full border bg-background/80 px-4 py-2 text-xs font-semibold uppercase tracking-[0.25em] text-muted-foreground">
            About Glide ID
          </div>
          <h1 className="text-4xl font-display font-semibold tracking-tight sm:text-5xl">
            Built for teams that can’t afford attendance fraud
          </h1>
          <p className="text-lg text-muted-foreground">
            Glide ID is a security-first, multi-tenant attendance and physical access platform.
            We help regulated teams prove identity, enforce compliance, and remove manual payroll
            reconciliation without slowing down check-ins.
          </p>
          <div className="grid gap-4 sm:grid-cols-2">
            {[
              { k: 'Mission', v: 'Make biometric attendance defensible and auditable.' },
              { k: 'Vision', v: 'Eliminate time fraud across global workforces.' },
              { k: 'Focus', v: 'Security, speed, and privacy-preserving design.' },
              { k: 'Model', v: 'Multi-tenant SaaS with strict data isolation.' },
            ].map((item) => (
              <div key={item.k} className="rounded-2xl border bg-background/80 p-4">
                <div className="text-sm uppercase tracking-[0.2em] text-muted-foreground">
                  {item.k}
                </div>
                <div className="mt-2 text-sm text-foreground">{item.v}</div>
              </div>
            ))}
          </div>
          <div className="flex flex-wrap gap-3">
            <Link href="/onboarding">
              <Button>Start onboarding</Button>
            </Link>
            <Link href="/contact">
              <Button variant="outline">Talk to the team</Button>
            </Link>
          </div>
        </div>
        <div className="rounded-3xl border bg-muted/30 p-6">
          <img
            src="/hero-visual.svg"
            alt="Glide ID security mesh"
            className="w-full rounded-2xl border bg-background"
          />
          <div className="mt-4 text-sm text-muted-foreground">
            Built with encrypted vectors, liveness detection, and kiosk-level HMAC signing.
          </div>
        </div>
      </section>

      <section className="grid gap-8 lg:grid-cols-[1.1fr_0.9fr] lg:items-center">
        <div className="space-y-4">
          <h2 className="text-3xl font-display font-semibold">How multi-tenant security works</h2>
          <p className="text-muted-foreground">
            Glide ID isolates every tenant at the database level, encrypts biometric vectors, and
            enforces least-privilege access for administrators.
          </p>
          <div className="grid gap-3">
            {[
              {
                title: 'Tenant provisioning',
                text: 'Every organization is created as a tenant with its own SSO domain, kiosk code, and limits.',
              },
              {
                title: 'Tenant-scoped identity',
                text: 'Admins and employees belong to one tenant only; all queries are RLS-protected.',
              },
              {
                title: 'Kiosk authentication',
                text: 'Each kiosk uses device-bound secrets and optional mTLS for device trust.',
              },
              {
                title: 'Audit-first workflows',
                text: 'Immutable logs ensure every attendance event is traceable.',
              },
            ].map((item) => (
              <div key={item.title} className="rounded-2xl border bg-background/80 p-4">
                <div className="font-semibold">{item.title}</div>
                <div className="mt-2 text-sm text-muted-foreground">{item.text}</div>
              </div>
            ))}
          </div>
        </div>
        <div className="rounded-3xl border bg-background/80 p-6">
          <h3 className="text-xl font-display font-semibold">Trust signals</h3>
          <div className="mt-4 grid gap-3">
            {[
              'SOC 2 aligned controls',
              'GDPR/CCPA retention policies',
              'HMAC + mTLS kiosk hardening',
              'AES-256 encrypted vectors',
              'Row-level security enforced',
            ].map((item) => (
              <div key={item} className="rounded-xl border bg-muted/30 px-4 py-3 text-sm">
                {item}
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="space-y-6">
        <h2 className="text-3xl font-display font-semibold">Leadership & team</h2>
        <div className="grid gap-6 md:grid-cols-3">
          {[
            { name: 'Basil Joy', role: 'Founder & Engineer', img: '/team/basil-joy.svg' },
            { name: 'Security Lead', role: 'Liveness & Threat Modeling', img: '/team/avatar-neutral.svg' },
            { name: 'Platform Ops', role: 'SRE & Compliance', img: '/team/avatar-neutral.svg' },
          ].map((member) => (
            <div key={member.name} className="rounded-2xl border bg-background/80 p-4">
              <img src={member.img} alt={member.name} className="w-full rounded-xl border" />
              <div className="mt-4 font-semibold">{member.name}</div>
              <div className="text-sm text-muted-foreground">{member.role}</div>
            </div>
          ))}
        </div>
      </section>

      <section className="grid gap-8 lg:grid-cols-[0.9fr_1.1fr] lg:items-center">
        <div className="rounded-3xl border bg-muted/30 p-6">
          <h3 className="text-xl font-display font-semibold">Milestones</h3>
          <div className="mt-4 space-y-4 text-sm text-muted-foreground">
            <div className="rounded-xl border bg-background/80 px-4 py-3">
              2023 — Prototype deployments with offline kiosk resilience
            </div>
            <div className="rounded-xl border bg-background/80 px-4 py-3">
              2024 — ArcFace vector pipeline + HNSW indexing
            </div>
            <div className="rounded-xl border bg-background/80 px-4 py-3">
              2025 — HRMS integrations and compliance automation
            </div>
            <div className="rounded-xl border bg-background/80 px-4 py-3">
              2026 — Global enterprise rollout program
            </div>
          </div>
        </div>
        <div className="space-y-4">
          <h2 className="text-3xl font-display font-semibold">What we stand for</h2>
          <ul className="grid gap-3 text-sm text-muted-foreground">
            <li className="rounded-2xl border bg-background/80 px-4 py-3">
              Privacy-by-design. We never store raw facial imagery.
            </li>
            <li className="rounded-2xl border bg-background/80 px-4 py-3">
              Security-first architecture with strict tenant isolation.
            </li>
            <li className="rounded-2xl border bg-background/80 px-4 py-3">
              Fast check-ins to keep real-world operations moving.
            </li>
            <li className="rounded-2xl border bg-background/80 px-4 py-3">
              Compliance-ready processes for regulated environments.
            </li>
          </ul>
        </div>
      </section>
    </div>
  )
}
