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
  late_arrivals: number
  early_departures: number
}

type ShiftSummary = {
  shift_start_time?: string | null
  shift_end_time?: string | null
  users: number
  check_ins: number
  check_outs: number
  late_arrivals: number
  early_departures: number
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
    late_arrivals: number
    early_departures: number
  }
  shift_summary?: ShiftSummary[]
}

type Department = { id: string; name: string }

type ReportSchedule = {
  id: string
  report_type: string
  name?: string | null
  frequency: string
  day_of_week?: number | null
  time_of_day: string
  timezone: string
  recipients: string[]
  is_active: boolean
  last_sent_at?: string | null
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
  const [departmentId, setDepartmentId] = useState('')
  const [employeeId, setEmployeeId] = useState('')
  const [lateGrace, setLateGrace] = useState('10')
  const [earlyGrace, setEarlyGrace] = useState('10')
  const [report, setReport] = useState<ReportResponse | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [isExporting, setIsExporting] = useState(false)
  const [isExportingPdf, setIsExportingPdf] = useState(false)
  const [departments, setDepartments] = useState<Department[]>([])

  const [schedules, setSchedules] = useState<ReportSchedule[]>([])
  const [scheduleFrequency, setScheduleFrequency] = useState('weekly')
  const [scheduleDay, setScheduleDay] = useState('1')
  const [scheduleTime, setScheduleTime] = useState('09:00')
  const [scheduleRecipients, setScheduleRecipients] = useState('')
  const [isScheduling, setIsScheduling] = useState(false)

  const [sendRecipients, setSendRecipients] = useState('')
  const [isSendingNow, setIsSendingNow] = useState(false)

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
    void loadDepartments()
    void loadSchedules()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, user?.role])

  const loadDepartments = async () => {
    try {
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(`${base}/api/v1/departments`, { headers })
      if (!resp.ok) return
      const data = await resp.json()
      setDepartments(Array.isArray(data) ? data : [])
    } catch {
      setDepartments([])
    }
  }

  const loadSchedules = async () => {
    try {
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(`${base}/api/v1/reports/schedules`, { headers })
      if (!resp.ok) return
      const data = await resp.json()
      setSchedules(Array.isArray(data) ? data : [])
    } catch {
      setSchedules([])
    }
  }

  const load = async (s = startDate, e = endDate) => {
    if (!s || !e || e < s) {
      toast.error('Please choose a valid date range')
      return
    }
    try {
      setIsLoading(true)
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`
      const params = new URLSearchParams()
      params.set('start_date', s)
      params.set('end_date', e)
      if (departmentId) params.set('department_id', departmentId)
      if (employeeId.trim()) params.set('employee_id', employeeId.trim())
      params.set('late_grace_minutes', lateGrace || '10')
      params.set('early_grace_minutes', earlyGrace || '10')

      const resp = await fetch(`${base}/api/v1/reports/attendance?${params.toString()}`, { headers })
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
      const params = new URLSearchParams()
      params.set('start_date', startDate)
      params.set('end_date', endDate)
      if (departmentId) params.set('department_id', departmentId)
      if (employeeId.trim()) params.set('employee_id', employeeId.trim())

      const resp = await fetch(`${base}/api/v1/reports/export?${params.toString()}`, {
        method: 'POST',
        headers,
      })
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

  const exportPdf = async () => {
    if (!startDate || !endDate || endDate < startDate) {
      toast.error('Please choose a valid date range')
      return
    }
    try {
      setIsExportingPdf(true)
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`
      const params = new URLSearchParams()
      params.set('start_date', startDate)
      params.set('end_date', endDate)
      if (departmentId) params.set('department_id', departmentId)
      if (employeeId.trim()) params.set('employee_id', employeeId.trim())

      const resp = await fetch(`${base}/api/v1/reports/attendance/pdf?${params.toString()}`, {
        headers,
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to export PDF')
      }
      const blob = await resp.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `attendance-report-${startDate}-to-${endDate}.pdf`
      document.body.appendChild(a)
      a.click()
      a.remove()
      URL.revokeObjectURL(url)
      toast.success('PDF downloaded')
    } catch (e: any) {
      toast.error(e.message || 'Failed to export PDF')
    } finally {
      setIsExportingPdf(false)
    }
  }

  const createSchedule = async () => {
    if (!scheduleRecipients.trim()) {
      toast.error('Recipients are required')
      return
    }
    try {
      setIsScheduling(true)
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (token) headers.Authorization = `Bearer ${token}`
      const recipients = scheduleRecipients
        .split(',')
        .map((r) => r.trim())
        .filter(Boolean)
      const body = {
        report_type: 'attendance',
        frequency: scheduleFrequency,
        day_of_week: scheduleFrequency === 'weekly' ? Number(scheduleDay) : undefined,
        time_of_day: scheduleTime,
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC',
        recipients,
        filters: {
          department_id: departmentId || undefined,
          employee_id: employeeId.trim() || undefined,
          start_date: startDate,
          end_date: endDate,
        },
      }
      const resp = await fetch(`${base}/api/v1/reports/schedules`, {
        method: 'POST',
        headers,
        body: JSON.stringify(body),
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to create schedule')
      }
      toast.success('Schedule created')
      setScheduleRecipients('')
      await loadSchedules()
    } catch (e: any) {
      toast.error(e.message || 'Failed to create schedule')
    } finally {
      setIsScheduling(false)
    }
  }

  const runSchedule = async (id: string) => {
    try {
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(`${base}/api/v1/reports/schedules/${id}/run`, {
        method: 'POST',
        headers,
        body: JSON.stringify({ message: 'Manual dispatch from admin portal' }),
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to run schedule')
      }
      toast.success('Report queued')
      await loadSchedules()
    } catch (e: any) {
      toast.error(e.message || 'Failed to run schedule')
    }
  }

  const deleteSchedule = async (id: string) => {
    try {
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(`${base}/api/v1/reports/schedules/${id}`, {
        method: 'DELETE',
        headers,
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to delete schedule')
      }
      toast.success('Schedule deleted')
      await loadSchedules()
    } catch (e: any) {
      toast.error(e.message || 'Failed to delete schedule')
    }
  }

  const sendNow = async () => {
    if (!sendRecipients.trim()) {
      toast.error('Recipients are required')
      return
    }
    try {
      setIsSendingNow(true)
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (token) headers.Authorization = `Bearer ${token}`
      const recipients = sendRecipients
        .split(',')
        .map((r) => r.trim())
        .filter(Boolean)

      const resp = await fetch(`${base}/api/v1/reports/send-now`, {
        method: 'POST',
        headers,
        body: JSON.stringify({
          recipients,
          start_date: startDate,
          end_date: endDate,
          department_id: departmentId || undefined,
          employee_id: employeeId.trim() || undefined,
          late_grace_minutes: Number(lateGrace || 10),
          early_grace_minutes: Number(earlyGrace || 10),
        }),
      })
      const payload = await resp.json().catch(() => ({}))
      if (!resp.ok) {
        throw new Error(payload.error || 'Failed to send report')
      }
      if (payload.message_id) {
        toast.success(`Report sent (Message ID: ${payload.message_id})`)
      } else {
        toast.success('Report sent')
      }
      setSendRecipients('')
    } catch (e: any) {
      toast.error(e.message || 'Failed to send report')
    } finally {
      setIsSendingNow(false)
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

      <div className="bg-card border border-border rounded-lg p-4 shadow-sm space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
          <div>
            <div className="text-sm font-medium mb-1">Start date</div>
            <Input type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} />
          </div>
          <div>
            <div className="text-sm font-medium mb-1">End date</div>
            <Input type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} />
          </div>
          <div>
            <div className="text-sm font-medium mb-1">Department</div>
            <select
              value={departmentId}
              onChange={(e) => setDepartmentId(e.target.value)}
              className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            >
              <option value="">All departments</option>
              {departments.map((d) => (
                <option key={d.id} value={d.id}>
                  {d.name}
                </option>
              ))}
            </select>
          </div>
          <div>
            <div className="text-sm font-medium mb-1">Employee ID</div>
            <Input value={employeeId} onChange={(e) => setEmployeeId(e.target.value)} placeholder="Optional" />
          </div>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
          <div>
            <div className="text-sm font-medium mb-1">Late grace (min)</div>
            <Input value={lateGrace} onChange={(e) => setLateGrace(e.target.value)} />
          </div>
          <div>
            <div className="text-sm font-medium mb-1">Early grace (min)</div>
            <Input value={earlyGrace} onChange={(e) => setEarlyGrace(e.target.value)} />
          </div>
          <div className="flex md:items-end">
            <Button onClick={() => load()} disabled={isLoading} className="w-full">
              {isLoading ? 'Loading…' : 'Run report'}
            </Button>
          </div>
          <div className="flex md:items-end gap-2">
            <Button variant="outline" onClick={exportCsv} disabled={isExporting} className="w-full">
              {isExporting ? 'Exporting…' : 'Export CSV'}
            </Button>
            <Button variant="outline" onClick={exportPdf} disabled={isExportingPdf} className="w-full">
              {isExportingPdf ? 'Exporting…' : 'Export PDF'}
            </Button>
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
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
        <div className="space-y-3">
          <div className="skeleton h-28 w-full" />
          <div className="skeleton h-64 w-full" />
        </div>
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
                  <span className="text-muted-foreground">Late arrivals</span>
                  <span className="font-medium">{report.totals.late_arrivals.toLocaleString()}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Early departures</span>
                  <span className="font-medium">{report.totals.early_departures.toLocaleString()}</span>
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
                    <th className="text-right font-medium py-2 px-3">Late</th>
                    <th className="text-right font-medium py-2 px-3">Early</th>
                    <th className="text-right font-medium py-2 pl-3">Anomalies</th>
                  </tr>
                </thead>
                <tbody>
                  {report.days.map((d) => (
                    <tr key={d.date} className="border-b last:border-b-0">
                      <td className="py-2 pr-3">{d.date}</td>
                      <td className="py-2 px-3 text-right">{d.check_ins.toLocaleString()}</td>
                      <td className="py-2 px-3 text-right">{d.check_outs.toLocaleString()}</td>
                      <td className="py-2 px-3 text-right">{d.late_arrivals.toLocaleString()}</td>
                      <td className="py-2 px-3 text-right">{d.early_departures.toLocaleString()}</td>
                      <td className="py-2 pl-3 text-right">{d.anomalies.toLocaleString()}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          {report.shift_summary && report.shift_summary.length > 0 ? (
            <div className="bg-card border border-border rounded-lg p-4 shadow-sm">
              <div className="text-sm font-medium text-muted-foreground mb-3">Shift summary</div>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead className="text-muted-foreground">
                    <tr className="border-b">
                      <th className="text-left font-medium py-2 pr-3">Shift</th>
                      <th className="text-right font-medium py-2 px-3">Users</th>
                      <th className="text-right font-medium py-2 px-3">Check-ins</th>
                      <th className="text-right font-medium py-2 px-3">Check-outs</th>
                      <th className="text-right font-medium py-2 px-3">Late</th>
                      <th className="text-right font-medium py-2 pl-3">Early</th>
                    </tr>
                  </thead>
                  <tbody>
                    {report.shift_summary.map((s, idx) => (
                      <tr key={`${s.shift_start_time || 'na'}-${idx}`} className="border-b last:border-b-0">
                        <td className="py-2 pr-3">
                          {s.shift_start_time || '—'} - {s.shift_end_time || '—'}
                        </td>
                        <td className="py-2 px-3 text-right">{s.users.toLocaleString()}</td>
                        <td className="py-2 px-3 text-right">{s.check_ins.toLocaleString()}</td>
                        <td className="py-2 px-3 text-right">{s.check_outs.toLocaleString()}</td>
                        <td className="py-2 px-3 text-right">{s.late_arrivals.toLocaleString()}</td>
                        <td className="py-2 pl-3 text-right">{s.early_departures.toLocaleString()}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          ) : null}
        </>
      )}

      <div className="bg-card border border-border rounded-lg p-4 shadow-sm space-y-4">
        <div>
          <div className="text-lg font-semibold">Scheduled report delivery</div>
          <div className="text-sm text-muted-foreground">Email automated attendance summaries to your team.</div>
        </div>
        <div className="border rounded-md p-3 space-y-2">
          <div className="text-sm font-medium">Send now</div>
          <div className="grid grid-cols-1 md:grid-cols-[1fr_auto] gap-2">
            <Input
              value={sendRecipients}
              onChange={(e) => setSendRecipients(e.target.value)}
              placeholder="hr@company.com, ops@company.com"
            />
            <Button onClick={sendNow} disabled={isSendingNow}>
              {isSendingNow ? 'Sending…' : 'Send now'}
            </Button>
          </div>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
          <div>
            <div className="text-sm font-medium mb-1">Frequency</div>
            <select
              value={scheduleFrequency}
              onChange={(e) => setScheduleFrequency(e.target.value)}
              className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            >
              <option value="daily">Daily</option>
              <option value="weekly">Weekly</option>
              <option value="monthly">Monthly</option>
            </select>
          </div>
          <div>
            <div className="text-sm font-medium mb-1">Day</div>
            <select
              value={scheduleDay}
              onChange={(e) => setScheduleDay(e.target.value)}
              className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              disabled={scheduleFrequency !== 'weekly'}
            >
              <option value="1">Monday</option>
              <option value="2">Tuesday</option>
              <option value="3">Wednesday</option>
              <option value="4">Thursday</option>
              <option value="5">Friday</option>
              <option value="6">Saturday</option>
              <option value="0">Sunday</option>
            </select>
          </div>
          <div>
            <div className="text-sm font-medium mb-1">Time</div>
            <Input type="time" value={scheduleTime} onChange={(e) => setScheduleTime(e.target.value)} />
          </div>
          <div>
            <div className="text-sm font-medium mb-1">Recipients</div>
            <Input
              value={scheduleRecipients}
              onChange={(e) => setScheduleRecipients(e.target.value)}
              placeholder="hr@company.com, ops@company.com"
            />
          </div>
        </div>
        <div className="flex justify-end">
          <Button onClick={createSchedule} disabled={isScheduling}>
            {isScheduling ? 'Saving…' : 'Create schedule'}
          </Button>
        </div>
        <div className="border rounded-md">
          <div className="border-b px-3 py-2 text-sm font-medium text-muted-foreground">Active schedules</div>
          {schedules.length === 0 ? (
            <div className="p-3 text-sm text-muted-foreground">No schedules configured.</div>
          ) : (
            <div className="divide-y">
              {schedules.map((s) => (
                <div key={s.id} className="p-3 flex flex-col md:flex-row md:items-center md:justify-between gap-3">
                  <div>
                    <div className="font-medium">{s.report_type.toUpperCase()} report</div>
                    <div className="text-xs text-muted-foreground">
                      {s.frequency} at {s.time_of_day} ({s.timezone})
                      {s.day_of_week !== null && s.day_of_week !== undefined ? ` • Day ${s.day_of_week}` : ''}
                    </div>
                    <div className="text-xs text-muted-foreground">Recipients: {s.recipients.join(', ')}</div>
                  </div>
                  <div className="flex gap-2">
                    <Button size="sm" variant="outline" onClick={() => runSchedule(s.id)}>
                      Run now
                    </Button>
                    <Button size="sm" variant="outline" onClick={() => deleteSchedule(s.id)}>
                      Delete
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
