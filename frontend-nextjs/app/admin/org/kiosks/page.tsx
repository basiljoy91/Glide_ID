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
  location?: string | null
  last_heartbeat_at?: string | null
}

interface KioskHistory {
  date: string
  activity_count: number
  anomalies: number
  last_activity_at?: string | null
}

const STATUS_OPTIONS = ['all', 'active', 'inactive', 'maintenance', 'revoked'] as const
type StatusFilter = (typeof STATUS_OPTIONS)[number]

export default function OrgKiosksPage() {
  const { isAuthenticated, user, token } = useAuthStore()
  const router = useRouter()
  const [kiosks, setKiosks] = useState<Kiosk[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [status, setStatus] = useState<StatusFilter>('all')
  const [pendingRevokeId, setPendingRevokeId] = useState<string | null>(null)
  const [pendingRotateId, setPendingRotateId] = useState<string | null>(null)
  const [busyId, setBusyId] = useState<string | null>(null)
  const [rotatedSecret, setRotatedSecret] = useState<{ code: string; secret: string } | null>(null)

  // New features state
  const [isAddOpen, setIsAddOpen] = useState(false)
  const [addName, setAddName] = useState('')
  const [addCode, setAddCode] = useState('')
  const [addLocation, setAddLocation] = useState('')
  const [isSaving, setIsSaving] = useState(false)

  const [editId, setEditId] = useState<string | null>(null)
  const [editName, setEditName] = useState('')
  const [editLocation, setEditLocation] = useState('')

  const [historyKioskId, setHistoryKioskId] = useState<string | null>(null)
  const [historyData, setHistoryData] = useState<KioskHistory[]>([])
  const [isLoadingHistory, setIsLoadingHistory] = useState(false)

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
      const resp = await fetch(`${base}/api/v1/kiosks${qs}`, { headers })
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

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!addName.trim() || !addCode.trim()) {
      toast.error('Name and Code are required')
      return
    }
    try {
      setIsSaving(true)
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/kiosks`,
        {
          method: 'POST',
          headers,
          body: JSON.stringify({ name: addName, code: addCode, location: addLocation }),
        }
      )
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to create kiosk')
      }
      toast.success('Kiosk created successfully')
      setIsAddOpen(false)
      setAddName('')
      setAddCode('')
      setAddLocation('')
      await fetchKiosks()
    } catch (e: any) {
      toast.error(e.message || 'Failed to create kiosk')
    } finally {
      setIsSaving(false)
    }
  }

  const beginEdit = (k: Kiosk) => {
    setEditId(k.id)
    setEditName(k.name)
    setEditLocation(k.location || '')
  }

  const saveEdit = async () => {
    if (!editId) return
    try {
      setIsSaving(true)
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/kiosks/${editId}`,
        {
          method: 'PUT',
          headers,
          body: JSON.stringify({ name: editName, location: editLocation }),
        }
      )
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to update kiosk')
      }
      toast.success('Kiosk updated')
      setEditId(null)
      await fetchKiosks()
    } catch (e: any) {
      toast.error(e.message || 'Failed to update kiosk')
    } finally {
      setIsSaving(false)
    }
  }

  const viewHistory = async (id: string) => {
    try {
      setHistoryKioskId(id)
      setIsLoadingHistory(true)
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/kiosks/${id}/history`,
        { headers }
      )
      if (resp.ok) {
        const data = await resp.json()
        setHistoryData(data)
      } else {
        throw new Error('Could not fetch history')
      }
    } catch (e: any) {
      toast.error(e.message || 'Failed to fetch kiosk history')
      setHistoryKioskId(null)
    } finally {
      setIsLoadingHistory(false)
    }
  }

  const revokeKiosk = async (id: string) => {
    try {
      setBusyId(id)
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
      setPendingRevokeId(null)
      await fetchKiosks()
    } catch (e: any) {
      toast.error(e.message || 'Failed to revoke kiosk')
    } finally {
      setBusyId(null)
    }
  }

  const rotateSecret = async (id: string, code: string) => {
    try {
      setBusyId(id)
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

      setRotatedSecret({ code, secret })
      setPendingRotateId(null)
    } catch (e: any) {
      toast.error(e.message || 'Failed to rotate kiosk secret')
    } finally {
      setBusyId(null)
    }
  }

  const isHealthy = (k: Kiosk) => {
    if (!k.last_heartbeat_at) return false
    const t = new Date(k.last_heartbeat_at).getTime()
    if (!Number.isFinite(t)) return false
    return Date.now() - t <= 10 * 60 * 1000
  }

  const maxActivityCount = historyData.reduce(
    (max, day) => Math.max(max, day.activity_count),
    0
  )

  if (!isAuthenticated || !user) return null

  return (
    <div className="container mx-auto px-4 py-8 space-y-8">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold mb-2">Kiosks</h1>
          <p className="text-muted-foreground">
            Monitor and manage active check-in kiosks. Revoke compromised devices instantly.
          </p>
        </div>
        <Button onClick={() => setIsAddOpen(true)}>Add New Kiosk</Button>
      </div>

      {isAddOpen && (
        <form onSubmit={handleAdd} className="border rounded-lg p-4 space-y-4 bg-card">
          <h2 className="font-semibold">Add New Kiosk</h2>
          <div className="grid md:grid-cols-3 gap-4">
            <div>
              <label className="text-sm font-medium mb-1 block">Name *</label>
              <input
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
                value={addName}
                onChange={(e) => setAddName(e.target.value)}
                placeholder="Lobby iPad"
              />
            </div>
            <div>
              <label className="text-sm font-medium mb-1 block">Pairing Code *</label>
              <input
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
                value={addCode}
                onChange={(e) => setAddCode(e.target.value)}
                placeholder="10-digit code"
              />
            </div>
            <div>
              <label className="text-sm font-medium mb-1 block">Location / Floor</label>
              <input
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
                value={addLocation}
                onChange={(e) => setAddLocation(e.target.value)}
                placeholder="Ground Floor Reception"
              />
            </div>
          </div>
          <div className="flex gap-2">
            <Button type="submit" disabled={isSaving}>
              {isSaving ? 'Creating...' : 'Create Kiosk'}
            </Button>
            <Button type="button" variant="outline" onClick={() => setIsAddOpen(false)}>
              Cancel
            </Button>
          </div>
        </form>
      )}

      {historyKioskId && (
        <div className="border rounded-lg bg-card p-4 space-y-4">
          <div className="flex justify-between items-center">
            <h2 className="font-semibold">Recorded Activity (Last 7 Days)</h2>
            <Button size="sm" variant="ghost" onClick={() => setHistoryKioskId(null)}>Close</Button>
          </div>
          {isLoadingHistory ? (
            <div className="skeleton h-20 w-full" />
          ) : (
            <div className="space-y-3">
              <div className="text-sm text-muted-foreground">
                This view shows actual attendance activity recorded by the kiosk, not inferred uptime.
              </div>
              <div className="flex justify-between items-end gap-2 overflow-x-auto pb-2">
              {historyData.map((day, i) => {
                const height =
                  maxActivityCount > 0 ? Math.max(12, Math.round((day.activity_count / maxActivityCount) * 120)) : 12
                return (
                  <div key={i} className="flex flex-col items-center gap-2 flex-col-reverse group relative">
                    <div className="text-xs text-muted-foreground whitespace-nowrap">{new Date(day.date).toLocaleDateString(undefined, { weekday: 'short' })}</div>
                    <div className="w-10 rounded-t-sm" style={{ 
                      height: `${height}px`, 
                      backgroundColor: day.anomalies > 0 ? 'hsl(0 84.2% 60.2%)' : day.activity_count > 0 ? 'hsl(142.1 76.2% 36.3%)' : 'hsl(215.4 16.3% 46.9%)'
                    }} />
                    <div className="absolute -top-16 w-40 bg-popover text-popover-foreground text-xs p-2 rounded opacity-0 group-hover:opacity-100 transition-opacity shadow">
                      <div>Activity logs: {day.activity_count}</div>
                      <div>Anomalies: {day.anomalies}</div>
                      <div>Last activity: {day.last_activity_at ? new Date(day.last_activity_at).toLocaleString() : '—'}</div>
                    </div>
                  </div>
                )
              })}
              </div>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                {historyData.map((day) => (
                  <div key={day.date} className="rounded-md border p-3 text-sm">
                    <div className="font-medium">{new Date(day.date).toLocaleDateString()}</div>
                    <div className="text-muted-foreground">Activity logs: {day.activity_count}</div>
                    <div className="text-muted-foreground">Anomalies: {day.anomalies}</div>
                    <div className="text-muted-foreground">
                      Last activity: {day.last_activity_at ? new Date(day.last_activity_at).toLocaleString() : '—'}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {rotatedSecret && (
        <div className="border rounded-lg bg-card p-4 space-y-3">
          <div className="text-sm font-semibold">New secret issued for kiosk {rotatedSecret.code}</div>
          <div className="text-xs text-muted-foreground">
            This value is shown once. Update the kiosk device credential immediately.
          </div>
          <pre className="rounded-md bg-muted p-3 text-xs overflow-x-auto">{rotatedSecret.secret}</pre>
          <div className="flex gap-2">
            <Button
              size="sm"
              onClick={async () => {
                try {
                  await navigator.clipboard.writeText(rotatedSecret.secret)
                  toast.success('Secret copied')
                } catch {
                  toast.error('Copy failed')
                }
              }}
            >
              Copy Secret
            </Button>
            <Button size="sm" variant="outline" onClick={() => setRotatedSecret(null)}>
              Dismiss
            </Button>
          </div>
        </div>
      )}

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
          <div className="p-4 space-y-3">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="skeleton h-14 w-full" />
            ))}
          </div>
        ) : kiosks.length === 0 ? (
          <div className="p-4 text-sm text-muted-foreground">
            No kiosks registered yet. The onboarding flow creates the first kiosk automatically.
          </div>
        ) : (
          <>
            <div className="md:hidden divide-y">
              {kiosks.map((k) => (
                <div key={k.id} className="p-4 space-y-3">
                  <div className="flex items-start justify-between gap-3">
                    {editId === k.id ? (
                      <div className="flex flex-col gap-2 w-full">
                        <input className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm" value={editName} onChange={e => setEditName(e.target.value)} />
                        <input className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm" value={editLocation} onChange={e => setEditLocation(e.target.value)} placeholder="Location" />
                        <div className="flex gap-2">
                          <Button size="sm" onClick={saveEdit} disabled={isSaving}>Save</Button>
                          <Button size="sm" variant="outline" onClick={() => setEditId(null)}>Cancel</Button>
                        </div>
                      </div>
                    ) : (
                      <>
                        <div>
                          <div className="font-medium">{k.name}</div>
                          <div className="text-xs font-mono text-muted-foreground">{k.code}</div>
                          {k.location && <div className="text-xs text-muted-foreground mt-1">📍 {k.location}</div>}
                        </div>
                        <span
                          className={
                            k.status === 'active'
                              ? 'text-xs text-green-600 dark:text-green-300'
                              : 'text-xs text-muted-foreground'
                          }
                        >
                          {k.status}
                        </span>
                      </>
                    )}
                  </div>
                  <div className="text-xs text-muted-foreground">
                    Health:{' '}
                    {k.status !== 'active' ? '—' : isHealthy(k) ? 'Healthy' : 'Stale'}
                  </div>
                  <div className="text-xs text-muted-foreground">
                    Last heartbeat: {k.last_heartbeat_at ? new Date(k.last_heartbeat_at).toLocaleString() : '—'}
                  </div>
                  {k.status === 'active' && (
                    <div className="flex flex-wrap gap-2">
                      {pendingRotateId === k.id ? (
                        <>
                          <Button size="sm" onClick={() => void rotateSecret(k.id, k.code)} disabled={busyId === k.id}>
                            Confirm Rotate
                          </Button>
                          <Button size="sm" variant="outline" onClick={() => setPendingRotateId(null)} disabled={busyId === k.id}>
                            Cancel
                          </Button>
                        </>
                      ) : (
                        <Button size="sm" variant="outline" onClick={() => setPendingRotateId(k.id)}>
                          Rotate Secret
                        </Button>
                      )}

                      {pendingRevokeId === k.id ? (
                        <>
                          <Button size="sm" onClick={() => void revokeKiosk(k.id)} disabled={busyId === k.id} className="bg-destructive text-destructive-foreground hover:bg-destructive/90">
                            Confirm Revoke
                          </Button>
                              <Button size="sm" variant="outline" onClick={() => setPendingRevokeId(null)} disabled={busyId === k.id}>
                                Cancel
                              </Button>
                            </>
                          ) : (
                            <Button size="sm" variant="outline" onClick={() => setPendingRevokeId(k.id)} className="text-destructive hover:text-destructive/80">
                              Revoke
                            </Button>
                          )}

                          {editId !== k.id && (
                            <Button size="sm" variant="outline" onClick={() => beginEdit(k)}>
                              Edit
                            </Button>
                          )}
                          <Button size="sm" variant="outline" onClick={() => viewHistory(k.id)}>
                            Activity
                          </Button>
                        </div>
                  )}
                </div>
              ))}
            </div>

            <div className="hidden md:block overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b text-left">
                    <th className="px-4 py-2">Name</th>
                    <th className="px-4 py-2">Code</th>
                    <th className="px-4 py-2">Location</th>
                    <th className="px-4 py-2">Status</th>
                    <th className="px-4 py-2">Health</th>
                    <th className="px-4 py-2">Last Heartbeat</th>
                    <th className="px-4 py-2 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {kiosks.map((k) => (
                    <tr key={k.id} className="border-b last:border-b-0 align-top">
                      <td className="px-4 py-2">
                        {editId === k.id ? (
                          <input className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm" value={editName} onChange={e => setEditName(e.target.value)} />
                        ) : (
                          k.name
                        )}
                      </td>
                      <td className="px-4 py-2 font-mono">{k.code}</td>
                      <td className="px-4 py-2">
                        {editId === k.id ? (
                          <input className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm" value={editLocation} onChange={e => setEditLocation(e.target.value)} placeholder="Waitroom" />
                        ) : (
                          k.location || '—'
                        )}
                      </td>
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
                      <td className="px-4 py-2">
                        {k.last_heartbeat_at ? new Date(k.last_heartbeat_at).toLocaleString() : '—'}
                      </td>
                      <td className="px-4 py-2 text-right">
                        {editId === k.id ? (
                          <div className="flex justify-end gap-2">
                            <Button size="sm" onClick={saveEdit} disabled={isSaving}>Save</Button>
                            <Button size="sm" variant="outline" onClick={() => setEditId(null)}>Cancel</Button>
                          </div>
                        ) : k.status === 'active' && (
                          <div className="flex justify-end gap-2">
                            <Button size="sm" variant="outline" onClick={() => viewHistory(k.id)}>
                              Activity
                            </Button>
                            <Button size="sm" variant="outline" onClick={() => beginEdit(k)}>
                              Edit
                            </Button>
                            {pendingRotateId === k.id ? (
                              <>
                                <Button size="sm" onClick={() => void rotateSecret(k.id, k.code)} disabled={busyId === k.id}>
                                  Confirm Rotate
                                </Button>
                                <Button size="sm" variant="outline" onClick={() => setPendingRotateId(null)} disabled={busyId === k.id}>
                                  Cancel
                                </Button>
                              </>
                            ) : (
                              <Button size="sm" variant="outline" onClick={() => setPendingRotateId(k.id)}>
                                Rotate Secret
                              </Button>
                            )}
                            {pendingRevokeId === k.id ? (
                              <>
                                <Button size="sm" onClick={() => void revokeKiosk(k.id)} disabled={busyId === k.id} className="bg-destructive text-destructive-foreground hover:bg-destructive/90">
                                  Confirm Revoke
                                </Button>
                                <Button size="sm" variant="outline" onClick={() => setPendingRevokeId(null)} disabled={busyId === k.id}>
                                  Cancel
                                </Button>
                              </>
                            ) : (
                              <Button size="sm" variant="outline" onClick={() => setPendingRevokeId(k.id)} className="text-destructive hover:text-destructive/80">
                                Revoke
                              </Button>
                            )}
                          </div>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </>
        )}
      </div>
    </div>
  )
}
