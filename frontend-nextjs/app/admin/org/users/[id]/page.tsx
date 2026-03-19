'use client'

import { useEffect, useMemo, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import toast from 'react-hot-toast'
import { useAuthStore } from '@/store/useStore'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface EmployeeSummary {
  id: string
  first_name: string
  last_name: string
  email: string
  role: string
}

interface EmployeeUser {
  id: string
  employee_id: string
  email: string
  first_name: string
  last_name: string
  phone?: string | null
  department_id?: string | null
  designation?: string | null
  date_of_joining: string
  role: string
  is_active: boolean
  manager_id?: string | null
  employment_type?: string | null
  work_location?: string | null
  cost_center?: string | null
  invite_status?: string | null
  invite_sent_at?: string | null
  offboarded_at?: string | null
  offboarding_reason?: string | null
}

interface EmergencyContact {
  id: string
  name: string
  relationship?: string | null
  phone: string
  email?: string | null
  is_primary: boolean
}

interface EmployeeDocument {
  id: string
  document_type: string
  name: string
  file_url: string
  expires_at?: string | null
}

interface ProfileResponse {
  user: EmployeeUser
  department_name?: string | null
  manager?: EmployeeSummary | null
  direct_reports: EmployeeSummary[]
  emergency_contacts: EmergencyContact[]
  documents: EmployeeDocument[]
}

interface Department {
  id: string
  name: string
}

export default function EmployeeProfilePage() {
  const params = useParams<{ id: string }>()
  const router = useRouter()
  const { user, token, isAuthenticated } = useAuthStore()
  const [profile, setProfile] = useState<ProfileResponse | null>(null)
  const [departments, setDepartments] = useState<Department[]>([])
  const [managers, setManagers] = useState<EmployeeSummary[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isSaving, setIsSaving] = useState(false)
  const [contactForm, setContactForm] = useState({ name: '', relationship: '', phone: '', email: '', is_primary: false })
  const [documentForm, setDocumentForm] = useState({ document_type: 'contract', name: '', file_url: '', expires_at: '' })
  const [offboardingReason, setOffboardingReason] = useState('')

  const base = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
  const employeeId = Array.isArray(params?.id) ? params.id[0] : params?.id

  const [form, setForm] = useState({
    email: '',
    first_name: '',
    last_name: '',
    phone: '',
    department_id: '',
    designation: '',
    date_of_joining: '',
    is_active: true,
    manager_id: '',
    employment_type: 'full_time',
    work_location: '',
    cost_center: '',
  })

  const headers = useMemo(
    () => ({
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    }),
    [token]
  )

  useEffect(() => {
    if (!isAuthenticated || !user) {
      router.replace('/admin/login')
      return
    }
    if (!['org_admin', 'hr'].includes(user.role)) {
      router.replace('/dashboard')
      return
    }
    if (!employeeId) return
    void Promise.all([loadProfile(), loadDepartments(), loadManagers()])
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [employeeId, isAuthenticated, user?.role])

  const loadProfile = async () => {
    if (!employeeId) return
    try {
      setIsLoading(true)
      const resp = await fetch(`${base}/api/v1/workforce/employees/${employeeId}/profile`, { headers })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to load employee profile')
      }
      const data = await resp.json()
      setProfile(data)
      setForm({
        email: data.user.email || '',
        first_name: data.user.first_name || '',
        last_name: data.user.last_name || '',
        phone: data.user.phone || '',
        department_id: data.user.department_id || '',
        designation: data.user.designation || '',
        date_of_joining: data.user.date_of_joining?.slice(0, 10) || '',
        is_active: Boolean(data.user.is_active),
        manager_id: data.user.manager_id || '',
        employment_type: data.user.employment_type || 'full_time',
        work_location: data.user.work_location || '',
        cost_center: data.user.cost_center || '',
      })
      setOffboardingReason(data.user.offboarding_reason || '')
    } catch (e: any) {
      toast.error(e.message || 'Failed to load employee profile')
    } finally {
      setIsLoading(false)
    }
  }

  const loadDepartments = async () => {
    const resp = await fetch(`${base}/api/v1/departments`, { headers })
    if (!resp.ok) return
    const data = await resp.json()
    setDepartments(Array.isArray(data) ? data : [])
  }

  const loadManagers = async () => {
    const resp = await fetch(`${base}/api/v1/users?limit=200`, { headers })
    if (!resp.ok) return
    const data = await resp.json()
    const rows = Array.isArray(data?.data) ? data.data : Array.isArray(data) ? data : []
    setManagers(
      rows
        .filter((row: any) => row.is_active)
        .map((row: any) => ({
          id: row.id,
          first_name: row.first_name,
          last_name: row.last_name,
          email: row.email,
          role: row.role,
        }))
    )
  }

  const saveProfile = async () => {
    if (!employeeId) return
    try {
      setIsSaving(true)
      const resp = await fetch(`${base}/api/v1/workforce/employees/${employeeId}/profile`, {
        method: 'PUT',
        headers,
        body: JSON.stringify({
          ...form,
          phone: form.phone || null,
          department_id: form.department_id || null,
          designation: form.designation || null,
          manager_id: form.manager_id || null,
          work_location: form.work_location || null,
          cost_center: form.cost_center || null,
        }),
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to update profile')
      }
      toast.success('Employee profile updated')
      await loadProfile()
    } catch (e: any) {
      toast.error(e.message || 'Failed to update profile')
    } finally {
      setIsSaving(false)
    }
  }

  const resendInvite = async () => {
    if (!employeeId) return
    try {
      const resp = await fetch(`${base}/api/v1/workforce/employees/${employeeId}/invite/resend`, {
        method: 'POST',
        headers,
      })
      const data = await resp.json().catch(() => ({}))
      if (!resp.ok) throw new Error(data.error || 'Failed to resend invite')
      if (data.invite_url) {
        await navigator.clipboard.writeText(data.invite_url)
      }
      toast.success('Invite resent and link copied')
      await loadProfile()
    } catch (e: any) {
      toast.error(e.message || 'Failed to resend invite')
    }
  }

  const offboardEmployee = async () => {
    if (!employeeId) return
    if (!window.confirm('Offboard this employee? Access will be revoked immediately.')) return
    try {
      const resp = await fetch(`${base}/api/v1/workforce/employees/${employeeId}/offboard`, {
        method: 'POST',
        headers,
        body: JSON.stringify({ reason: offboardingReason }),
      })
      const data = await resp.json().catch(() => ({}))
      if (!resp.ok) throw new Error(data.error || 'Failed to offboard employee')
      toast.success('Employee offboarded')
      await loadProfile()
    } catch (e: any) {
      toast.error(e.message || 'Failed to offboard employee')
    }
  }

  const addContact = async () => {
    if (!employeeId) return
    try {
      const resp = await fetch(`${base}/api/v1/workforce/employees/${employeeId}/emergency-contacts`, {
        method: 'POST',
        headers,
        body: JSON.stringify({
          ...contactForm,
          relationship: contactForm.relationship || null,
          email: contactForm.email || null,
        }),
      })
      const data = await resp.json().catch(() => ({}))
      if (!resp.ok) throw new Error(data.error || 'Failed to add emergency contact')
      toast.success('Emergency contact added')
      setContactForm({ name: '', relationship: '', phone: '', email: '', is_primary: false })
      await loadProfile()
    } catch (e: any) {
      toast.error(e.message || 'Failed to add emergency contact')
    }
  }

  const deleteContact = async (contactId: string) => {
    if (!employeeId) return
    try {
      const resp = await fetch(`${base}/api/v1/workforce/employees/${employeeId}/emergency-contacts/${contactId}`, {
        method: 'DELETE',
        headers,
      })
      const data = await resp.json().catch(() => ({}))
      if (!resp.ok) throw new Error(data.error || 'Failed to delete emergency contact')
      toast.success('Emergency contact removed')
      await loadProfile()
    } catch (e: any) {
      toast.error(e.message || 'Failed to delete emergency contact')
    }
  }

  const addDocument = async () => {
    if (!employeeId) return
    try {
      const resp = await fetch(`${base}/api/v1/workforce/employees/${employeeId}/documents`, {
        method: 'POST',
        headers,
        body: JSON.stringify({
          ...documentForm,
          expires_at: documentForm.expires_at ? `${documentForm.expires_at}T00:00:00Z` : null,
        }),
      })
      const data = await resp.json().catch(() => ({}))
      if (!resp.ok) throw new Error(data.error || 'Failed to add document')
      toast.success('Document saved')
      setDocumentForm({ document_type: 'contract', name: '', file_url: '', expires_at: '' })
      await loadProfile()
    } catch (e: any) {
      toast.error(e.message || 'Failed to add document')
    }
  }

  const deleteDocument = async (documentId: string) => {
    if (!employeeId) return
    try {
      const resp = await fetch(`${base}/api/v1/workforce/employees/${employeeId}/documents/${documentId}`, {
        method: 'DELETE',
        headers,
      })
      const data = await resp.json().catch(() => ({}))
      if (!resp.ok) throw new Error(data.error || 'Failed to delete document')
      toast.success('Document removed')
      await loadProfile()
    } catch (e: any) {
      toast.error(e.message || 'Failed to delete document')
    }
  }

  if (!isAuthenticated || !user) return null

  return (
    <div className="container mx-auto px-4 py-8 space-y-8">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">Employee Profile</h1>
          <p className="text-muted-foreground">
            {profile ? `${profile.user.first_name} ${profile.user.last_name} · ${profile.user.employee_id}` : 'Loading employee profile'}
          </p>
        </div>
        <div className="flex gap-2">
          <Link href="/admin/org/users">
            <Button variant="outline">Back to Employees</Button>
          </Link>
          <Link href="/admin/org/operations">
            <Button variant="outline">Attendance Operations</Button>
          </Link>
        </div>
      </div>

      {isLoading || !profile ? (
        <div className="rounded-lg border bg-card p-6 text-sm text-muted-foreground">Loading employee profile...</div>
      ) : (
        <>
          <div className="grid gap-6 lg:grid-cols-[2fr,1fr]">
            <div className="rounded-lg border bg-card p-6 space-y-4">
              <div className="flex items-center justify-between">
                <h2 className="text-lg font-semibold">Core Profile</h2>
                <Button onClick={() => void saveProfile()} disabled={isSaving}>
                  {isSaving ? 'Saving...' : 'Save Profile'}
                </Button>
              </div>
              <div className="grid gap-4 md:grid-cols-2">
                <div>
                  <Label>Email</Label>
                  <Input value={form.email} onChange={(e) => setForm((prev) => ({ ...prev, email: e.target.value }))} className="mt-1" />
                </div>
                <div>
                  <Label>Phone</Label>
                  <Input value={form.phone} onChange={(e) => setForm((prev) => ({ ...prev, phone: e.target.value }))} className="mt-1" />
                </div>
                <div>
                  <Label>First Name</Label>
                  <Input value={form.first_name} onChange={(e) => setForm((prev) => ({ ...prev, first_name: e.target.value }))} className="mt-1" />
                </div>
                <div>
                  <Label>Last Name</Label>
                  <Input value={form.last_name} onChange={(e) => setForm((prev) => ({ ...prev, last_name: e.target.value }))} className="mt-1" />
                </div>
                <div>
                  <Label>Department</Label>
                  <select
                    value={form.department_id}
                    onChange={(e) => setForm((prev) => ({ ...prev, department_id: e.target.value }))}
                    className="mt-1 flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                  >
                    <option value="">Unassigned</option>
                    {departments.map((department) => (
                      <option key={department.id} value={department.id}>
                        {department.name}
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <Label>Reporting Manager</Label>
                  <select
                    value={form.manager_id}
                    onChange={(e) => setForm((prev) => ({ ...prev, manager_id: e.target.value }))}
                    className="mt-1 flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                  >
                    <option value="">None</option>
                    {managers
                      .filter((manager) => manager.id !== profile.user.id)
                      .map((manager) => (
                        <option key={manager.id} value={manager.id}>
                          {manager.first_name} {manager.last_name} · {manager.role}
                        </option>
                      ))}
                  </select>
                </div>
                <div>
                  <Label>Designation</Label>
                  <Input value={form.designation} onChange={(e) => setForm((prev) => ({ ...prev, designation: e.target.value }))} className="mt-1" />
                </div>
                <div>
                  <Label>Date of Joining</Label>
                  <Input type="date" value={form.date_of_joining} onChange={(e) => setForm((prev) => ({ ...prev, date_of_joining: e.target.value }))} className="mt-1" />
                </div>
                <div>
                  <Label>Employment Type</Label>
                  <select
                    value={form.employment_type}
                    onChange={(e) => setForm((prev) => ({ ...prev, employment_type: e.target.value }))}
                    className="mt-1 flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                  >
                    <option value="full_time">Full time</option>
                    <option value="part_time">Part time</option>
                    <option value="contract">Contract</option>
                    <option value="intern">Intern</option>
                  </select>
                </div>
                <div>
                  <Label>Work Location</Label>
                  <Input value={form.work_location} onChange={(e) => setForm((prev) => ({ ...prev, work_location: e.target.value }))} className="mt-1" />
                </div>
                <div>
                  <Label>Cost Center</Label>
                  <Input value={form.cost_center} onChange={(e) => setForm((prev) => ({ ...prev, cost_center: e.target.value }))} className="mt-1" />
                </div>
                <div className="flex items-center gap-2 pt-6">
                  <input
                    id="is_active"
                    type="checkbox"
                    checked={form.is_active}
                    onChange={(e) => setForm((prev) => ({ ...prev, is_active: e.target.checked }))}
                  />
                  <Label htmlFor="is_active">Active employee</Label>
                </div>
              </div>
            </div>

            <div className="rounded-lg border bg-card p-6 space-y-4">
              <h2 className="text-lg font-semibold">Status</h2>
              <div className="text-sm text-muted-foreground space-y-2">
                <div>Invite status: <span className="font-medium text-foreground">{profile.user.invite_status || 'not_invited'}</span></div>
                <div>Invite sent: {profile.user.invite_sent_at ? new Date(profile.user.invite_sent_at).toLocaleString() : '—'}</div>
                <div>Department: {profile.department_name || '—'}</div>
                <div>Current manager: {profile.manager ? `${profile.manager.first_name} ${profile.manager.last_name}` : '—'}</div>
                <div>Direct reports: {profile.direct_reports.length}</div>
                <div>Offboarded at: {profile.user.offboarded_at ? new Date(profile.user.offboarded_at).toLocaleString() : '—'}</div>
              </div>
              <div className="space-y-3 border-t pt-4">
                <Button variant="outline" className="w-full" onClick={() => void resendInvite()}>
                  Resend Invite
                </Button>
                <div className="space-y-2">
                  <Label>Offboarding Reason</Label>
                  <Input value={offboardingReason} onChange={(e) => setOffboardingReason(e.target.value)} />
                </div>
                <Button variant="destructive" className="w-full" onClick={() => void offboardEmployee()}>
                  Offboard Employee
                </Button>
              </div>
            </div>
          </div>

          <div className="grid gap-6 lg:grid-cols-2">
            <div className="rounded-lg border bg-card p-6 space-y-4">
              <h2 className="text-lg font-semibold">Emergency Contacts</h2>
              <div className="space-y-3">
                {profile.emergency_contacts.length === 0 ? (
                  <div className="text-sm text-muted-foreground">No emergency contacts recorded.</div>
                ) : (
                  profile.emergency_contacts.map((contact) => (
                    <div key={contact.id} className="rounded-md border p-3 flex items-start justify-between gap-3">
                      <div>
                        <div className="font-medium">
                          {contact.name} {contact.is_primary ? <span className="text-xs text-emerald-600">Primary</span> : null}
                        </div>
                        <div className="text-sm text-muted-foreground">{contact.relationship || '—'} · {contact.phone}</div>
                        <div className="text-sm text-muted-foreground">{contact.email || '—'}</div>
                      </div>
                      <Button variant="ghost" size="sm" onClick={() => void deleteContact(contact.id)}>
                        Remove
                      </Button>
                    </div>
                  ))
                )}
              </div>
              <div className="grid gap-3 md:grid-cols-2 border-t pt-4">
                <Input placeholder="Name" value={contactForm.name} onChange={(e) => setContactForm((prev) => ({ ...prev, name: e.target.value }))} />
                <Input placeholder="Relationship" value={contactForm.relationship} onChange={(e) => setContactForm((prev) => ({ ...prev, relationship: e.target.value }))} />
                <Input placeholder="Phone" value={contactForm.phone} onChange={(e) => setContactForm((prev) => ({ ...prev, phone: e.target.value }))} />
                <Input placeholder="Email" value={contactForm.email} onChange={(e) => setContactForm((prev) => ({ ...prev, email: e.target.value }))} />
                <label className="flex items-center gap-2 text-sm text-muted-foreground">
                  <input type="checkbox" checked={contactForm.is_primary} onChange={(e) => setContactForm((prev) => ({ ...prev, is_primary: e.target.checked }))} />
                  Primary contact
                </label>
                <Button onClick={() => void addContact()}>Add Contact</Button>
              </div>
            </div>

            <div className="rounded-lg border bg-card p-6 space-y-4">
              <h2 className="text-lg font-semibold">Documents And Contracts</h2>
              <div className="space-y-3">
                {profile.documents.length === 0 ? (
                  <div className="text-sm text-muted-foreground">No employee documents stored.</div>
                ) : (
                  profile.documents.map((document) => (
                    <div key={document.id} className="rounded-md border p-3 flex items-start justify-between gap-3">
                      <div>
                        <div className="font-medium">{document.name}</div>
                        <div className="text-sm text-muted-foreground">{document.document_type} · {document.expires_at ? new Date(document.expires_at).toLocaleDateString() : 'No expiry'}</div>
                        <a href={document.file_url} target="_blank" className="text-sm text-blue-600 underline break-all" rel="noreferrer">
                          {document.file_url}
                        </a>
                      </div>
                      <Button variant="ghost" size="sm" onClick={() => void deleteDocument(document.id)}>
                        Remove
                      </Button>
                    </div>
                  ))
                )}
              </div>
              <div className="grid gap-3 md:grid-cols-2 border-t pt-4">
                <select
                  value={documentForm.document_type}
                  onChange={(e) => setDocumentForm((prev) => ({ ...prev, document_type: e.target.value }))}
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                >
                  <option value="contract">Contract</option>
                  <option value="policy_ack">Policy acknowledgement</option>
                  <option value="id_proof">ID proof</option>
                  <option value="visa">Visa</option>
                </select>
                <Input placeholder="Document name" value={documentForm.name} onChange={(e) => setDocumentForm((prev) => ({ ...prev, name: e.target.value }))} />
                <Input className="md:col-span-2" placeholder="File URL" value={documentForm.file_url} onChange={(e) => setDocumentForm((prev) => ({ ...prev, file_url: e.target.value }))} />
                <Input type="date" value={documentForm.expires_at} onChange={(e) => setDocumentForm((prev) => ({ ...prev, expires_at: e.target.value }))} />
                <Button onClick={() => void addDocument()}>Save Document</Button>
              </div>
            </div>
          </div>

          <div className="rounded-lg border bg-card p-6">
            <h2 className="text-lg font-semibold mb-4">Manager Hierarchy</h2>
            {profile.direct_reports.length === 0 ? (
              <div className="text-sm text-muted-foreground">This employee currently has no direct reports.</div>
            ) : (
              <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
                {profile.direct_reports.map((report) => (
                  <div key={report.id} className="rounded-md border p-3">
                    <div className="font-medium">{report.first_name} {report.last_name}</div>
                    <div className="text-sm text-muted-foreground">{report.email}</div>
                    <div className="text-xs text-muted-foreground uppercase mt-2">{report.role}</div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  )
}
