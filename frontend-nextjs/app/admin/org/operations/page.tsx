'use client'

import { useEffect, useMemo, useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import toast from 'react-hot-toast'
import { useAuthStore } from '@/store/useStore'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface UserOption {
  id: string
  employee_id: string
  first_name: string
  last_name: string
  role: string
}

interface WorkflowUser {
  user_id: string
  employee_id: string
  first_name: string
  last_name: string
  department_name?: string | null
}

interface LeaveRequestRow {
  id: string
  user: WorkflowUser
  leave_type: string
  start_date: string
  end_date: string
  day_count: number
  reason?: string | null
  status: string
}

interface RegularizationRow {
  id: string
  user: WorkflowUser
  requested_status: string
  requested_punch_time: string
  reason: string
  status: string
}

interface OvertimeRow {
  id: string
  user: WorkflowUser
  work_date: string
  requested_minutes: number
  approved_minutes: number
  reason?: string | null
  status: string
}

interface ShiftRow {
  id: string
  user: WorkflowUser
  shift_name: string
  start_date: string
  end_date: string
  start_time: string
  end_time: string
  work_days: string[]
  is_rota: boolean
  notes?: string | null
}

interface ExceptionRow {
  id: string
  attendance_log_id: string
  assigned_to: WorkflowUser
  employee: WorkflowUser
  punch_time: string
  status: string
  sla_due_at?: string | null
  note?: string | null
  anomaly_reason?: string | null
}

interface AnomalyRow {
  id: string
  employee_id: string
  first_name: string
  last_name: string
  punch_time: string
  anomaly_reason?: string | null
}

interface AttendanceOpsSettings {
  allow_remote_attendance: boolean
  geofencing_enabled: boolean
  geofence_latitude?: number | null
  geofence_longitude?: number | null
  geofence_radius_meters: number
  break_tracking_enabled: boolean
  exception_sla_hours: number
}

export default function AttendanceOperationsPage() {
  const { user, token, isAuthenticated } = useAuthStore()
  const router = useRouter()
  const [isLoading, setIsLoading] = useState(true)
  const [users, setUsers] = useState<UserOption[]>([])
  const [settings, setSettings] = useState<AttendanceOpsSettings>({
    allow_remote_attendance: false,
    geofencing_enabled: false,
    geofence_latitude: null,
    geofence_longitude: null,
    geofence_radius_meters: 200,
    break_tracking_enabled: true,
    exception_sla_hours: 24,
  })
  const [leaveRequests, setLeaveRequests] = useState<LeaveRequestRow[]>([])
  const [regularizations, setRegularizations] = useState<RegularizationRow[]>([])
  const [overtimeRequests, setOvertimeRequests] = useState<OvertimeRow[]>([])
  const [shifts, setShifts] = useState<ShiftRow[]>([])
  const [exceptions, setExceptions] = useState<ExceptionRow[]>([])
  const [anomalies, setAnomalies] = useState<AnomalyRow[]>([])
  const [shiftForm, setShiftForm] = useState({
    user_id: '',
    shift_name: 'General Shift',
    start_date: '',
    end_date: '',
    start_time: '09:00',
    end_time: '18:00',
    work_days: 'monday,tuesday,wednesday,thursday,friday',
    is_rota: false,
    notes: '',
  })
  const [exceptionForm, setExceptionForm] = useState({
    attendance_log_id: '',
    assigned_to: '',
    sla_due_at: '',
    note: '',
  })

  const base = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
  const headers = useMemo(
    () => ({
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    }),
    [token]
  )

  useEffect(() => {
    if (!isAuthenticated || !user) {
      router.replace('/admin/login')
      return
    }
    if (!['org_admin', 'hr', 'dept_manager'].includes(user.role)) {
      router.replace('/dashboard')
      return
    }
    void loadAll()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, user?.role])

  const loadAll = async () => {
    try {
      setIsLoading(true)
      const [settingsResp, leaveResp, regResp, overtimeResp, shiftsResp, exceptionsResp, usersResp, anomaliesResp] = await Promise.all([
        fetch(`${base}/api/v1/attendance-ops/settings`, { headers }),
        fetch(`${base}/api/v1/attendance-ops/leave-requests`, { headers }),
        fetch(`${base}/api/v1/attendance-ops/regularizations`, { headers }),
        fetch(`${base}/api/v1/attendance-ops/overtime-requests`, { headers }),
        fetch(`${base}/api/v1/attendance-ops/shifts`, { headers }),
        fetch(`${base}/api/v1/attendance-ops/exceptions`, { headers }),
        fetch(`${base}/api/v1/users?limit=200`, { headers }),
        fetch(`${base}/api/v1/reports/anomalies?state=unresolved&limit=50`, { headers }),
      ])

      if (settingsResp.ok) setSettings(await settingsResp.json())
      if (leaveResp.ok) setLeaveRequests(await leaveResp.json())
      if (regResp.ok) setRegularizations(await regResp.json())
      if (overtimeResp.ok) setOvertimeRequests(await overtimeResp.json())
      if (shiftsResp.ok) setShifts(await shiftsResp.json())
      if (exceptionsResp.ok) setExceptions(await exceptionsResp.json())
      if (usersResp.ok) {
        const data = await usersResp.json()
        const rows = Array.isArray(data?.data) ? data.data : Array.isArray(data) ? data : []
        setUsers(rows)
      }
      if (anomaliesResp.ok) setAnomalies(await anomaliesResp.json())
    } catch (e: any) {
      toast.error(e.message || 'Failed to load attendance operations')
    } finally {
      setIsLoading(false)
    }
  }

  const saveSettings = async () => {
    try {
      const resp = await fetch(`${base}/api/v1/attendance-ops/settings`, {
        method: 'PUT',
        headers,
        body: JSON.stringify(settings),
      })
      const data = await resp.json().catch(() => ({}))
      if (!resp.ok) throw new Error(data.error || 'Failed to save operations settings')
      toast.success('Operations settings updated')
      setSettings(data)
    } catch (e: any) {
      toast.error(e.message || 'Failed to save operations settings')
    }
  }

  const reviewRequest = async (kind: 'leave' | 'regularization' | 'overtime', id: string, status: 'approved' | 'rejected', approvedMinutes?: number) => {
    try {
      const payload: any = { status }
      if (kind === 'overtime' && status === 'approved') {
        payload.approved_minutes = approvedMinutes || 0
      }
      const endpoint =
        kind === 'leave'
          ? `${base}/api/v1/attendance-ops/leave-requests/${id}/review`
          : kind === 'regularization'
            ? `${base}/api/v1/attendance-ops/regularizations/${id}/review`
            : `${base}/api/v1/attendance-ops/overtime-requests/${id}/review`
      const resp = await fetch(endpoint, {
        method: 'PATCH',
        headers,
        body: JSON.stringify(payload),
      })
      const data = await resp.json().catch(() => ({}))
      if (!resp.ok) throw new Error(data.error || 'Failed to review request')
      toast.success(`${kind} request ${status}`)
      await loadAll()
    } catch (e: any) {
      toast.error(e.message || 'Failed to review request')
    }
  }

  const createShift = async () => {
    try {
      const resp = await fetch(`${base}/api/v1/attendance-ops/shifts`, {
        method: 'POST',
        headers,
        body: JSON.stringify({
          ...shiftForm,
          work_days: shiftForm.work_days.split(',').map((item) => item.trim()).filter(Boolean),
          notes: shiftForm.notes || null,
        }),
      })
      const data = await resp.json().catch(() => ({}))
      if (!resp.ok) throw new Error(data.error || 'Failed to create shift assignment')
      toast.success('Shift assignment created')
      await loadAll()
    } catch (e: any) {
      toast.error(e.message || 'Failed to create shift assignment')
    }
  }

  const deleteShift = async (id: string) => {
    try {
      const resp = await fetch(`${base}/api/v1/attendance-ops/shifts/${id}`, {
        method: 'DELETE',
        headers,
      })
      const data = await resp.json().catch(() => ({}))
      if (!resp.ok) throw new Error(data.error || 'Failed to delete shift assignment')
      toast.success('Shift assignment deleted')
      await loadAll()
    } catch (e: any) {
      toast.error(e.message || 'Failed to delete shift assignment')
    }
  }

  const assignException = async () => {
    try {
      const resp = await fetch(`${base}/api/v1/attendance-ops/exceptions`, {
        method: 'POST',
        headers,
        body: JSON.stringify({
          ...exceptionForm,
          sla_due_at: exceptionForm.sla_due_at ? new Date(exceptionForm.sla_due_at).toISOString() : null,
          note: exceptionForm.note || null,
        }),
      })
      const data = await resp.json().catch(() => ({}))
      if (!resp.ok) throw new Error(data.error || 'Failed to assign attendance exception')
      toast.success('Attendance exception assigned')
      setExceptionForm({ attendance_log_id: '', assigned_to: '', sla_due_at: '', note: '' })
      await loadAll()
    } catch (e: any) {
      toast.error(e.message || 'Failed to assign attendance exception')
    }
  }

  const resolveException = async (id: string) => {
    try {
      const resp = await fetch(`${base}/api/v1/attendance-ops/exceptions/${id}`, {
        method: 'PATCH',
        headers,
        body: JSON.stringify({ status: 'resolved' }),
      })
      const data = await resp.json().catch(() => ({}))
      if (!resp.ok) throw new Error(data.error || 'Failed to resolve attendance exception')
      toast.success('Attendance exception resolved')
      await loadAll()
    } catch (e: any) {
      toast.error(e.message || 'Failed to resolve attendance exception')
    }
  }

  if (!isAuthenticated || !user) return null

  return (
    <div className="container mx-auto px-4 py-8 space-y-8">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">Attendance Operations</h1>
          <p className="text-muted-foreground">Manage leave, corrections, overtime, shift planning, remote attendance policy, and exception assignments.</p>
        </div>
        <Link href="/admin/org/users">
          <Button variant="outline">Back to Employees</Button>
        </Link>
      </div>

      <div className="rounded-lg border bg-card p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">Remote Attendance And Break Policy</h2>
          <Button onClick={() => void saveSettings()}>Save Settings</Button>
        </div>
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <label className="flex items-center gap-2 text-sm text-muted-foreground">
            <input type="checkbox" checked={settings.allow_remote_attendance} onChange={(e) => setSettings((prev) => ({ ...prev, allow_remote_attendance: e.target.checked }))} />
            Allow remote attendance
          </label>
          <label className="flex items-center gap-2 text-sm text-muted-foreground">
            <input type="checkbox" checked={settings.geofencing_enabled} onChange={(e) => setSettings((prev) => ({ ...prev, geofencing_enabled: e.target.checked }))} />
            Enforce geofencing
          </label>
          <label className="flex items-center gap-2 text-sm text-muted-foreground">
            <input type="checkbox" checked={settings.break_tracking_enabled} onChange={(e) => setSettings((prev) => ({ ...prev, break_tracking_enabled: e.target.checked }))} />
            Track breaks
          </label>
          <div>
            <Label>Exception SLA (hours)</Label>
            <Input type="number" value={settings.exception_sla_hours} onChange={(e) => setSettings((prev) => ({ ...prev, exception_sla_hours: Number(e.target.value) || 24 }))} className="mt-1" />
          </div>
          <div>
            <Label>Geofence latitude</Label>
            <Input value={settings.geofence_latitude ?? ''} onChange={(e) => setSettings((prev) => ({ ...prev, geofence_latitude: e.target.value ? Number(e.target.value) : null }))} className="mt-1" />
          </div>
          <div>
            <Label>Geofence longitude</Label>
            <Input value={settings.geofence_longitude ?? ''} onChange={(e) => setSettings((prev) => ({ ...prev, geofence_longitude: e.target.value ? Number(e.target.value) : null }))} className="mt-1" />
          </div>
          <div>
            <Label>Geofence radius (meters)</Label>
            <Input type="number" value={settings.geofence_radius_meters} onChange={(e) => setSettings((prev) => ({ ...prev, geofence_radius_meters: Number(e.target.value) || 200 }))} className="mt-1" />
          </div>
        </div>
      </div>

      <div className="grid gap-6 xl:grid-cols-2">
        <div className="rounded-lg border bg-card p-6 space-y-4">
          <h2 className="text-lg font-semibold">Leave And Absence Requests</h2>
          {isLoading ? <div className="text-sm text-muted-foreground">Loading...</div> : leaveRequests.length === 0 ? <div className="text-sm text-muted-foreground">No leave requests.</div> : (
            <div className="space-y-3">
              {leaveRequests.map((request) => (
                <div key={request.id} className="rounded-md border p-3">
                  <div className="flex items-start justify-between gap-4">
                    <div>
                      <div className="font-medium">{request.user.first_name} {request.user.last_name} · {request.leave_type}</div>
                      <div className="text-sm text-muted-foreground">{request.start_date} to {request.end_date} · {request.day_count} days</div>
                      <div className="text-sm text-muted-foreground">{request.reason || 'No reason provided'}</div>
                    </div>
                    <div className="text-xs uppercase text-muted-foreground">{request.status}</div>
                  </div>
                  {request.status === 'pending' ? (
                    <div className="flex gap-2 mt-3">
                      <Button size="sm" onClick={() => void reviewRequest('leave', request.id, 'approved')}>Approve</Button>
                      <Button size="sm" variant="outline" onClick={() => void reviewRequest('leave', request.id, 'rejected')}>Reject</Button>
                    </div>
                  ) : null}
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="rounded-lg border bg-card p-6 space-y-4">
          <h2 className="text-lg font-semibold">Attendance Corrections</h2>
          {isLoading ? <div className="text-sm text-muted-foreground">Loading...</div> : regularizations.length === 0 ? <div className="text-sm text-muted-foreground">No regularization requests.</div> : (
            <div className="space-y-3">
              {regularizations.map((request) => (
                <div key={request.id} className="rounded-md border p-3">
                  <div className="flex items-start justify-between gap-4">
                    <div>
                      <div className="font-medium">{request.user.first_name} {request.user.last_name} · {request.requested_status.replace('_', ' ')}</div>
                      <div className="text-sm text-muted-foreground">{new Date(request.requested_punch_time).toLocaleString()}</div>
                      <div className="text-sm text-muted-foreground">{request.reason}</div>
                    </div>
                    <div className="text-xs uppercase text-muted-foreground">{request.status}</div>
                  </div>
                  {request.status === 'pending' ? (
                    <div className="flex gap-2 mt-3">
                      <Button size="sm" onClick={() => void reviewRequest('regularization', request.id, 'approved')}>Approve</Button>
                      <Button size="sm" variant="outline" onClick={() => void reviewRequest('regularization', request.id, 'rejected')}>Reject</Button>
                    </div>
                  ) : null}
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="rounded-lg border bg-card p-6 space-y-4">
          <h2 className="text-lg font-semibold">Overtime Tracking</h2>
          {isLoading ? <div className="text-sm text-muted-foreground">Loading...</div> : overtimeRequests.length === 0 ? <div className="text-sm text-muted-foreground">No overtime requests.</div> : (
            <div className="space-y-3">
              {overtimeRequests.map((request) => (
                <div key={request.id} className="rounded-md border p-3">
                  <div className="flex items-start justify-between gap-4">
                    <div>
                      <div className="font-medium">{request.user.first_name} {request.user.last_name}</div>
                      <div className="text-sm text-muted-foreground">{request.work_date} · requested {request.requested_minutes} min</div>
                      <div className="text-sm text-muted-foreground">{request.reason || 'No reason provided'}</div>
                    </div>
                    <div className="text-xs uppercase text-muted-foreground">{request.status}</div>
                  </div>
                  {request.status === 'pending' ? (
                    <div className="flex gap-2 mt-3">
                      <Button size="sm" onClick={() => void reviewRequest('overtime', request.id, 'approved', request.requested_minutes)}>Approve Requested</Button>
                      <Button size="sm" variant="outline" onClick={() => void reviewRequest('overtime', request.id, 'rejected')}>Reject</Button>
                    </div>
                  ) : null}
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="rounded-lg border bg-card p-6 space-y-4">
          <h2 className="text-lg font-semibold">Shift Planning And Rota</h2>
          <div className="grid gap-3 md:grid-cols-2 border-b pb-4">
            <select value={shiftForm.user_id} onChange={(e) => setShiftForm((prev) => ({ ...prev, user_id: e.target.value }))} className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm">
              <option value="">Select employee</option>
              {users.map((item) => (
                <option key={item.id} value={item.id}>{item.first_name} {item.last_name} · {item.employee_id}</option>
              ))}
            </select>
            <Input value={shiftForm.shift_name} onChange={(e) => setShiftForm((prev) => ({ ...prev, shift_name: e.target.value }))} placeholder="Shift name" />
            <Input type="date" value={shiftForm.start_date} onChange={(e) => setShiftForm((prev) => ({ ...prev, start_date: e.target.value }))} />
            <Input type="date" value={shiftForm.end_date} onChange={(e) => setShiftForm((prev) => ({ ...prev, end_date: e.target.value }))} />
            <Input type="time" value={shiftForm.start_time} onChange={(e) => setShiftForm((prev) => ({ ...prev, start_time: e.target.value }))} />
            <Input type="time" value={shiftForm.end_time} onChange={(e) => setShiftForm((prev) => ({ ...prev, end_time: e.target.value }))} />
            <Input className="md:col-span-2" value={shiftForm.work_days} onChange={(e) => setShiftForm((prev) => ({ ...prev, work_days: e.target.value }))} placeholder="monday,tuesday,..." />
            <Input className="md:col-span-2" value={shiftForm.notes} onChange={(e) => setShiftForm((prev) => ({ ...prev, notes: e.target.value }))} placeholder="Notes" />
            <label className="flex items-center gap-2 text-sm text-muted-foreground">
              <input type="checkbox" checked={shiftForm.is_rota} onChange={(e) => setShiftForm((prev) => ({ ...prev, is_rota: e.target.checked }))} />
              Mark as rota-based assignment
            </label>
            <Button onClick={() => void createShift()}>Create Shift Assignment</Button>
          </div>
          <div className="space-y-3">
            {shifts.map((shift) => (
              <div key={shift.id} className="rounded-md border p-3 flex items-start justify-between gap-3">
                <div>
                  <div className="font-medium">{shift.user.first_name} {shift.user.last_name} · {shift.shift_name}</div>
                  <div className="text-sm text-muted-foreground">{shift.start_date} to {shift.end_date} · {shift.start_time} - {shift.end_time}</div>
                  <div className="text-sm text-muted-foreground">{shift.work_days.join(', ')}</div>
                </div>
                <Button size="sm" variant="outline" onClick={() => void deleteShift(shift.id)}>Delete</Button>
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="grid gap-6 xl:grid-cols-2">
        <div className="rounded-lg border bg-card p-6 space-y-4">
          <h2 className="text-lg font-semibold">Attendance Exception Assignment</h2>
          <div className="grid gap-3 md:grid-cols-2 border-b pb-4">
            <select value={exceptionForm.attendance_log_id} onChange={(e) => setExceptionForm((prev) => ({ ...prev, attendance_log_id: e.target.value }))} className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm">
              <option value="">Select unresolved anomaly</option>
              {anomalies.map((anomaly) => (
                <option key={anomaly.id} value={anomaly.id}>{anomaly.first_name} {anomaly.last_name} · {new Date(anomaly.punch_time).toLocaleString()}</option>
              ))}
            </select>
            <select value={exceptionForm.assigned_to} onChange={(e) => setExceptionForm((prev) => ({ ...prev, assigned_to: e.target.value }))} className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm">
              <option value="">Assign to</option>
              {users.map((item) => (
                <option key={item.id} value={item.id}>{item.first_name} {item.last_name} · {item.role}</option>
              ))}
            </select>
            <Input type="datetime-local" value={exceptionForm.sla_due_at} onChange={(e) => setExceptionForm((prev) => ({ ...prev, sla_due_at: e.target.value }))} />
            <Input value={exceptionForm.note} onChange={(e) => setExceptionForm((prev) => ({ ...prev, note: e.target.value }))} placeholder="Assignment note" />
            <Button className="md:col-span-2" onClick={() => void assignException()}>Assign Exception</Button>
          </div>
          <div className="space-y-3">
            {exceptions.map((item) => (
              <div key={item.id} className="rounded-md border p-3 flex items-start justify-between gap-3">
                <div>
                  <div className="font-medium">{item.employee.first_name} {item.employee.last_name}</div>
                  <div className="text-sm text-muted-foreground">Assigned to {item.assigned_to.first_name} {item.assigned_to.last_name}</div>
                  <div className="text-sm text-muted-foreground">SLA: {item.sla_due_at ? new Date(item.sla_due_at).toLocaleString() : '—'}</div>
                  <div className="text-sm text-muted-foreground">{item.anomaly_reason || item.note || 'No note'}</div>
                </div>
                <div className="flex flex-col gap-2 items-end">
                  <span className="text-xs uppercase text-muted-foreground">{item.status}</span>
                  {item.status !== 'resolved' ? <Button size="sm" onClick={() => void resolveException(item.id)}>Resolve</Button> : null}
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="rounded-lg border bg-card p-6 space-y-4">
          <h2 className="text-lg font-semibold">Operations Snapshot</h2>
          <div className="grid gap-4 md:grid-cols-2">
            <div className="rounded-md border p-4">
              <div className="text-sm text-muted-foreground">Pending leave requests</div>
              <div className="text-3xl font-bold mt-2">{leaveRequests.filter((item) => item.status === 'pending').length}</div>
            </div>
            <div className="rounded-md border p-4">
              <div className="text-sm text-muted-foreground">Pending corrections</div>
              <div className="text-3xl font-bold mt-2">{regularizations.filter((item) => item.status === 'pending').length}</div>
            </div>
            <div className="rounded-md border p-4">
              <div className="text-sm text-muted-foreground">Pending overtime approvals</div>
              <div className="text-3xl font-bold mt-2">{overtimeRequests.filter((item) => item.status === 'pending').length}</div>
            </div>
            <div className="rounded-md border p-4">
              <div className="text-sm text-muted-foreground">Open exceptions</div>
              <div className="text-3xl font-bold mt-2">{exceptions.filter((item) => item.status !== 'resolved').length}</div>
            </div>
          </div>
          <div className="text-sm text-muted-foreground border-t pt-4 space-y-2">
            <div>Remote attendance: {settings.allow_remote_attendance ? 'enabled' : 'disabled'}</div>
            <div>Geofencing: {settings.geofencing_enabled ? `enabled (${settings.geofence_radius_meters}m)` : 'disabled'}</div>
            <div>Break tracking: {settings.break_tracking_enabled ? 'required' : 'disabled'}</div>
            <div>Exception SLA: {settings.exception_sla_hours} hours</div>
          </div>
        </div>
      </div>
    </div>
  )
}
