'use client'

import { useEffect, useState } from 'react'
import { useAuthStore } from '@/store/useStore'
import { useRouter } from 'next/navigation'
import { DataCard, DataCardGrid } from '@/components/data/DataCard'
import {
  Activity,
  Building2,
  CheckCircle2,
  CreditCard,
  DollarSign,
  Settings,
  TrendingUp,
  Users,
} from 'lucide-react'
import { SkeletonCard } from '@/components/ui/skeleton-card'
import toast from 'react-hot-toast'

interface GlobalMetrics {
  totalOrganizations: number
  totalUsers: number
  totalCheckIns: number
  monthlyRevenue: number
  activeOrganizations: number
  growthRate: number
}

export default function SuperAdminDashboard() {
  const { user, isAuthenticated } = useAuthStore()
  const router = useRouter()
  const [metrics, setMetrics] = useState<GlobalMetrics | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    if (!isAuthenticated || user?.role !== 'super_admin') {
      router.push('/admin/login')
      return
    }
    fetchMetrics()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, user?.role])

  const fetchMetrics = async () => {
    try {
      setIsLoading(true)
      const token = useAuthStore.getState().token
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/admin/super/metrics`,
        {
          headers: token ? { Authorization: `Bearer ${token}` } : {},
        }
      )

      if (!response.ok) {
        const err = await response.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to fetch metrics')
      }

      const data = await response.json()
      // Accept either snake_case or camelCase from backend
      setMetrics({
        totalOrganizations: data.totalOrganizations ?? data.total_organizations ?? 0,
        totalUsers: data.totalUsers ?? data.total_users ?? 0,
        totalCheckIns: data.totalCheckIns ?? data.total_check_ins ?? 0,
        monthlyRevenue: data.monthlyRevenue ?? data.monthly_revenue ?? 0,
        activeOrganizations: data.activeOrganizations ?? data.active_organizations ?? 0,
        growthRate: data.growthRate ?? data.growth_rate ?? 0,
      })
    } catch (error: any) {
      toast.error(error.message || 'Failed to load metrics')
      // fallback mock data
      setMetrics({
        totalOrganizations: 0,
        totalUsers: 0,
        totalCheckIns: 0,
        monthlyRevenue: 0,
        activeOrganizations: 0,
        growthRate: 0,
      })
    } finally {
      setIsLoading(false)
    }
  }

  if (!isAuthenticated || user?.role !== 'super_admin') return null

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="mb-8">
        <h1 className="text-3xl font-bold mb-2">Super Admin Dashboard</h1>
        <p className="text-muted-foreground">Global platform overview and management</p>
      </div>

      {isLoading ? (
        <DataCardGrid>
          <SkeletonCard />
          <SkeletonCard />
          <SkeletonCard />
          <SkeletonCard />
        </DataCardGrid>
      ) : metrics ? (
        <>
          <DataCardGrid>
            <DataCard
              title="Total Organizations"
              value={metrics.totalOrganizations.toLocaleString()}
              icon={<Building2 className="h-6 w-6" />}
              subtitle={`${metrics.activeOrganizations} active`}
            />
            <DataCard
              title="Total Users"
              value={metrics.totalUsers.toLocaleString()}
              icon={<Users className="h-6 w-6" />}
              subtitle="Across all organizations"
            />
            <DataCard
              title="Total Check-Ins"
              value={metrics.totalCheckIns.toLocaleString()}
              icon={<CheckCircle2 className="h-6 w-6" />}
              subtitle="All-time records"
            />
            <DataCard
              title="Monthly Revenue"
              value={`$${metrics.monthlyRevenue.toLocaleString()}`}
              icon={<DollarSign className="h-6 w-6" />}
              subtitle={`${metrics.growthRate > 0 ? '+' : ''}${metrics.growthRate}% growth`}
            />
          </DataCardGrid>

          <div className="mt-8 grid md:grid-cols-2 gap-6">
            <div className="border rounded-lg p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="font-semibold">Platform Growth</h3>
                <TrendingUp className="h-5 w-5 text-primary" />
              </div>
              <div className="space-y-2">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Active Rate</span>
                  <span className="font-medium">
                    {metrics.totalOrganizations > 0
                      ? Math.round((metrics.activeOrganizations / metrics.totalOrganizations) * 100)
                      : 0}
                    %
                  </span>
                </div>
              </div>
            </div>

            <div className="border rounded-lg p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="font-semibold">System Activity</h3>
                <Activity className="h-5 w-5 text-primary" />
              </div>
              <div className="space-y-2">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Avg. Users per Org</span>
                  <span className="font-medium">
                    {metrics.totalOrganizations > 0
                      ? Math.floor(metrics.totalUsers / metrics.totalOrganizations)
                      : 0}
                  </span>
                </div>
              </div>
            </div>
          </div>

          <div className="mt-8 border rounded-lg p-6">
            <h3 className="font-semibold mb-4">Quick Actions</h3>
            <div className="grid md:grid-cols-3 gap-4">
              <a
                href="/admin/super/organizations"
                className="p-4 border rounded-lg hover:bg-muted transition-colors"
              >
                <Building2 className="h-5 w-5 mb-2 text-primary" />
                <p className="font-medium">Manage Organizations</p>
                <p className="text-sm text-muted-foreground">View, upgrade, or deactivate</p>
              </a>
              <a
                href="/admin/super/billing"
                className="p-4 border rounded-lg hover:bg-muted transition-colors"
              >
                <CreditCard className="h-5 w-5 mb-2 text-primary" />
                <p className="font-medium">Billing Overview</p>
                <p className="text-sm text-muted-foreground">Subscriptions and revenue</p>
              </a>
              <a
                href="/admin/super/settings"
                className="p-4 border rounded-lg hover:bg-muted transition-colors"
              >
                <Settings className="h-5 w-5 mb-2 text-primary" />
                <p className="font-medium">Platform Settings</p>
                <p className="text-sm text-muted-foreground">Global configuration</p>
              </a>
            </div>
          </div>
        </>
      ) : null}
    </div>
  )
}


