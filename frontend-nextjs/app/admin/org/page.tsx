'use client'

import { useEffect, useMemo, useState } from 'react'
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
  rangeCheckIns: number
  rangeAnomalies: number
  rangeAttendanceLogs: number
  rangeStart?: string
  rangeEnd?: string
}

export default function OrgAdminDashboardPage() {
  const { user, isAuthenticated, token } = useAuthStore()
  const router = useRouter()
  const [metrics, setMetrics] = useState<OrgMetrics | null>(null)
  const [chart7d, setChart7d] = useState<ChartPoint7d[]>([])
  const [recentAnomalies, setRecentAnomalies] = useState<AnomalyPreviewRow[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [rangeStart, setRangeStart] = useState(
    new Date(Date.now() - 29 * 86400000).toISOString().slice(0, 10)
  )
  const [rangeEnd, setRangeEnd] = useState(new Date().toISOString().slice(0, 10))
  const [autoRefresh, setAutoRefresh] = useState(true)

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

  useEffect(() => {
    if (!autoRefresh) return
    const interval = setInterval(() => {
      fetchAll()
    }, 60000)
    return () => clearInterval(interval)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [autoRefresh, rangeStart, rangeEnd])

  useEffect(() => {
    fetchAll()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rangeStart, rangeEnd])

  const fetchAll = async () => {
    try {
      setIsLoading(true)

      const base = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`
      const params = new URLSearchParams()
      if (rangeStart) params.set('start', rangeStart)
      if (rangeEnd) params.set('end', rangeEnd)
      const metricsUrl = `${base}/api/v1/reports/org-metrics?${params.toString()}`

      const [mResp, cResp, aResp] = await Promise.all([
        fetch(metricsUrl, { headers }),
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
        rangeCheckIns: 0,
        rangeAnomalies: 0,
        rangeAttendanceLogs: 0,
      })
      setChart7d([])
      setRecentAnomalies([])
    } finally {
      setIsLoading(false)
    }
  }

  if (!isAuthenticated || !user) return null

  const rangeLabel = useMemo(() => {
    if (!metrics?.rangeStart || !metrics?.rangeEnd) return 'Selected range'
    return `${metrics.rangeStart} → ${metrics.rangeEnd}`
  }, [metrics?.rangeStart, metrics?.rangeEnd])

  const exportMetrics = async () => {
    try {
      const base = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      const params = new URLSearchParams()
      if (rangeStart) params.set('start', rangeStart)
      if (rangeEnd) params.set('end', rangeEnd)
      const resp = await fetch(`${base}/api/v1/reports/org-metrics/export?${params.toString()}`, {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Export failed')
      }
      const blob = await resp.blob()
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `org-metrics-${rangeStart}-${rangeEnd}.csv`
      link.click()
      window.URL.revokeObjectURL(url)
    } catch (e: any) {
      toast.error(e.message || 'Export failed')
    }
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div>
        <h1 className="text-3xl font-bold mb-2">Organization Dashboard</h1>
        <p className="text-muted-foreground">
          Welcome, {user.firstName} {user.lastName}
        </p>
      </div>

      <div className="border rounded-lg bg-card p-4 flex flex-col lg:flex-row lg:items-center gap-3 lg:justify-between">
        <div className="flex flex-wrap gap-3 items-center">
          <div className="text-sm font-medium text-muted-foreground">Date range</div>
          <input
            type="date"
            value={rangeStart}
            onChange={(e) => setRangeStart(e.target.value)}
            className="h-10 rounded-md border border-input bg-background px-3 text-sm"
          />
          <span className="text-sm text-muted-foreground">to</span>
          <input
            type="date"
            value={rangeEnd}
            onChange={(e) => setRangeEnd(e.target.value)}
            className="h-10 rounded-md border border-input bg-background px-3 text-sm"
          />
          <label className="flex items-center gap-2 text-sm text-muted-foreground">
            <input
              type="checkbox"
              checked={autoRefresh}
              onChange={(e) => setAutoRefresh(e.target.checked)}
            />
            Auto-refresh (60s)
          </label>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={fetchAll}>
            Refresh
          </Button>
          <Button variant="outline" size="sm" onClick={exportMetrics}>
            Export CSV
          </Button>
        </div>
      </div>

      <DataCardGrid>
        <DataCard
          title="Total Employees"
          value={isLoading || !metrics ? '—' : metrics.totalEmployees.toLocaleString()}
          subtitle="Active employees in this tenant"
        />
        <DataCard
          title="Check-Ins (range)"
          value={isLoading || !metrics ? '—' : metrics.rangeCheckIns.toLocaleString()}
          subtitle={rangeLabel}
        />
        <DataCard
          title="Anomalies (range)"
          value={isLoading || !metrics ? '—' : metrics.rangeAnomalies.toLocaleString()}
          subtitle="Flagged check-ins in range"
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
        <DataCard
          title="Attendance Logs (range)"
          value={isLoading || !metrics ? '—' : metrics.rangeAttendanceLogs.toLocaleString()}
          subtitle={rangeLabel}
        />
      </DataCardGrid>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <div className="bg-card border border-border rounded-lg p-4 shadow-sm">
          <div className="text-sm font-medium text-muted-foreground mb-2">Operations Snapshot</div>
          <div className="space-y-1 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Employees</span>
              <span className="font-medium">{isLoading || !metrics ? '—' : metrics.totalEmployees}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Check-ins today</span>
              <span className="font-medium">{isLoading || !metrics ? '—' : metrics.todayCheckIns}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Offline kiosks</span>
              <span className="font-medium">{isLoading || !metrics ? '—' : metrics.offlineKiosks}</span>
            </div>
          </div>
        </div>
        <div className="bg-card border border-border rounded-lg p-4 shadow-sm lg:col-span-2">
          <div className="text-sm font-medium text-muted-foreground mb-3">Quick Actions</div>
          <div className="flex flex-wrap gap-2">
            <Link href="/admin/org/users">
              <Button variant="outline" size="sm">Manage employees</Button>
            </Link>
            <Link href="/admin/org/kiosks">
              <Button variant="outline" size="sm">Review kiosk health</Button>
            </Link>
            <Link href="/admin/org/reports/attendance">
              <Button variant="outline" size="sm">Run attendance report</Button>
            </Link>
            <Link href="/admin/org/reviews/anomalies">
              <Button variant="outline" size="sm">Resolve anomalies</Button>
            </Link>
          </div>
        </div>
      </div>

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
