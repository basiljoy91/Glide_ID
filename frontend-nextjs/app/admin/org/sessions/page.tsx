'use client'

import { useEffect, useMemo, useState } from 'react'
import { useRouter } from 'next/navigation'
import toast from 'react-hot-toast'
import { Button } from '@/components/ui/button'
import { useAuthStore } from '@/store/useStore'

type SessionRecord = {
  id: string
  user_id: string
  first_name: string
  last_name: string
  email: string
  role: string
  ip_address?: string | null
  user_agent?: string | null
  last_seen_at: string
  expires_at: string
  revoked_at?: string | null
  created_at: string
}

export default function SessionsPage() {
  const { isAuthenticated, user, token } = useAuthStore()
  const router = useRouter()
  const base = useMemo(() => process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080', [])
  const canManage = user?.role === 'org_admin' || user?.permissions?.includes('sessions.manage')

  const [sessions, setSessions] = useState<SessionRecord[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [busySessionID, setBusySessionID] = useState<string | null>(null)

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
      const resp = await fetch(`${base}/api/v1/org/sessions`, { headers: authHeaders() })
      if (!resp.ok) {
        const error = await resp.json().catch(() => ({}))
        throw new Error(error.error || 'Failed to load sessions')
      }
      const data = await resp.json()
      setSessions(Array.isArray(data.sessions) ? data.sessions : [])
    } catch (error: any) {
      toast.error(error.message || 'Failed to load sessions')
    } finally {
      setIsLoading(false)
    }
  }

  const revokeSession = async (id: string) => {
    try {
      setBusySessionID(id)
      const resp = await fetch(`${base}/api/v1/org/sessions/${id}/revoke`, {
        method: 'POST',
        headers: authHeaders(),
      })
      if (!resp.ok) {
        const error = await resp.json().catch(() => ({}))
        throw new Error(error.error || 'Failed to revoke session')
      }
      toast.success('Session revoked')
      await load()
    } catch (error: any) {
      toast.error(error.message || 'Failed to revoke session')
    } finally {
      setBusySessionID(null)
    }
  }

  const revokeUserSessions = async (userID: string) => {
    try {
      setBusySessionID(userID)
      const resp = await fetch(`${base}/api/v1/org/sessions/revoke-user`, {
        method: 'POST',
        headers: authHeaders(),
        body: JSON.stringify({ user_id: userID }),
      })
      if (!resp.ok) {
        const error = await resp.json().catch(() => ({}))
        throw new Error(error.error || 'Failed to revoke user sessions')
      }
      const data = await resp.json()
      toast.success(`${data.revoked_count || 0} sessions revoked`)
      await load()
    } catch (error: any) {
      toast.error(error.message || 'Failed to revoke user sessions')
    } finally {
      setBusySessionID(null)
    }
  }

  if (!isAuthenticated || !user || !canManage) return null

  return (
    <div className="container mx-auto space-y-6 p-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold">Session Management</h1>
          <p className="text-muted-foreground">Inspect active devices and revoke a single session or every session for a compromised account.</p>
        </div>
        <Button variant="outline" onClick={() => void load()}>Refresh</Button>
      </div>

      {isLoading ? (
        <div className="rounded-lg border p-6 text-sm text-muted-foreground">Loading sessions...</div>
      ) : sessions.length === 0 ? (
        <div className="rounded-lg border border-dashed p-6 text-sm text-muted-foreground">No tracked sessions found.</div>
      ) : (
        <div className="overflow-x-auto rounded-lg border bg-card">
          <table className="min-w-[1120px] w-full text-sm">
            <thead className="text-left text-muted-foreground">
              <tr className="border-b">
                <th className="px-4 py-3">User</th>
                <th className="px-4 py-3">Role</th>
                <th className="px-4 py-3">IP</th>
                <th className="px-4 py-3">User Agent</th>
                <th className="px-4 py-3">Last Seen</th>
                <th className="px-4 py-3">Expires</th>
                <th className="px-4 py-3">Status</th>
                <th className="px-4 py-3">Actions</th>
              </tr>
            </thead>
            <tbody>
              {sessions.map((session) => (
                <tr key={session.id} className="border-b last:border-b-0 align-top">
                  <td className="px-4 py-3">
                    <div className="font-medium">{session.first_name} {session.last_name}</div>
                    <div className="text-xs text-muted-foreground">{session.email}</div>
                  </td>
                  <td className="px-4 py-3">{session.role}</td>
                  <td className="px-4 py-3 whitespace-nowrap">{session.ip_address || '—'}</td>
                  <td className="px-4 py-3 text-xs text-muted-foreground">{session.user_agent || '—'}</td>
                  <td className="px-4 py-3 whitespace-nowrap">{new Date(session.last_seen_at).toLocaleString()}</td>
                  <td className="px-4 py-3 whitespace-nowrap">{new Date(session.expires_at).toLocaleString()}</td>
                  <td className="px-4 py-3">{session.revoked_at ? 'Revoked' : 'Active'}</td>
                  <td className="px-4 py-3">
                    <div className="flex gap-2">
                      <Button variant="outline" disabled={!!session.revoked_at || busySessionID === session.id} onClick={() => void revokeSession(session.id)}>
                        {busySessionID === session.id ? 'Working...' : 'Revoke'}
                      </Button>
                      <Button variant="ghost" disabled={busySessionID === session.user_id} onClick={() => void revokeUserSessions(session.user_id)}>
                        {busySessionID === session.user_id ? 'Working...' : 'Revoke User Sessions'}
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
