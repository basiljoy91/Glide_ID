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

  useEffect(() => {
    if (!isAuthenticated || !user) {
      router.push('/admin/login')
      return
    }
    if (!['org_admin', 'hr'].includes(user.role)) {
      router.push('/dashboard')
      return
    }
    fetchIntegrations()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, user?.role])

  const fetchIntegrations = async () => {
    try {
      setIsLoading(true)
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/hrms/integrations`,
        {
          headers,
        }
      )
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to load integrations')
      }
      const data = await resp.json()
      setIntegrations(data)
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

  if (!isAuthenticated || !user) return null

  return (
    <div className="container mx-auto px-4 py-8 space-y-8">
      <div>
        <h1 className="text-2xl font-bold mb-2">Integration Hub</h1>
        <p className="text-muted-foreground">
          Connect HR and payroll systems via secure webhooks and API keys.
        </p>
      </div>

      <form
        onSubmit={handleCreate}
        className="border rounded-lg p-4 space-y-4 bg-card"
      >
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
              placeholder="Paste provider API key or secret"
            />
          </div>
        </div>

        <div className="grid md:grid-cols-3 gap-4">
          <div className="md:col-span-1">
            <Label htmlFor="api-secret">API Secret (optional)</Label>
            <Input
              id="api-secret"
              value={apiSecret}
              onChange={(e) => setApiSecret(e.target.value)}
              className="mt-1"
              placeholder="Optional secret (used for webhook HMAC)"
            />
          </div>
          <div className="md:col-span-2">
            <Label htmlFor="config">Config (JSON)</Label>
            <textarea
              id="config"
              value={configJson}
              onChange={(e) => setConfigJson(e.target.value)}
              className="mt-1 flex min-h-[88px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              placeholder='{"region":"us","endpoint":"..."}'
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
          <div className="p-4 text-sm text-muted-foreground">
            No integrations configured yet.
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left">
                <th className="px-4 py-2">Provider</th>
                <th className="px-4 py-2">Status</th>
                <th className="px-4 py-2">Last Sync</th>
              </tr>
            </thead>
            <tbody>
              {integrations.map((i) => (
                <tr key={i.id} className="border-b last:border-b-0">
                  <td className="px-4 py-2">{i.provider}</td>
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
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

