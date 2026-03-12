'use client'

import Link from 'next/link'
import { Button } from '@/components/ui/button'
import {
  ShieldCheck,
  WifiOff,
  ServerCrash,
  BarChart3,
  Network,
  FileCheck2,
  LockKeyhole,
  BellRing,
  Smartphone,
  BookOpen,
  UsersRound
} from 'lucide-react'

const FEATURES = [
  {
    title: 'Multi-tenant SaaS & Data Isolation',
    desc: 'Tenant-scoped data model with database Row-Level Security (RLS) ensuring strict tenant isolation.',
    icon: <ShieldCheck className="h-6 w-6 text-primary" />,
  },
  {
    title: 'Offline-First Kiosk Sync',
    desc: 'Encrypted offline queue guarantees no data loss. Auto-reconciliation to the cloud when connectivity returns.',
    icon: <WifiOff className="h-6 w-6 text-primary" />,
  },
  {
    title: 'Edge Security & mTLS',
    desc: 'HMAC request signing for off-the-shelf kiosks, and mTLS termination at gateway for managed corporate devices.',
    icon: <ServerCrash className="h-6 w-6 text-primary" />,
  },
  {
    title: 'Enterprise Analytics',
    desc: 'Tenant-scoped metrics, real-time attendance tracking, and AI-powered anomaly review workflows.',
    icon: <BarChart3 className="h-6 w-6 text-primary" />,
  },
  {
    title: 'HRMS Integration Hub',
    desc: 'Connect your existing HR software. Provider integrations are stored per tenant with robust webhook processing.',
    icon: <Network className="h-6 w-6 text-primary" />,
  },
  {
    title: 'Compliance & Data Purgation',
    desc: 'Built-in consent capture and automated data purging schedules for terminated employees to meet GDPR/CCPA.',
    icon: <FileCheck2 className="h-6 w-6 text-primary" />,
  },
  {
    title: 'SSO & SAML Integration',
    desc: 'Seamlessly integrate with Okta, Azure AD, Google Workspace, and other major identity providers.',
    icon: <LockKeyhole className="h-6 w-6 text-primary" />,
  },
  {
    title: 'Real-Time Notifications',
    desc: 'Get instantly alerted via Email, Slack, or MS Teams when anomalies or critical events are detected.',
    icon: <BellRing className="h-6 w-6 text-primary" />,
  },
  {
    title: 'Mobile App Management',
    desc: 'Admins and managers can approve time logs, review anomalies, and check metrics on the go from iOS & Android.',
    icon: <Smartphone className="h-6 w-6 text-primary" />,
  },
  {
    title: 'Comprehensive API Documentation',
    desc: 'Build custom workflows using our fully documented REST and GraphQL endpoints for ultimate flexibility.',
    icon: <BookOpen className="h-6 w-6 text-primary" />,
  },
  {
    title: 'Role-Based Access Controls',
    desc: 'Granular RBAC ensures users only see what they should. Define custom policies for managers, HR, and admins.',
    icon: <UsersRound className="h-6 w-6 text-primary" />,
  },
]

export default function FeaturesPage() {
  return (
    <div className="container mx-auto px-4 py-16 space-y-24">
      
      {/* HEADER */}
      <div className="text-center max-w-3xl mx-auto">
        <h1 className="text-4xl md:text-5xl font-bold tracking-tight">Built for scale. Secured by design.</h1>
        <p className="mt-4 text-muted-foreground text-lg leading-relaxed">
          Glide ID combines robust edge AI with enterprise-grade cloud security to deliver 
          a flawless identity and attendance experience.
        </p>
        <div className="mt-8 flex items-center justify-center gap-4">
          <Link href="/onboarding">
            <Button size="lg" className="h-12 px-8">Get started</Button>
          </Link>
          <Link href="/contact">
            <Button size="lg" variant="outline" className="h-12 px-8">Talk to us</Button>
          </Link>
        </div>
      </div>

      {/* HIGHLIGHT MOCKUP 1: Anomaly Review Dashboard */}
      <div className="grid md:grid-cols-2 gap-12 items-center max-w-6xl mx-auto">
        <div className="space-y-6">
          <div className="inline-flex items-center rounded-lg bg-primary/10 px-3 py-1 text-sm font-medium text-primary">
            AI-Powered Anomaly Detection
          </div>
          <h2 className="text-3xl font-bold tracking-tight">Catch buddy-punching automatically</h2>
          <p className="text-lg text-muted-foreground">
            Our facial recognition engine flags mismatches with 99.8% accuracy.
            Managers get a dedicated dashboard to quickly review flagged photos and approve or reject attendance logs.
          </p>
          <ul className="space-y-3">
            <li className="flex items-center gap-3">
              <div className="h-6 w-6 rounded-full bg-primary/20 flex items-center justify-center">
                <CheckIcon className="h-4 w-4 text-primary" />
              </div>
              <span className="font-medium">Liveness detection built-in</span>
            </li>
            <li className="flex items-center gap-3">
              <div className="h-6 w-6 rounded-full bg-primary/20 flex items-center justify-center">
                <CheckIcon className="h-4 w-4 text-primary" />
              </div>
              <span className="font-medium">Real-time alerts to managers</span>
            </li>
            <li className="flex items-center gap-3">
              <div className="h-6 w-6 rounded-full bg-primary/20 flex items-center justify-center">
                <CheckIcon className="h-4 w-4 text-primary" />
              </div>
              <span className="font-medium">Audit logs for HR compliance</span>
            </li>
          </ul>
        </div>
        <div className="relative h-[400px] rounded-xl border bg-card shadow-xl overflow-hidden flex flex-col">
          {/* Dashboard Mockup UI */}
          <div className="h-12 border-b bg-muted/30 flex items-center px-4 space-x-2">
            <div className="h-3 w-3 rounded-full bg-red-400" />
            <div className="h-3 w-3 rounded-full bg-amber-400" />
            <div className="h-3 w-3 rounded-full bg-green-400" />
          </div>
          <div className="p-6 flex-1 bg-muted/10 grid grid-cols-2 gap-4">
            <div className="space-y-4">
              <div className="h-8 w-3/4 rounded bg-muted animate-pulse" />
              <div className="h-32 w-full rounded-lg bg-red-100 dark:bg-red-900/20 border-2 border-red-500/50 flex items-center justify-center text-red-500 font-medium">Flagged Photo</div>
            </div>
            <div className="space-y-4">
              <div className="h-8 w-1/2 rounded bg-muted animate-pulse" />
              <div className="h-32 w-full rounded-lg bg-muted flex items-center justify-center text-muted-foreground">Original ID Profile</div>
            </div>
            <div className="col-span-2 flex justify-end gap-2 mt-auto">
              <div className="h-10 w-24 rounded bg-red-500/10 text-red-500 flex items-center justify-center font-medium text-sm">Reject</div>
              <div className="h-10 w-24 rounded bg-primary text-primary-foreground flex items-center justify-center font-medium text-sm">Approve</div>
            </div>
          </div>
        </div>
      </div>

      {/* HIGHLIGHT MOCKUP 2: Offline Sync */}
      <div className="grid md:grid-cols-2 gap-12 items-center max-w-6xl mx-auto pt-12">
        <div className="relative h-[400px] rounded-xl border bg-card shadow-xl overflow-hidden flex flex-col order-2 md:order-1">
          {/* App Mockup UI */}
          <div className="h-12 border-b bg-muted/30 flex items-center justify-between px-4">
            <div className="font-semibold text-sm">Main Reception Kiosk</div>
            <div className="flex items-center gap-2 text-xs font-medium text-amber-500 bg-amber-500/10 px-2 py-1 rounded-full">
              <WifiOff className="h-3 w-3" /> Offline (42 pending)
            </div>
          </div>
          <div className="p-6 flex-1 bg-muted/10 flex flex-col items-center justify-center text-center space-y-4">
            <div className="h-32 w-32 rounded-full border-4 border-primary/20 flex flex-col items-center justify-center text-primary relative">
              <span className="text-4xl font-bold">10:42</span>
              <span className="text-sm font-medium">AM</span>
              <div className="absolute -bottom-2 bg-primary text-primary-foreground text-xs font-bold px-3 py-1 rounded-full">
                Scanning...
              </div>
            </div>
            <p className="text-muted-foreground font-medium max-w-[200px]">Recording attendance securely to local storage.</p>
          </div>
        </div>
        <div className="space-y-6 order-1 md:order-2">
          <div className="inline-flex items-center rounded-lg bg-primary/10 px-3 py-1 text-sm font-medium text-primary">
            Edge Resilience
          </div>
          <h2 className="text-3xl font-bold tracking-tight">Fail-safe offline mode</h2>
          <p className="text-lg text-muted-foreground">
            Internet goes down? Work doesn&apos;t stop. Our edge client caches attendance events locally 
            using strong cryptography, instantly and securely syncing them to the cloud the moment the connection is restored.
          </p>
          <ul className="space-y-3">
            <li className="flex items-center gap-3">
              <div className="h-6 w-6 rounded-full bg-primary/20 flex items-center justify-center">
                <CheckIcon className="h-4 w-4 text-primary" />
              </div>
              <span className="font-medium">Encrypted SQLite queue</span>
            </li>
            <li className="flex items-center gap-3">
              <div className="h-6 w-6 rounded-full bg-primary/20 flex items-center justify-center">
                <CheckIcon className="h-4 w-4 text-primary" />
              </div>
              <span className="font-medium">Zero data loss guarantee</span>
            </li>
            <li className="flex items-center gap-3">
              <div className="h-6 w-6 rounded-full bg-primary/20 flex items-center justify-center">
                <CheckIcon className="h-4 w-4 text-primary" />
              </div>
              <span className="font-medium">Automatic deduplication</span>
            </li>
          </ul>
        </div>
      </div>

      {/* FULL FEATURE GRID */}
      <div className="pt-24 border-t">
        <div className="text-center max-w-2xl mx-auto mb-16">
          <h2 className="text-3xl font-bold tracking-tight">Everything you need</h2>
          <p className="mt-4 text-muted-foreground text-lg">
            A comprehensive suite of tools built for IT, HR, and facility managers.
          </p>
        </div>

        <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-x-8 gap-y-12">
          {FEATURES.map((f) => (
            <div key={f.title} className="group relative">
              <div className="mb-4 inline-flex h-12 w-12 items-center justify-center rounded-lg bg-primary/10 transition-colors group-hover:bg-primary group-hover:text-primary-foreground">
                <div className="transition-transform group-hover:scale-110">
                  {f.icon}
                </div>
              </div>
              <h3 className="text-xl font-semibold mb-2">{f.title}</h3>
              <p className="text-muted-foreground leading-relaxed">
                {f.desc}
              </p>
            </div>
          ))}
        </div>
      </div>

    </div>
  )
}

function CheckIcon(props: React.SVGProps<SVGSVGElement>) {
  return (
    <svg
      {...props}
      fill="none"
      stroke="currentColor"
      viewBox="0 0 24 24"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={3}
        d="M5 13l4 4L19 7"
      />
    </svg>
  )
}

