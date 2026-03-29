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

type BillableOrg = {
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

export default function SuperAdminBillingPage() {
  const { isAuthenticated, user, token } = useAuthStore()
  const router = useRouter()
  const [overview, setOverview] = useState<BillingOverview | null>(null)
  const [invoices, setInvoices] = useState<Invoice[]>([])
  const [organizations, setOrganizations] = useState<BillableOrg[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [invoiceStatus, setInvoiceStatus] = useState('all')
  const [invoiceTenantFilter, setInvoiceTenantFilter] = useState('')
  const [invoicePage, setInvoicePage] = useState(0)
  const [invoicePageSize, setInvoicePageSize] = useState(25)
  const [hasNextInvoicePage, setHasNextInvoicePage] = useState(false)

  const [orgQuery, setOrgQuery] = useState('')
  const [tenantId, setTenantId] = useState('')
  const [periodStart, setPeriodStart] = useState(new Date(Date.now() - 30 * 86400000).toISOString().slice(0, 10))
  const [periodEnd, setPeriodEnd] = useState(new Date().toISOString().slice(0, 10))
  const [taxCents, setTaxCents] = useState('0')
  const [creating, setCreating] = useState(false)
  const [paymentForms, setPaymentForms] = useState<Record<string, { payment_reference: string; payment_amount_cents: string }>>({})
  const [markingPaidId, setMarkingPaidId] = useState<string | null>(null)

  useEffect(() => {
    if (!isAuthenticated || user?.role !== 'super_admin') {
      router.push('/admin/login')
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, user?.role])

  const load = async () => {
    try {
      setIsLoading(true)
      const headers: Record<string, string> = {}
      if (token) headers.Authorization = `Bearer ${token}`
      const base = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      const invoiceQS = new URLSearchParams()
      invoiceQS.set('limit', String(invoicePageSize))
      invoiceQS.set('offset', String(invoicePage * invoicePageSize))
      if (invoiceStatus !== 'all') invoiceQS.set('status', invoiceStatus)
      if (invoiceTenantFilter) invoiceQS.set('tenant_id', invoiceTenantFilter)

      const [overviewResp, invoicesResp, organizationsResp] = await Promise.all([
        fetch(`${base}/api/v1/admin/super/billing/overview`, { headers }),
        fetch(`${base}/api/v1/admin/super/billing/invoices?${invoiceQS.toString()}`, { headers }),
        fetch(`${base}/api/v1/admin/super/organizations?limit=200`, { headers }),
      ])

      if (!overviewResp.ok) {
        const err = await overviewResp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to load billing overview')
      }
      if (!invoicesResp.ok) {
        const err = await invoicesResp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to load invoices')
      }
      if (!organizationsResp.ok) {
        const err = await organizationsResp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to load organizations')
      }

      setOverview(await overviewResp.json())
      const invoiceData = await invoicesResp.json()
      setInvoices(invoiceData)
      setHasNextInvoicePage(invoiceData.length === invoicePageSize)
      setOrganizations(await organizationsResp.json())
    } catch (e: any) {
      toast.error(e.message || 'Failed to load billing data')
      setHasNextInvoicePage(false)
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    if (!isAuthenticated || user?.role !== 'super_admin') return
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [invoiceStatus, invoiceTenantFilter, invoicePage, invoicePageSize])

  const filteredOrganizations = organizations.filter((org) => {
    const q = orgQuery.trim().toLowerCase()
    if (!q) return true
    return org.name.toLowerCase().includes(q) || org.slug.toLowerCase().includes(q)
  })

  const selectedOrganization = organizations.find((org) => org.id === tenantId) || null
  const invoicePreviewSubtotal = selectedOrganization
    ? selectedOrganization.base_amount_cents + selectedOrganization.per_seat_amount_cents * selectedOrganization.seat_count
    : 0
  const invoicePreviewTotal = invoicePreviewSubtotal + (Number(taxCents) || 0)
  const canCreateInvoice =
    !!selectedOrganization &&
    selectedOrganization.is_active &&
    ['trialing', 'active', 'past_due'].includes(selectedOrganization.billing_status)

  const createInvoice = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!selectedOrganization) {
      toast.error('Select an organization')
      return
    }
    if (!canCreateInvoice) {
      toast.error('The selected organization must be active and billable')
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
            tenant_id: selectedOrganization.id,
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
      setTenantId('')
      setOrgQuery('')
      setInvoicePage(0)
      await load()
    } catch (e: any) {
      toast.error(e.message || 'Failed to create invoice')
    } finally {
      setCreating(false)
    }
  }

  const markPaid = async (invoice: Invoice) => {
    const paymentForm = paymentForms[invoice.id] || {
      payment_reference: '',
      payment_amount_cents: String(invoice.total_cents),
    }
    if (!paymentForm.payment_reference.trim()) {
      toast.error('Payment reference is required')
      return
    }
    if (Number(paymentForm.payment_amount_cents) !== invoice.total_cents) {
      toast.error('Payment amount must match the invoice total')
      return
    }
    try {
      setMarkingPaidId(invoice.id)
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/admin/super/billing/invoices/${invoice.id}/mark-paid`,
        {
          method: 'PATCH',
          headers,
          body: JSON.stringify({
            payment_reference: paymentForm.payment_reference.trim(),
            payment_amount_cents: Number(paymentForm.payment_amount_cents),
          }),
        }
      )
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to mark invoice paid')
      }
      toast.success('Invoice marked paid')
      setPaymentForms((prev) => {
        const next = { ...prev }
        delete next[invoice.id]
        return next
      })
      await load()
    } catch (e: any) {
      toast.error(e.message || 'Failed to mark invoice paid')
    } finally {
      setMarkingPaidId(null)
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
        <div className="grid gap-3 lg:grid-cols-[1.4fr_1fr]">
          <div className="space-y-3">
            <Input
              value={orgQuery}
              onChange={(e) => setOrgQuery(e.target.value)}
              placeholder="Search organization by name or slug"
            />
            <select
              value={tenantId}
              onChange={(e) => setTenantId(e.target.value)}
              className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
            >
              <option value="">Select organization</option>
              {filteredOrganizations.map((org) => (
                <option key={org.id} value={org.id}>
                  {org.name} ({org.slug})
                </option>
              ))}
            </select>
            <div className="grid md:grid-cols-3 gap-3">
              <Input type="date" value={periodStart} onChange={(e) => setPeriodStart(e.target.value)} />
              <Input type="date" value={periodEnd} onChange={(e) => setPeriodEnd(e.target.value)} />
              <Input type="number" value={taxCents} onChange={(e) => setTaxCents(e.target.value)} placeholder="Tax cents" />
            </div>
          </div>
          <div className="rounded-lg border border-border bg-muted/30 p-4 text-sm space-y-2">
            <div className="font-medium">Invoice preview</div>
            {selectedOrganization ? (
              <>
                <div className="flex justify-between gap-3">
                  <span className="text-muted-foreground">Organization</span>
                  <span>{selectedOrganization.name}</span>
                </div>
                <div className="flex justify-between gap-3">
                  <span className="text-muted-foreground">Tier</span>
                  <span>{selectedOrganization.subscription_tier}</span>
                </div>
                <div className="flex justify-between gap-3">
                  <span className="text-muted-foreground">Billing status</span>
                  <span>{selectedOrganization.billing_status}</span>
                </div>
                <div className="flex justify-between gap-3">
                  <span className="text-muted-foreground">Seats</span>
                  <span>
                    {selectedOrganization.seat_count} billed / {selectedOrganization.users_count} users
                  </span>
                </div>
                <div className="flex justify-between gap-3">
                  <span className="text-muted-foreground">Base</span>
                  <span>${(selectedOrganization.base_amount_cents / 100).toLocaleString()}</span>
                </div>
                <div className="flex justify-between gap-3">
                  <span className="text-muted-foreground">Per seat</span>
                  <span>${(selectedOrganization.per_seat_amount_cents / 100).toLocaleString()}</span>
                </div>
                <div className="flex justify-between gap-3 border-t pt-2">
                  <span className="text-muted-foreground">Preview total</span>
                  <span className="font-medium">${(invoicePreviewTotal / 100).toLocaleString()}</span>
                </div>
                {!canCreateInvoice ? (
                  <div className="rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-amber-900">
                    This organization cannot be invoiced until it is active and has a billable subscription state.
                  </div>
                ) : null}
              </>
            ) : (
              <div className="text-muted-foreground">Select an organization to preview pricing and invoice eligibility.</div>
            )}
          </div>
        </div>
        <Button type="submit" disabled={creating || !canCreateInvoice}>
          {creating ? 'Creating…' : 'Create invoice'}
        </Button>
      </form>

      <div className="bg-card border border-border rounded-lg p-4 shadow-sm space-y-3">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <h2 className="font-semibold">Invoices</h2>
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <select
              value={invoiceTenantFilter}
              onChange={(e) => {
                setInvoicePage(0)
                setInvoiceTenantFilter(e.target.value)
              }}
              className="h-10 rounded-md border border-input bg-background px-3 text-sm"
            >
              <option value="">All organizations</option>
              {organizations.map((org) => (
                <option key={org.id} value={org.id}>
                  {org.name}
                </option>
              ))}
            </select>
            <select
              value={invoiceStatus}
              onChange={(e) => {
                setInvoicePage(0)
                setInvoiceStatus(e.target.value)
              }}
              className="h-10 rounded-md border border-input bg-background px-3 text-sm"
            >
              <option value="all">All status</option>
              <option value="draft">Draft</option>
              <option value="open">Open</option>
              <option value="paid">Paid</option>
              <option value="void">Void</option>
              <option value="uncollectible">Uncollectible</option>
            </select>
            <select
              value={String(invoicePageSize)}
              onChange={(e) => {
                setInvoicePage(0)
                setInvoicePageSize(Number(e.target.value))
              }}
              className="h-10 rounded-md border border-input bg-background px-3 text-sm"
            >
              <option value="10">10 / page</option>
              <option value="25">25 / page</option>
              <option value="50">50 / page</option>
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
                  <th className="text-right font-medium py-2 pl-3">Settlement</th>
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
                      {i.status === 'open' ? (
                        <div className="flex flex-col items-end gap-2">
                          <Input
                            value={paymentForms[i.id]?.payment_reference ?? ''}
                            onChange={(e) =>
                              setPaymentForms((prev) => ({
                                ...prev,
                                [i.id]: {
                                  payment_reference: e.target.value,
                                  payment_amount_cents: prev[i.id]?.payment_amount_cents ?? String(i.total_cents),
                                },
                              }))
                            }
                            placeholder="Payment reference"
                            className="w-48"
                          />
                          <div className="flex items-center gap-2">
                            <Input
                              type="number"
                              value={paymentForms[i.id]?.payment_amount_cents ?? String(i.total_cents)}
                              onChange={(e) =>
                                setPaymentForms((prev) => ({
                                  ...prev,
                                  [i.id]: {
                                    payment_reference: prev[i.id]?.payment_reference ?? '',
                                    payment_amount_cents: e.target.value,
                                  },
                                }))
                              }
                              className="w-36"
                            />
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => void markPaid(i)}
                              disabled={markingPaidId === i.id}
                            >
                              {markingPaidId === i.id ? 'Saving…' : 'Mark paid'}
                            </Button>
                          </div>
                        </div>
                      ) : i.status === 'paid' ? (
                        <span className="text-xs text-green-700 dark:text-green-300">Paid</span>
                      ) : (
                        <span className="text-xs text-muted-foreground">Status locked</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <div className="flex flex-col gap-3 rounded-lg border border-border bg-card p-4 shadow-sm sm:flex-row sm:items-center sm:justify-between">
        <div className="text-sm text-muted-foreground">
          Page {invoicePage + 1}
          {invoices.length > 0 ? ` • Showing ${invoicePage * invoicePageSize + 1}-${invoicePage * invoicePageSize + invoices.length}` : ''}
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            onClick={() => setInvoicePage((prev) => Math.max(prev - 1, 0))}
            disabled={invoicePage === 0 || isLoading}
          >
            Previous
          </Button>
          <Button
            variant="outline"
            onClick={() => setInvoicePage((prev) => prev + 1)}
            disabled={!hasNextInvoicePage || isLoading}
          >
            Next
          </Button>
        </div>
      </div>
    </div>
  )
}
