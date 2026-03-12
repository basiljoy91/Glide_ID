import type { Metadata } from 'next'
import { BlogIndex } from '@/components/public/BlogIndex'

const siteUrl = process.env.NEXT_PUBLIC_SITE_URL || 'https://glide-id.example'

export const metadata: Metadata = {
  title: 'Glide ID Journal | Security & Product Updates',
  description:
    'Product updates, security notes, and implementation guides for enterprise biometric attendance.',
  keywords: [
    'biometric security',
    'attendance automation',
    'liveness detection',
    'HRMS integrations',
    'compliance',
  ],
  metadataBase: new URL(siteUrl),
  alternates: { canonical: '/blog' },
  openGraph: {
    title: 'Glide ID Journal',
    description:
      'Product updates, security notes, and implementation guides for enterprise biometric attendance.',
    url: '/blog',
    siteName: 'Glide ID',
    images: [{ url: '/hero-visual.svg', width: 1200, height: 630, alt: 'Glide ID journal' }],
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Glide ID Journal',
    description:
      'Product updates, security notes, and implementation guides for enterprise biometric attendance.',
    images: ['/hero-visual.svg'],
  },
}

export default function BlogPage() {
  return <BlogIndex />
}
