import { LandingPage } from '@/components/public/LandingPage'
import type { Metadata } from 'next'

const siteUrl = process.env.NEXT_PUBLIC_SITE_URL || 'https://glide-id.example'

export const metadata: Metadata = {
  title: 'Glide ID | Enterprise Facial Recognition Attendance',
  description:
    'Secure, scalable facial recognition attendance and access control with liveness checks, offline kiosks, and compliance automation.',
  keywords: [
    'facial recognition attendance',
    'biometric access control',
    'liveness detection',
    'offline kiosk',
    'pgvector',
    'enterprise RBAC',
  ],
  metadataBase: new URL(siteUrl),
  alternates: { canonical: '/landing' },
  openGraph: {
    title: 'Glide ID | Enterprise Facial Recognition Attendance',
    description:
      'Secure, scalable facial recognition attendance and access control with liveness checks, offline kiosks, and compliance automation.',
    url: '/landing',
    siteName: 'Glide ID',
    images: [{ url: '/hero-visual.svg', width: 1200, height: 630, alt: 'Glide ID overview' }],
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Glide ID | Enterprise Facial Recognition Attendance',
    description:
      'Secure, scalable facial recognition attendance and access control with liveness checks, offline kiosks, and compliance automation.',
    images: ['/hero-visual.svg'],
  },
}

export default function LandingRoutePage() {
  return <LandingPage />
}
