'use client'

import { useEffect, useState } from 'react'
import { useAuthStore } from '@/store/useStore'
import { useRouter } from 'next/navigation'
import toast from 'react-hot-toast'
import Link from 'next/link'
import { Button } from '@/components/ui/button'
import {
  AnomaliesPreviewTable,
  type AnomalyPreviewRow,
} from '@/components/reviews/AnomaliesPreviewTable'

export default function AnomalyReviewsPage() {
  const { user, isAuthenticated, token } = useAuthStore()
  const router = useRouter()
  const [rows, setRows] = useState<AnomalyPreviewRow[]>([])
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    if (!isAuthenticated || !user) {
      router.push('/admin/login')
      return
    }
    if (!['org_admin', 'hr', 'dept_manager'].includes(user.role)) {
      router.push('/admin/login')
      return
    }
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, user?.role])

  const load = async () => {
    try {
      setIsLoading(true)
      const base = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(`${base}/api/v1/reports/anomalies?limit=50`, {
        headers,
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to load anomalies')
      }
      setRows(await resp.json())
    } catch (e: any) {
      toast.error(e.message || 'Failed to load anomalies')
      setRows([])
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold mb-2">Anomaly Reviews</h1>
          <p className="text-muted-foreground">
            Review and resolve anomalous attendance events.
          </p>
        </div>
        <Link href="/admin/org">
          <Button variant="outline">Back to dashboard</Button>
        </Link>
      </div>

      <div className="bg-card border border-border rounded-lg p-4 shadow-sm">
        {isLoading ? (
          <div className="h-40 bg-muted rounded" />
        ) : (
          <AnomaliesPreviewTable rows={rows} emptyText="No anomalies to review." />
        )}
      </div>
    </div>
  )
}

