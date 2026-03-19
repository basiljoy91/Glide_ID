'use client'

import { useEffect, useMemo, useState } from 'react'
import { useAuthStore } from '@/store/useStore'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import toast from 'react-hot-toast'

type Ticket = {
  id: string
  category: string
  priority: string
  subject: string
  description: string
  status: string
  created_at: string
  resolved_at?: string | null
}

export default function OrgSupportPage() {
  const { isAuthenticated, user, token } = useAuthStore()
  const router = useRouter()
  const base = useMemo(() => process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080', [])
  const [tickets, setTickets] = useState<Ticket[]>([])
  const [subject, setSubject] = useState('')
  const [description, setDescription] = useState('')
  const [category, setCategory] = useState('general')
  const [priority, setPriority] = useState('normal')
  const [isLoading, setIsLoading] = useState(true)
  const [isSubmitting, setIsSubmitting] = useState(false)

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
      const resp = await fetch(`${base}/api/v1/org/support/tickets`, { headers })
      if (!resp.ok) throw new Error('Failed to load support tickets')
      setTickets(await resp.json())
    } catch (e: any) {
      toast.error(e.message || 'Failed to load support tickets')
    } finally {
      setIsLoading(false)
    }
  }

  const createTicket = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!subject.trim() || !description.trim()) {
      toast.error('Subject and description are required')
      return
    }
    try {
      setIsSubmitting(true)
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (token) headers.Authorization = `Bearer ${token}`
      const resp = await fetch(`${base}/api/v1/org/support/tickets`, {
        method: 'POST',
        headers,
        body: JSON.stringify({ category, priority, subject, description }),
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to submit support ticket')
      }
      toast.success('Support ticket submitted')
      setSubject('')
      setDescription('')
      setCategory('general')
      setPriority('normal')
      await load()
    } catch (e: any) {
      toast.error(e.message || 'Failed to submit support ticket')
    } finally {
      setIsSubmitting(false)
    }
  }

  if (!isAuthenticated || !user) return null

  return (
    <div className="container mx-auto px-4 py-8 space-y-6">
      <div>
        <h1 className="text-2xl font-bold mb-2">Support</h1>
        <p className="text-muted-foreground">Report issues and track organization support requests.</p>
      </div>

      <form onSubmit={createTicket} className="bg-card border border-border rounded-lg p-4 shadow-sm space-y-3">
        <h2 className="font-semibold">Open a ticket</h2>
        <div className="grid md:grid-cols-3 gap-3">
          <select value={category} onChange={(e) => setCategory(e.target.value)} className="h-10 rounded-md border border-input bg-background px-3 text-sm">
            <option value="general">General</option>
            <option value="billing">Billing</option>
            <option value="integration">Integration</option>
            <option value="kiosk">Kiosk</option>
          </select>
          <select value={priority} onChange={(e) => setPriority(e.target.value)} className="h-10 rounded-md border border-input bg-background px-3 text-sm">
            <option value="low">Low</option>
            <option value="normal">Normal</option>
            <option value="high">High</option>
            <option value="urgent">Urgent</option>
          </select>
          <Input value={subject} onChange={(e) => setSubject(e.target.value)} placeholder="Subject" />
        </div>
        <textarea value={description} onChange={(e) => setDescription(e.target.value)} className="flex min-h-[120px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm" placeholder="Describe the issue, impact, and expected outcome." />
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting ? 'Submitting…' : 'Submit ticket'}
        </Button>
      </form>

      <div className="bg-card border border-border rounded-lg p-4 shadow-sm">
        <div className="font-semibold mb-3">Open and historical tickets</div>
        {isLoading ? (
          <div className="text-sm text-muted-foreground">Loading tickets...</div>
        ) : tickets.length === 0 ? (
          <div className="text-sm text-muted-foreground">No tickets submitted yet.</div>
        ) : (
          <div className="space-y-3">
            {tickets.map((ticket) => (
              <div key={ticket.id} className="rounded-md border p-3">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="font-medium">{ticket.subject}</div>
                    <div className="text-xs text-muted-foreground">
                      {ticket.category} • {ticket.priority} • {ticket.status}
                    </div>
                  </div>
                  <div className="text-xs text-muted-foreground">{new Date(ticket.created_at).toLocaleString()}</div>
                </div>
                <div className="mt-2 text-sm text-muted-foreground">{ticket.description}</div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
