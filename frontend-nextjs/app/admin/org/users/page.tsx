'use client'

import { useEffect, useMemo, useState } from 'react'
import { useAuthStore } from '@/store/useStore'
import { useRouter } from 'next/navigation'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import toast from 'react-hot-toast'

interface UserRow {
  id: string
  employee_id: string
  email: string
  first_name: string
  last_name: string
  role: 'org_admin' | 'hr' | 'dept_manager' | 'employee'
  is_active: boolean
  phone?: string | null
  designation?: string | null
  department_id?: string | null
  date_of_joining?: string
}

interface Department {
  id: string
  name: string
}

type Role = 'org_admin' | 'hr' | 'dept_manager' | 'employee'

const ROLE_OPTIONS: Role[] = ['employee', 'dept_manager', 'hr', 'org_admin']

export default function OrgUsersPage() {
  const { isAuthenticated, user, token } = useAuthStore()
  const router = useRouter()
  const [users, setUsers] = useState<UserRow[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [departments, setDepartments] = useState<Department[]>([])

  const [employeeId, setEmployeeId] = useState('')
  const [email, setEmail] = useState('')
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [phone, setPhone] = useState('')
  const [designation, setDesignation] = useState('')
  const [departmentId, setDepartmentId] = useState<string>('')
  const [role, setRole] = useState<Role>('employee')
  const [authMethod, setAuthMethod] = useState<'password' | 'sso'>('sso')
  const [password, setPassword] = useState('')
  const [saving, setSaving] = useState(false)

  const [query, setQuery] = useState('')
  const [roleFilter, setRoleFilter] = useState<'all' | Role>('all')
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'inactive'>('all')
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editingUser, setEditingUser] = useState<Partial<UserRow>>({})
  const [selectedRows, setSelectedRows] = useState<Record<string, boolean>>({})
  const [bulkRunning, setBulkRunning] = useState(false)
  const [importing, setImporting] = useState(false)

  const base = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

  useEffect(() => {
    if (!isAuthenticated || !user) {
      router.push('/admin/login')
      return
    }
    if (!['org_admin', 'hr'].includes(user.role)) {
      router.push('/dashboard')
      return
    }
    void fetchUsers()
    void fetchDepartments()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, user?.role])

  useEffect(() => {
    if (role === 'employee') {
      setAuthMethod('sso')
      setPassword('')
    }
  }, [role])

  const fetchUsers = async () => {
    try {
      setIsLoading(true)
      const resp = await fetch(`${base}/api/v1/users?limit=300`, {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to load users')
      }
      const data = await resp.json()
      setUsers(Array.isArray(data) ? data : [])
      setSelectedRows({})
    } catch (e: any) {
      toast.error(e.message || 'Failed to load users')
    } finally {
      setIsLoading(false)
    }
  }

  const fetchDepartments = async () => {
    try {
      const resp = await fetch(`${base}/api/v1/departments`, {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      })
      if (!resp.ok) return
      const data = await resp.json()
      setDepartments(Array.isArray(data) ? data : [])
    } catch {
      // best effort
    }
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!employeeId.trim() || !email.trim() || !firstName.trim() || !lastName.trim()) {
      toast.error('Employee ID, email, first name and last name are required')
      return
    }
    if (['org_admin', 'hr', 'dept_manager'].includes(role) && authMethod === 'password' && !password.trim()) {
      toast.error('Password is required for admin/HR/manager users')
      return
    }

    try {
      setSaving(true)
      const resp = await fetch(`${base}/api/v1/users`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({
          employee_id: employeeId.trim(),
          email: email.trim(),
          first_name: firstName.trim(),
          last_name: lastName.trim(),
          role,
          is_active: true,
          phone: phone.trim() || undefined,
          designation: designation.trim() || undefined,
          department_id: departmentId || undefined,
          date_of_joining: new Date().toISOString(),
          auth_method: role === 'employee' ? 'sso' : authMethod,
          password:
            role !== 'employee' && authMethod === 'password' && password.trim()
              ? password
              : undefined,
        }),
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to create user')
      }
      toast.success('User created')
      setEmployeeId('')
      setEmail('')
      setFirstName('')
      setLastName('')
      setPhone('')
      setDesignation('')
      setDepartmentId('')
      setAuthMethod('sso')
      setPassword('')
      setRole('employee')
      await fetchUsers()
    } catch (e: any) {
      toast.error(e.message || 'Failed to create user')
    } finally {
      setSaving(false)
    }
  }

  const filteredUsers = useMemo(() => {
    const q = query.trim().toLowerCase()
    return users.filter((u) => {
      if (roleFilter !== 'all' && u.role !== roleFilter) return false
      if (statusFilter === 'active' && !u.is_active) return false
      if (statusFilter === 'inactive' && u.is_active) return false
      if (!q) return true
      const name = `${u.first_name} ${u.last_name}`.toLowerCase()
      return (
        name.includes(q) ||
        u.employee_id.toLowerCase().includes(q) ||
        u.email.toLowerCase().includes(q)
      )
    })
  }, [users, query, roleFilter, statusFilter])

  const beginEdit = (u: UserRow) => {
    setEditingId(u.id)
    setEditingUser({
      email: u.email,
      first_name: u.first_name,
      last_name: u.last_name,
      phone: u.phone || '',
      designation: u.designation || '',
      role: u.role,
      is_active: u.is_active,
      department_id: u.department_id || '',
    })
  }

  const cancelEdit = () => {
    setEditingId(null)
    setEditingUser({})
  }

  const saveEdit = async (u: UserRow) => {
    if (!editingUser.email || !editingUser.first_name || !editingUser.last_name || !editingUser.role) {
      toast.error('Email, first name, last name and role are required')
      return
    }
    try {
      const resp = await fetch(`${base}/api/v1/users/${u.id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({
          employee_id: u.employee_id,
          email: editingUser.email,
          first_name: editingUser.first_name,
          last_name: editingUser.last_name,
          phone: editingUser.phone || null,
          designation: editingUser.designation || null,
          role: editingUser.role,
          is_active: Boolean(editingUser.is_active),
          department_id: editingUser.department_id || null,
          date_of_joining: u.date_of_joining,
        }),
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to update user')
      }
      toast.success('User updated')
      setEditingId(null)
      setEditingUser({})
      await fetchUsers()
    } catch (e: any) {
      toast.error(e.message || 'Failed to update user')
    }
  }

  const toggleActive = async (u: UserRow) => {
    try {
      const resp = await fetch(`${base}/api/v1/users/${u.id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({
          employee_id: u.employee_id,
          email: u.email,
          first_name: u.first_name,
          last_name: u.last_name,
          phone: u.phone || null,
          designation: u.designation || null,
          role: u.role,
          is_active: !u.is_active,
          department_id: u.department_id || null,
          date_of_joining: u.date_of_joining,
        }),
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to update status')
      }
      toast.success(u.is_active ? 'User deactivated' : 'User reactivated')
      await fetchUsers()
    } catch (e: any) {
      toast.error(e.message || 'Failed to update status')
    }
  }

  const copyEnrollLink = async (userId: string) => {
    try {
      const resp = await fetch(`${base}/api/v1/users/${userId}/enroll-link`, {
        method: 'POST',
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to create enrollment link')
      }
      const data = await resp.json()
      const url = `${window.location.origin}/enroll/${data.token}`
      await navigator.clipboard.writeText(url)
      toast.success('Enrollment link copied')
    } catch (e: any) {
      toast.error(e.message || 'Failed to copy link')
    }
  }

  const selectedUserIds = useMemo(
    () => Object.keys(selectedRows).filter((id) => selectedRows[id]),
    [selectedRows]
  )

  const toggleSelectAllVisible = (checked: boolean) => {
    if (!checked) {
      setSelectedRows({})
      return
    }
    const next: Record<string, boolean> = {}
    filteredUsers.forEach((u) => {
      next[u.id] = true
    })
    setSelectedRows(next)
  }

  const runBulkAction = async (action: 'activate' | 'deactivate' | 'delete') => {
    if (!selectedUserIds.length) {
      toast.error('Select at least one employee')
      return
    }
    try {
      setBulkRunning(true)
      const resp = await fetch(`${base}/api/v1/users/bulk/action`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({
          user_ids: selectedUserIds,
          action,
        }),
      })
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Bulk action failed')
      }
      toast.success(`Bulk ${action} completed`)
      await fetchUsers()
    } catch (e: any) {
      toast.error(e.message || 'Bulk action failed')
    } finally {
      setBulkRunning(false)
    }
  }

  const parseCsv = (text: string) => {
    const lines = text
      .split(/\r?\n/)
      .map((l) => l.trim())
      .filter(Boolean)
    if (!lines.length) return []
    const headers = lines[0].split(',').map((h) => h.trim().toLowerCase())
    const rows: any[] = []

    for (let i = 1; i < lines.length; i++) {
      const values = lines[i].split(',').map((v) => v.trim())
      const row: any = {}
      headers.forEach((h, idx) => {
        row[h] = values[idx] ?? ''
      })
      rows.push({
        employee_id: row.employee_id || row.emp_id || '',
        email: row.email || '',
        first_name: row.first_name || row.firstname || '',
        last_name: row.last_name || row.lastname || '',
        phone: row.phone || undefined,
        designation: row.designation || undefined,
        department_id: row.department_id || undefined,
        role: row.role || 'employee',
        auth_method: row.auth_method || (row.role === 'employee' ? 'sso' : 'password'),
        password: row.password || undefined,
        date_of_joining: row.date_of_joining || undefined,
      })
    }
    return rows
  }

  const importCsvFile = async (file: File) => {
    try {
      setImporting(true)
      const text = await file.text()
      const rows = parseCsv(text)
      if (!rows.length) {
        toast.error('CSV has no data rows')
        return
      }
      const resp = await fetch(`${base}/api/v1/users/bulk/import`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({ rows }),
      })
      const result = await resp.json().catch(() => ({}))
      if (!resp.ok) {
        throw new Error(result.error || 'Import failed')
      }
      toast.success(`Imported ${result.success_count || 0} users, failed ${result.failed_count || 0}`)
      await fetchUsers()
    } catch (e: any) {
      toast.error(e.message || 'Import failed')
    } finally {
      setImporting(false)
    }
  }

  if (!isAuthenticated || !user) return null

  return (
    <div className="container mx-auto px-4 py-8 space-y-8">
      <div>
        <h1 className="text-2xl font-bold mb-2">Employees</h1>
        <p className="text-muted-foreground">Manage users, roles, access, and enrollment links.</p>
      </div>

      <form onSubmit={handleCreate} className="border rounded-lg p-4 space-y-4 bg-card">
        <h2 className="font-semibold">Add Employee</h2>
        <div className="grid md:grid-cols-4 gap-4">
          <div>
            <Label htmlFor="emp-id">Employee ID *</Label>
            <Input id="emp-id" value={employeeId} onChange={(e) => setEmployeeId(e.target.value)} className="mt-1" />
          </div>
          <div>
            <Label htmlFor="emp-email">Email *</Label>
            <Input id="emp-email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} className="mt-1" />
          </div>
          <div>
            <Label htmlFor="emp-first">First Name *</Label>
            <Input id="emp-first" value={firstName} onChange={(e) => setFirstName(e.target.value)} className="mt-1" />
          </div>
          <div>
            <Label htmlFor="emp-last">Last Name *</Label>
            <Input id="emp-last" value={lastName} onChange={(e) => setLastName(e.target.value)} className="mt-1" />
          </div>
        </div>
        <div className="grid md:grid-cols-4 gap-4">
          <div>
            <Label htmlFor="emp-role">Role</Label>
            <select
              id="emp-role"
              value={role}
              onChange={(e) => setRole(e.target.value as Role)}
              className="mt-1 flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            >
              {ROLE_OPTIONS.map((r) => (
                <option key={r} value={r}>
                  {r}
                </option>
              ))}
            </select>
          </div>
          <div>
            <Label htmlFor="emp-dept">Department</Label>
            <select
              id="emp-dept"
              value={departmentId}
              onChange={(e) => setDepartmentId(e.target.value)}
              className="mt-1 flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            >
              <option value="">Unassigned</option>
              {departments.map((d) => (
                <option key={d.id} value={d.id}>
                  {d.name}
                </option>
              ))}
            </select>
          </div>
          <div>
            <Label htmlFor="emp-designation">Designation</Label>
            <Input
              id="emp-designation"
              value={designation}
              onChange={(e) => setDesignation(e.target.value)}
              className="mt-1"
              placeholder="e.g. Senior Engineer"
            />
          </div>
          <div>
            <Label htmlFor="emp-phone">Phone</Label>
            <Input
              id="emp-phone"
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              className="mt-1"
              placeholder="+1 555 555 5555"
            />
          </div>
        </div>
        <div className="grid md:grid-cols-3 gap-4">
          <div>
            <Label htmlFor="emp-auth-method">Auth method</Label>
            <select
              id="emp-auth-method"
              value={role === 'employee' ? 'sso' : authMethod}
              onChange={(e) => setAuthMethod(e.target.value as 'password' | 'sso')}
              disabled={role === 'employee'}
              className="mt-1 flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm disabled:opacity-60"
            >
              <option value="sso">SSO only</option>
              <option value="password">Password login</option>
            </select>
          </div>
          {role !== 'employee' && authMethod === 'password' ? (
            <div>
              <Label htmlFor="emp-password">Initial Password *</Label>
              <Input
                id="emp-password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="mt-1"
                placeholder="Set initial password"
              />
            </div>
          ) : (
            <div className="text-xs text-muted-foreground flex items-end pb-2">
              Employees use SSO/biometric enrollment and do not require local passwords.
            </div>
          )}
        </div>
        <Button type="submit" disabled={saving}>
          {saving ? 'Saving...' : 'Add Employee'}
        </Button>
      </form>

      <div className="border rounded-lg bg-card">
        <div className="border-b px-4 py-3 flex flex-col md:flex-row md:items-center gap-3 md:justify-between">
          <div className="text-sm font-semibold text-muted-foreground">Existing Employees</div>
          <div className="flex flex-col md:flex-row gap-2 md:items-center">
            <Input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search name, email, employee ID"
              className="md:w-72"
            />
            <select
              value={roleFilter}
              onChange={(e) => setRoleFilter(e.target.value as 'all' | Role)}
              className="h-10 rounded-md border border-input bg-background px-3 text-sm"
            >
              <option value="all">All roles</option>
              {ROLE_OPTIONS.map((r) => (
                <option key={r} value={r}>
                  {r}
                </option>
              ))}
            </select>
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value as 'all' | 'active' | 'inactive')}
              className="h-10 rounded-md border border-input bg-background px-3 text-sm"
            >
              <option value="all">All status</option>
              <option value="active">Active</option>
              <option value="inactive">Inactive</option>
            </select>
            <label className="h-10 inline-flex items-center rounded-md border border-input bg-background px-3 text-sm cursor-pointer">
              {importing ? 'Importing...' : 'Import CSV'}
              <input
                type="file"
                accept=".csv,text/csv"
                className="hidden"
                disabled={importing}
                onChange={(e) => {
                  const f = e.target.files?.[0]
                  if (f) void importCsvFile(f)
                  e.currentTarget.value = ''
                }}
              />
            </label>
          </div>
        </div>

        <div className="px-4 py-3 border-b flex flex-wrap gap-2 items-center justify-between">
          <div className="text-xs text-muted-foreground">
            Selected: {selectedUserIds.length}
          </div>
          <div className="flex gap-2">
            <Button size="sm" variant="outline" disabled={bulkRunning || !selectedUserIds.length} onClick={() => void runBulkAction('activate')}>
              Activate selected
            </Button>
            <Button size="sm" variant="outline" disabled={bulkRunning || !selectedUserIds.length} onClick={() => void runBulkAction('deactivate')}>
              Deactivate selected
            </Button>
            <Button size="sm" variant="outline" disabled={bulkRunning || !selectedUserIds.length} onClick={() => void runBulkAction('delete')}>
              Delete selected
            </Button>
          </div>
        </div>

        {isLoading ? (
          <div className="p-4 space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="skeleton h-16 w-full" />
            ))}
          </div>
        ) : filteredUsers.length === 0 ? (
          <div className="p-4 text-sm text-muted-foreground">No employees found.</div>
        ) : (
          <>
            <div className="md:hidden divide-y">
              {filteredUsers.map((u) => {
                const isEditing = editingId === u.id
                return (
                  <div key={u.id} className="p-4 space-y-3">
                    <div className="flex items-start justify-between gap-3">
                      <label className="inline-flex items-center gap-2">
                        <input
                          type="checkbox"
                          checked={Boolean(selectedRows[u.id])}
                          onChange={(e) =>
                            setSelectedRows((prev) => ({ ...prev, [u.id]: e.target.checked }))
                          }
                        />
                        <span className="font-medium">{u.employee_id}</span>
                      </label>
                      <span
                        className={
                          u.is_active
                            ? 'text-xs text-green-600 dark:text-green-300'
                            : 'text-xs text-muted-foreground'
                        }
                      >
                        {u.is_active ? 'Active' : 'Inactive'}
                      </span>
                    </div>
                    <div className="text-sm text-muted-foreground">
                      {u.first_name} {u.last_name} · {u.email}
                    </div>
                    <div className="text-xs text-muted-foreground">
                      Department:{' '}
                      {u.department_id ? departments.find((d) => d.id === u.department_id)?.name || '—' : '—'}
                    </div>
                    <div className="text-xs text-muted-foreground">Role: {u.role}</div>
                    {isEditing && (
                      <div className="space-y-2">
                        <Input
                          value={editingUser.first_name || ''}
                          onChange={(e) =>
                            setEditingUser((prev) => ({ ...prev, first_name: e.target.value }))
                          }
                          placeholder="First name"
                        />
                        <Input
                          value={editingUser.last_name || ''}
                          onChange={(e) =>
                            setEditingUser((prev) => ({ ...prev, last_name: e.target.value }))
                          }
                          placeholder="Last name"
                        />
                        <Input
                          type="email"
                          value={editingUser.email || ''}
                          onChange={(e) =>
                            setEditingUser((prev) => ({ ...prev, email: e.target.value }))
                          }
                          placeholder="Email"
                        />
                        <select
                          value={(editingUser.department_id as string) || ''}
                          onChange={(e) =>
                            setEditingUser((prev) => ({ ...prev, department_id: e.target.value }))
                          }
                          className="h-10 rounded-md border border-input bg-background px-3 text-sm w-full"
                        >
                          <option value="">Unassigned</option>
                          {departments.map((d) => (
                            <option key={d.id} value={d.id}>
                              {d.name}
                            </option>
                          ))}
                        </select>
                        <select
                          value={(editingUser.role as Role) || u.role}
                          onChange={(e) =>
                            setEditingUser((prev) => ({ ...prev, role: e.target.value as Role }))
                          }
                          className="h-10 rounded-md border border-input bg-background px-3 text-sm w-full"
                        >
                          {ROLE_OPTIONS.map((r) => (
                            <option key={r} value={r}>
                              {r}
                            </option>
                          ))}
                        </select>
                      </div>
                    )}
                    <div className="flex flex-wrap gap-2">
                      {isEditing ? (
                        <>
                          <Button size="sm" onClick={() => void saveEdit(u)}>
                            Save
                          </Button>
                          <Button size="sm" variant="outline" onClick={cancelEdit}>
                            Cancel
                          </Button>
                        </>
                      ) : (
                        <>
                          <Button size="sm" variant="outline" onClick={() => beginEdit(u)}>
                            Edit
                          </Button>
                          <Button size="sm" variant="outline" onClick={() => void toggleActive(u)}>
                            {u.is_active ? 'Deactivate' : 'Activate'}
                          </Button>
                          <Button size="sm" variant="ghost" onClick={() => void copyEnrollLink(u.id)}>
                            Copy enroll link
                          </Button>
                        </>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>

            <div className="hidden md:block overflow-x-auto">
              <table className="w-full text-sm min-w-[1024px]">
                <thead>
                  <tr className="border-b text-left">
                    <th className="px-4 py-2">
                      <input
                        type="checkbox"
                        checked={
                          filteredUsers.length > 0 &&
                          filteredUsers.every((u) => Boolean(selectedRows[u.id]))
                        }
                        onChange={(e) => toggleSelectAllVisible(e.target.checked)}
                      />
                    </th>
                    <th className="px-4 py-2">Employee ID</th>
                    <th className="px-4 py-2">Name</th>
                    <th className="px-4 py-2">Email</th>
                    <th className="px-4 py-2">Department</th>
                    <th className="px-4 py-2">Role</th>
                    <th className="px-4 py-2">Status</th>
                    <th className="px-4 py-2 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredUsers.map((u) => {
                    const isEditing = editingId === u.id
                    return (
                      <tr key={u.id} className="border-b last:border-b-0 align-top">
                        <td className="px-4 py-2">
                          <input
                            type="checkbox"
                            checked={Boolean(selectedRows[u.id])}
                            onChange={(e) =>
                              setSelectedRows((prev) => ({ ...prev, [u.id]: e.target.checked }))
                            }
                          />
                        </td>
                        <td className="px-4 py-2">{u.employee_id}</td>
                        <td className="px-4 py-2">
                          {isEditing ? (
                            <div className="grid grid-cols-1 gap-2">
                              <Input
                                value={editingUser.first_name || ''}
                                onChange={(e) =>
                                  setEditingUser((prev) => ({ ...prev, first_name: e.target.value }))
                                }
                                placeholder="First name"
                              />
                              <Input
                                value={editingUser.last_name || ''}
                                onChange={(e) =>
                                  setEditingUser((prev) => ({ ...prev, last_name: e.target.value }))
                                }
                                placeholder="Last name"
                              />
                            </div>
                          ) : (
                            `${u.first_name} ${u.last_name}`
                          )}
                        </td>
                        <td className="px-4 py-2">
                          {isEditing ? (
                            <Input
                              type="email"
                              value={editingUser.email || ''}
                              onChange={(e) =>
                                setEditingUser((prev) => ({ ...prev, email: e.target.value }))
                              }
                            />
                          ) : (
                            u.email
                          )}
                        </td>
                        <td className="px-4 py-2">
                          {isEditing ? (
                            <select
                              value={(editingUser.department_id as string) || ''}
                              onChange={(e) =>
                                setEditingUser((prev) => ({ ...prev, department_id: e.target.value }))
                              }
                              className="h-10 rounded-md border border-input bg-background px-3 text-sm w-full"
                            >
                              <option value="">Unassigned</option>
                              {departments.map((d) => (
                                <option key={d.id} value={d.id}>
                                  {d.name}
                                </option>
                              ))}
                            </select>
                          ) : u.department_id ? (
                            departments.find((d) => d.id === u.department_id)?.name || '—'
                          ) : (
                            '—'
                          )}
                        </td>
                        <td className="px-4 py-2">
                          {isEditing ? (
                            <select
                              value={(editingUser.role as Role) || u.role}
                              onChange={(e) =>
                                setEditingUser((prev) => ({ ...prev, role: e.target.value as Role }))
                              }
                              className="h-10 rounded-md border border-input bg-background px-3 text-sm w-full"
                            >
                              {ROLE_OPTIONS.map((r) => (
                                <option key={r} value={r}>
                                  {r}
                                </option>
                              ))}
                            </select>
                          ) : (
                            u.role
                          )}
                        </td>
                        <td className="px-4 py-2">
                          <span
                            className={
                              u.is_active
                                ? 'text-xs text-green-600 dark:text-green-300'
                                : 'text-xs text-muted-foreground'
                            }
                          >
                            {u.is_active ? 'Active' : 'Inactive'}
                          </span>
                        </td>
                        <td className="px-4 py-2">
                          <div className="flex justify-end flex-wrap gap-2">
                            {isEditing ? (
                              <>
                                <Button size="sm" onClick={() => void saveEdit(u)}>
                                  Save
                                </Button>
                                <Button size="sm" variant="outline" onClick={cancelEdit}>
                                  Cancel
                                </Button>
                              </>
                            ) : (
                              <>
                                <Button size="sm" variant="outline" onClick={() => beginEdit(u)}>
                                  Edit
                                </Button>
                                <Button
                                  size="sm"
                                  variant="outline"
                                  onClick={() => void toggleActive(u)}
                                >
                                  {u.is_active ? 'Deactivate' : 'Activate'}
                                </Button>
                                <Button
                                  size="sm"
                                  variant="ghost"
                                  onClick={() => void copyEnrollLink(u.id)}
                                >
                                  Copy enroll link
                                </Button>
                              </>
                            )}
                          </div>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          </>
        )}
      </div>
    </div>
  )
}
