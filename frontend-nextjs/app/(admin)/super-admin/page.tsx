'use client'

import { useEffect, useState } from 'react'
import { useAuthStore } from '@/store/useStore'
import { useRouter } from 'next/navigation'
import { DataCard, DataCardGrid } from '@/components/data/DataCard'
import { Building2, Users, CheckCircle2, DollarSign, TrendingUp, Activity, Settings } from 'lucide-react'
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
    // Check authentication
    if (!isAuthenticated || user?.role !== 'super_admin') {
      router.push('/admin/login')
      return
    }

    // Fetch global metrics
    fetchMetrics()
  }, [isAuthenticated, user, router])

  const fetchMetrics = async () => {
    try {
      setIsLoading(true)
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/admin/super/metrics`,
        {
          headers: {
            Authorization: `Bearer ${useAuthStore.getState().token}`,
          },
        }
      )

      if (!response.ok) {
        throw new Error('Failed to fetch metrics')
      }

      const data = await response.json()
      setMetrics(data)
    } catch (error: any) {
      toast.error(error.message || 'Failed to load metrics')
      // Use mock data for now
      setMetrics({
        totalOrganizations: 125,
        totalUsers: 12500,
        totalCheckIns: 450000,
        monthlyRevenue: 125000,
        activeOrganizations: 98,
        growthRate: 12.5,
      })
    } finally {
      setIsLoading(false)
    }
  }

  if (!isAuthenticated || user?.role !== 'super_admin') {
    return null
  }

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="mb-8">
        <h1 className="text-3xl font-bold mb-2">Super Admin Dashboard</h1>
        <p className="text-muted-foreground">
          Global platform overview and management
        </p>
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

          {/* Additional Metrics */}
          <div className="mt-8 grid md:grid-cols-2 gap-6">
            <div className="border rounded-lg p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="font-semibold">Platform Growth</h3>
                <TrendingUp className="h-5 w-5 text-primary" />
              </div>
              <div className="space-y-2">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">New Organizations (30d)</span>
                  <span className="font-medium">+{Math.floor(metrics.totalOrganizations * 0.1)}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Active Rate</span>
                  <span className="font-medium">
                    {Math.round((metrics.activeOrganizations / metrics.totalOrganizations) * 100)}%
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
                  <span className="text-muted-foreground">Check-Ins Today</span>
                  <span className="font-medium">
                    {Math.floor(metrics.totalCheckIns / 365).toLocaleString()}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Avg. per Organization</span>
                  <span className="font-medium">
                    {Math.floor(metrics.totalUsers / metrics.totalOrganizations)}
                  </span>
                </div>
              </div>
            </div>
          </div>

          {/* Quick Actions */}
          <div className="mt-8 border rounded-lg p-6">
            <h3 className="font-semibold mb-4">Quick Actions</h3>
            <div className="grid md:grid-cols-3 gap-4">
              <a
                href="/admin/super/organizations"
                className="p-4 border rounded-lg hover:bg-muted transition-colors"
              >
                <Building2 className="h-5 w-5 mb-2 text-primary" />
                <p className="font-medium">Manage Organizations</p>
                <p className="text-sm text-muted-foreground">
                  View, upgrade, or deactivate organizations
                </p>
              </a>
              <a
                href="/admin/super/billing"
                className="p-4 border rounded-lg hover:bg-muted transition-colors"
              >
                <DollarSign className="h-5 w-5 mb-2 text-primary" />
                <p className="font-medium">Billing Overview</p>
                <p className="text-sm text-muted-foreground">
                  Review subscriptions and revenue
                </p>
              </a>
              <a
                href="/admin/super/settings"
                className="p-4 border rounded-lg hover:bg-muted transition-colors"
              >
                <Settings className="h-5 w-5 mb-2 text-primary" />
                <p className="font-medium">Platform Settings</p>
                <p className="text-sm text-muted-foreground">
                  Configure global platform settings
                </p>
              </a>
            </div>
          </div>
        </>
      ) : null}
    </div>
  )
}

