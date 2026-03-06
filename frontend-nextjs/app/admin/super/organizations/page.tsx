'use client'

import { useEffect, useState } from 'react'
import { useAuthStore } from '@/store/useStore'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import toast from 'react-hot-toast'

type OrgRow = {
  id: string
  name: string
  slug: string
  subscription_tier: 'free' | 'starter' | 'professional' | 'enterprise'
  billing_status: 'trialing' | 'active' | 'past_due' | 'canceled' | 'inactive'
  seat_count: number
  base_amount_cents: number
  per_seat_amount_cents: number
  estimated_mrr_cents: number
  users_count: number
  is_active: boolean
}

const TIERS = ['free', 'starter', 'professional', 'enterprise'] as const
const STATUSES = ['trialing', 'active', 'past_due', 'canceled'] as const

export default function SuperAdminOrganizationsPage() {
  const { isAuthenticated, user, token } = useAuthStore()
  const router = useRouter()
  const [rows, setRows] = useState<OrgRow[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [query, setQuery] = useState('')
  const [editingId, setEditingId] = useState<string | null>(null)
  const [form, setForm] = useState<any>({})

  useEffect(() => {
    if (!isAuthenticated || user?.role !== 'super_admin') {
      router.push('/admin/login')
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
      const q = query.trim() ? `?q=${encodeURIComponent(query.trim())}` : ''
      const resp = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/admin/super/organizations${q}`,
        { headers }
      )
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to load organizations')
      }
      setRows(await resp.json())
    } catch (e: any) {
      toast.error(e.message || 'Failed to load organizations')
      setRows([])
    } finally {
      setIsLoading(false)
    }
  }

  const beginEdit = (r: OrgRow) => {
    setEditingId(r.id)
    setForm({
      plan_tier: r.subscription_tier,
      status: r.billing_status === 'inactive' ? 'active' : r.billing_status,
      seat_count: r.seat_count,
      base_amount_cents: r.base_amount_cents,
      per_seat_amount_cents: r.per_seat_amount_cents,
      is_active: r.is_active,
    })
  }

  const cancelEdit = () => {
    setEditingId(null)
    setForm({})
  }

  const saveEdit = async (id: string) => {
    try {
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (token) headers.Authorization = `Bearer ${token}`

      const subResp = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/admin/super/organizations/${id}/subscription`,
        {
          method: 'PATCH',
          headers,
          body: JSON.stringify({
            plan_tier: form.plan_tier,
            status: form.status,
            seat_count: Number(form.seat_count),
            base_amount_cents: Number(form.base_amount_cents),
            per_seat_amount_cents: Number(form.per_seat_amount_cents),
          }),
        }
      )
      if (!subResp.ok) {
        const err = await subResp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to update subscription')
      }

      const statusResp = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/admin/super/organizations/${id}/status`,
        {
          method: 'PATCH',
          headers,
          body: JSON.stringify({
            is_active: Boolean(form.is_active),
          }),
        }
      )
      if (!statusResp.ok) {
        const err = await statusResp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to update organization status')
      }

      toast.success('Organization updated')
      cancelEdit()
      await load()
    } catch (e: any) {
      toast.error(e.message || 'Failed to update organization')
    }
  }

  if (!isAuthenticated || user?.role !== 'super_admin') return null

  return (
    <div className="container mx-auto px-4 py-8 space-y-6">
      <div>
        <h1 className="text-2xl font-bold mb-2">Organizations</h1>
        <p className="text-muted-foreground">Manage plan, billing configuration, and organization status.</p>
      </div>

      <div className="bg-card border border-border rounded-lg p-4 shadow-sm flex gap-2">
        <Input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search by name or slug"
        />
        <Button onClick={() => void load()} disabled={isLoading}>
          {isLoading ? 'Loading…' : 'Search'}
        </Button>
      </div>

      <div className="border rounded-lg bg-card overflow-x-auto">
        {isLoading ? (
          <div className="p-4 space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="skeleton h-12 w-full min-w-[760px]" />
            ))}
          </div>
        ) : rows.length === 0 ? (
          <div className="p-4 text-sm text-muted-foreground">No organizations found.</div>
        ) : (
          <table className="w-full text-sm min-w-[1100px]">
            <thead>
              <tr className="border-b text-left">
                <th className="px-4 py-2">Organization</th>
                <th className="px-4 py-2">Tier</th>
                <th className="px-4 py-2">Billing Status</th>
                <th className="px-4 py-2">Seats</th>
                <th className="px-4 py-2">Pricing (cents)</th>
                <th className="px-4 py-2">MRR</th>
                <th className="px-4 py-2">Org Active</th>
                <th className="px-4 py-2 text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.id} className="border-b last:border-b-0 align-top">
                  <td className="px-4 py-2">
                    <div className="font-medium">{r.name}</div>
                    <div className="text-xs text-muted-foreground">{r.slug}</div>
                  </td>
                  <td className="px-4 py-2">
                    {editingId === r.id ? (
                      <select
                        value={form.plan_tier}
                        onChange={(e) => setForm((prev: any) => ({ ...prev, plan_tier: e.target.value }))}
                        className="h-10 rounded-md border border-input bg-background px-3 text-sm"
                      >
                        {TIERS.map((t) => (
                          <option key={t} value={t}>
                            {t}
                          </option>
                        ))}
                      </select>
                    ) : (
                      r.subscription_tier
                    )}
                  </td>
                  <td className="px-4 py-2">
                    {editingId === r.id ? (
                      <select
                        value={form.status}
                        onChange={(e) => setForm((prev: any) => ({ ...prev, status: e.target.value }))}
                        className="h-10 rounded-md border border-input bg-background px-3 text-sm"
                      >
                        {STATUSES.map((s) => (
                          <option key={s} value={s}>
                            {s}
                          </option>
                        ))}
                      </select>
                    ) : (
                      r.billing_status
                    )}
                  </td>
                  <td className="px-4 py-2">
                    {editingId === r.id ? (
                      <Input
                        type="number"
                        value={form.seat_count}
                        onChange={(e) => setForm((prev: any) => ({ ...prev, seat_count: e.target.value }))}
                      />
                    ) : (
                      `${r.seat_count} (${r.users_count} users)`
                    )}
                  </td>
                  <td className="px-4 py-2">
                    {editingId === r.id ? (
                      <div className="grid grid-cols-1 gap-2">
                        <Input
                          type="number"
                          value={form.base_amount_cents}
                          onChange={(e) =>
                            setForm((prev: any) => ({ ...prev, base_amount_cents: e.target.value }))
                          }
                          placeholder="Base"
                        />
                        <Input
                          type="number"
                          value={form.per_seat_amount_cents}
                          onChange={(e) =>
                            setForm((prev: any) => ({ ...prev, per_seat_amount_cents: e.target.value }))
                          }
                          placeholder="Per seat"
                        />
                      </div>
                    ) : (
                      <div>
                        <div>base: {r.base_amount_cents}</div>
                        <div>seat: {r.per_seat_amount_cents}</div>
                      </div>
                    )}
                  </td>
                  <td className="px-4 py-2">${(r.estimated_mrr_cents / 100).toLocaleString()}</td>
                  <td className="px-4 py-2">
                    {editingId === r.id ? (
                      <select
                        value={form.is_active ? 'true' : 'false'}
                        onChange={(e) =>
                          setForm((prev: any) => ({ ...prev, is_active: e.target.value === 'true' }))
                        }
                        className="h-10 rounded-md border border-input bg-background px-3 text-sm"
                      >
                        <option value="true">Active</option>
                        <option value="false">Inactive</option>
                      </select>
                    ) : r.is_active ? (
                      'Active'
                    ) : (
                      'Inactive'
                    )}
                  </td>
                  <td className="px-4 py-2 text-right">
                    {editingId === r.id ? (
                      <div className="flex justify-end gap-2">
                        <Button size="sm" onClick={() => void saveEdit(r.id)}>
                          Save
                        </Button>
                        <Button size="sm" variant="outline" onClick={cancelEdit}>
                          Cancel
                        </Button>
                      </div>
                    ) : (
                      <Button size="sm" variant="outline" onClick={() => beginEdit(r)}>
                        Edit
                      </Button>
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
