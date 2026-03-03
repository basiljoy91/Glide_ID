'use client'

import { useEffect, useMemo, useState } from 'react'
import { useAuthStore } from '@/store/useStore'
import { useRouter } from 'next/navigation'
import toast from 'react-hot-toast'
import Link from 'next/link'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { format } from 'date-fns'

type AnomalyRow = {
  id: string
  punch_time: string
  status: string
  verification_method: string
  anomaly_detected: boolean
  anomaly_reason?: string | null
  notes?: string | null
  user_id: string
  employee_id: string
  first_name: string
  last_name: string
  kiosk_code?: string | null
}

export default function AnomalyReviewsPage() {
  const { user, isAuthenticated, token } = useAuthStore()
  const router = useRouter()
  const [rows, setRows] = useState<AnomalyRow[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [selected, setSelected] = useState<Record<string, boolean>>({})
  const [bulkNote, setBulkNote] = useState('')
  const [isResolvingBulk, setIsResolvingBulk] = useState(false)

  const today = new Date().toISOString().slice(0, 10)
  const [searchQ, setSearchQ] = useState('')
  const [method, setMethod] = useState('all')
  const [state, setState] = useState<'unresolved' | 'resolved' | 'all'>('unresolved')
  const [startDate, setStartDate] = useState(new Date(Date.now() - 30 * 86400000).toISOString().slice(0, 10))
  const [endDate, setEndDate] = useState(today)

  useEffect(() => {
    if (!isAuthenticated || !user) {
      router.push('/admin/login')
      return
    }
    if (!['org_admin', 'hr', 'dept_manager'].includes(user.role)) {
      router.push('/admin/login')
      return
    }
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, user?.role])

  const load = async () => {
    try {
      setIsLoading(true)
      setSelected({})
      const base = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`

      const query = new URLSearchParams()
      query.set('limit', '200')
      query.set('state', state)
      if (startDate) query.set('start_date', startDate)
      if (endDate) query.set('end_date', endDate)
      if (searchQ.trim()) query.set('q', searchQ.trim())
      if (method !== 'all') query.set('method', method)

      const resp = await fetch(`${base}/api/v1/reports/anomalies?${query.toString()}`, {
        headers,
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to load anomalies')
      }
      setRows(await resp.json())
    } catch (e: any) {
      toast.error(e.message || 'Failed to load anomalies')
      setRows([])
    } finally {
      setIsLoading(false)
    }
  }

  const selectedIds = useMemo(() => Object.keys(selected).filter((id) => selected[id]), [selected])
  const unresolvedSelectedCount = useMemo(
    () => rows.filter((r) => selected[r.id] && r.anomaly_detected).length,
    [rows, selected]
  )

  const toggleAll = (checked: boolean) => {
    if (!checked) {
      setSelected({})
      return
    }
    const next: Record<string, boolean> = {}
    rows.forEach((r) => {
      if (r.anomaly_detected) next[r.id] = true
    })
    setSelected(next)
  }

  const bulkResolve = async () => {
    if (!selectedIds.length) {
      toast.error('Select at least one anomaly')
      return
    }
    if (!unresolvedSelectedCount) {
      toast.error('Only unresolved anomalies can be resolved')
      return
    }
    try {
      setIsResolvingBulk(true)
      const base = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(`${base}/api/v1/reports/anomalies/bulk-resolve`, {
        method: 'PATCH',
        headers,
        body: JSON.stringify({ ids: selectedIds, note: bulkNote }),
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to resolve anomalies')
      }
      toast.success('Anomalies resolved')
      setBulkNote('')
      await load()
    } catch (e: any) {
      toast.error(e.message || 'Failed to resolve anomalies')
    } finally {
      setIsResolvingBulk(false)
    }
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold mb-2">Anomaly Review Queue</h1>
          <p className="text-muted-foreground">Filter, investigate, and bulk-resolve flagged attendance logs.</p>
        </div>
        <Link href="/admin/org">
          <Button variant="outline">Back to dashboard</Button>
        </Link>
      </div>

      <div className="bg-card border border-border rounded-lg p-4 shadow-sm space-y-3">
        <div className="grid grid-cols-1 md:grid-cols-5 gap-3">
          <Input
            value={searchQ}
            onChange={(e) => setSearchQ(e.target.value)}
            placeholder="Search employee name/ID"
          />
          <select
            value={state}
            onChange={(e) => setState(e.target.value as 'unresolved' | 'resolved' | 'all')}
            className="h-10 rounded-md border border-input bg-background px-3 text-sm"
          >
            <option value="unresolved">Unresolved</option>
            <option value="resolved">Resolved</option>
            <option value="all">All</option>
          </select>
          <select
            value={method}
            onChange={(e) => setMethod(e.target.value)}
            className="h-10 rounded-md border border-input bg-background px-3 text-sm"
          >
            <option value="all">All methods</option>
            <option value="biometric">Biometric</option>
            <option value="pin">PIN</option>
            <option value="manual">Manual</option>
          </select>
          <Input type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} />
          <Input type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} />
        </div>
        <div className="flex justify-end">
          <Button onClick={() => void load()} disabled={isLoading}>
            {isLoading ? 'Loading…' : 'Apply filters'}
          </Button>
        </div>
      </div>

      <div className="bg-card border border-border rounded-lg p-4 shadow-sm space-y-3">
        <div className="flex flex-col md:flex-row gap-2 md:items-center md:justify-between">
          <div className="text-sm text-muted-foreground">{rows.length} records</div>
          <div className="flex gap-2">
            <Input
              value={bulkNote}
              onChange={(e) => setBulkNote(e.target.value)}
              placeholder="Bulk resolution note (optional)"
              className="w-80"
            />
            <Button onClick={bulkResolve} disabled={isResolvingBulk || !selectedIds.length}>
              {isResolvingBulk ? 'Resolving…' : `Resolve selected (${selectedIds.length})`}
            </Button>
          </div>
        </div>

        {isLoading ? (
          <div className="h-40 bg-muted rounded" />
        ) : rows.length === 0 ? (
          <div className="text-sm text-muted-foreground">No anomalies to review.</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="text-muted-foreground">
                <tr className="border-b">
                  <th className="py-2 pr-2">
                    <input
                      type="checkbox"
                      onChange={(e) => toggleAll(e.target.checked)}
                      checked={rows.length > 0 && rows.every((r) => !r.anomaly_detected || selected[r.id])}
                    />
                  </th>
                  <th className="text-left font-medium py-2 pr-3">Time</th>
                  <th className="text-left font-medium py-2 pr-3">Employee</th>
                  <th className="text-left font-medium py-2 pr-3">Method</th>
                  <th className="text-left font-medium py-2 pr-3">Reason</th>
                  <th className="text-left font-medium py-2 pr-3">State</th>
                  <th className="text-right font-medium py-2 pl-3">Action</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((r) => (
                  <tr key={r.id} className="border-b last:border-b-0">
                    <td className="py-2 pr-2">
                      <input
                        type="checkbox"
                        checked={Boolean(selected[r.id])}
                        disabled={!r.anomaly_detected}
                        onChange={(e) =>
                          setSelected((prev) => ({ ...prev, [r.id]: e.target.checked }))
                        }
                      />
                    </td>
                    <td className="py-2 pr-3 whitespace-nowrap">{format(new Date(r.punch_time), 'MMM d, h:mm a')}</td>
                    <td className="py-2 pr-3">
                      <div className="font-medium">
                        {r.first_name} {r.last_name}
                      </div>
                      <div className="text-xs text-muted-foreground">{r.employee_id}</div>
                    </td>
                    <td className="py-2 pr-3">{r.verification_method}</td>
                    <td className="py-2 pr-3 text-muted-foreground">{r.anomaly_reason || '—'}</td>
                    <td className="py-2 pr-3">
                      {r.anomaly_detected ? (
                        <span className="text-xs text-yellow-700 dark:text-yellow-300">Unresolved</span>
                      ) : (
                        <span className="text-xs text-green-700 dark:text-green-300">Resolved</span>
                      )}
                    </td>
                    <td className="py-2 pl-3 text-right">
                      <Link href={`/admin/org/reviews/anomalies/${r.id}`}>
                        <Button size="sm" variant="outline">
                          Review
                        </Button>
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}
