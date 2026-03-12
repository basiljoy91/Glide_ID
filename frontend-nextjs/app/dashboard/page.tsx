'use client'

import { useEffect, useState } from 'react'
import { DataCard, DataCardGrid } from '@/components/data/DataCard'
import { useAuthStore } from '@/store/useStore'
import { useRouter } from 'next/navigation'
import toast from 'react-hot-toast'
import { format } from 'date-fns'

interface AttendanceLogSimple {
  id: string
  status: string
  punchTime: string
}

interface UpcomingShift {
  title: string
  startTime: string
  endTime: string
}

interface LeaveBalance {
  annual: number
  sick: number
}

interface EmployeeDashboardResponse {
  todayCheckIns: AttendanceLogSimple[]
  recentHistory: AttendanceLogSimple[]
  upcomingShift?: UpcomingShift
  leaveBalance: LeaveBalance
}

export default function DashboardPage() {
  const { user, token, isAuthenticated } = useAuthStore()
  const router = useRouter()
  const [data, setData] = useState<EmployeeDashboardResponse | null>(null)
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
    
    // Valid employee - fetch personal dashboard metrics
    void fetchDashboard()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, user?.role, router])

  const fetchDashboard = async () => {
    try {
      setIsLoading(true)
      const base = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(`${base}/api/v1/employee/dashboard`, { headers })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to load employee dashboard')
      }
      setData(await resp.json())
    } catch (e: any) {
      toast.error(e.message || 'Failed to load employee dashboard')
      setData(null)
    } finally {
      setIsLoading(false)
    }
  }

  // Helper to format date string to local time
  const formatTime = (ts: string) => {
    if (!ts) return ''
    try {
      return format(new Date(ts), 'h:mm a')
    } catch {
      return ts
    }
  }

  const formatDateTime = (ts: string) => {
    if (!ts) return ''
    try {
      return format(new Date(ts), 'MMM d, yyyy h:mm a')
    } catch {
      return ts
    }
  }

  return (
    <div className="container mx-auto p-6 space-y-8">
      <div>
        <h1 className="text-3xl font-bold mb-2">My Dashboard</h1>
        {user && (
          <p className="text-muted-foreground">
            Welcome back, {user.firstName} {user.lastName}
          </p>
        )}
      </div>

      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        {/* Today's Check-ins */}
        <div className="bg-card border rounded-lg p-6 shadow-sm col-span-1 md:col-span-2 lg:col-span-1 border-primary/20 bg-primary/5">
          <div className="flex items-center gap-3 mb-4">
            <span className="text-2xl">🕒</span>
            <h2 className="text-xl font-semibold">Today's Activity</h2>
          </div>
          
          {isLoading ? (
            <div className="flex justify-center items-center h-24 text-muted-foreground">Loading...</div>
          ) : data?.todayCheckIns && data.todayCheckIns.length > 0 ? (
            <div className="space-y-3">
              {data.todayCheckIns.map(log => (
                <div key={log.id} className="flex justify-between items-center p-3 rounded-md bg-background border">
                  <div className="flex items-center gap-2">
                    <div className={`w-2 h-2 rounded-full ${log.status === 'check_in' ? 'bg-green-500' : 'bg-orange-500'}`} />
                    <span className="font-medium capitalize">{log.status.replace('_', ' ')}</span>
                  </div>
                  <span className="text-muted-foreground">{formatTime(log.punchTime)}</span>
                </div>
              ))}
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center p-6 border border-dashed rounded-lg bg-background text-muted-foreground">
              <p>No check-ins yet today.</p>
              <p className="text-sm mt-1">Use a kiosk to check in for your shift.</p>
            </div>
          )}
        </div>

        {/* Upcoming Shift */}
        <div className="bg-card border rounded-lg p-6 shadow-sm">
          <div className="flex items-center gap-3 mb-4">
            <span className="text-2xl">📅</span>
            <h2 className="text-xl font-semibold">Upcoming Shift</h2>
          </div>
          
          {isLoading ? (
             <div className="flex justify-center items-center h-24 text-muted-foreground">Loading...</div>
          ) : data?.upcomingShift ? (
            <div className="flex flex-col p-4 rounded-md border bg-muted/30 h-[calc(100%-3rem)]">
              <span className="font-semibold text-lg">{data.upcomingShift.title}</span>
              <span className="text-muted-foreground mt-2">
                {data.upcomingShift.startTime} - {data.upcomingShift.endTime}
              </span>
              <div className="mt-auto pt-4 flex gap-2">
                <span className="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">
                  Scheduled
                </span>
              </div>
            </div>
          ) : (
             <div className="flex items-center justify-center h-24 text-muted-foreground">No upcoming shifts.</div>
          )}
        </div>

        {/* Leave Balance */}
        <div className="bg-card border rounded-lg p-6 shadow-sm">
          <div className="flex items-center gap-3 mb-4">
            <span className="text-2xl">⛱️</span>
            <h2 className="text-xl font-semibold">Leave Balance</h2>
          </div>
          
          {isLoading ? (
             <div className="flex justify-center items-center h-24 text-muted-foreground">Loading...</div>
          ) : data ? (
            <div className="grid grid-cols-2 gap-4 h-[calc(100%-3rem)]">
              <div className="flex flex-col items-center justify-center p-4 rounded-md border bg-background">
                <span className="text-3xl font-bold text-primary">{data.leaveBalance.annual}</span>
                <span className="text-sm text-muted-foreground mt-1 text-center">Annual (days)</span>
              </div>
              <div className="flex flex-col items-center justify-center p-4 rounded-md border bg-background">
                <span className="text-3xl font-bold text-orange-500">{data.leaveBalance.sick}</span>
                <span className="text-sm text-muted-foreground mt-1 text-center">Sick (days)</span>
              </div>
            </div>
          ) : null}
        </div>
      </div>

      {/* Recent History */}
      <div className="bg-card border rounded-lg shadow-sm overflow-hidden">
        <div className="p-6 border-b">
          <h2 className="text-xl font-semibold">Recent Attendance History</h2>
          <p className="text-sm text-muted-foreground">Your last 10 punches</p>
        </div>
        
        <div className="overflow-x-auto">
          {isLoading ? (
             <div className="flex justify-center items-center h-32 text-muted-foreground">Loading history...</div>
          ) : data?.recentHistory && data.recentHistory.length > 0 ? (
            <table className="w-full text-sm text-left">
              <thead className="text-xs text-muted-foreground uppercase bg-muted/40">
                <tr>
                  <th scope="col" className="px-6 py-3 rounded-tl-lg">Type</th>
                  <th scope="col" className="px-6 py-3 rounded-tr-lg">Timestamp</th>
                </tr>
              </thead>
              <tbody>
                {data.recentHistory.map((log) => (
                  <tr key={log.id} className="border-b last:border-0 hover:bg-muted/20">
                    <td className="px-6 py-4 font-medium">
                      <div className="flex items-center gap-2">
                        <div className={`w-2 h-2 rounded-full ${log.status === 'check_in' ? 'bg-green-500' : 'bg-orange-500'}`} />
                        <span className="capitalize">{log.status.replace('_', ' ')}</span>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      {formatDateTime(log.punchTime)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
              <span className="text-3xl mb-2">📋</span>
              <p>No recent attendance history found.</p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

