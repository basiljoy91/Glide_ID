import Link from 'next/link'
import { Button } from '@/components/ui/button'
import { Shield, Users, Clock, Zap, Lock, BarChart3 } from 'lucide-react'
import { KioskCodeLauncher } from '@/components/kiosk/KioskCodeLauncher'

export default function LandingPage() {
  return (
    <div className="flex flex-col">
      {/* Hero Section */}
      <section className="container mx-auto px-4 py-24 text-center">
        <h1 className="text-5xl font-bold tracking-tight mb-6">
          Enterprise Facial Recognition
          <br />
          <span className="text-primary">Attendance & Identity System</span>
        </h1>
        <p className="text-xl text-muted-foreground mb-8 max-w-2xl mx-auto">
          Secure, scalable, and compliant. Transform your workforce management with
          AI-powered biometric authentication and seamless physical access control.
        </p>
        <div className="flex flex-wrap gap-4 justify-center">
          <Link href="/onboarding">
            <Button size="lg" className="text-lg px-8">
              Get Started
            </Button>
          </Link>
          <Link href="/admin/login">
            <Button size="lg" variant="outline" className="text-lg px-8">
              Admin Login
            </Button>
          </Link>
          <Link href="/kiosk">
            <Button size="lg" variant="secondary" className="text-lg px-8">
              Kiosk Check-In
            </Button>
          </Link>
          <Link href="/pricing">
            <Button size="lg" variant="outline" className="text-lg px-8">
              View Pricing
            </Button>
          </Link>
        </div>

        <KioskCodeLauncher />
      </section>

      {/* Features Section */}
      <section className="container mx-auto px-4 py-24">
        <h2 className="text-3xl font-bold text-center mb-12">
          Enterprise-Grade Features
        </h2>
        <div className="grid md:grid-cols-3 gap-8">
          <FeatureCard
            icon={<Shield className="h-8 w-8" />}
            title="Military-Grade Security"
            description="AES-256 encryption, mTLS, HMAC signing, and Row-Level Security ensure your biometric data is protected."
          />
          <FeatureCard
            icon={<Users className="h-8 w-8" />}
            title="Multi-Tenant SaaS"
            description="Complete data isolation with tenant-specific workspaces. Scale from startups to enterprises."
          />
          <FeatureCard
            icon={<Clock className="h-8 w-8" />}
            title="Offline-First Kiosks"
            description="Monotonic clock technology ensures accurate time tracking even when network is down."
          />
          <FeatureCard
            icon={<Zap className="h-8 w-8" />}
            title="Real-Time Processing"
            description="HNSW indexing enables instant 1:N face matching for thousands of employees simultaneously."
          />
          <FeatureCard
            icon={<Lock className="h-8 w-8" />}
            title="Physical Access Control"
            description="IoT door relay integration with Wiegand Protocol support for seamless building access."
          />
          <FeatureCard
            icon={<BarChart3 className="h-8 w-8" />}
            title="HRMS Integration"
            description="Connect with Workday, SAP, BambooHR. Automated payroll exports and webhook provisioning."
          />
        </div>
      </section>

      {/* CTA Section */}
      <section className="container mx-auto px-4 py-24 text-center bg-muted/50 rounded-lg">
        <h2 className="text-3xl font-bold mb-4">Ready to Get Started?</h2>
        <p className="text-muted-foreground mb-8 max-w-xl mx-auto">
          Join leading enterprises using Glide ID for secure, compliant workforce management.
        </p>
        <div className="flex flex-wrap gap-3 justify-center">
          <Link href="/onboarding">
            <Button size="lg" className="text-lg px-8">
              Start Free Trial
            </Button>
          </Link>
          <Link href="/about">
            <Button size="lg" variant="outline" className="text-lg px-8">
              Learn about multi-tenant security
            </Button>
          </Link>
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
    <div className="p-6 border rounded-lg hover:shadow-lg transition-shadow">
      <div className="text-primary mb-4">{icon}</div>
      <h3 className="text-xl font-semibold mb-2">{title}</h3>
      <p className="text-muted-foreground">{description}</p>
    </div>
  )
}

