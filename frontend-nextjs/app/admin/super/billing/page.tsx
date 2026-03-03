'use client'

import { useEffect, useState } from 'react'
import { useAuthStore } from '@/store/useStore'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { DataCard, DataCardGrid } from '@/components/data/DataCard'
import toast from 'react-hot-toast'

type BillingOverview = {
  active_subscriptions: number
  monthly_recurring_revenue_cents: number
  paid_this_month_cents: number
  outstanding_amount_cents: number
  overdue_invoices: number
  open_invoices: number
}

type Invoice = {
  id: string
  tenant_id: string
  tenant_name: string
  invoice_number: string
  status: string
  period_start: string
  period_end: string
  subtotal_cents: number
  tax_cents: number
  total_cents: number
  due_at?: string | null
  paid_at?: string | null
  payment_reference?: string | null
  created_at: string
}

export default function SuperAdminBillingPage() {
  const { isAuthenticated, user, token } = useAuthStore()
  const router = useRouter()
  const [overview, setOverview] = useState<BillingOverview | null>(null)
  const [invoices, setInvoices] = useState<Invoice[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [invoiceStatus, setInvoiceStatus] = useState('all')

  const [tenantId, setTenantId] = useState('')
  const [periodStart, setPeriodStart] = useState(new Date(Date.now() - 30 * 86400000).toISOString().slice(0, 10))
  const [periodEnd, setPeriodEnd] = useState(new Date().toISOString().slice(0, 10))
  const [taxCents, setTaxCents] = useState('0')
  const [creating, setCreating] = useState(false)

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
      const base = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      const invoiceQS = new URLSearchParams()
      if (invoiceStatus !== 'all') invoiceQS.set('status', invoiceStatus)

      const [overviewResp, invoicesResp] = await Promise.all([
        fetch(`${base}/api/v1/admin/super/billing/overview`, { headers }),
        fetch(`${base}/api/v1/admin/super/billing/invoices?${invoiceQS.toString()}`, { headers }),
      ])

      if (!overviewResp.ok) {
        const err = await overviewResp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to load billing overview')
      }
      if (!invoicesResp.ok) {
        const err = await invoicesResp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to load invoices')
      }

      setOverview(await overviewResp.json())
      setInvoices(await invoicesResp.json())
    } catch (e: any) {
      toast.error(e.message || 'Failed to load billing data')
    } finally {
      setIsLoading(false)
    }
  }

  const createInvoice = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!tenantId.trim()) {
      toast.error('Tenant ID is required')
      return
    }
    try {
      setCreating(true)
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/admin/super/billing/invoices`,
        {
          method: 'POST',
          headers,
          body: JSON.stringify({
            tenant_id: tenantId.trim(),
            period_start: periodStart,
            period_end: periodEnd,
            tax_cents: Number(taxCents) || 0,
            status: 'open',
          }),
        }
      )
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to create invoice')
      }
      toast.success('Invoice created')
      await load()
    } catch (e: any) {
      toast.error(e.message || 'Failed to create invoice')
    } finally {
      setCreating(false)
    }
  }

  const markPaid = async (id: string) => {
    try {
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/admin/super/billing/invoices/${id}/mark-paid`,
        {
          method: 'PATCH',
          headers,
          body: JSON.stringify({}),
        }
      )
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to mark invoice paid')
      }
      toast.success('Invoice marked paid')
      await load()
    } catch (e: any) {
      toast.error(e.message || 'Failed to mark invoice paid')
    }
  }

  if (!isAuthenticated || user?.role !== 'super_admin') return null

  return (
    <div className="container mx-auto px-4 py-8 space-y-6">
      <div>
        <h1 className="text-2xl font-bold mb-2">Billing</h1>
        <p className="text-muted-foreground">Monitor recurring revenue and manage invoice lifecycle.</p>
      </div>

      {overview ? (
        <DataCardGrid>
          <DataCard title="Active Subs" value={overview.active_subscriptions} subtitle="Active/trialing/past-due" />
          <DataCard title="MRR" value={`$${(overview.monthly_recurring_revenue_cents / 100).toLocaleString()}`} subtitle="Current recurring projection" />
          <DataCard title="Paid This Month" value={`$${(overview.paid_this_month_cents / 100).toLocaleString()}`} subtitle="Realized cash this month" />
          <DataCard title="Outstanding" value={`$${(overview.outstanding_amount_cents / 100).toLocaleString()}`} subtitle={`${overview.open_invoices} open (${overview.overdue_invoices} overdue)`} />
        </DataCardGrid>
      ) : isLoading ? (
        <div className="text-sm text-muted-foreground">Loading overview...</div>
      ) : null}

      <form onSubmit={createInvoice} className="bg-card border border-border rounded-lg p-4 shadow-sm space-y-3">
        <h2 className="font-semibold">Create Invoice</h2>
        <div className="grid md:grid-cols-4 gap-3">
          <Input value={tenantId} onChange={(e) => setTenantId(e.target.value)} placeholder="Tenant UUID" />
          <Input type="date" value={periodStart} onChange={(e) => setPeriodStart(e.target.value)} />
          <Input type="date" value={periodEnd} onChange={(e) => setPeriodEnd(e.target.value)} />
          <Input type="number" value={taxCents} onChange={(e) => setTaxCents(e.target.value)} placeholder="Tax cents" />
        </div>
        <Button type="submit" disabled={creating}>
          {creating ? 'Creating…' : 'Create invoice'}
        </Button>
      </form>

      <div className="bg-card border border-border rounded-lg p-4 shadow-sm space-y-3">
        <div className="flex justify-between items-center">
          <h2 className="font-semibold">Invoices</h2>
          <div className="flex gap-2 items-center">
            <select
              value={invoiceStatus}
              onChange={(e) => setInvoiceStatus(e.target.value)}
              className="h-10 rounded-md border border-input bg-background px-3 text-sm"
            >
              <option value="all">All status</option>
              <option value="draft">Draft</option>
              <option value="open">Open</option>
              <option value="paid">Paid</option>
              <option value="void">Void</option>
              <option value="uncollectible">Uncollectible</option>
            </select>
            <Button variant="outline" onClick={() => void load()} disabled={isLoading}>
              Refresh
            </Button>
          </div>
        </div>
        {isLoading ? (
          <div className="text-sm text-muted-foreground">Loading invoices...</div>
        ) : invoices.length === 0 ? (
          <div className="text-sm text-muted-foreground">No invoices found.</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm min-w-[980px]">
              <thead className="text-muted-foreground">
                <tr className="border-b">
                  <th className="text-left font-medium py-2 pr-3">Invoice</th>
                  <th className="text-left font-medium py-2 pr-3">Tenant</th>
                  <th className="text-left font-medium py-2 pr-3">Period</th>
                  <th className="text-right font-medium py-2 px-3">Amount</th>
                  <th className="text-left font-medium py-2 px-3">Status</th>
                  <th className="text-right font-medium py-2 pl-3">Action</th>
                </tr>
              </thead>
              <tbody>
                {invoices.map((i) => (
                  <tr key={i.id} className="border-b last:border-b-0">
                    <td className="py-2 pr-3">
                      <div className="font-medium">{i.invoice_number}</div>
                      <div className="text-xs text-muted-foreground">{new Date(i.created_at).toLocaleString()}</div>
                    </td>
                    <td className="py-2 pr-3">{i.tenant_name}</td>
                    <td className="py-2 pr-3">
                      {i.period_start} to {i.period_end}
                    </td>
                    <td className="py-2 px-3 text-right">${(i.total_cents / 100).toLocaleString()}</td>
                    <td className="py-2 px-3">{i.status}</td>
                    <td className="py-2 pl-3 text-right">
                      {i.status !== 'paid' ? (
                        <Button size="sm" variant="outline" onClick={() => void markPaid(i.id)}>
                          Mark paid
                        </Button>
                      ) : (
                        <span className="text-xs text-green-700 dark:text-green-300">Paid</span>
                      )}
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
