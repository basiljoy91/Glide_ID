'use client'

import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import toast from 'react-hot-toast'

export default function ContactPage() {
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [message, setMessage] = useState('')
  const [isSending, setIsSending] = useState(false)

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim() || !email.trim() || !message.trim()) {
      toast.error('Please fill all fields')
      return
    }
    setIsSending(true)
    try {
      // No backend mailer yet; keep as UX-complete demo.
      await new Promise((r) => setTimeout(r, 600))
      toast.success('Message received (demo). We will contact you soon.')
      setName('')
      setEmail('')
      setMessage('')
    } finally {
      setIsSending(false)
    }
  }

  return (
    <div className="container mx-auto px-4 py-16">
      <div className="max-w-2xl mx-auto space-y-6">
        <div>
          <h1 className="text-4xl font-bold tracking-tight">Contact</h1>
          <p className="mt-3 text-muted-foreground">
            Tell us about your organization and kiosk requirements. We’ll help you plan onboarding,
            SSO, and rollout.
          </p>
        </div>

        <form onSubmit={submit} className="border rounded-lg p-6 bg-card space-y-4">
          <div className="grid sm:grid-cols-2 gap-4">
            <div>
              <Label htmlFor="name">Name</Label>
              <Input id="name" value={name} onChange={(e) => setName(e.target.value)} className="mt-1" />
            </div>
            <div>
              <Label htmlFor="email">Email</Label>
              <Input id="email" value={email} onChange={(e) => setEmail(e.target.value)} className="mt-1" />
            </div>
          </div>
          <div>
            <Label htmlFor="message">Message</Label>
            <textarea
              id="message"
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              className="mt-1 flex min-h-[140px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              placeholder="What do you want to build? How many employees/kiosks? Any compliance requirements?"
            />
          </div>
          <Button type="submit" disabled={isSending}>
            {isSending ? 'Sending…' : 'Send message'}
          </Button>
          <div className="text-xs text-muted-foreground">
            Note: This form is currently a demo UI (no email delivery wired yet).
          </div>
        </form>
      </div>
    </div>
  )
}

