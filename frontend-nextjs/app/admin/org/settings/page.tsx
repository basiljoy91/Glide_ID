'use client'

import { useEffect, useMemo, useState } from 'react'
import { useRouter } from 'next/navigation'
import toast from 'react-hot-toast'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useAuthStore } from '@/store/useStore'

type Holiday = { date: string; name: string }
type ShiftTemplate = {
  id?: string
  name: string
  start_time: string
  end_time: string
  days: string[]
  grace_minutes: number
  notes: string
  is_default: boolean
}

type SettingsPayload = {
  company_profile: {
    display_name: string
    legal_name: string
    brand_color: string
    logo_url: string
    support_email: string
    support_phone: string
  }
  operational: {
    timezone: string
    work_week: string[]
    holiday_calendar: Holiday[]
  }
  attendance_policy: {
    late_grace_minutes: number
    early_departure_grace_minutes: number
    break_grace_minutes: number
    auto_checkout_hours: number
    regularization_requires_approval: boolean
    allow_manual_attendance_adjustments: boolean
  }
  kiosk_defaults: {
    heartbeat_grace_minutes: number
    offline_sync_window_hours: number
    require_pin_fallback: boolean
    default_location: string
  }
  data_retention: {
    attendance_log_days: number
    audit_log_days: number
    inactive_user_purge_days: number
  }
  shift_templates: ShiftTemplate[]
}

const emptySettings: SettingsPayload = {
  company_profile: {
    display_name: '',
    legal_name: '',
    brand_color: '#111827',
    logo_url: '',
    support_email: '',
    support_phone: '',
  },
  operational: {
    timezone: 'UTC',
    work_week: ['monday', 'tuesday', 'wednesday', 'thursday', 'friday'],
    holiday_calendar: [],
  },
  attendance_policy: {
    late_grace_minutes: 10,
    early_departure_grace_minutes: 10,
    break_grace_minutes: 5,
    auto_checkout_hours: 16,
    regularization_requires_approval: true,
    allow_manual_attendance_adjustments: false,
  },
  kiosk_defaults: {
    heartbeat_grace_minutes: 15,
    offline_sync_window_hours: 24,
    require_pin_fallback: true,
    default_location: '',
  },
  data_retention: {
    attendance_log_days: 365,
    audit_log_days: 730,
    inactive_user_purge_days: 365,
  },
  shift_templates: [],
}

const weekdayOptions = ['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday']

export default function OrgSettingsPage() {
  const { isAuthenticated, user, token } = useAuthStore()
  const router = useRouter()
  const base = useMemo(() => process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080', [])
  const canManage = user?.role === 'org_admin' || user?.permissions?.includes('settings.manage')

  const [settings, setSettings] = useState<SettingsPayload>(emptySettings)
  const [newHoliday, setNewHoliday] = useState<Holiday>({ date: '', name: '' })
  const [shiftDraft, setShiftDraft] = useState<ShiftTemplate>({
    name: '',
    start_time: '09:00',
    end_time: '18:00',
    days: ['monday', 'tuesday', 'wednesday', 'thursday', 'friday'],
    grace_minutes: 10,
    notes: '',
    is_default: false,
  })
  const [isLoading, setIsLoading] = useState(true)
  const [isSaving, setIsSaving] = useState(false)
  const [isSavingShift, setIsSavingShift] = useState(false)

  useEffect(() => {
    if (!isAuthenticated || !user) {
      router.push('/admin/login')
      return
    }
    if (!canManage) {
      router.push('/admin/org')
      return
    }
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, user?.id, canManage])

  const authHeaders = () => ({
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  })

  const load = async () => {
    try {
      setIsLoading(true)
      const resp = await fetch(`${base}/api/v1/org/settings`, { headers: authHeaders() })
      if (!resp.ok) {
        const error = await resp.json().catch(() => ({}))
        throw new Error(error.error || 'Failed to load settings')
      }
      const data = await resp.json()
      setSettings({ ...emptySettings, ...data, shift_templates: Array.isArray(data.shift_templates) ? data.shift_templates : [] })
    } catch (error: any) {
      toast.error(error.message || 'Failed to load settings')
    } finally {
      setIsLoading(false)
    }
  }

  const saveSettings = async () => {
    try {
      setIsSaving(true)
      const resp = await fetch(`${base}/api/v1/org/settings`, {
        method: 'PUT',
        headers: authHeaders(),
        body: JSON.stringify(settings),
      })
      if (!resp.ok) {
        const error = await resp.json().catch(() => ({}))
        throw new Error(error.error || 'Failed to save settings')
      }
      const data = await resp.json()
      setSettings({ ...emptySettings, ...data, shift_templates: Array.isArray(data.shift_templates) ? data.shift_templates : [] })
      toast.success('Organization settings updated')
    } catch (error: any) {
      toast.error(error.message || 'Failed to save settings')
    } finally {
      setIsSaving(false)
    }
  }

  const saveShift = async () => {
    try {
      setIsSavingShift(true)
      const method = shiftDraft.id ? 'PUT' : 'POST'
      const endpoint = shiftDraft.id
        ? `${base}/api/v1/org/settings/shifts/${shiftDraft.id}`
        : `${base}/api/v1/org/settings/shifts`
      const resp = await fetch(endpoint, {
        method,
        headers: authHeaders(),
        body: JSON.stringify(shiftDraft),
      })
      if (!resp.ok) {
        const error = await resp.json().catch(() => ({}))
        throw new Error(error.error || 'Failed to save shift template')
      }
      toast.success('Shift template saved')
      setShiftDraft({
        name: '',
        start_time: '09:00',
        end_time: '18:00',
        days: ['monday', 'tuesday', 'wednesday', 'thursday', 'friday'],
        grace_minutes: 10,
        notes: '',
        is_default: false,
      })
      await load()
    } catch (error: any) {
      toast.error(error.message || 'Failed to save shift template')
    } finally {
      setIsSavingShift(false)
    }
  }

  const deleteShift = async (id?: string) => {
    if (!id) return
    try {
      const resp = await fetch(`${base}/api/v1/org/settings/shifts/${id}`, {
        method: 'DELETE',
        headers: authHeaders(),
      })
      if (!resp.ok) {
        const error = await resp.json().catch(() => ({}))
        throw new Error(error.error || 'Failed to delete shift template')
      }
      toast.success('Shift template deleted')
      await load()
    } catch (error: any) {
      toast.error(error.message || 'Failed to delete shift template')
    }
  }

  const toggleWorkday = (day: string) => {
    setSettings((current) => ({
      ...current,
      operational: {
        ...current.operational,
        work_week: current.operational.work_week.includes(day)
          ? current.operational.work_week.filter((item) => item !== day)
          : [...current.operational.work_week, day],
      },
    }))
  }

  const toggleShiftDay = (day: string) => {
    setShiftDraft((current) => ({
      ...current,
      days: current.days.includes(day)
        ? current.days.filter((item) => item !== day)
        : [...current.days, day],
    }))
  }

  const addHoliday = () => {
    if (!newHoliday.date || !newHoliday.name.trim()) {
      toast.error('Enter both holiday date and name')
      return
    }
    setSettings((current) => ({
      ...current,
      operational: {
        ...current.operational,
        holiday_calendar: [...current.operational.holiday_calendar, newHoliday],
      },
    }))
    setNewHoliday({ date: '', name: '' })
  }

  if (!isAuthenticated || !user || !canManage) return null

  return (
    <div className="container mx-auto space-y-6 p-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold">Organization Settings</h1>
          <p className="text-muted-foreground">Manage company profile, work rules, kiosk defaults, retention, and shift templates.</p>
        </div>
        <Button onClick={saveSettings} disabled={isSaving || isLoading}>
          {isSaving ? 'Saving...' : 'Save Settings'}
        </Button>
      </div>

      {isLoading ? (
        <div className="rounded-lg border p-6 text-sm text-muted-foreground">Loading settings...</div>
      ) : (
        <>
          <section className="grid gap-6 rounded-lg border bg-card p-6 lg:grid-cols-2">
            <div className="space-y-4">
              <h2 className="text-lg font-semibold">Company Profile & Branding</h2>
              <Input placeholder="Display name" value={settings.company_profile.display_name} onChange={(e) => setSettings((current) => ({ ...current, company_profile: { ...current.company_profile, display_name: e.target.value } }))} />
              <Input placeholder="Legal name" value={settings.company_profile.legal_name} onChange={(e) => setSettings((current) => ({ ...current, company_profile: { ...current.company_profile, legal_name: e.target.value } }))} />
              <Input placeholder="Brand color" value={settings.company_profile.brand_color} onChange={(e) => setSettings((current) => ({ ...current, company_profile: { ...current.company_profile, brand_color: e.target.value } }))} />
              <Input placeholder="Logo URL" value={settings.company_profile.logo_url} onChange={(e) => setSettings((current) => ({ ...current, company_profile: { ...current.company_profile, logo_url: e.target.value } }))} />
              <Input placeholder="Support email" value={settings.company_profile.support_email} onChange={(e) => setSettings((current) => ({ ...current, company_profile: { ...current.company_profile, support_email: e.target.value } }))} />
              <Input placeholder="Support phone" value={settings.company_profile.support_phone} onChange={(e) => setSettings((current) => ({ ...current, company_profile: { ...current.company_profile, support_phone: e.target.value } }))} />
            </div>

            <div className="space-y-4">
              <h2 className="text-lg font-semibold">Operational Calendar</h2>
              <Input placeholder="Timezone" value={settings.operational.timezone} onChange={(e) => setSettings((current) => ({ ...current, operational: { ...current.operational, timezone: e.target.value } }))} />
              <div className="flex flex-wrap gap-2">
                {weekdayOptions.map((day) => (
                  <button
                    key={day}
                    type="button"
                    onClick={() => toggleWorkday(day)}
                    className={`rounded-full border px-3 py-1 text-sm ${settings.operational.work_week.includes(day) ? 'border-primary bg-primary text-primary-foreground' : 'border-border bg-background text-foreground'}`}
                  >
                    {day.slice(0, 3).toUpperCase()}
                  </button>
                ))}
              </div>
              <div className="grid gap-2 md:grid-cols-[180px_1fr_auto]">
                <Input type="date" value={newHoliday.date} onChange={(e) => setNewHoliday((current) => ({ ...current, date: e.target.value }))} />
                <Input placeholder="Holiday name" value={newHoliday.name} onChange={(e) => setNewHoliday((current) => ({ ...current, name: e.target.value }))} />
                <Button type="button" variant="outline" onClick={addHoliday}>Add</Button>
              </div>
              <div className="space-y-2 text-sm">
                {settings.operational.holiday_calendar.length === 0 ? (
                  <div className="rounded-md border border-dashed p-3 text-muted-foreground">No holidays configured.</div>
                ) : (
                  settings.operational.holiday_calendar.map((holiday, index) => (
                    <div key={`${holiday.date}-${index}`} className="flex items-center justify-between rounded-md border px-3 py-2">
                      <div>
                        <div className="font-medium">{holiday.name}</div>
                        <div className="text-xs text-muted-foreground">{holiday.date}</div>
                      </div>
                      <Button variant="ghost" onClick={() => setSettings((current) => ({ ...current, operational: { ...current.operational, holiday_calendar: current.operational.holiday_calendar.filter((_, itemIndex) => itemIndex !== index) } }))}>Remove</Button>
                    </div>
                  ))
                )}
              </div>
            </div>
          </section>

          <section className="grid gap-6 rounded-lg border bg-card p-6 lg:grid-cols-3">
            <div className="space-y-3">
              <h2 className="text-lg font-semibold">Attendance Policy</h2>
              <Input type="number" placeholder="Late grace minutes" value={settings.attendance_policy.late_grace_minutes} onChange={(e) => setSettings((current) => ({ ...current, attendance_policy: { ...current.attendance_policy, late_grace_minutes: Number(e.target.value) || 0 } }))} />
              <Input type="number" placeholder="Early departure grace" value={settings.attendance_policy.early_departure_grace_minutes} onChange={(e) => setSettings((current) => ({ ...current, attendance_policy: { ...current.attendance_policy, early_departure_grace_minutes: Number(e.target.value) || 0 } }))} />
              <Input type="number" placeholder="Break grace minutes" value={settings.attendance_policy.break_grace_minutes} onChange={(e) => setSettings((current) => ({ ...current, attendance_policy: { ...current.attendance_policy, break_grace_minutes: Number(e.target.value) || 0 } }))} />
              <Input type="number" placeholder="Auto checkout hours" value={settings.attendance_policy.auto_checkout_hours} onChange={(e) => setSettings((current) => ({ ...current, attendance_policy: { ...current.attendance_policy, auto_checkout_hours: Number(e.target.value) || 0 } }))} />
              <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={settings.attendance_policy.regularization_requires_approval} onChange={(e) => setSettings((current) => ({ ...current, attendance_policy: { ...current.attendance_policy, regularization_requires_approval: e.target.checked } }))} /> Require approval for regularization</label>
              <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={settings.attendance_policy.allow_manual_attendance_adjustments} onChange={(e) => setSettings((current) => ({ ...current, attendance_policy: { ...current.attendance_policy, allow_manual_attendance_adjustments: e.target.checked } }))} /> Allow manual attendance adjustments</label>
            </div>

            <div className="space-y-3">
              <h2 className="text-lg font-semibold">Kiosk Defaults</h2>
              <Input type="number" placeholder="Heartbeat grace minutes" value={settings.kiosk_defaults.heartbeat_grace_minutes} onChange={(e) => setSettings((current) => ({ ...current, kiosk_defaults: { ...current.kiosk_defaults, heartbeat_grace_minutes: Number(e.target.value) || 0 } }))} />
              <Input type="number" placeholder="Offline sync window hours" value={settings.kiosk_defaults.offline_sync_window_hours} onChange={(e) => setSettings((current) => ({ ...current, kiosk_defaults: { ...current.kiosk_defaults, offline_sync_window_hours: Number(e.target.value) || 0 } }))} />
              <Input placeholder="Default location label" value={settings.kiosk_defaults.default_location} onChange={(e) => setSettings((current) => ({ ...current, kiosk_defaults: { ...current.kiosk_defaults, default_location: e.target.value } }))} />
              <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={settings.kiosk_defaults.require_pin_fallback} onChange={(e) => setSettings((current) => ({ ...current, kiosk_defaults: { ...current.kiosk_defaults, require_pin_fallback: e.target.checked } }))} /> Require PIN fallback</label>
            </div>

            <div className="space-y-3">
              <h2 className="text-lg font-semibold">Data Retention</h2>
              <Input type="number" placeholder="Attendance log days" value={settings.data_retention.attendance_log_days} onChange={(e) => setSettings((current) => ({ ...current, data_retention: { ...current.data_retention, attendance_log_days: Number(e.target.value) || 0 } }))} />
              <Input type="number" placeholder="Audit log days" value={settings.data_retention.audit_log_days} onChange={(e) => setSettings((current) => ({ ...current, data_retention: { ...current.data_retention, audit_log_days: Number(e.target.value) || 0 } }))} />
              <Input type="number" placeholder="Inactive user purge days" value={settings.data_retention.inactive_user_purge_days} onChange={(e) => setSettings((current) => ({ ...current, data_retention: { ...current.data_retention, inactive_user_purge_days: Number(e.target.value) || 0 } }))} />
            </div>
          </section>

          <section className="grid gap-6 rounded-lg border bg-card p-6 lg:grid-cols-[1.1fr_0.9fr]">
            <div className="space-y-4">
              <div>
                <h2 className="text-lg font-semibold">Shift Templates & Roster Rules</h2>
                <p className="text-sm text-muted-foreground">Define standard shifts for roster planning and attendance rule defaults.</p>
              </div>
              <div className="space-y-3">
                {settings.shift_templates.length === 0 ? (
                  <div className="rounded-md border border-dashed p-4 text-sm text-muted-foreground">No shift templates saved yet.</div>
                ) : (
                  settings.shift_templates.map((template) => (
                    <div key={template.id || template.name} className="rounded-md border p-4">
                      <div className="flex items-start justify-between gap-4">
                        <div>
                          <div className="flex items-center gap-2">
                            <h3 className="font-medium">{template.name}</h3>
                            {template.is_default && <span className="rounded-full bg-primary px-2 py-0.5 text-xs text-primary-foreground">Default</span>}
                          </div>
                          <div className="text-sm text-muted-foreground">{template.start_time} - {template.end_time} • Grace {template.grace_minutes} min</div>
                          <div className="text-xs text-muted-foreground">{template.days.join(', ') || 'No days selected'}</div>
                          {template.notes ? <div className="mt-1 text-sm">{template.notes}</div> : null}
                        </div>
                        <div className="flex gap-2">
                          <Button variant="outline" onClick={() => setShiftDraft(template)}>Edit</Button>
                          <Button variant="ghost" onClick={() => void deleteShift(template.id)}>Delete</Button>
                        </div>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>

            <div className="space-y-3 rounded-lg border bg-background p-4">
              <h3 className="font-semibold">{shiftDraft.id ? 'Edit Shift Template' : 'New Shift Template'}</h3>
              <Input placeholder="Template name" value={shiftDraft.name} onChange={(e) => setShiftDraft((current) => ({ ...current, name: e.target.value }))} />
              <div className="grid gap-3 md:grid-cols-2">
                <Input type="time" value={shiftDraft.start_time} onChange={(e) => setShiftDraft((current) => ({ ...current, start_time: e.target.value }))} />
                <Input type="time" value={shiftDraft.end_time} onChange={(e) => setShiftDraft((current) => ({ ...current, end_time: e.target.value }))} />
              </div>
              <Input type="number" placeholder="Grace minutes" value={shiftDraft.grace_minutes} onChange={(e) => setShiftDraft((current) => ({ ...current, grace_minutes: Number(e.target.value) || 0 }))} />
              <textarea className="min-h-[96px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm" placeholder="Notes or roster rules" value={shiftDraft.notes} onChange={(e) => setShiftDraft((current) => ({ ...current, notes: e.target.value }))} />
              <div className="flex flex-wrap gap-2">
                {weekdayOptions.map((day) => (
                  <button
                    key={day}
                    type="button"
                    onClick={() => toggleShiftDay(day)}
                    className={`rounded-full border px-3 py-1 text-sm ${shiftDraft.days.includes(day) ? 'border-primary bg-primary text-primary-foreground' : 'border-border bg-background text-foreground'}`}
                  >
                    {day.slice(0, 3).toUpperCase()}
                  </button>
                ))}
              </div>
              <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={shiftDraft.is_default} onChange={(e) => setShiftDraft((current) => ({ ...current, is_default: e.target.checked }))} /> Set as default shift</label>
              <div className="flex gap-2">
                <Button onClick={() => void saveShift()} disabled={isSavingShift}>{isSavingShift ? 'Saving...' : 'Save Shift'}</Button>
                <Button variant="outline" onClick={() => setShiftDraft({ name: '', start_time: '09:00', end_time: '18:00', days: ['monday', 'tuesday', 'wednesday', 'thursday', 'friday'], grace_minutes: 10, notes: '', is_default: false })}>Clear</Button>
              </div>
            </div>
          </section>
        </>
      )}
    </div>
  )
}
