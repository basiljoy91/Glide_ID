import Link from 'next/link'
import Image from 'next/image'
import { Button } from '@/components/ui/button'
import { Shield, Users, Clock, Zap, Lock, BarChart3, Play } from 'lucide-react'
import { KioskCodeLauncher } from '@/components/kiosk/KioskCodeLauncher'
import { MetricCounters } from '@/components/public/MetricCounters'

export function LandingPage() {
  return (
    <div className="flex flex-col">
      <section className="relative overflow-hidden">
        <div className="pointer-events-none absolute inset-0 hero-grid opacity-60" />
        <div className="pointer-events-none absolute -top-32 -right-20 h-72 w-72 rounded-full bg-emerald-400/20 blur-3xl" />
        <div className="pointer-events-none absolute -bottom-40 -left-20 h-80 w-80 rounded-full bg-sky-500/20 blur-3xl" />
        <div className="container mx-auto relative z-10 grid gap-12 px-4 py-20 lg:grid-cols-[1.1fr_0.9fr] lg:items-center">
          <div className="space-y-6">
            <div className="inline-flex items-center gap-2 rounded-full border bg-background/80 px-4 py-2 text-xs font-semibold uppercase tracking-[0.25em] text-muted-foreground">
              Zero trust attendance
              <span className="h-1 w-1 rounded-full bg-primary" />
              ArcFace-grade matching
            </div>
            <h1 className="text-5xl font-display font-semibold leading-tight sm:text-6xl">
              Enterprise facial recognition
              <span className="block text-primary">attendance & access control</span>
            </h1>
            <p className="text-lg text-muted-foreground">
              Secure, scalable, and compliant. Glide ID combines on-device liveness, encrypted
              offline queues, and high-precision vector search to remove attendance fraud without
              adding friction to teams.
            </p>
            <div className="flex flex-wrap gap-3">
              <Link href="/onboarding">
                <Button size="lg" className="px-8">
                  Get Started
                </Button>
              </Link>
              <Link href="/pricing">
                <Button size="lg" variant="outline" className="px-8">
                  View Pricing
                </Button>
              </Link>
              <Link href="/admin/login">
                <Button size="lg" variant="secondary" className="px-8">
                  Admin Login
                </Button>
              </Link>
            </div>
            <div className="rounded-2xl border bg-background/80 p-4 shadow-sm">
              <div className="text-sm font-semibold text-foreground">Try a kiosk demo</div>
              <div className="text-sm text-muted-foreground">
                Use your 10-digit kiosk code to launch a live check-in flow.
              </div>
              <div className="mt-3">
                <KioskCodeLauncher variant="compact" />
              </div>
            </div>
          </div>
          <div className="relative">
            <div className="absolute -top-6 -left-6 h-16 w-16 rounded-2xl bg-primary/20 blur-xl" />
            <Image
              src="/hero-visual.svg"
              alt="Biometric mesh visualization"
              width={1000}
              height={760}
              className="w-full rounded-3xl border bg-background shadow-xl animate-float-slow"
            />
            <div className="absolute -bottom-6 left-6 rounded-2xl border bg-background/90 p-4 shadow-lg animate-float">
              <div className="text-xs uppercase tracking-[0.2em] text-muted-foreground">
                Liveness
              </div>
              <div className="text-xl font-display font-semibold text-foreground">98.4% pass</div>
              <div className="text-xs text-muted-foreground">Passive + active checks</div>
            </div>
          </div>
        </div>
      </section>

      <section className="container mx-auto px-4 py-12">
        <div className="mb-6 text-center text-sm uppercase tracking-[0.3em] text-muted-foreground">
          Trusted by high-security teams
        </div>
        <div className="grid gap-4 text-center text-sm text-muted-foreground sm:grid-cols-2 lg:grid-cols-5">
          {['Nimbus Labs', 'Altura Finance', 'Helio Works', 'Vantage Bio', 'KiteWorks AI'].map(
            (name) => (
              <div key={name} className="rounded-xl border bg-background/80 px-4 py-3">
                {name}
              </div>
            )
          )}
        </div>
      </section>

      <section className="container mx-auto px-4 py-12">
        <MetricCounters />
      </section>

      <section className="container mx-auto px-4 py-16">
        <div className="grid gap-8 lg:grid-cols-[1.1fr_0.9fr] lg:items-center">
          <div className="space-y-4">
            <h2 className="text-3xl font-display font-semibold">
              Built for scale, auditability, and speed
            </h2>
            <p className="text-muted-foreground">
              Every check-in is cryptographically signed and logged. Face vectors stay encrypted in
              `pgvector` with HNSW search for high-performance matching.
            </p>
            <div className="grid gap-4 sm:grid-cols-2">
              <FeatureCard
                icon={<Shield className="h-6 w-6" />}
                title="Security by default"
                description="AES-256, mTLS, HMAC signing, and row-level security guard every request."
              />
              <FeatureCard
                icon={<Users className="h-6 w-6" />}
                title="Multi-tenant control"
                description="Tenant-scoped RBAC, SSO, and managed kiosk provisioning."
              />
              <FeatureCard
                icon={<Clock className="h-6 w-6" />}
                title="Offline ready"
                description="Monotonic clock + encrypted queues prevent time spoofing."
              />
              <FeatureCard
                icon={<Zap className="h-6 w-6" />}
                title="Instant matching"
                description="HNSW indexing keeps check-ins under 2 seconds."
              />
              <FeatureCard
                icon={<Lock className="h-6 w-6" />}
                title="Physical access"
                description="Wiegand relay integration for door unlock workflows."
              />
              <FeatureCard
                icon={<BarChart3 className="h-6 w-6" />}
                title="HRMS automation"
                description="Workday, SAP, BambooHR connectors and webhooks."
              />
            </div>
          </div>
          <div className="rounded-3xl border bg-muted/30 p-6">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-xs uppercase tracking-[0.3em] text-muted-foreground">
                  90-sec demo
                </div>
                <h3 className="text-xl font-display font-semibold">See Glide ID in action</h3>
              </div>
              <Button size="icon" className="glow-ring">
                <Play className="h-4 w-4" />
              </Button>
            </div>
            <Image
              src="/demo-frame.svg"
              alt="Product demo frame"
              width={900}
              height={560}
              className="mt-6 rounded-2xl"
            />
          </div>
        </div>
      </section>

      <section className="container mx-auto px-4 py-16">
        <div className="grid gap-8 lg:grid-cols-[0.9fr_1.1fr] lg:items-center">
          <div className="rounded-3xl border bg-background/80 p-6 shadow-sm">
            <div className="text-xs uppercase tracking-[0.3em] text-muted-foreground">
              Case study
            </div>
            <h3 className="mt-3 text-2xl font-display font-semibold">
              Atlas Manufacturing cut payroll variance by 41%
            </h3>
            <p className="mt-3 text-muted-foreground">
              With 22 facilities and shift-based operations, Atlas used Glide ID to remove buddy
              punching, automate shift reconciliation, and enforce door access compliance.
            </p>
            <div className="mt-6 grid gap-4 sm:grid-cols-3">
              {[
                { k: '41%', v: 'Payroll variance reduced' },
                { k: '2.1s', v: 'Average check-in' },
                { k: '0', v: 'Spoof incidents after launch' },
              ].map((stat) => (
                <div key={stat.k} className="rounded-2xl border bg-muted/40 p-4 text-center">
                  <div className="text-xl font-display font-semibold">{stat.k}</div>
                  <div className="text-xs text-muted-foreground">{stat.v}</div>
                </div>
              ))}
            </div>
            <div className="mt-6">
              <Button variant="outline">Read the case study</Button>
            </div>
          </div>
          <div className="space-y-4">
            <h2 className="text-3xl font-display font-semibold">What teams say</h2>
            <div className="grid gap-4 sm:grid-cols-2">
              {[
                {
                  name: 'Priya Nair',
                  role: 'HR Director, Helio Works',
                  quote:
                    'We stopped manual audits entirely. The anomaly queue is crisp and payroll sync is clean.',
                },
                {
                  name: 'Jared M.',
                  role: 'Security Ops, Nimbus Labs',
                  quote:
                    'Liveness checks are fast and consistent. We finally trust the door unlock flow.',
                },
                {
                  name: 'Samuel K.',
                  role: 'IT Lead, Vantage Bio',
                  quote:
                    'Deployment across 12 sites was painless. Offline mode saved us during outages.',
                },
                {
                  name: 'Rhea Patel',
                  role: 'People Ops, Altura Finance',
                  quote:
                    'The admin dashboard gives us actual operational clarity, not just charts.',
                },
              ].map((t) => (
                <div key={t.name} className="rounded-2xl border bg-background/80 p-5 shadow-sm">
                  <div className="text-sm text-muted-foreground">{t.role}</div>
                  <div className="mt-2 text-sm text-foreground">“{t.quote}”</div>
                  <div className="mt-4 text-sm font-semibold">{t.name}</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      <section className="container mx-auto px-4 py-16">
        <div className="grid gap-10 lg:grid-cols-[1.1fr_0.9fr]">
          <div>
            <h2 className="text-3xl font-display font-semibold">Frequently asked</h2>
            <div className="mt-6 space-y-4">
              {[
                {
                  q: 'Do you store raw photos?',
                  a: 'No. We vectorize on the AI service, store only encrypted vectors, and purge biometrics after termination windows.',
                },
                {
                  q: 'How do you prevent clock spoofing in offline mode?',
                  a: 'Kiosks use monotonic clocks and capture offsets from the last server ping. The server reconstructs real time on sync.',
                },
                {
                  q: 'Can we integrate with our HRMS?',
                  a: 'Yes. We ship connectors and webhooks for Workday, SAP, BambooHR, and custom systems.',
                },
              ].map((item) => (
                <div key={item.q} className="rounded-2xl border bg-background/80 p-5">
                  <div className="font-semibold">{item.q}</div>
                  <div className="mt-2 text-sm text-muted-foreground">{item.a}</div>
                </div>
              ))}
            </div>
          </div>
          <div className="rounded-3xl border bg-muted/30 p-6">
            <div className="text-xs uppercase tracking-[0.3em] text-muted-foreground">
              Compliance stack
            </div>
            <div className="mt-4 grid gap-3">
              {['SOC 2 aligned controls', 'GDPR/CCPA retention workflows', 'Audit log immutability'].map(
                (item) => (
                  <div key={item} className="rounded-xl border bg-background/80 px-4 py-3 text-sm">
                    {item}
                  </div>
                )
              )}
            </div>
            <div className="mt-6">
              <Button variant="outline">Download compliance brief</Button>
            </div>
          </div>
        </div>
      </section>

      <section className="container mx-auto px-4 py-16">
        <div className="rounded-3xl border bg-gradient-to-br from-slate-900 via-slate-900 to-emerald-900 px-8 py-12 text-white">
          <div className="grid gap-6 lg:grid-cols-[1.2fr_0.8fr] lg:items-center">
            <div>
              <h2 className="text-3xl font-display font-semibold">
                Ready to modernize attendance?
              </h2>
              <p className="mt-3 text-slate-200">
                Launch your tenant workspace in minutes and connect kiosks globally.
              </p>
            </div>
            <div className="flex flex-wrap gap-3 lg:justify-end">
              <Link href="/onboarding">
                <Button size="lg" className="px-8">
                  Start Free Trial
                </Button>
              </Link>
              <Link href="/contact">
                <Button size="lg" variant="secondary" className="px-8">
                  Talk to Sales
                </Button>
              </Link>
            </div>
          </div>
        </div>
      </section>
    </div>
  )
}

function FeatureCard({
  icon,
  title,
  description,
}: {
  icon: React.ReactNode
  title: string
  description: string
}) {
  return (
    <div className="rounded-2xl border bg-background/80 p-4 shadow-sm">
      <div className="text-primary mb-3">{icon}</div>
      <h3 className="text-lg font-semibold">{title}</h3>
      <p className="mt-2 text-sm text-muted-foreground">{description}</p>
    </div>
  )
}
