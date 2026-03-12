export type BlogPost = {
  slug: string
  title: string
  excerpt: string
  category: string
  readTime: string
  date: string
  content: string[]
}

export const blogPosts: BlogPost[] = [
  {
    slug: 'spoof-resistant-liveness',
    title: 'Designing spoof-resistant liveness checks',
    excerpt:
      'How we combine passive and active liveness signals with ArcFace confidence to reduce spoof acceptance in the real world.',
    category: 'Security',
    readTime: '6 min read',
    date: 'Mar 04, 2026',
    content: [
      'Liveness is more than a single score. We use a layered pipeline that blends passive signals (texture, moire detection, motion micro-variance) with active prompts (head tilt, blink, depth cues).',
      'The goal is to minimize false rejections while keeping spoof acceptance extremely low. We treat the liveness score as a gating mechanism and only allow fallback paths for low-risk scenarios.',
      'Operationally, every liveness decision is logged alongside the kiosk device, camera metadata, and confidence bands for audit review.',
    ],
  },
  {
    slug: 'offline-attendance-without-time-spoofing',
    title: 'Offline attendance without time spoofing',
    excerpt:
      'A deep dive into monotonic clocks, encrypted queues, and reconciliation strategies for kiosk deployments.',
    category: 'Engineering',
    readTime: '8 min read',
    date: 'Feb 19, 2026',
    content: [
      'Offline mode is only safe if you can trust the timestamp. We use a monotonic clock offset relative to the last server ping to reconstruct real punch time.',
      'Kiosks encrypt offline payloads using the public key and never store raw images. On sync, the backend validates the offset window and rejects tampered payloads.',
      'This approach removes time travel attacks while preserving operational continuity.',
    ],
  },
  {
    slug: 'hrms-integrations-payroll-drift',
    title: 'HRMS integrations that remove payroll drift',
    excerpt:
      'Patterns for syncing Workday, SAP, and BambooHR with automated anomaly review workflows.',
    category: 'Operations',
    readTime: '5 min read',
    date: 'Jan 28, 2026',
    content: [
      'We treat HRMS as the system of record and Glide ID as the execution layer. Inbound hires trigger enrollment; outbound terminations trigger timed biometric purges.',
      'Payroll export is blocked until anomalies are reviewed, preventing errors from propagating to downstream systems.',
      'Webhook retries and audit trails ensure every integration event is visible and recoverable.',
    ],
  },
]

export const blogCategories = ['All', 'Security', 'Engineering', 'Operations', 'Compliance', 'Product']
