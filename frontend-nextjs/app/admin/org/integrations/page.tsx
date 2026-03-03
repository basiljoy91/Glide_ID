'use client'

import { useEffect, useState } from 'react'
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

const PROVIDERS = ['workday', 'sap', 'bamboohr', 'custom']

export default function OrgIntegrationsPage() {
  const { isAuthenticated, user, token } = useAuthStore()
  const router = useRouter()
  const [integrations, setIntegrations] = useState<Integration[]>([])
  const [isLoading, setIsLoading] = useState(true)

  const [provider, setProvider] = useState('workday')
  const [apiKey, setApiKey] = useState('')
  const [apiSecret, setApiSecret] = useState('')
  const [configJson, setConfigJson] = useState('{}')
  const [creating, setCreating] = useState(false)

  const [editingId, setEditingId] = useState<string | null>(null)
  const [editProvider, setEditProvider] = useState('workday')
  const [editApiKey, setEditApiKey] = useState('')
  const [editApiSecret, setEditApiSecret] = useState('')
  const [editConfigJson, setEditConfigJson] = useState('{}')

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
      const resp = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/hrms/integrations`,
        { headers }
      )
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to load integrations')
      }
      const data = await resp.json()
      setIntegrations(Array.isArray(data) ? data : [])
    } catch (e: any) {
      toast.error(e.message || 'Failed to load integrations')
    } finally {
      setIsLoading(false)
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

      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/hrms/integrations`,
        {
          method: 'POST',
          headers,
          body: JSON.stringify({
            provider,
            api_key: apiKey,
            api_secret: apiSecret.trim() ? apiSecret : undefined,
            config: parsedConfig,
          }),
        }
      )
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to create integration')
      }
      toast.success('Integration created')
      setApiKey('')
      setApiSecret('')
      setConfigJson('{}')
      setProvider('workday')
      await fetchIntegrations()
    } catch (e: any) {
      toast.error(e.message || 'Failed to create integration')
    } finally {
      setCreating(false)
    }
  }

  const beginEdit = (i: Integration) => {
    setEditingId(i.id)
    setEditProvider(i.provider)
    setEditApiKey('')
    setEditApiSecret('')
    setEditConfigJson(JSON.stringify(i.config || {}, null, 2))
  }

  const cancelEdit = () => {
    setEditingId(null)
    setEditProvider('workday')
    setEditApiKey('')
    setEditApiSecret('')
    setEditConfigJson('{}')
  }

  const saveEdit = async (id: string) => {
    let parsedConfig: any = {}
    try {
      parsedConfig = editConfigJson.trim() ? JSON.parse(editConfigJson) : {}
    } catch {
      toast.error('Config must be valid JSON')
      return
    }

    try {
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/hrms/integrations/${id}`,
        {
          method: 'PUT',
          headers,
          body: JSON.stringify({
            provider: editProvider,
            api_key: editApiKey.trim() ? editApiKey : undefined,
            api_secret: editApiSecret.trim() ? editApiSecret : undefined,
            config: parsedConfig,
          }),
        }
      )
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to update integration')
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
      const resp = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/hrms/integrations/${id}/toggle`,
        {
          method: 'PATCH',
          headers,
          body: JSON.stringify({ is_active: !isActive }),
        }
      )
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
      const resp = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/hrms/integrations/${id}/test`,
        {
          method: 'POST',
          headers,
        }
      )
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
          <div className="md:col-span-2">
            <Label htmlFor="api-key">API Key</Label>
            <Input
              id="api-key"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              className="mt-1"
              placeholder="Provider API key"
            />
          </div>
        </div>
        <div className="grid md:grid-cols-3 gap-4">
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
          <div className="md:col-span-2">
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
        <Button type="submit" disabled={creating}>
          {creating ? 'Saving...' : 'Create Integration'}
        </Button>
      </form>

      <div className="border rounded-lg bg-card">
        <div className="border-b px-4 py-2 text-sm font-semibold text-muted-foreground">
          Existing Integrations
        </div>
        {isLoading ? (
          <div className="p-4 text-sm text-muted-foreground">Loading...</div>
        ) : integrations.length === 0 ? (
          <div className="p-4 text-sm text-muted-foreground">No integrations configured yet.</div>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left">
                <th className="px-4 py-2">Provider</th>
                <th className="px-4 py-2">Status</th>
                <th className="px-4 py-2">Last Sync</th>
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
                      i.provider
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
                        <textarea
                          value={editConfigJson}
                          onChange={(e) => setEditConfigJson(e.target.value)}
                          className="flex min-h-[88px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                        />
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
                      </div>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
