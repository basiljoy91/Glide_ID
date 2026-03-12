'use client'

import Link from 'next/link'
import { Button } from '@/components/ui/button'

export default function BlogPage() {
  return (
    <div className="container mx-auto px-4 py-16 space-y-10">
      <div className="max-w-3xl">
        <h1 className="text-4xl font-bold tracking-tight">Blog</h1>
        <p className="mt-4 text-muted-foreground text-lg">
          Product updates, security notes, and implementation guides.
        </p>
      </div>

      <div className="border rounded-lg p-6 bg-card text-sm text-muted-foreground">
        Coming soon.
      </div>

      <Link href="/landing">
        <Button variant="outline">Back to landing</Button>
      </Link>
    </div>
  )
}

