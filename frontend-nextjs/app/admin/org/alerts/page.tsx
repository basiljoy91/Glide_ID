'use client'

import { useEffect, useMemo, useState } from 'react'
import { useAuthStore } from '@/store/useStore'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import toast from 'react-hot-toast'

type Notification = {
  id: string
  notification_type: string
  title: string
  body: string
  severity: string
  is_read: boolean
  action_url?: string | null
  created_at: string
}

export default function OrgAlertsPage() {
  const { isAuthenticated, user, token } = useAuthStore()
  const router = useRouter()
  const base = useMemo(() => process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080', [])
  const [notifications, setNotifications] = useState<Notification[]>([])
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    if (!isAuthenticated || !user) {
      router.push('/admin/login')
      return
    }
    if (!['org_admin', 'hr', 'dept_manager'].includes(user.role)) {
      router.push('/dashboard')
      return
    }
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, user?.role])

  const load = async () => {
    try {
      setIsLoading(true)
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(`${base}/api/v1/org/notifications`, { headers })
      if (!resp.ok) throw new Error('Failed to load notifications')
      setNotifications(await resp.json())
    } catch (e: any) {
      toast.error(e.message || 'Failed to load notifications')
    } finally {
      setIsLoading(false)
    }
  }

  const markRead = async (id: string) => {
    try {
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(`${base}/api/v1/org/notifications/${id}/read`, { method: 'POST', headers })
      if (!resp.ok) throw new Error('Failed to update notification')
      setNotifications((current) => current.map((item) => (item.id === id ? { ...item, is_read: true } : item)))
    } catch (e: any) {
      toast.error(e.message || 'Failed to update notification')
    }
  }

  if (!isAuthenticated || !user) return null

  return (
    <div className="container mx-auto px-4 py-8 space-y-6">
      <div>
        <h1 className="text-2xl font-bold mb-2">Alert Center</h1>
        <p className="text-muted-foreground">Integration failures, sync conflicts, support updates, and operational alerts.</p>
      </div>

      <div className="bg-card border border-border rounded-lg p-4 shadow-sm">
        {isLoading ? (
          <div className="text-sm text-muted-foreground">Loading alerts...</div>
        ) : notifications.length === 0 ? (
          <div className="text-sm text-muted-foreground">No alerts right now.</div>
        ) : (
          <div className="space-y-3">
            {notifications.map((notification) => (
              <div key={notification.id} className={`rounded-md border p-3 ${notification.is_read ? 'opacity-70' : ''}`}>
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="font-medium">{notification.title}</div>
                    <div className="text-xs text-muted-foreground">
                      {notification.notification_type} • {notification.severity}
                    </div>
                  </div>
                  <div className="text-xs text-muted-foreground">{new Date(notification.created_at).toLocaleString()}</div>
                </div>
                <div className="mt-2 text-sm text-muted-foreground">{notification.body}</div>
                <div className="mt-3 flex gap-2">
                  {!notification.is_read && (
                    <Button size="sm" variant="outline" onClick={() => void markRead(notification.id)}>
                      Mark read
                    </Button>
                  )}
                  {notification.action_url && (
                    <Button size="sm" variant="outline" onClick={() => router.push(notification.action_url || '/admin/org')}>
                      Open
                    </Button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
