'use client'

import { useEffect, useState } from 'react'
import { useAuthStore } from '@/store/useStore'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import toast from 'react-hot-toast'

interface Kiosk {
  id: string
  name: string
  code: string
  status: string
  last_heartbeat_at?: string | null
}

const STATUS_OPTIONS = ['all', 'active', 'inactive', 'maintenance', 'revoked'] as const
type StatusFilter = (typeof STATUS_OPTIONS)[number]

export default function OrgKiosksPage() {
  const { isAuthenticated, user, token } = useAuthStore()
  const router = useRouter()
  const [kiosks, setKiosks] = useState<Kiosk[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [status, setStatus] = useState<StatusFilter>('all')

  useEffect(() => {
    if (!isAuthenticated || !user) {
      router.push('/admin/login')
      return
    }
    if (!['org_admin'].includes(user.role)) {
      router.push('/dashboard')
      return
    }
    fetchKiosks('all')
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, user?.role])

  const fetchKiosks = async (statusFilter: StatusFilter = status) => {
    try {
      setIsLoading(true)
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`
      const base = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      const qs = statusFilter && statusFilter !== 'all' ? `?status=${encodeURIComponent(statusFilter)}` : ''
      const resp = await fetch(
        `${base}/api/v1/kiosks${qs}`,
        {
          headers,
        }
      )
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to load kiosks')
      }
      const data = await resp.json()
      setKiosks(data)
    } catch (e: any) {
      toast.error(e.message || 'Failed to load kiosks')
    } finally {
      setIsLoading(false)
    }
  }

  const handleRevoke = async (id: string) => {
    if (!confirm('Revoke this kiosk? It will no longer be able to check in.')) return
    try {
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/kiosks/${id}`,
        {
          method: 'DELETE',
          headers,
        }
      )
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to revoke kiosk')
      }
      toast.success('Kiosk revoked')
      await fetchKiosks()
    } catch (e: any) {
      toast.error(e.message || 'Failed to revoke kiosk')
    }
  }

  const handleRotateSecret = async (id: string, code: string) => {
    if (!confirm(`Rotate HMAC secret for kiosk ${code}? Existing kiosk clients must be updated.`)) return
    try {
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/kiosks/${id}/rotate-secret`,
        {
          method: 'POST',
          headers,
        }
      )
      const data = await resp.json().catch(() => ({}))
      if (!resp.ok) {
        throw new Error(data.error || 'Failed to rotate kiosk secret')
      }

      const secret = data.hmac_secret as string | undefined
      if (!secret) {
        throw new Error('Secret rotation succeeded but no secret was returned')
      }

      try {
        await navigator.clipboard.writeText(secret)
        toast.success('New kiosk secret generated and copied to clipboard')
      } catch {
        toast.success('New kiosk secret generated')
      }

      window.alert(
        `Kiosk ${code} new HMAC secret (save this now):\n\n${secret}\n\nThis value is shown only now.`
      )
    } catch (e: any) {
      toast.error(e.message || 'Failed to rotate kiosk secret')
    }
  }

  const isHealthy = (k: Kiosk) => {
    if (!k.last_heartbeat_at) return false
    const t = new Date(k.last_heartbeat_at).getTime()
    if (!Number.isFinite(t)) return false
    return Date.now() - t <= 10 * 60 * 1000
  }

  if (!isAuthenticated || !user) return null

  return (
    <div className="container mx-auto px-4 py-8 space-y-8">
      <div>
        <h1 className="text-2xl font-bold mb-2">Kiosks</h1>
        <p className="text-muted-foreground">
          Monitor and manage active check-in kiosks. Revoke compromised devices instantly.
        </p>
      </div>

      <div className="border rounded-lg bg-card">
        <div className="border-b px-4 py-3 flex flex-col md:flex-row md:items-center md:justify-between gap-2">
          <div className="text-sm font-semibold text-muted-foreground">Registered Kiosks</div>
          <div className="flex items-center gap-2">
            <div className="text-xs text-muted-foreground">Status</div>
            <select
              value={status}
              onChange={async (e) => {
                const v = e.target.value as StatusFilter
                setStatus(v)
                await fetchKiosks(v)
              }}
              className="flex h-9 rounded-md border border-input bg-background px-3 text-sm"
            >
              {STATUS_OPTIONS.map((s) => (
                <option key={s} value={s}>
                  {s}
                </option>
              ))}
            </select>
          </div>
        </div>
        {isLoading ? (
          <div className="p-4 text-sm text-muted-foreground">Loading...</div>
        ) : kiosks.length === 0 ? (
          <div className="p-4 text-sm text-muted-foreground">
            No kiosks registered yet. The onboarding flow creates the first kiosk automatically.
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left">
                <th className="px-4 py-2">Name</th>
                <th className="px-4 py-2">Code</th>
                <th className="px-4 py-2">Status</th>
                <th className="px-4 py-2">Health</th>
                <th className="px-4 py-2 hidden md:table-cell">Last Heartbeat</th>
                <th className="px-4 py-2 text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {kiosks.map((k) => (
                <tr key={k.id} className="border-b last:border-b-0">
                  <td className="px-4 py-2">{k.name}</td>
                  <td className="px-4 py-2 font-mono">{k.code}</td>
                  <td className="px-4 py-2">
                    <span
                      className={
                        k.status === 'active'
                          ? 'text-xs text-green-600 dark:text-green-300'
                          : 'text-xs text-muted-foreground'
                      }
                    >
                      {k.status}
                    </span>
                  </td>
                  <td className="px-4 py-2">
                    {k.status !== 'active' ? (
                      <span className="text-xs text-muted-foreground">—</span>
                    ) : isHealthy(k) ? (
                      <span className="text-xs text-green-600 dark:text-green-300">Healthy</span>
                    ) : (
                      <span className="text-xs text-yellow-700 dark:text-yellow-300">Stale</span>
                    )}
                  </td>
                  <td className="px-4 py-2 hidden md:table-cell">
                    {k.last_heartbeat_at
                      ? new Date(k.last_heartbeat_at).toLocaleString()
                      : '—'}
                  </td>
                  <td className="px-4 py-2 text-right">
                    {k.status === 'active' && (
                      <div className="flex justify-end gap-2">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleRotateSecret(k.id, k.code)}
                        >
                          Rotate Secret
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleRevoke(k.id)}
                          className="text-destructive hover:text-destructive/80"
                        >
                          Revoke
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

