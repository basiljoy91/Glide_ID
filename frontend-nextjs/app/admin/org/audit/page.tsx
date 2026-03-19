'use client'

import { useEffect, useMemo, useState } from 'react'
import { useAuthStore } from '@/store/useStore'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import toast from 'react-hot-toast'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

type AuditLogRow = {
  id: string
  actor_name?: string | null
  actor_email?: string | null
  target_name?: string | null
  target_email?: string | null
  action: string
  resource_type?: string | null
  ip_address?: string | null
  created_at: string
  details?: Record<string, unknown> | null
}

const ACTION_OPTIONS = [
  'all',
  'admin_login',
  'user_created',
  'user_updated',
  'user_deleted',
  'user_activated',
  'user_deactivated',
  'department_created',
  'department_updated',
  'department_deleted',
  'export_generated',
  'report_generated',
] as const

export default function OrgAuditPage() {
  const { isAuthenticated, user, token } = useAuthStore()
  const router = useRouter()
  const base = useMemo(() => process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080', [])

  const [rows, setRows] = useState<AuditLogRow[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [query, setQuery] = useState('')
  const [debouncedQuery, setDebouncedQuery] = useState('')
  const [action, setAction] = useState<(typeof ACTION_OPTIONS)[number]>('all')
  const [page, setPage] = useState(1)
  const [limit] = useState(25)
  const [total, setTotal] = useState(0)

  useEffect(() => {
    if (!isAuthenticated || !user) {
      router.push('/admin/login')
      return
    }
    if (!['org_admin', 'hr'].includes(user.role)) {
      router.push('/dashboard')
      return
    }
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, user?.role])

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(query.trim())
      setPage(1)
    }, 250)
    return () => clearTimeout(timer)
  }, [query])

  useEffect(() => {
    if (!isAuthenticated || !user) return
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, debouncedQuery, action])

  const load = async () => {
    try {
      setIsLoading(true)
      const params = new URLSearchParams()
      params.set('page', String(page))
      params.set('limit', String(limit))
      if (debouncedQuery) params.set('q', debouncedQuery)
      if (action !== 'all') params.set('action', action)

      const resp = await fetch(`${base}/api/v1/audit?${params.toString()}`, {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to load audit logs')
      }
      const payload = await resp.json()
      setRows(Array.isArray(payload.data) ? payload.data : [])
      setTotal(payload?.meta?.total || 0)
    } catch (e: any) {
      toast.error(e.message || 'Failed to load audit logs')
      setRows([])
      setTotal(0)
    } finally {
      setIsLoading(false)
    }
  }

  const pageCount = Math.max(1, Math.ceil(total / limit))

  const formatAction = (value: string) =>
    value
      .split('_')
      .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
      .join(' ')

  const formatDate = (value: string) => {
    const date = new Date(value)
    if (Number.isNaN(date.getTime())) return value
    return date.toLocaleString()
  }

  const detailsPreview = (details?: Record<string, unknown> | null) => {
    if (!details || Object.keys(details).length === 0) return '—'
    const preview = Object.entries(details)
      .slice(0, 2)
      .map(([key, value]) => `${key}: ${String(value)}`)
      .join(' • ')
    return preview || '—'
  }

  if (!isAuthenticated || !user) return null

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold mb-2">Audit Logs</h1>
          <p className="text-muted-foreground">Track admin access and operational changes in your organization.</p>
        </div>
        <Link href="/admin/org">
          <Button variant="outline">Back to dashboard</Button>
        </Link>
      </div>

      <div className="bg-card border border-border rounded-lg p-4 shadow-sm space-y-3">
        <div className="grid grid-cols-1 md:grid-cols-[1fr_220px_auto] gap-3">
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search actor, target, action"
          />
          <select
            value={action}
            onChange={(e) => {
              setAction(e.target.value as (typeof ACTION_OPTIONS)[number])
              setPage(1)
            }}
            className="h-10 rounded-md border border-input bg-background px-3 text-sm"
          >
            {ACTION_OPTIONS.map((item) => (
              <option key={item} value={item}>
                {item === 'all' ? 'All actions' : formatAction(item)}
              </option>
            ))}
          </select>
          <Button variant="outline" onClick={() => void load()} disabled={isLoading}>
            Refresh
          </Button>
        </div>
      </div>

      <div className="bg-card border border-border rounded-lg shadow-sm overflow-hidden">
        <div className="border-b px-4 py-3 text-sm text-muted-foreground">
          {total.toLocaleString()} log entries
        </div>
        {isLoading ? (
          <div className="p-4 space-y-3">
            {Array.from({ length: 6 }).map((_, index) => (
              <div key={index} className="h-14 rounded bg-muted" />
            ))}
          </div>
        ) : rows.length === 0 ? (
          <div className="p-4 text-sm text-muted-foreground">No audit logs matched your filters.</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm min-w-[960px]">
              <thead className="text-muted-foreground">
                <tr className="border-b text-left">
                  <th className="px-4 py-2">Time</th>
                  <th className="px-4 py-2">Actor</th>
                  <th className="px-4 py-2">Action</th>
                  <th className="px-4 py-2">Target</th>
                  <th className="px-4 py-2">Resource</th>
                  <th className="px-4 py-2">IP</th>
                  <th className="px-4 py-2">Details</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((row) => (
                  <tr key={row.id} className="border-b last:border-b-0 align-top">
                    <td className="px-4 py-3 whitespace-nowrap">{formatDate(row.created_at)}</td>
                    <td className="px-4 py-3">
                      <div className="font-medium">{row.actor_name || 'System'}</div>
                      <div className="text-xs text-muted-foreground">{row.actor_email || '—'}</div>
                    </td>
                    <td className="px-4 py-3 font-medium">{formatAction(row.action)}</td>
                    <td className="px-4 py-3">
                      <div className="font-medium">{row.target_name || '—'}</div>
                      <div className="text-xs text-muted-foreground">{row.target_email || '—'}</div>
                    </td>
                    <td className="px-4 py-3">{row.resource_type || '—'}</td>
                    <td className="px-4 py-3 whitespace-nowrap">{row.ip_address || '—'}</td>
                    <td className="px-4 py-3 text-xs text-muted-foreground">{detailsPreview(row.details)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <div className="flex items-center justify-between">
        <div className="text-sm text-muted-foreground">
          Page {page} of {pageCount}
        </div>
        <div className="flex gap-2">
          <Button variant="outline" disabled={page <= 1} onClick={() => setPage((value) => value - 1)}>
            Previous
          </Button>
          <Button
            variant="outline"
            disabled={page >= pageCount}
            onClick={() => setPage((value) => value + 1)}
          >
            Next
          </Button>
        </div>
      </div>
    </div>
  )
}
