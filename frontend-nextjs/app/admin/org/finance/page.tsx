'use client'

import { useEffect, useMemo, useState } from 'react'
import { useAuthStore } from '@/store/useStore'
import { useRouter } from 'next/navigation'
import { DataCard, DataCardGrid } from '@/components/data/DataCard'
import { Button } from '@/components/ui/button'
import toast from 'react-hot-toast'

type BillingOverview = {
  plan_tier: string
  status: string
  billing_cycle: string
  seat_count: number
  active_users: number
  overage: number
  base_amount_cents: number
  per_seat_amount_cents: number
  projected_total_cents: number
  projected_overage_cents: number
  next_invoice_at?: string
  current_period_end?: string
}

type Invoice = {
  id: string
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

export default function OrgFinancePage() {
  const { isAuthenticated, user, token } = useAuthStore()
  const router = useRouter()
  const base = useMemo(() => process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080', [])
  const [overview, setOverview] = useState<BillingOverview | null>(null)
  const [invoices, setInvoices] = useState<Invoice[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [statusFilter, setStatusFilter] = useState('all')

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

  const load = async () => {
    try {
      setIsLoading(true)
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`
      const invoiceParams = new URLSearchParams()
      if (statusFilter !== 'all') invoiceParams.set('status', statusFilter)
      const [overviewResp, invoicesResp] = await Promise.all([
        fetch(`${base}/api/v1/org/finance/overview`, { headers }),
        fetch(`${base}/api/v1/org/finance/invoices?${invoiceParams.toString()}`, { headers }),
      ])
      if (!overviewResp.ok || !invoicesResp.ok) {
        throw new Error('Failed to load finance data')
      }
      setOverview(await overviewResp.json())
      setInvoices(await invoicesResp.json())
    } catch (e: any) {
      toast.error(e.message || 'Failed to load finance data')
    } finally {
      setIsLoading(false)
    }
  }

  if (!isAuthenticated || !user) return null

  return (
    <div className="container mx-auto px-4 py-8 space-y-6">
      <div>
        <h1 className="text-2xl font-bold mb-2">Finance</h1>
        <p className="text-muted-foreground">Subscription status, seat usage, overage, and invoice history.</p>
      </div>

      {overview && (
        <DataCardGrid>
          <DataCard title="Plan" value={overview.plan_tier} subtitle={`${overview.status} • ${overview.billing_cycle}`} />
          <DataCard title="Seats" value={`${overview.active_users}/${overview.seat_count}`} subtitle={`${overview.overage} overage`} />
          <DataCard title="Projected Bill" value={`$${(overview.projected_total_cents / 100).toLocaleString()}`} subtitle={`Overage $${(overview.projected_overage_cents / 100).toLocaleString()}`} />
          <DataCard title="Next Invoice" value={overview.next_invoice_at ? new Date(overview.next_invoice_at).toLocaleDateString() : '—'} subtitle={overview.current_period_end ? `Period ends ${new Date(overview.current_period_end).toLocaleDateString()}` : 'No active cycle'} />
        </DataCardGrid>
      )}

      <div className="bg-card border border-border rounded-lg p-4 shadow-sm space-y-3">
        <div className="flex items-center justify-between gap-3">
          <h2 className="font-semibold">Invoice history</h2>
          <div className="flex gap-2">
            <select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)} className="h-10 rounded-md border border-input bg-background px-3 text-sm">
              <option value="all">All</option>
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
          <div className="text-sm text-muted-foreground">No invoices available for this organization.</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="text-muted-foreground">
                <tr className="border-b">
                  <th className="text-left py-2 pr-3">Invoice</th>
                  <th className="text-left py-2 pr-3">Period</th>
                  <th className="text-left py-2 pr-3">Status</th>
                  <th className="text-right py-2 pr-3">Amount</th>
                  <th className="text-left py-2">Due / Paid</th>
                </tr>
              </thead>
              <tbody>
                {invoices.map((invoice) => (
                  <tr key={invoice.id} className="border-b last:border-b-0">
                    <td className="py-2 pr-3">
                      <div className="font-medium">{invoice.invoice_number}</div>
                      <div className="text-xs text-muted-foreground">{new Date(invoice.created_at).toLocaleString()}</div>
                    </td>
                    <td className="py-2 pr-3">
                      {invoice.period_start} to {invoice.period_end}
                    </td>
                    <td className="py-2 pr-3">{invoice.status}</td>
                    <td className="py-2 pr-3 text-right">${(invoice.total_cents / 100).toLocaleString()}</td>
                    <td className="py-2">
                      <div>{invoice.due_at ? new Date(invoice.due_at).toLocaleDateString() : '—'}</div>
                      <div className="text-xs text-muted-foreground">{invoice.paid_at ? `Paid ${new Date(invoice.paid_at).toLocaleDateString()}` : invoice.payment_reference || 'Unpaid'}</div>
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
