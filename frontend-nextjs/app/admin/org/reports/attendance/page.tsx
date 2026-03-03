'use client'

import { useEffect, useMemo, useState } from 'react'
import { useAuthStore } from '@/store/useStore'
import { useRouter } from 'next/navigation'
import toast from 'react-hot-toast'
import Link from 'next/link'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { MiniBarChart7d, type ChartPoint7d } from '@/components/reports/MiniBarChart7d'

type ReportDay = {
  date: string
  check_ins: number
  check_outs: number
  anomalies: number
}

type ReportResponse = {
  start_date: string
  end_date: string
  days: ReportDay[]
  totals: {
    check_ins: number
    check_outs: number
    anomalies: number
    logs: number
  }
}

export default function AttendanceReportPage() {
  const { user, isAuthenticated, token } = useAuthStore()
  const router = useRouter()
  const base = useMemo(() => process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080', [])

  const today = new Date()
  const defaultEnd = today.toISOString().slice(0, 10)
  const defaultStart = new Date(today.getTime() - 6 * 86400000).toISOString().slice(0, 10)

  const [startDate, setStartDate] = useState(defaultStart)
  const [endDate, setEndDate] = useState(defaultEnd)
  const [report, setReport] = useState<ReportResponse | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [isExporting, setIsExporting] = useState(false)

  useEffect(() => {
    if (!isAuthenticated || !user) {
      router.push('/admin/login')
      return
    }
    if (!['org_admin', 'hr', 'dept_manager'].includes(user.role)) {
      router.push('/admin/login')
      return
    }
    void load(defaultStart, defaultEnd)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, user?.role])

  const load = async (s = startDate, e = endDate) => {
    if (!s || !e || e < s) {
      toast.error('Please choose a valid date range')
      return
    }
    try {
      setIsLoading(true)
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(
        `${base}/api/v1/reports/attendance?start_date=${encodeURIComponent(s)}&end_date=${encodeURIComponent(e)}`,
        { headers }
      )
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to load report')
      }
      setReport(await resp.json())
    } catch (e: any) {
      toast.error(e.message || 'Failed to load report')
      setReport(null)
    } finally {
      setIsLoading(false)
    }
  }

  const chartPoints: ChartPoint7d[] =
    report?.days?.map((d) => ({ date: d.date, count: d.check_ins + d.check_outs })) ?? []

  const applyQuickRange = (days: number) => {
    const end = new Date()
    const start = new Date(end.getTime() - (days - 1) * 86400000)
    const s = start.toISOString().slice(0, 10)
    const e = end.toISOString().slice(0, 10)
    setStartDate(s)
    setEndDate(e)
    void load(s, e)
  }

  const exportCsv = async () => {
    if (!startDate || !endDate || endDate < startDate) {
      toast.error('Please choose a valid date range')
      return
    }
    try {
      setIsExporting(true)
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(
        `${base}/api/v1/reports/export?start_date=${encodeURIComponent(startDate)}&end_date=${encodeURIComponent(endDate)}`,
        { method: 'POST', headers }
      )
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to export CSV')
      }
      const blob = await resp.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `attendance-report-${startDate}-to-${endDate}.csv`
      document.body.appendChild(a)
      a.click()
      a.remove()
      URL.revokeObjectURL(url)
      toast.success('CSV downloaded')
    } catch (e: any) {
      toast.error(e.message || 'Failed to export CSV')
    } finally {
      setIsExporting(false)
    }
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold mb-2">Attendance Report</h1>
          <p className="text-muted-foreground">Daily attendance totals for a date range.</p>
        </div>
        <Link href="/admin/org">
          <Button variant="outline">Back to dashboard</Button>
        </Link>
      </div>

      <div className="bg-card border border-border rounded-lg p-4 shadow-sm">
        <div className="flex flex-col md:flex-row md:items-end gap-3">
          <div className="flex-1">
            <div className="text-sm font-medium mb-1">Start date</div>
            <Input type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} />
          </div>
          <div className="flex-1">
            <div className="text-sm font-medium mb-1">End date</div>
            <Input type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} />
          </div>
          <div className="flex md:justify-end">
            <Button onClick={() => load()} disabled={isLoading}>
              {isLoading ? 'Loading…' : 'Run report'}
            </Button>
          </div>
          <div className="flex md:justify-end">
            <Button variant="outline" onClick={exportCsv} disabled={isExporting}>
              {isExporting ? 'Exporting…' : 'Export CSV'}
            </Button>
          </div>
        </div>
        <div className="flex flex-wrap gap-2 mt-3">
          <Button size="sm" variant="outline" onClick={() => applyQuickRange(7)}>
            Last 7 days
          </Button>
          <Button size="sm" variant="outline" onClick={() => applyQuickRange(30)}>
            Last 30 days
          </Button>
          <Button size="sm" variant="outline" onClick={() => applyQuickRange(90)}>
            Last 90 days
          </Button>
        </div>
      </div>

      {isLoading ? (
        <div className="h-40 bg-muted rounded" />
      ) : !report ? (
        <div className="text-sm text-muted-foreground">No data.</div>
      ) : (
        <>
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
            <div className="bg-card border border-border rounded-lg p-4 shadow-sm lg:col-span-1">
              <div className="text-sm font-medium text-muted-foreground mb-2">Totals</div>
              <div className="space-y-1 text-sm">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Check-ins</span>
                  <span className="font-medium">{report.totals.check_ins.toLocaleString()}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Check-outs</span>
                  <span className="font-medium">{report.totals.check_outs.toLocaleString()}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Anomalies</span>
                  <span className="font-medium">{report.totals.anomalies.toLocaleString()}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Total logs</span>
                  <span className="font-medium">{report.totals.logs.toLocaleString()}</span>
                </div>
              </div>
            </div>

            <div className="bg-card border border-border rounded-lg p-4 shadow-sm lg:col-span-2">
              <div className="text-sm font-medium text-muted-foreground mb-3">
                Daily activity (check-ins + check-outs)
              </div>
              <MiniBarChart7d points={chartPoints} />
            </div>
          </div>

          <div className="bg-card border border-border rounded-lg p-4 shadow-sm">
            <div className="text-sm font-medium text-muted-foreground mb-3">Daily breakdown</div>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="text-muted-foreground">
                  <tr className="border-b">
                    <th className="text-left font-medium py-2 pr-3">Date</th>
                    <th className="text-right font-medium py-2 px-3">Check-ins</th>
                    <th className="text-right font-medium py-2 px-3">Check-outs</th>
                    <th className="text-right font-medium py-2 pl-3">Anomalies</th>
                  </tr>
                </thead>
                <tbody>
                  {report.days.map((d) => (
                    <tr key={d.date} className="border-b last:border-b-0">
                      <td className="py-2 pr-3">{d.date}</td>
                      <td className="py-2 px-3 text-right">{d.check_ins.toLocaleString()}</td>
                      <td className="py-2 px-3 text-right">{d.check_outs.toLocaleString()}</td>
                      <td className="py-2 pl-3 text-right">{d.anomalies.toLocaleString()}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </>
      )}
    </div>
  )
}
