'use client'

import { useEffect, useState } from 'react'
import { DataCard, DataCardGrid } from '@/components/data/DataCard'
import { useAuthStore } from '@/store/useStore'
import { useRouter } from 'next/navigation'
import toast from 'react-hot-toast'

interface OrgMetrics {
  totalEmployees: number
  todayCheckIns: number
  anomaliesPending: number
  activeKiosks: number
  healthyKiosks: number
  offlineKiosks: number
}

export default function DashboardPage() {
  const { user, token, isAuthenticated } = useAuthStore()
  const router = useRouter()
  const [metrics, setMetrics] = useState<OrgMetrics | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    if (!isAuthenticated || !user) {
      router.push('/admin/login')
      return
    }
    // Route super_admin to their dashboard
    if (user.role === 'super_admin') {
      router.replace('/admin/super')
      return
    }
    // Route org roles to org dashboard
    if (['org_admin', 'hr', 'dept_manager'].includes(user.role)) {
      router.replace('/admin/org')
      return
    }
    // For regular employees, show a slim per-user metrics view
    void fetchMetrics()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, user?.role])

  const fetchMetrics = async () => {
    try {
      setIsLoading(true)
      const base = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(`${base}/api/v1/reports/org-metrics`, { headers })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to load metrics')
      }
      setMetrics(await resp.json())
    } catch (e: any) {
      toast.error(e.message || 'Failed to load metrics')
      setMetrics(null)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="container mx-auto p-6">
      <h1 className="text-3xl font-bold mb-6">Dashboard</h1>
      
      {user && (
        <div className="mb-6">
          <p className="text-muted-foreground">
            Welcome, {user.firstName} {user.lastName}
          </p>
          {user.role === 'employee' && (
            <p className="text-xs text-muted-foreground">
              Organization-wide metrics below. Personal attendance history coming soon.
            </p>
          )}
        </div>
      )}

      <DataCardGrid>
        <DataCard
          title="Total Employees"
          value={
            isLoading || !metrics ? '—' : metrics.totalEmployees.toLocaleString()
          }
          icon="👥"
          subtitle="Active employees in your organization"
        />
        <DataCard
          title="Today's Check-Ins"
          value={
            isLoading || !metrics ? '—' : metrics.todayCheckIns.toLocaleString()
          }
          icon="✅"
          subtitle="All check-ins recorded today"
        />
        <DataCard
          title="Pending Reviews"
          value={
            isLoading || !metrics ? '—' : metrics.anomaliesPending.toLocaleString()
          }
          icon="⚠️"
          subtitle="Anomalies flagged for HR review"
        />
        <DataCard
          title="Kiosks Active"
          value={
            isLoading || !metrics ? '—' : metrics.activeKiosks.toLocaleString()
          }
          icon="🖥️"
          subtitle={
            isLoading || !metrics
              ? 'From kiosk heartbeat data'
              : `${metrics.healthyKiosks} healthy, ${metrics.offlineKiosks} offline`
          }
        />
      </DataCardGrid>
    </div>
  )
}

