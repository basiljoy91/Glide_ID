'use client'

import { useEffect, useState } from 'react'
import { useAuthStore } from '@/store/useStore'
import { useRouter } from 'next/navigation'
import { DataCard, DataCardGrid } from '@/components/data/DataCard'
import toast from 'react-hot-toast'
import Link from 'next/link'
import { Button } from '@/components/ui/button'
import { MiniBarChart7d, type ChartPoint7d } from '@/components/reports/MiniBarChart7d'
import {
  AnomaliesPreviewTable,
  type AnomalyPreviewRow,
} from '@/components/reviews/AnomaliesPreviewTable'

interface OrgMetrics {
  totalEmployees: number
  todayCheckIns: number
  anomaliesPending: number
  activeKiosks: number
  healthyKiosks: number
  offlineKiosks: number
  totalAttendanceLogs: number
}

export default function OrgAdminDashboardPage() {
  const { user, isAuthenticated, token } = useAuthStore()
  const router = useRouter()
  const [metrics, setMetrics] = useState<OrgMetrics | null>(null)
  const [chart7d, setChart7d] = useState<ChartPoint7d[]>([])
  const [recentAnomalies, setRecentAnomalies] = useState<AnomalyPreviewRow[]>([])
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
    fetchAll()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, user?.role])

  const fetchAll = async () => {
    try {
      setIsLoading(true)

      const base = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`

      const [mResp, cResp, aResp] = await Promise.all([
        fetch(`${base}/api/v1/reports/org-metrics`, { headers }),
        fetch(`${base}/api/v1/reports/checkins-7d`, { headers }),
        fetch(`${base}/api/v1/reports/anomalies?limit=5`, { headers }),
      ])

      if (!mResp.ok) {
        const err = await mResp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to load metrics')
      }
      setMetrics(await mResp.json())

      if (cResp.ok) {
        setChart7d(await cResp.json())
      } else {
        setChart7d([])
      }

      if (aResp.ok) {
        setRecentAnomalies(await aResp.json())
      } else {
        setRecentAnomalies([])
      }
    } catch (e: any) {
      toast.error(e.message || 'Failed to load metrics')
      setMetrics({
        totalEmployees: 0,
        todayCheckIns: 0,
        anomaliesPending: 0,
        activeKiosks: 0,
        healthyKiosks: 0,
        offlineKiosks: 0,
        totalAttendanceLogs: 0,
      })
      setChart7d([])
      setRecentAnomalies([])
    } finally {
      setIsLoading(false)
    }
  }

  if (!isAuthenticated || !user) return null

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div>
        <h1 className="text-3xl font-bold mb-2">Organization Dashboard</h1>
        <p className="text-muted-foreground">
          Welcome, {user.firstName} {user.lastName}
        </p>
      </div>

      <DataCardGrid>
        <DataCard
          title="Total Employees"
          value={isLoading || !metrics ? '—' : metrics.totalEmployees.toLocaleString()}
          subtitle="Active employees in this tenant"
        />
        <DataCard
          title="Today's Check-Ins"
          value={isLoading || !metrics ? '—' : metrics.todayCheckIns.toLocaleString()}
          subtitle="All check-ins recorded today"
        />
        <DataCard
          title="Pending Reviews"
          value={isLoading || !metrics ? '—' : metrics.anomaliesPending.toLocaleString()}
          subtitle="Anomalies flagged for HR review"
        />
        <DataCard
          title="Active Kiosks"
          value={isLoading || !metrics ? '—' : metrics.activeKiosks.toLocaleString()}
          subtitle={
            isLoading || !metrics
              ? 'From kiosk heartbeat data'
              : `${metrics.healthyKiosks} healthy, ${metrics.offlineKiosks} offline`
          }
        />
      </DataCardGrid>

      <div className="grid grid-cols-1 lg:grid-cols-5 gap-4">
        <div className="bg-card border border-border rounded-lg p-4 shadow-sm lg:col-span-2">
          <div className="flex items-center justify-between mb-3">
            <div>
              <div className="text-sm font-medium text-muted-foreground">Check-ins (last 7 days)</div>
              <div className="text-xs text-muted-foreground">Daily total logs (check-ins + check-outs)</div>
            </div>
            <Link href="/admin/org/reports/attendance">
              <Button size="sm" variant="outline">
                Open report
              </Button>
            </Link>
          </div>
          {isLoading ? (
            <div className="h-28 bg-muted rounded" />
          ) : (
            <MiniBarChart7d points={chart7d} />
          )}
        </div>

        <div className="bg-card border border-border rounded-lg p-4 shadow-sm lg:col-span-3">
          <div className="flex items-center justify-between mb-3">
            <div>
              <div className="text-sm font-medium text-muted-foreground">Most recent anomalies</div>
              <div className="text-xs text-muted-foreground">Quick access to HR review workflow</div>
            </div>
            <Link href="/admin/org/reviews/anomalies">
              <Button size="sm" variant="outline">
                View all
              </Button>
            </Link>
          </div>
          {isLoading ? (
            <div className="h-28 bg-muted rounded" />
          ) : (
            <AnomaliesPreviewTable rows={recentAnomalies} compact />
          )}
        </div>
      </div>
    </div>
  )
}

