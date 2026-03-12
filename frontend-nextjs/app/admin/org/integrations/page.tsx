'use client'

import { useEffect, useMemo, useState } from 'react'
import { useAuthStore } from '@/store/useStore'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import toast from 'react-hot-toast'

interface Integration {
  id: string
  provider: string
  is_active: boolean
  last_sync_at?: string | null
  webhook_url?: string | null
  config?: Record<string, any>
}

interface SyncSchedule {
  id: string
  integration_id: string
  frequency: string
  day_of_week?: number | null
  time_of_day: string
  timezone: string
  is_active: boolean
  last_run_at?: string | null
  next_run_at?: string | null
}

interface SyncLog {
  id: string
  status: string
  message?: string | null
  started_at: string
  completed_at?: string | null
}

const PROVIDERS = ['workday', 'sap', 'bamboohr', 'custom']

type MappingRow = { source: string; target: string }

export default function OrgIntegrationsPage() {
  const { isAuthenticated, user, token } = useAuthStore()
  const router = useRouter()
  const base = useMemo(() => process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080', [])
  const [integrations, setIntegrations] = useState<Integration[]>([])
  const [isLoading, setIsLoading] = useState(true)

  const [provider, setProvider] = useState('workday')
  const [apiKey, setApiKey] = useState('')
  const [apiSecret, setApiSecret] = useState('')
  const [webhookUrl, setWebhookUrl] = useState('')
  const [configJson, setConfigJson] = useState('{}')
  const [mappingRows, setMappingRows] = useState<MappingRow[]>([{ source: 'employee_id', target: 'employee_id' }])
  const [syncFrequency, setSyncFrequency] = useState('manual')
  const [syncDay, setSyncDay] = useState('1')
  const [syncTime, setSyncTime] = useState('09:00')
  const [syncTimezone, setSyncTimezone] = useState(
    Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
  )
  const [creating, setCreating] = useState(false)

  const [editingId, setEditingId] = useState<string | null>(null)
  const [editProvider, setEditProvider] = useState('workday')
  const [editApiKey, setEditApiKey] = useState('')
  const [editApiSecret, setEditApiSecret] = useState('')
  const [editWebhookUrl, setEditWebhookUrl] = useState('')
  const [editConfigJson, setEditConfigJson] = useState('{}')
  const [editMappingRows, setEditMappingRows] = useState<MappingRow[]>([])
  const [editSyncFrequency, setEditSyncFrequency] = useState('manual')
  const [editSyncDay, setEditSyncDay] = useState('1')
  const [editSyncTime, setEditSyncTime] = useState('09:00')
  const [editSyncTimezone, setEditSyncTimezone] = useState(
    Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
  )

  const [syncSchedules, setSyncSchedules] = useState<Record<string, SyncSchedule | null>>({})
  const [syncLogs, setSyncLogs] = useState<Record<string, SyncLog[]>>({})
  const [openLogsId, setOpenLogsId] = useState<string | null>(null)

  useEffect(() => {
    if (!isAuthenticated || !user) {
      router.push('/admin/login')
      return
    }
    if (!['org_admin', 'hr'].includes(user.role)) {
      router.push('/dashboard')
      return
    }
    void fetchIntegrations()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, user?.role])

  const fetchIntegrations = async () => {
    try {
      setIsLoading(true)
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(`${base}/api/v1/hrms/integrations`, { headers })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to load integrations')
      }
      const data = await resp.json()
      const list = Array.isArray(data) ? data : []
      setIntegrations(list)
      await Promise.all(
        list.map(async (i: Integration) => {
          await loadSchedule(i.id)
        })
      )
    } catch (e: any) {
      toast.error(e.message || 'Failed to load integrations')
    } finally {
      setIsLoading(false)
    }
  }

  const loadSchedule = async (id: string): Promise<SyncSchedule | null> => {
    try {
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(`${base}/api/v1/hrms/integrations/${id}/schedule`, { headers })
      if (!resp.ok) return null
      const data = await resp.json()
      const schedule = data?.id ? data : null
      setSyncSchedules((prev) => ({ ...prev, [id]: schedule }))
      return schedule
    } catch {
      setSyncSchedules((prev) => ({ ...prev, [id]: null }))
      return null
    }
  }

  const loadLogs = async (id: string) => {
    try {
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(`${base}/api/v1/hrms/integrations/${id}/sync-logs?limit=20`, { headers })
      if (!resp.ok) return
      const data = await resp.json()
      setSyncLogs((prev) => ({ ...prev, [id]: Array.isArray(data) ? data : [] }))
    } catch {
      setSyncLogs((prev) => ({ ...prev, [id]: [] }))
    }
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!apiKey.trim()) {
      toast.error('API key is required')
      return
    }
    try {
      setCreating(true)
      let parsedConfig: any = {}
      try {
        parsedConfig = configJson.trim() ? JSON.parse(configJson) : {}
      } catch {
        toast.error('Config must be valid JSON')
        return
      }

      const mapping = mappingRows.filter((r) => r.source.trim() && r.target.trim())
      if (mapping.length) {
        parsedConfig.field_mapping = mapping
      }

      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(`${base}/api/v1/hrms/integrations`, {
        method: 'POST',
        headers,
        body: JSON.stringify({
          provider,
          api_key: apiKey,
          api_secret: apiSecret.trim() ? apiSecret : undefined,
          webhook_url: webhookUrl.trim() ? webhookUrl.trim() : undefined,
          config: parsedConfig,
        }),
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to create integration')
      }
      const created = await resp.json()
      if (syncFrequency !== 'manual') {
        await saveSchedule(created.id, {
          frequency: syncFrequency,
          day_of_week: syncFrequency === 'weekly' ? Number(syncDay) : undefined,
          time_of_day: syncTime,
          timezone: syncTimezone,
        })
      }
      toast.success('Integration created')
      setApiKey('')
      setApiSecret('')
      setConfigJson('{}')
      setProvider('workday')
      setWebhookUrl('')
      setMappingRows([{ source: 'employee_id', target: 'employee_id' }])
      setSyncFrequency('manual')
      await fetchIntegrations()
    } catch (e: any) {
      toast.error(e.message || 'Failed to create integration')
    } finally {
      setCreating(false)
    }
  }

  const beginEdit = async (i: Integration) => {
    setEditingId(i.id)
    setEditProvider(i.provider)
    setEditApiKey('')
    setEditApiSecret('')
    setEditWebhookUrl(i.webhook_url || '')
    setEditConfigJson(JSON.stringify(i.config || {}, null, 2))
    const mapping = (i.config?.field_mapping as MappingRow[]) || []
    setEditMappingRows(mapping.length ? mapping : [{ source: 'employee_id', target: 'employee_id' }])
    const existing = await loadSchedule(i.id)
    if (existing) {
      setEditSyncFrequency(existing.frequency)
      setEditSyncDay(String(existing.day_of_week ?? 1))
      setEditSyncTime(existing.time_of_day)
      setEditSyncTimezone(existing.timezone || 'UTC')
    } else {
      setEditSyncFrequency('manual')
      setEditSyncDay('1')
      setEditSyncTime('09:00')
      setEditSyncTimezone(Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC')
    }
  }

  const cancelEdit = () => {
    setEditingId(null)
    setEditProvider('workday')
    setEditApiKey('')
    setEditApiSecret('')
    setEditWebhookUrl('')
    setEditConfigJson('{}')
    setEditMappingRows([])
  }

  const saveSchedule = async (id: string, schedule: any) => {
    const headers: Record<string, string> = { 'Content-Type': 'application/json' }
    if (token) headers.Authorization = `Bearer ${token}`
    const resp = await fetch(`${base}/api/v1/hrms/integrations/${id}/schedule`, {
      method: 'PUT',
      headers,
      body: JSON.stringify(schedule),
    })
    if (!resp.ok) {
      const err = await resp.json().catch(() => ({}))
      throw new Error(err.error || 'Failed to save schedule')
    }
    await loadSchedule(id)
  }

  const saveEdit = async (id: string) => {
    let parsedConfig: any = {}
    try {
      parsedConfig = editConfigJson.trim() ? JSON.parse(editConfigJson) : {}
    } catch {
      toast.error('Config must be valid JSON')
      return
    }

    const mapping = editMappingRows.filter((r) => r.source.trim() && r.target.trim())
    if (mapping.length) {
      parsedConfig.field_mapping = mapping
    }

    try {
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(`${base}/api/v1/hrms/integrations/${id}`, {
        method: 'PUT',
        headers,
        body: JSON.stringify({
          provider: editProvider,
          api_key: editApiKey.trim() ? editApiKey : undefined,
          api_secret: editApiSecret.trim() ? editApiSecret : undefined,
          webhook_url: editWebhookUrl.trim() ? editWebhookUrl.trim() : undefined,
          config: parsedConfig,
        }),
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to update integration')
      }

      if (editSyncFrequency !== 'manual') {
        await saveSchedule(id, {
          frequency: editSyncFrequency,
          day_of_week: editSyncFrequency === 'weekly' ? Number(editSyncDay) : undefined,
          time_of_day: editSyncTime,
          timezone: editSyncTimezone,
        })
      }

      toast.success('Integration updated')
      cancelEdit()
      await fetchIntegrations()
    } catch (e: any) {
      toast.error(e.message || 'Failed to update integration')
    }
  }

  const toggleIntegration = async (id: string, isActive: boolean) => {
    try {
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(`${base}/api/v1/hrms/integrations/${id}/toggle`, {
        method: 'PATCH',
        headers,
        body: JSON.stringify({ is_active: !isActive }),
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to toggle integration')
      }
      toast.success(isActive ? 'Integration disabled' : 'Integration enabled')
      await fetchIntegrations()
    } catch (e: any) {
      toast.error(e.message || 'Failed to toggle integration')
    }
  }

  const testIntegration = async (id: string) => {
    try {
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(`${base}/api/v1/hrms/integrations/${id}/test`, {
        method: 'POST',
        headers,
      })
      const data = await resp.json().catch(() => ({}))
      if (!resp.ok) {
        throw new Error(data.error || 'Failed to test integration')
      }
      toast.success(data.message || 'Integration test finished')
      await fetchIntegrations()
    } catch (e: any) {
      toast.error(e.message || 'Failed to test integration')
    }
  }

  const runSync = async (id: string) => {
    try {
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(`${base}/api/v1/hrms/integrations/${id}/sync`, {
        method: 'POST',
        headers,
        body: JSON.stringify({ message: 'Manual sync from admin portal' }),
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to sync integration')
      }
      toast.success('Sync triggered')
      await loadLogs(id)
      await fetchIntegrations()
    } catch (e: any) {
      toast.error(e.message || 'Failed to sync integration')
    }
  }

  const toggleLogs = async (id: string) => {
    if (openLogsId === id) {
      setOpenLogsId(null)
      return
    }
    setOpenLogsId(id)
    await loadLogs(id)
  }

  const updateMappingRow = (
    rows: MappingRow[],
    idx: number,
    key: 'source' | 'target',
    value: string
  ) => rows.map((r, i) => (i === idx ? { ...r, [key]: value } : r))

  if (!isAuthenticated || !user) return null

  return (
    <div className="container mx-auto px-4 py-8 space-y-8">
      <div>
        <h1 className="text-2xl font-bold mb-2">Integration Hub</h1>
        <p className="text-muted-foreground">Connect HR/payroll systems and operate integrations.</p>
      </div>

      <form onSubmit={handleCreate} className="border rounded-lg p-4 space-y-4 bg-card">
        <h2 className="font-semibold">Add Integration</h2>
        <div className="grid md:grid-cols-3 gap-4">
          <div>
            <Label htmlFor="provider">Provider</Label>
            <select
              id="provider"
              value={provider}
              onChange={(e) => setProvider(e.target.value)}
              className="mt-1 flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            >
              {PROVIDERS.map((p) => (
                <option key={p} value={p}>
                  {p}
                </option>
              ))}
            </select>
          </div>
          <div>
            <Label htmlFor="api-key">API Key</Label>
            <Input
              id="api-key"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              className="mt-1"
              placeholder="Provider API key"
            />
          </div>
          <div>
            <Label htmlFor="api-secret">API Secret</Label>
            <Input
              id="api-secret"
              value={apiSecret}
              onChange={(e) => setApiSecret(e.target.value)}
              className="mt-1"
              placeholder="Optional secret"
            />
          </div>
        </div>
        <div className="grid md:grid-cols-3 gap-4">
          <div className="md:col-span-2">
            <Label htmlFor="webhook">Webhook URL</Label>
            <Input
              id="webhook"
              value={webhookUrl}
              onChange={(e) => setWebhookUrl(e.target.value)}
              className="mt-1"
              placeholder="https://your-system/webhooks/hrms"
            />
          </div>
          <div>
            <Label htmlFor="config">Config (JSON)</Label>
            <textarea
              id="config"
              value={configJson}
              onChange={(e) => setConfigJson(e.target.value)}
              className="mt-1 flex min-h-[88px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              placeholder='{"endpoint":"https://..."}'
            />
          </div>
        </div>
        <div className="border rounded-md p-3 space-y-2">
          <div className="text-sm font-medium">Field mappings</div>
          {mappingRows.map((row, idx) => (
            <div key={idx} className="grid grid-cols-2 gap-2">
              <Input
                value={row.source}
                onChange={(e) => setMappingRows(updateMappingRow(mappingRows, idx, 'source', e.target.value))}
                placeholder="Source field"
              />
              <Input
                value={row.target}
                onChange={(e) => setMappingRows(updateMappingRow(mappingRows, idx, 'target', e.target.value))}
                placeholder="Target field"
              />
            </div>
          ))}
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => setMappingRows((prev) => [...prev, { source: '', target: '' }])}
          >
            Add mapping
          </Button>
        </div>
        <div className="grid md:grid-cols-4 gap-4">
          <div>
            <Label>Sync frequency</Label>
            <select
              value={syncFrequency}
              onChange={(e) => setSyncFrequency(e.target.value)}
              className="mt-1 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            >
              <option value="manual">Manual</option>
              <option value="hourly">Hourly</option>
              <option value="daily">Daily</option>
              <option value="weekly">Weekly</option>
              <option value="monthly">Monthly</option>
            </select>
          </div>
          <div>
            <Label>Day</Label>
            <select
              value={syncDay}
              onChange={(e) => setSyncDay(e.target.value)}
              className="mt-1 h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              disabled={syncFrequency !== 'weekly'}
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
            <Label>Time</Label>
            <Input type="time" value={syncTime} onChange={(e) => setSyncTime(e.target.value)} />
          </div>
          <div>
            <Label>Timezone</Label>
            <Input value={syncTimezone} onChange={(e) => setSyncTimezone(e.target.value)} />
          </div>
        </div>
        <Button type="submit" disabled={creating}>
          {creating ? 'Saving...' : 'Create Integration'}
        </Button>
      </form>

      <div className="border rounded-lg bg-card">
        <div className="border-b px-4 py-2 text-sm font-semibold text-muted-foreground">
          Existing Integrations
        </div>
        {isLoading ? (
          <div className="p-4 space-y-3">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="skeleton h-16 w-full" />
            ))}
          </div>
        ) : integrations.length === 0 ? (
          <div className="p-4 text-sm text-muted-foreground">No integrations configured yet.</div>
        ) : (
          <>
            <div className="md:hidden divide-y">
              {integrations.map((i) => (
                <div key={i.id} className="p-4 space-y-3">
                  <div className="flex items-start justify-between gap-3">
                    <div className="font-medium">{i.provider}</div>
                    <span
                      className={
                        i.is_active
                          ? 'text-xs text-green-600 dark:text-green-300'
                          : 'text-xs text-muted-foreground'
                      }
                    >
                      {i.is_active ? 'Active' : 'Inactive'}
                    </span>
                  </div>
                  <div className="text-xs text-muted-foreground">
                    Last sync: {i.last_sync_at ? new Date(i.last_sync_at).toLocaleString() : '—'}
                  </div>
                  <div className="text-xs text-muted-foreground">Webhook: {i.webhook_url || '—'}</div>
                  {syncSchedules[i.id] ? (
                    <div className="text-xs text-muted-foreground">
                      Schedule: {syncSchedules[i.id]?.frequency} at {syncSchedules[i.id]?.time_of_day}{' '}
                      {syncSchedules[i.id]?.timezone}
                    </div>
                  ) : null}
                  {editingId === i.id ? (
                    <div className="space-y-2">
                      <select
                        value={editProvider}
                        onChange={(e) => setEditProvider(e.target.value)}
                        className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                      >
                        {PROVIDERS.map((p) => (
                          <option key={p} value={p}>
                            {p}
                          </option>
                        ))}
                      </select>
                      <Input
                        value={editApiKey}
                        onChange={(e) => setEditApiKey(e.target.value)}
                        placeholder="New API key (optional)"
                      />
                      <Input
                        value={editApiSecret}
                        onChange={(e) => setEditApiSecret(e.target.value)}
                        placeholder="New API secret (optional)"
                      />
                      <Input
                        value={editWebhookUrl}
                        onChange={(e) => setEditWebhookUrl(e.target.value)}
                        placeholder="Webhook URL"
                      />
                      <textarea
                        value={editConfigJson}
                        onChange={(e) => setEditConfigJson(e.target.value)}
                        className="flex min-h-[88px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                      />
                      <div className="border rounded-md p-2 space-y-2">
                        <div className="text-xs font-medium">Field mappings</div>
                        {editMappingRows.map((row, idx) => (
                          <div key={idx} className="grid grid-cols-2 gap-2">
                            <Input
                              value={row.source}
                              onChange={(e) =>
                                setEditMappingRows(updateMappingRow(editMappingRows, idx, 'source', e.target.value))
                              }
                            />
                            <Input
                              value={row.target}
                              onChange={(e) =>
                                setEditMappingRows(updateMappingRow(editMappingRows, idx, 'target', e.target.value))
                              }
                            />
                          </div>
                        ))}
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          onClick={() => setEditMappingRows((prev) => [...prev, { source: '', target: '' }])}
                        >
                          Add mapping
                        </Button>
                      </div>
                      <div className="grid grid-cols-2 gap-2">
                        <select
                          value={editSyncFrequency}
                          onChange={(e) => setEditSyncFrequency(e.target.value)}
                          className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                        >
                          <option value="manual">Manual</option>
                          <option value="hourly">Hourly</option>
                          <option value="daily">Daily</option>
                          <option value="weekly">Weekly</option>
                          <option value="monthly">Monthly</option>
                        </select>
                        <Input type="time" value={editSyncTime} onChange={(e) => setEditSyncTime(e.target.value)} />
                        <select
                          value={editSyncDay}
                          onChange={(e) => setEditSyncDay(e.target.value)}
                          className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                          disabled={editSyncFrequency !== 'weekly'}
                        >
                          <option value="1">Monday</option>
                          <option value="2">Tuesday</option>
                          <option value="3">Wednesday</option>
                          <option value="4">Thursday</option>
                          <option value="5">Friday</option>
                          <option value="6">Saturday</option>
                          <option value="0">Sunday</option>
                        </select>
                        <Input
                          value={editSyncTimezone}
                          onChange={(e) => setEditSyncTimezone(e.target.value)}
                          placeholder="Timezone"
                        />
                      </div>
                      <div className="flex gap-2">
                        <Button size="sm" onClick={() => void saveEdit(i.id)}>
                          Save
                        </Button>
                        <Button size="sm" variant="outline" onClick={cancelEdit}>
                          Cancel
                        </Button>
                      </div>
                    </div>
                  ) : (
                    <div className="flex flex-wrap gap-2">
                      <Button size="sm" variant="outline" onClick={() => beginEdit(i)}>
                        Edit
                      </Button>
                      <Button size="sm" variant="outline" onClick={() => void toggleIntegration(i.id, i.is_active)}>
                        {i.is_active ? 'Disable' : 'Enable'}
                      </Button>
                      <Button size="sm" variant="outline" onClick={() => void testIntegration(i.id)}>
                        Test
                      </Button>
                      <Button size="sm" variant="outline" onClick={() => void runSync(i.id)}>
                        Sync now
                      </Button>
                      <Button size="sm" variant="outline" onClick={() => void toggleLogs(i.id)}>
                        {openLogsId === i.id ? 'Hide logs' : 'View logs'}
                      </Button>
                    </div>
                  )}
                  {openLogsId === i.id ? (
                    <div className="text-xs text-muted-foreground">
                      {(syncLogs[i.id] || []).length === 0
                        ? 'No sync history'
                        : (syncLogs[i.id] || []).map((log) => (
                            <div key={log.id} className="py-1">
                              {new Date(log.started_at).toLocaleString()} • {log.status}
                            </div>
                          ))}
                    </div>
                  ) : null}
                </div>
              ))}
            </div>

            <div className="hidden md:block overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b text-left">
                    <th className="px-4 py-2">Provider</th>
                    <th className="px-4 py-2">Status</th>
                    <th className="px-4 py-2">Last Sync</th>
                    <th className="px-4 py-2">Webhook</th>
                    <th className="px-4 py-2 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {integrations.map((i) => (
                    <tr key={i.id} className="border-b last:border-b-0 align-top">
                      <td className="px-4 py-2">
                        {editingId === i.id ? (
                          <select
                            value={editProvider}
                            onChange={(e) => setEditProvider(e.target.value)}
                            className="h-10 rounded-md border border-input bg-background px-3 text-sm"
                          >
                            {PROVIDERS.map((p) => (
                              <option key={p} value={p}>
                                {p}
                              </option>
                            ))}
                          </select>
                        ) : (
                          <div>
                            <div>{i.provider}</div>
                            {syncSchedules[i.id] ? (
                              <div className="text-xs text-muted-foreground">
                                {syncSchedules[i.id]?.frequency} at {syncSchedules[i.id]?.time_of_day}{' '}
                                {syncSchedules[i.id]?.timezone}
                              </div>
                            ) : null}
                          </div>
                        )}
                      </td>
                      <td className="px-4 py-2">
                        <span
                          className={
                            i.is_active
                              ? 'text-xs text-green-600 dark:text-green-300'
                              : 'text-xs text-muted-foreground'
                          }
                        >
                          {i.is_active ? 'Active' : 'Inactive'}
                        </span>
                      </td>
                      <td className="px-4 py-2">
                        {i.last_sync_at ? new Date(i.last_sync_at).toLocaleString() : '—'}
                      </td>
                      <td className="px-4 py-2">{i.webhook_url || '—'}</td>
                      <td className="px-4 py-2">
                        {editingId === i.id ? (
                          <div className="space-y-2">
                            <Input
                              value={editApiKey}
                              onChange={(e) => setEditApiKey(e.target.value)}
                              placeholder="New API key (optional)"
                            />
                            <Input
                              value={editApiSecret}
                              onChange={(e) => setEditApiSecret(e.target.value)}
                              placeholder="New API secret (optional)"
                            />
                            <Input
                              value={editWebhookUrl}
                              onChange={(e) => setEditWebhookUrl(e.target.value)}
                              placeholder="Webhook URL"
                            />
                            <textarea
                              value={editConfigJson}
                              onChange={(e) => setEditConfigJson(e.target.value)}
                              className="flex min-h-[88px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                            />
                            <div className="border rounded-md p-2 space-y-2">
                              <div className="text-xs font-medium">Field mappings</div>
                              {editMappingRows.map((row, idx) => (
                                <div key={idx} className="grid grid-cols-2 gap-2">
                                  <Input
                                    value={row.source}
                                    onChange={(e) =>
                                      setEditMappingRows(updateMappingRow(editMappingRows, idx, 'source', e.target.value))
                                    }
                                  />
                                  <Input
                                    value={row.target}
                                    onChange={(e) =>
                                      setEditMappingRows(updateMappingRow(editMappingRows, idx, 'target', e.target.value))
                                    }
                                  />
                                </div>
                              ))}
                              <Button
                                type="button"
                                size="sm"
                                variant="outline"
                                onClick={() => setEditMappingRows((prev) => [...prev, { source: '', target: '' }])}
                              >
                                Add mapping
                              </Button>
                            </div>
                            <div className="grid grid-cols-2 gap-2">
                              <select
                                value={editSyncFrequency}
                                onChange={(e) => setEditSyncFrequency(e.target.value)}
                                className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                              >
                                <option value="manual">Manual</option>
                                <option value="hourly">Hourly</option>
                                <option value="daily">Daily</option>
                                <option value="weekly">Weekly</option>
                                <option value="monthly">Monthly</option>
                              </select>
                              <Input type="time" value={editSyncTime} onChange={(e) => setEditSyncTime(e.target.value)} />
                              <select
                                value={editSyncDay}
                                onChange={(e) => setEditSyncDay(e.target.value)}
                                className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                                disabled={editSyncFrequency !== 'weekly'}
                              >
                                <option value="1">Monday</option>
                                <option value="2">Tuesday</option>
                                <option value="3">Wednesday</option>
                                <option value="4">Thursday</option>
                                <option value="5">Friday</option>
                                <option value="6">Saturday</option>
                                <option value="0">Sunday</option>
                              </select>
                              <Input
                                value={editSyncTimezone}
                                onChange={(e) => setEditSyncTimezone(e.target.value)}
                                placeholder="Timezone"
                              />
                            </div>
                            <div className="flex justify-end gap-2">
                              <Button size="sm" onClick={() => void saveEdit(i.id)}>
                                Save
                              </Button>
                              <Button size="sm" variant="outline" onClick={cancelEdit}>
                                Cancel
                              </Button>
                            </div>
                          </div>
                        ) : (
                          <div className="flex justify-end gap-2">
                            <Button size="sm" variant="outline" onClick={() => beginEdit(i)}>
                              Edit
                            </Button>
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => void toggleIntegration(i.id, i.is_active)}
                            >
                              {i.is_active ? 'Disable' : 'Enable'}
                            </Button>
                            <Button size="sm" variant="outline" onClick={() => void testIntegration(i.id)}>
                              Test
                            </Button>
                            <Button size="sm" variant="outline" onClick={() => void runSync(i.id)}>
                              Sync now
                            </Button>
                            <Button size="sm" variant="outline" onClick={() => void toggleLogs(i.id)}>
                              {openLogsId === i.id ? 'Hide logs' : 'View logs'}
                            </Button>
                          </div>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              {openLogsId ? (
                <div className="p-4 text-xs text-muted-foreground">
                  <div className="font-medium text-foreground mb-2">Sync history</div>
                  {(syncLogs[openLogsId] || []).length === 0 ? (
                    <div>No sync history yet.</div>
                  ) : (
                    (syncLogs[openLogsId] || []).map((log) => (
                      <div key={log.id} className="py-1">
                        {new Date(log.started_at).toLocaleString()} • {log.status}
                        {log.message ? ` • ${log.message}` : ''}
                      </div>
                    ))
                  )}
                </div>
              ) : null}
            </div>
          </>
        )}
      </div>
    </div>
  )
}
