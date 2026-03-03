'use client'

import { useEffect, useMemo, useState } from 'react'
import { useAuthStore } from '@/store/useStore'
import { useRouter } from 'next/navigation'
import toast from 'react-hot-toast'
import Link from 'next/link'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { format } from 'date-fns'

type AnomalyDetail = {
  id: string
  punch_time: string
  status: string
  verification_method: string
  anomaly_reason?: string | null
  user_id: string
  employee_id: string
  first_name: string
  last_name: string
  kiosk_code?: string | null
}

export default function AnomalyDetailPage({ params }: { params: { id: string } }) {
  const { user, isAuthenticated, token } = useAuthStore()
  const router = useRouter()
  const [detail, setDetail] = useState<AnomalyDetail | null>(null)
  const [note, setNote] = useState('')
  const [isLoading, setIsLoading] = useState(true)
  const [isResolving, setIsResolving] = useState(false)

  const base = useMemo(
    () => process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080',
    []
  )

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
  }, [isAuthenticated, user?.role, params.id])

  const load = async () => {
    try {
      setIsLoading(true)
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(`${base}/api/v1/reports/anomalies/${params.id}`, {
        headers,
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to load anomaly')
      }
      setDetail(await resp.json())
    } catch (e: any) {
      toast.error(e.message || 'Failed to load anomaly')
      setDetail(null)
    } finally {
      setIsLoading(false)
    }
  }

  const resolve = async () => {
    try {
      setIsResolving(true)
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(`${base}/api/v1/reports/anomalies/${params.id}/resolve`, {
        method: 'PATCH',
        headers,
        body: JSON.stringify({ note }),
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to resolve anomaly')
      }
      toast.success('Resolved')
      router.push('/admin/org/reviews/anomalies')
    } catch (e: any) {
      toast.error(e.message || 'Failed to resolve')
    } finally {
      setIsResolving(false)
    }
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold mb-2">Anomaly Review</h1>
          <p className="text-muted-foreground">Investigate and resolve an anomalous event.</p>
        </div>
        <div className="flex gap-2">
          <Link href="/admin/org/reviews/anomalies">
            <Button variant="outline">Back</Button>
          </Link>
        </div>
      </div>

      {isLoading ? (
        <div className="h-40 bg-muted rounded" />
      ) : !detail ? (
        <div className="text-sm text-muted-foreground">Not found.</div>
      ) : (
        <div className="bg-card border border-border rounded-lg p-4 shadow-sm space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <div className="text-xs text-muted-foreground">Employee</div>
              <div className="font-medium">
                {detail.first_name} {detail.last_name}
              </div>
              <div className="text-sm text-muted-foreground">{detail.employee_id}</div>
            </div>
            <div>
              <div className="text-xs text-muted-foreground">Time</div>
              <div className="font-medium">
                {format(new Date(detail.punch_time), 'MMM d, yyyy h:mm a')}
              </div>
              <div className="text-sm text-muted-foreground">
                {detail.kiosk_code ? `Kiosk: ${detail.kiosk_code}` : 'Kiosk: —'}
              </div>
            </div>
            <div>
              <div className="text-xs text-muted-foreground">Status</div>
              <div className="font-medium">{detail.status}</div>
            </div>
            <div>
              <div className="text-xs text-muted-foreground">Method</div>
              <div className="font-medium">{detail.verification_method}</div>
            </div>
          </div>

          <div>
            <div className="text-xs text-muted-foreground">Anomaly reason</div>
            <div className="text-sm">{detail.anomaly_reason || '—'}</div>
          </div>

          <div className="space-y-2">
            <div className="text-sm font-medium">Resolution note (optional)</div>
            <Input
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="e.g., Confirmed late arrival, approved by manager"
            />
            <div className="flex justify-end">
              <Button onClick={resolve} disabled={isResolving}>
                {isResolving ? 'Resolving…' : 'Mark as resolved'}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

