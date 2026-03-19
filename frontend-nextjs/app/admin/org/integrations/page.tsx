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
  config?: Record<string, unknown>
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

interface WebhookEvent {
  id: string
  status: string
  retry_count: number
  error_message?: string | null
  next_retry_at?: string | null
  created_at: string
  processed_at?: string | null
}

interface SyncConflict {
  id: string
  external_record_id: string
  field_name: string
  local_value: unknown
  external_value: unknown
  status: string
  created_at: string
  resolved_at?: string | null
}

interface MappingTestResult {
  mapped: Record<string, unknown>
  missing_fields: string[]
}

interface DryRunPreview {
  mapped: Record<string, unknown>
  missing_fields: string[]
}

interface DryRunConflict {
  external_record_id: string
  field_name: string
  local_value: unknown
  external_value: unknown
}

interface DryRunResult {
  valid_count: number
  invalid_count: number
  preview: DryRunPreview[]
  conflicts: DryRunConflict[]
}

interface RotatedCredentials {
  api_key: string
  api_secret: string
}

type MappingRow = { source: string; target: string }

const PROVIDERS = [
  'workday',
  'sap',
  'bamboohr',
  'adp',
  'gusto',
  'quickbooks',
  'okta',
  'azure_ad',
  'google_workspace',
  'custom',
]

const DEFAULT_MAPPING_ROWS: MappingRow[] = [
  { source: 'employee_id', target: 'employee_id' },
  { source: 'first_name', target: 'first_name' },
  { source: 'last_name', target: 'last_name' },
  { source: 'email', target: 'email' },
  { source: 'designation', target: 'designation' },
]

const DEFAULT_SAMPLE = JSON.stringify(
  {
    employee_id: 'EMP-1001',
    first_name: 'Ada',
    last_name: 'Lovelace',
    email: 'ada@example.com',
    designation: 'Engineering Manager',
  },
  null,
  2
)

const DEFAULT_DRY_RUN = JSON.stringify(
  [
    {
      employee_id: 'EMP-1001',
      first_name: 'Ada',
      last_name: 'Lovelace',
      email: 'ada@example.com',
      designation: 'Engineering Manager',
    },
    {
      employee_id: 'EMP-1002',
      first_name: 'Grace',
      last_name: 'Hopper',
      email: 'grace@example.com',
      designation: 'Staff Engineer',
    },
  ],
  null,
  2
)

function formatJson(value: unknown) {
  return JSON.stringify(value, null, 2)
}

function formatSchedule(schedule: SyncSchedule | null | undefined) {
  if (!schedule) return 'Manual only'
  return `${schedule.frequency} at ${schedule.time_of_day} ${schedule.timezone}`
}

function getMappingRowsFromConfig(config?: Record<string, unknown>): MappingRow[] {
  const raw = config?.field_mapping
  if (!Array.isArray(raw)) return DEFAULT_MAPPING_ROWS
  const rows = raw
    .map((item) => {
      const row = item as Record<string, unknown>
      return {
        source: String(row?.source || '').trim(),
        target: String(row?.target || '').trim(),
      }
    })
    .filter((row) => row.source && row.target)
  return rows.length ? rows : DEFAULT_MAPPING_ROWS
}

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
  const [mappingRows, setMappingRows] = useState<MappingRow[]>(DEFAULT_MAPPING_ROWS)
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
  const [openOpsId, setOpenOpsId] = useState<string | null>(null)

  const [opsMappingRows, setOpsMappingRows] = useState<Record<string, MappingRow[]>>({})
  const [mappingSamples, setMappingSamples] = useState<Record<string, string>>({})
  const [mappingResults, setMappingResults] = useState<Record<string, MappingTestResult | null>>({})
  const [dryRunInputs, setDryRunInputs] = useState<Record<string, string>>({})
  const [dryRunResults, setDryRunResults] = useState<Record<string, DryRunResult | null>>({})
  const [webhookEvents, setWebhookEvents] = useState<Record<string, WebhookEvent[]>>({})
  const [syncConflicts, setSyncConflicts] = useState<Record<string, SyncConflict[]>>({})
  const [rotatedCredentials, setRotatedCredentials] = useState<Record<string, RotatedCredentials | null>>({})
  const [busyKey, setBusyKey] = useState<string | null>(null)

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

  const authHeaders = (includeJson = false): Record<string, string> => {
    const headers: Record<string, string> = {}
    if (includeJson) headers['Content-Type'] = 'application/json'
    if (token) headers.Authorization = `Bearer ${token}`
    return headers
  }

  const fetchJson = async <T,>(path: string, init?: RequestInit): Promise<T> => {
    const resp = await fetch(`${base}${path}`, init)
    const data = await resp.json().catch(() => ({}))
    if (!resp.ok) {
      const error = typeof data?.error === 'string' ? data.error : 'Request failed'
      throw new Error(error)
    }
    return data as T
  }

  const fetchIntegrations = async () => {
    try {
      setIsLoading(true)
      const list = await fetchJson<Integration[]>('/api/v1/hrms/integrations', {
        headers: authHeaders(),
      })
      setIntegrations(Array.isArray(list) ? list : [])
      await Promise.all(
        (Array.isArray(list) ? list : []).map(async (integration) => {
          await loadSchedule(integration.id)
        })
      )
    } catch (e: unknown) {
      const message = e instanceof Error ? e.message : 'Failed to load integrations'
      toast.error(message)
    } finally {
      setIsLoading(false)
    }
  }

  const loadSchedule = async (id: string): Promise<SyncSchedule | null> => {
    try {
      const data = await fetchJson<SyncSchedule | { schedule: null }>(`/api/v1/hrms/integrations/${id}/schedule`, {
        headers: authHeaders(),
      })
      const schedule = 'id' in data ? data : null
      setSyncSchedules((prev) => ({ ...prev, [id]: schedule }))
      return schedule
    } catch {
      setSyncSchedules((prev) => ({ ...prev, [id]: null }))
      return null
    }
  }

  const loadLogs = async (id: string) => {
    try {
      const data = await fetchJson<SyncLog[]>(`/api/v1/hrms/integrations/${id}/sync-logs?limit=20`, {
        headers: authHeaders(),
      })
      setSyncLogs((prev) => ({ ...prev, [id]: Array.isArray(data) ? data : [] }))
    } catch {
      setSyncLogs((prev) => ({ ...prev, [id]: [] }))
    }
  }

  const loadWebhookEvents = async (id: string) => {
    try {
      const data = await fetchJson<WebhookEvent[]>(`/api/v1/hrms/integrations/${id}/webhook-events`, {
        headers: authHeaders(),
      })
      setWebhookEvents((prev) => ({ ...prev, [id]: Array.isArray(data) ? data : [] }))
    } catch {
      setWebhookEvents((prev) => ({ ...prev, [id]: [] }))
    }
  }

  const loadConflicts = async (id: string) => {
    try {
      const data = await fetchJson<SyncConflict[]>(`/api/v1/hrms/integrations/${id}/conflicts`, {
        headers: authHeaders(),
      })
      setSyncConflicts((prev) => ({ ...prev, [id]: Array.isArray(data) ? data : [] }))
    } catch {
      setSyncConflicts((prev) => ({ ...prev, [id]: [] }))
    }
  }

  const parseJsonValue = <T,>(value: string, label: string): T => {
    try {
      return JSON.parse(value) as T
    } catch {
      throw new Error(`${label} must be valid JSON`)
    }
  }

  const getActiveMappingRows = (id: string, fallbackConfig?: Record<string, unknown>) => {
    const rows = opsMappingRows[id] || getMappingRowsFromConfig(fallbackConfig)
    return rows.filter((row) => row.source.trim() && row.target.trim())
  }

  const ensureOpsDefaults = (integration: Integration) => {
    setOpsMappingRows((prev) => {
      if (prev[integration.id]) return prev
      return { ...prev, [integration.id]: getMappingRowsFromConfig(integration.config) }
    })
    setMappingSamples((prev) => {
      if (prev[integration.id]) return prev
      return { ...prev, [integration.id]: DEFAULT_SAMPLE }
    })
    setDryRunInputs((prev) => {
      if (prev[integration.id]) return prev
      return { ...prev, [integration.id]: DEFAULT_DRY_RUN }
    })
  }

  const toggleOperations = async (integration: Integration) => {
    if (openOpsId === integration.id) {
      setOpenOpsId(null)
      return
    }
    ensureOpsDefaults(integration)
    setOpenOpsId(integration.id)
    await Promise.all([loadWebhookEvents(integration.id), loadConflicts(integration.id)])
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!apiKey.trim()) {
      toast.error('API key is required')
      return
    }

    try {
      setCreating(true)
      const parsedConfig = parseJsonValue<Record<string, unknown>>(configJson.trim() || '{}', 'Config')
      const filteredMapping = mappingRows.filter((row) => row.source.trim() && row.target.trim())
      if (filteredMapping.length) {
        parsedConfig.field_mapping = filteredMapping
      }

      const created = await fetchJson<Integration>('/api/v1/hrms/integrations', {
        method: 'POST',
        headers: authHeaders(true),
        body: JSON.stringify({
          provider,
          api_key: apiKey,
          api_secret: apiSecret.trim() ? apiSecret : undefined,
          webhook_url: webhookUrl.trim() ? webhookUrl.trim() : undefined,
          config: parsedConfig,
        }),
      })

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
      setMappingRows(DEFAULT_MAPPING_ROWS)
      setSyncFrequency('manual')
      setSyncDay('1')
      setSyncTime('09:00')
      await fetchIntegrations()
    } catch (e: unknown) {
      const message = e instanceof Error ? e.message : 'Failed to create integration'
      toast.error(message)
    } finally {
      setCreating(false)
    }
  }

  const beginEdit = async (integration: Integration) => {
    setEditingId(integration.id)
    setEditProvider(integration.provider)
    setEditApiKey('')
    setEditApiSecret('')
    setEditWebhookUrl(integration.webhook_url || '')
    setEditConfigJson(JSON.stringify(integration.config || {}, null, 2))
    setEditMappingRows(getMappingRowsFromConfig(integration.config))
    const existing = await loadSchedule(integration.id)
    if (existing) {
      setEditSyncFrequency(existing.frequency)
      setEditSyncDay(String(existing.day_of_week ?? 1))
      setEditSyncTime(existing.time_of_day)
      setEditSyncTimezone(existing.timezone || 'UTC')
      return
    }
    setEditSyncFrequency('manual')
    setEditSyncDay('1')
    setEditSyncTime('09:00')
    setEditSyncTimezone(Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC')
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

  const saveSchedule = async (id: string, schedule: Record<string, unknown>) => {
    await fetchJson<SyncSchedule>(`/api/v1/hrms/integrations/${id}/schedule`, {
      method: 'PUT',
      headers: authHeaders(true),
      body: JSON.stringify(schedule),
    })
    await loadSchedule(id)
  }

  const saveEdit = async (id: string) => {
    try {
      const parsedConfig = parseJsonValue<Record<string, unknown>>(editConfigJson.trim() || '{}', 'Config')
      const filteredMapping = editMappingRows.filter((row) => row.source.trim() && row.target.trim())
      if (filteredMapping.length) {
        parsedConfig.field_mapping = filteredMapping
      }

      await fetchJson<Integration>(`/api/v1/hrms/integrations/${id}`, {
        method: 'PUT',
        headers: authHeaders(true),
        body: JSON.stringify({
          provider: editProvider,
          api_key: editApiKey.trim() ? editApiKey : undefined,
          api_secret: editApiSecret.trim() ? editApiSecret : undefined,
          webhook_url: editWebhookUrl.trim() ? editWebhookUrl.trim() : undefined,
          config: parsedConfig,
        }),
      })

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
    } catch (e: unknown) {
      const message = e instanceof Error ? e.message : 'Failed to update integration'
      toast.error(message)
    }
  }

  const toggleIntegration = async (id: string, isActive: boolean) => {
    try {
      await fetchJson(`/api/v1/hrms/integrations/${id}/toggle`, {
        method: 'PATCH',
        headers: authHeaders(true),
        body: JSON.stringify({ is_active: !isActive }),
      })
      toast.success(isActive ? 'Integration disabled' : 'Integration enabled')
      await fetchIntegrations()
    } catch (e: unknown) {
      const message = e instanceof Error ? e.message : 'Failed to toggle integration'
      toast.error(message)
    }
  }

  const testIntegration = async (id: string) => {
    try {
      const data = await fetchJson<{ message?: string }>(`/api/v1/hrms/integrations/${id}/test`, {
        method: 'POST',
        headers: authHeaders(),
      })
      toast.success(data.message || 'Integration test finished')
      await fetchIntegrations()
    } catch (e: unknown) {
      const message = e instanceof Error ? e.message : 'Failed to test integration'
      toast.error(message)
    }
  }

  const runSync = async (id: string) => {
    try {
      await fetchJson(`/api/v1/hrms/integrations/${id}/sync`, {
        method: 'POST',
        headers: authHeaders(true),
        body: JSON.stringify({ message: 'Manual sync from admin portal' }),
      })
      toast.success('Sync triggered')
      await loadLogs(id)
      await loadConflicts(id)
      await fetchIntegrations()
    } catch (e: unknown) {
      const message = e instanceof Error ? e.message : 'Failed to sync integration'
      toast.error(message)
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
  ) => rows.map((row, index) => (index === idx ? { ...row, [key]: value } : row))

  const runMappingTest = async (integration: Integration) => {
    try {
      setBusyKey(`mapping-${integration.id}`)
      const sample = parseJsonValue<Record<string, unknown>>(
        mappingSamples[integration.id] || DEFAULT_SAMPLE,
        'Mapping sample'
      )
      const result = await fetchJson<MappingTestResult>(`/api/v1/hrms/integrations/${integration.id}/mapping-test`, {
        method: 'POST',
        headers: authHeaders(true),
        body: JSON.stringify({
          sample,
          field_mapping: getActiveMappingRows(integration.id, integration.config),
        }),
      })
      setMappingResults((prev) => ({ ...prev, [integration.id]: result }))
      toast.success('Mapping test completed')
    } catch (e: unknown) {
      const message = e instanceof Error ? e.message : 'Failed to test field mapping'
      toast.error(message)
    } finally {
      setBusyKey(null)
    }
  }

  const runDryRun = async (integration: Integration) => {
    try {
      setBusyKey(`dry-run-${integration.id}`)
      const records = parseJsonValue<Record<string, unknown>[]>(
        dryRunInputs[integration.id] || DEFAULT_DRY_RUN,
        'Dry-run records'
      )
      if (!Array.isArray(records) || records.length === 0) {
        throw new Error('Dry-run records must be a non-empty JSON array')
      }
      const result = await fetchJson<DryRunResult>(`/api/v1/hrms/integrations/${integration.id}/dry-run`, {
        method: 'POST',
        headers: authHeaders(true),
        body: JSON.stringify({
          records,
          field_mapping: getActiveMappingRows(integration.id, integration.config),
        }),
      })
      setDryRunResults((prev) => ({ ...prev, [integration.id]: result }))
      await loadConflicts(integration.id)
      toast.success('Dry-run validation finished')
    } catch (e: unknown) {
      const message = e instanceof Error ? e.message : 'Failed to run dry-run validation'
      toast.error(message)
    } finally {
      setBusyKey(null)
    }
  }

  const retryWebhook = async (integrationID: string, eventID: string) => {
    try {
      setBusyKey(`retry-${eventID}`)
      await fetchJson(`/api/v1/hrms/integrations/${integrationID}/webhook-events/${eventID}/retry`, {
        method: 'POST',
        headers: authHeaders(),
      })
      await loadWebhookEvents(integrationID)
      toast.success('Webhook event re-queued')
    } catch (e: unknown) {
      const message = e instanceof Error ? e.message : 'Failed to retry webhook event'
      toast.error(message)
    } finally {
      setBusyKey(null)
    }
  }

  const resolveConflict = async (integrationID: string, conflictID: string, resolution: string) => {
    try {
      setBusyKey(`conflict-${conflictID}`)
      await fetchJson(`/api/v1/hrms/integrations/${integrationID}/conflicts/${conflictID}`, {
        method: 'PATCH',
        headers: authHeaders(true),
        body: JSON.stringify({ resolution }),
      })
      await loadConflicts(integrationID)
      toast.success('Conflict resolved')
    } catch (e: unknown) {
      const message = e instanceof Error ? e.message : 'Failed to resolve conflict'
      toast.error(message)
    } finally {
      setBusyKey(null)
    }
  }

  const rotateCredentials = async (integrationID: string) => {
    try {
      setBusyKey(`rotate-${integrationID}`)
      const creds = await fetchJson<RotatedCredentials>(`/api/v1/hrms/integrations/${integrationID}/rotate-credentials`, {
        method: 'POST',
        headers: authHeaders(),
      })
      setRotatedCredentials((prev) => ({ ...prev, [integrationID]: creds }))
      toast.success('Credentials rotated')
    } catch (e: unknown) {
      const message = e instanceof Error ? e.message : 'Failed to rotate credentials'
      toast.error(message)
    } finally {
      setBusyKey(null)
    }
  }

  if (!isAuthenticated || !user) return null

  return (
    <div className="container mx-auto space-y-8 px-4 py-8">
      <div className="space-y-2">
        <h1 className="text-2xl font-bold">Integration Hub</h1>
        <p className="text-muted-foreground">
          Manage payroll connectors, directory sync, webhook failures, validation dry-runs, and credential rotation.
        </p>
        <p className="text-xs text-muted-foreground">
          Supported connectors include Workday, SAP, BambooHR, ADP, Gusto, QuickBooks, Okta, Azure AD, Google
          Workspace, and custom providers.
        </p>
      </div>

      <form onSubmit={handleCreate} className="space-y-4 rounded-lg border bg-card p-4">
        <h2 className="font-semibold">Add Integration</h2>
        <div className="grid gap-4 md:grid-cols-3">
          <div>
            <Label htmlFor="provider">Provider</Label>
            <select
              id="provider"
              value={provider}
              onChange={(e) => setProvider(e.target.value)}
              className="mt-1 flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            >
              {PROVIDERS.map((item) => (
                <option key={item} value={item}>
                  {item}
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

        <div className="grid gap-4 md:grid-cols-3">
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

        <div className="space-y-2 rounded-md border p-3">
          <div className="text-sm font-medium">Field mappings</div>
          {mappingRows.map((row, idx) => (
            <div key={`${row.source}-${idx}`} className="grid gap-2 md:grid-cols-2">
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

        <div className="grid gap-4 md:grid-cols-4">
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

      <div className="rounded-lg border bg-card">
        <div className="border-b px-4 py-3 text-sm font-semibold text-muted-foreground">Configured Integrations</div>
        {isLoading ? (
          <div className="space-y-3 p-4">
            {Array.from({ length: 4 }).map((_, index) => (
              <div key={index} className="skeleton h-28 w-full" />
            ))}
          </div>
        ) : integrations.length === 0 ? (
          <div className="p-4 text-sm text-muted-foreground">No integrations configured yet.</div>
        ) : (
          <div className="space-y-4 p-4">
            {integrations.map((integration) => {
              const schedule = syncSchedules[integration.id]
              const mappingRowsForOps = opsMappingRows[integration.id] || getMappingRowsFromConfig(integration.config)
              const logs = syncLogs[integration.id] || []
              const events = webhookEvents[integration.id] || []
              const conflicts = syncConflicts[integration.id] || []
              const dryRun = dryRunResults[integration.id]
              const mappingResult = mappingResults[integration.id]
              const rotated = rotatedCredentials[integration.id]

              return (
                <div key={integration.id} className="space-y-4 rounded-lg border p-4">
                  <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
                    <div className="space-y-2">
                      <div className="flex flex-wrap items-center gap-2">
                        <h3 className="text-lg font-semibold capitalize">{integration.provider.replace(/_/g, ' ')}</h3>
                        <span
                          className={
                            integration.is_active
                              ? 'rounded-full bg-green-100 px-2 py-1 text-xs font-medium text-green-700'
                              : 'rounded-full bg-slate-100 px-2 py-1 text-xs font-medium text-slate-600'
                          }
                        >
                          {integration.is_active ? 'Active' : 'Inactive'}
                        </span>
                      </div>
                      <div className="grid gap-2 text-sm text-muted-foreground md:grid-cols-2">
                        <div>Last sync: {integration.last_sync_at ? new Date(integration.last_sync_at).toLocaleString() : '—'}</div>
                        <div>Schedule: {formatSchedule(schedule)}</div>
                        <div className="md:col-span-2 break-all">Webhook URL: {integration.webhook_url || '—'}</div>
                      </div>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      {editingId === integration.id ? null : (
                        <>
                          <Button size="sm" variant="outline" onClick={() => beginEdit(integration)}>
                            Edit
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => void toggleIntegration(integration.id, integration.is_active)}
                          >
                            {integration.is_active ? 'Disable' : 'Enable'}
                          </Button>
                          <Button size="sm" variant="outline" onClick={() => void testIntegration(integration.id)}>
                            Test
                          </Button>
                          <Button size="sm" variant="outline" onClick={() => void runSync(integration.id)}>
                            Sync now
                          </Button>
                          <Button size="sm" variant="outline" onClick={() => void toggleLogs(integration.id)}>
                            {openLogsId === integration.id ? 'Hide logs' : 'View logs'}
                          </Button>
                          <Button size="sm" onClick={() => void toggleOperations(integration)}>
                            {openOpsId === integration.id ? 'Hide tools' : 'Operate'}
                          </Button>
                        </>
                      )}
                    </div>
                  </div>

                  {editingId === integration.id ? (
                    <div className="space-y-3 rounded-md border p-3">
                      <div className="grid gap-3 md:grid-cols-2">
                        <select
                          value={editProvider}
                          onChange={(e) => setEditProvider(e.target.value)}
                          className="h-10 rounded-md border border-input bg-background px-3 text-sm"
                        >
                          {PROVIDERS.map((item) => (
                            <option key={item} value={item}>
                              {item}
                            </option>
                          ))}
                        </select>
                        <Input
                          value={editWebhookUrl}
                          onChange={(e) => setEditWebhookUrl(e.target.value)}
                          placeholder="Webhook URL"
                        />
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
                      </div>

                      <textarea
                        value={editConfigJson}
                        onChange={(e) => setEditConfigJson(e.target.value)}
                        className="flex min-h-[110px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                      />

                      <div className="space-y-2 rounded-md border p-3">
                        <div className="text-sm font-medium">Field mappings</div>
                        {editMappingRows.map((row, idx) => (
                          <div key={`${row.source}-${idx}`} className="grid gap-2 md:grid-cols-2">
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

                      <div className="grid gap-3 md:grid-cols-4">
                        <select
                          value={editSyncFrequency}
                          onChange={(e) => setEditSyncFrequency(e.target.value)}
                          className="h-10 rounded-md border border-input bg-background px-3 text-sm"
                        >
                          <option value="manual">Manual</option>
                          <option value="hourly">Hourly</option>
                          <option value="daily">Daily</option>
                          <option value="weekly">Weekly</option>
                          <option value="monthly">Monthly</option>
                        </select>
                        <select
                          value={editSyncDay}
                          onChange={(e) => setEditSyncDay(e.target.value)}
                          className="h-10 rounded-md border border-input bg-background px-3 text-sm"
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
                        <Input type="time" value={editSyncTime} onChange={(e) => setEditSyncTime(e.target.value)} />
                        <Input
                          value={editSyncTimezone}
                          onChange={(e) => setEditSyncTimezone(e.target.value)}
                          placeholder="Timezone"
                        />
                      </div>

                      <div className="flex flex-wrap justify-end gap-2">
                        <Button size="sm" onClick={() => void saveEdit(integration.id)}>
                          Save
                        </Button>
                        <Button size="sm" variant="outline" onClick={cancelEdit}>
                          Cancel
                        </Button>
                      </div>
                    </div>
                  ) : null}

                  {openLogsId === integration.id ? (
                    <div className="rounded-md border bg-muted/20 p-3 text-sm">
                      <div className="mb-2 font-medium text-foreground">Sync history</div>
                      {logs.length === 0 ? (
                        <div className="text-muted-foreground">No sync history yet.</div>
                      ) : (
                        <div className="space-y-2">
                          {logs.map((log) => (
                            <div key={log.id} className="rounded border p-2">
                              <div className="font-medium">{log.status}</div>
                              <div className="text-xs text-muted-foreground">
                                Started {new Date(log.started_at).toLocaleString()}
                              </div>
                              {log.message ? <div className="text-xs text-muted-foreground">{log.message}</div> : null}
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  ) : null}

                  {openOpsId === integration.id ? (
                    <div className="grid gap-4 lg:grid-cols-2">
                      <div className="space-y-4">
                        <div className="space-y-3 rounded-md border p-3">
                          <div>
                            <div className="font-medium">Field-mapping tester</div>
                            <div className="text-xs text-muted-foreground">
                              Validate source payload mapping before enabling directory sync.
                            </div>
                          </div>
                          <div className="space-y-2">
                            {mappingRowsForOps.map((row, idx) => (
                              <div key={`${row.source}-${idx}`} className="grid gap-2 md:grid-cols-2">
                                <Input
                                  value={row.source}
                                  onChange={(e) =>
                                    setOpsMappingRows((prev) => ({
                                      ...prev,
                                      [integration.id]: updateMappingRow(
                                        prev[integration.id] || mappingRowsForOps,
                                        idx,
                                        'source',
                                        e.target.value
                                      ),
                                    }))
                                  }
                                  placeholder="Source field"
                                />
                                <Input
                                  value={row.target}
                                  onChange={(e) =>
                                    setOpsMappingRows((prev) => ({
                                      ...prev,
                                      [integration.id]: updateMappingRow(
                                        prev[integration.id] || mappingRowsForOps,
                                        idx,
                                        'target',
                                        e.target.value
                                      ),
                                    }))
                                  }
                                  placeholder="Target field"
                                />
                              </div>
                            ))}
                            <Button
                              type="button"
                              size="sm"
                              variant="outline"
                              onClick={() =>
                                setOpsMappingRows((prev) => ({
                                  ...prev,
                                  [integration.id]: [...(prev[integration.id] || mappingRowsForOps), { source: '', target: '' }],
                                }))
                              }
                            >
                              Add mapping
                            </Button>
                          </div>
                          <div>
                            <Label>Sample payload JSON</Label>
                            <textarea
                              value={mappingSamples[integration.id] || DEFAULT_SAMPLE}
                              onChange={(e) =>
                                setMappingSamples((prev) => ({ ...prev, [integration.id]: e.target.value }))
                              }
                              className="mt-1 flex min-h-[180px] w-full rounded-md border border-input bg-background px-3 py-2 font-mono text-xs"
                            />
                          </div>
                          <Button
                            size="sm"
                            onClick={() => void runMappingTest(integration)}
                            disabled={busyKey === `mapping-${integration.id}`}
                          >
                            {busyKey === `mapping-${integration.id}` ? 'Testing...' : 'Run mapping test'}
                          </Button>
                          {mappingResult ? (
                            <div className="rounded-md bg-muted/30 p-3 text-xs">
                              <div className="font-medium text-foreground">Mapped output</div>
                              <pre className="mt-2 overflow-x-auto whitespace-pre-wrap">{formatJson(mappingResult.mapped)}</pre>
                              <div className="mt-3 font-medium text-foreground">Missing fields</div>
                              <div className="mt-1 text-muted-foreground">
                                {mappingResult.missing_fields.length === 0
                                  ? 'None'
                                  : mappingResult.missing_fields.join(', ')}
                              </div>
                            </div>
                          ) : null}
                        </div>

                        <div className="space-y-3 rounded-md border p-3">
                          <div>
                            <div className="font-medium">Import validation dry-run</div>
                            <div className="text-xs text-muted-foreground">
                              Run directory-sync validation, inspect preview rows, and surface conflicts before commit.
                            </div>
                          </div>
                          <div>
                            <Label>Dry-run records JSON</Label>
                            <textarea
                              value={dryRunInputs[integration.id] || DEFAULT_DRY_RUN}
                              onChange={(e) =>
                                setDryRunInputs((prev) => ({ ...prev, [integration.id]: e.target.value }))
                              }
                              className="mt-1 flex min-h-[220px] w-full rounded-md border border-input bg-background px-3 py-2 font-mono text-xs"
                            />
                          </div>
                          <Button
                            size="sm"
                            onClick={() => void runDryRun(integration)}
                            disabled={busyKey === `dry-run-${integration.id}`}
                          >
                            {busyKey === `dry-run-${integration.id}` ? 'Running...' : 'Run dry-run'}
                          </Button>
                          {dryRun ? (
                            <div className="space-y-3 rounded-md bg-muted/30 p-3 text-xs">
                              <div className="grid gap-3 md:grid-cols-3">
                                <div>
                                  <div className="font-medium text-foreground">Valid</div>
                                  <div>{dryRun.valid_count}</div>
                                </div>
                                <div>
                                  <div className="font-medium text-foreground">Invalid</div>
                                  <div>{dryRun.invalid_count}</div>
                                </div>
                                <div>
                                  <div className="font-medium text-foreground">Conflicts</div>
                                  <div>{dryRun.conflicts.length}</div>
                                </div>
                              </div>
                              <div>
                                <div className="font-medium text-foreground">Preview</div>
                                <div className="mt-2 space-y-2">
                                  {dryRun.preview.slice(0, 3).map((preview, idx) => (
                                    <div key={idx} className="rounded border bg-background p-2">
                                      <div className="text-[11px] text-muted-foreground">
                                        Missing: {preview.missing_fields.length ? preview.missing_fields.join(', ') : 'None'}
                                      </div>
                                      <pre className="mt-2 overflow-x-auto whitespace-pre-wrap">{formatJson(preview.mapped)}</pre>
                                    </div>
                                  ))}
                                </div>
                              </div>
                            </div>
                          ) : null}
                        </div>
                      </div>

                      <div className="space-y-4">
                        <div className="space-y-3 rounded-md border p-3">
                          <div>
                            <div className="font-medium">Webhook retry queue</div>
                            <div className="text-xs text-muted-foreground">
                              Inspect failed webhook deliveries and retry them after fixing provider-side issues.
                            </div>
                          </div>
                          {events.length === 0 ? (
                            <div className="text-sm text-muted-foreground">No webhook retries pending.</div>
                          ) : (
                            <div className="space-y-2">
                              {events.map((event) => (
                                <div key={event.id} className="rounded border p-3 text-xs">
                                  <div className="flex flex-wrap items-center justify-between gap-2">
                                    <div className="font-medium text-foreground">{event.status}</div>
                                    <Button
                                      size="sm"
                                      variant="outline"
                                      onClick={() => void retryWebhook(integration.id, event.id)}
                                      disabled={busyKey === `retry-${event.id}`}
                                    >
                                      {busyKey === `retry-${event.id}` ? 'Retrying...' : 'Retry'}
                                    </Button>
                                  </div>
                                  <div className="mt-2 text-muted-foreground">
                                    Created {new Date(event.created_at).toLocaleString()} • Retries {event.retry_count}
                                  </div>
                                  {event.next_retry_at ? (
                                    <div className="text-muted-foreground">
                                      Next retry {new Date(event.next_retry_at).toLocaleString()}
                                    </div>
                                  ) : null}
                                  {event.error_message ? <div className="mt-2 text-red-600">{event.error_message}</div> : null}
                                </div>
                              ))}
                            </div>
                          )}
                        </div>

                        <div className="space-y-3 rounded-md border p-3">
                          <div>
                            <div className="font-medium">Sync conflict resolution</div>
                            <div className="text-xs text-muted-foreground">
                              Resolve field-level collisions from dry-run or live directory sync attempts.
                            </div>
                          </div>
                          {conflicts.length === 0 ? (
                            <div className="text-sm text-muted-foreground">No open sync conflicts.</div>
                          ) : (
                            <div className="space-y-2">
                              {conflicts.map((conflict) => (
                                <div key={conflict.id} className="rounded border p-3 text-xs">
                                  <div className="flex flex-wrap items-center justify-between gap-2">
                                    <div className="font-medium text-foreground">
                                      {conflict.external_record_id} • {conflict.field_name}
                                    </div>
                                    <div className="text-muted-foreground">{conflict.status}</div>
                                  </div>
                                  <div className="mt-2 grid gap-2 md:grid-cols-2">
                                    <div className="rounded bg-muted/30 p-2">
                                      <div className="font-medium text-foreground">Local</div>
                                      <pre className="mt-1 overflow-x-auto whitespace-pre-wrap">{formatJson(conflict.local_value)}</pre>
                                    </div>
                                    <div className="rounded bg-muted/30 p-2">
                                      <div className="font-medium text-foreground">External</div>
                                      <pre className="mt-1 overflow-x-auto whitespace-pre-wrap">{formatJson(conflict.external_value)}</pre>
                                    </div>
                                  </div>
                                  <div className="mt-3 flex flex-wrap gap-2">
                                    <Button
                                      size="sm"
                                      variant="outline"
                                      onClick={() => void resolveConflict(integration.id, conflict.id, 'keep_local')}
                                      disabled={busyKey === `conflict-${conflict.id}`}
                                    >
                                      Keep local
                                    </Button>
                                    <Button
                                      size="sm"
                                      onClick={() => void resolveConflict(integration.id, conflict.id, 'accept_external')}
                                      disabled={busyKey === `conflict-${conflict.id}`}
                                    >
                                      Accept external
                                    </Button>
                                  </div>
                                </div>
                              ))}
                            </div>
                          )}
                        </div>

                        <div className="space-y-3 rounded-md border p-3">
                          <div>
                            <div className="font-medium">Credentials rotation</div>
                            <div className="text-xs text-muted-foreground">
                              Rotate API credentials and use the newly issued key/secret for secure reprovisioning.
                            </div>
                          </div>
                          <Button
                            size="sm"
                            onClick={() => void rotateCredentials(integration.id)}
                            disabled={busyKey === `rotate-${integration.id}`}
                          >
                            {busyKey === `rotate-${integration.id}` ? 'Rotating...' : 'Rotate credentials'}
                          </Button>
                          {rotated ? (
                            <div className="rounded-md bg-amber-50 p-3 text-xs text-amber-900">
                              <div className="font-medium">New credentials</div>
                              <div className="mt-2 break-all">API key: {rotated.api_key}</div>
                              <div className="break-all">API secret: {rotated.api_secret}</div>
                              <div className="mt-2 text-[11px]">These values are shown once in the admin portal after rotation.</div>
                            </div>
                          ) : null}
                        </div>
                      </div>
                    </div>
                  ) : null}
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}
