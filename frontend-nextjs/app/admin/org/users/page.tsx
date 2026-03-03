'use client'

import { useEffect, useState } from 'react'
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
  role: string
  is_active: boolean
  department_id?: string | null
}

interface Department {
  id: string
  name: string
}

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
  const [role, setRole] = useState<'org_admin' | 'hr' | 'dept_manager' | 'employee'>('employee')
  const [authMethod, setAuthMethod] = useState<'password' | 'sso'>('password')
  const [password, setPassword] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (!isAuthenticated || !user) {
      router.push('/admin/login')
      return
    }
    if (!['org_admin', 'hr'].includes(user.role)) {
      router.push('/dashboard')
      return
    }
    fetchUsers()
    fetchDepartments()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, user?.role])

  const fetchUsers = async () => {
    try {
      setIsLoading(true)
      const resp = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/users?limit=100`,
        {
          headers: token ? { Authorization: `Bearer ${token}` } : {},
        }
      )
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to load users')
      }
      const data = await resp.json()
      setUsers(data)
    } catch (e: any) {
      toast.error(e.message || 'Failed to load users')
    } finally {
      setIsLoading(false)
    }
  }

  const fetchDepartments = async () => {
    try {
      const resp = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/departments`,
        {
          headers: token ? { Authorization: `Bearer ${token}` } : {},
        }
      )
      if (!resp.ok) return
      const data = await resp.json()
      setDepartments(data)
    } catch {
      // best-effort; silently ignore
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
      const resp = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/users`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            ...(token ? { Authorization: `Bearer ${token}` } : {}),
          },
          body: JSON.stringify({
            employee_id: employeeId,
            email,
            first_name: firstName,
            last_name: lastName,
            role,
            is_active: true,
            phone: phone || undefined,
            designation: designation || undefined,
            department_id: departmentId || undefined,
            date_of_joining: new Date().toISOString(),
            auth_method: authMethod,
            password: authMethod === 'password' ? password : undefined,
          }),
        }
      )
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
      setAuthMethod('password')
      setPassword('')
      setRole('employee')
      await fetchUsers()
    } catch (e: any) {
      toast.error(e.message || 'Failed to create user')
    } finally {
      setSaving(false)
    }
  }

  if (!isAuthenticated || !user) return null

  return (
    <div className="container mx-auto px-4 py-8 space-y-8">
      <div>
        <h1 className="text-2xl font-bold mb-2">Employees</h1>
        <p className="text-muted-foreground">
          Manage users, roles and access within your organization.
        </p>
      </div>

      {/* Create form */}
      <form
        onSubmit={handleCreate}
        className="border rounded-lg p-4 space-y-4 bg-card"
      >
        <h2 className="font-semibold">Add Employee</h2>
        <div className="grid md:grid-cols-4 gap-4">
          <div>
            <Label htmlFor="emp-id">Employee ID *</Label>
            <Input
              id="emp-id"
              value={employeeId}
              onChange={(e) => setEmployeeId(e.target.value)}
              className="mt-1"
            />
          </div>
          <div>
            <Label htmlFor="emp-email">Email *</Label>
            <Input
              id="emp-email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="mt-1"
            />
          </div>
          <div>
            <Label htmlFor="emp-first">First Name *</Label>
            <Input
              id="emp-first"
              value={firstName}
              onChange={(e) => setFirstName(e.target.value)}
              className="mt-1"
            />
          </div>
          <div>
            <Label htmlFor="emp-last">Last Name *</Label>
            <Input
              id="emp-last"
              value={lastName}
              onChange={(e) => setLastName(e.target.value)}
              className="mt-1"
            />
          </div>
        </div>
        <div className="grid md:grid-cols-3 gap-4">
          <div>
            <Label htmlFor="emp-role">Role</Label>
            <select
              id="emp-role"
              value={role}
              onChange={(e) =>
                setRole(e.target.value as 'org_admin' | 'hr' | 'dept_manager' | 'employee')
              }
              className="mt-1 flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            >
              <option value="employee">Employee</option>
              <option value="dept_manager">Department Manager</option>
              <option value="hr">HR</option>
              <option value="org_admin">Org Admin</option>
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
        </div>
        <div className="grid md:grid-cols-3 gap-4">
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
          <div>
            <Label htmlFor="emp-auth-method">Auth method</Label>
            <select
              id="emp-auth-method"
              value={authMethod}
              onChange={(e) => setAuthMethod(e.target.value as 'password' | 'sso')}
              className="mt-1 flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            >
              <option value="password">Password login</option>
              <option value="sso">SSO only</option>
            </select>
          </div>
          <div>
            <Label htmlFor="emp-password">
              Password {['org_admin', 'hr', 'dept_manager'].includes(role) && authMethod === 'password' ? '*' : '(optional)'}
            </Label>
            <Input
              id="emp-password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="mt-1"
              placeholder="Set initial password"
            />
          </div>
        </div>
        <Button type="submit" disabled={saving}>
          {saving ? 'Saving...' : 'Add Employee'}
        </Button>
      </form>

      {/* List */}
      <div className="border rounded-lg bg-card">
        <div className="border-b px-4 py-2 text-sm font-semibold text-muted-foreground">
          Existing Employees
        </div>
        {isLoading ? (
          <div className="p-4 text-sm text-muted-foreground">Loading...</div>
        ) : users.length === 0 ? (
          <div className="p-4 text-sm text-muted-foreground">
            No employees yet. Add your first employee above.
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left">
                <th className="px-4 py-2">Employee ID</th>
                <th className="px-4 py-2">Name</th>
                <th className="px-4 py-2 hidden md:table-cell">Email</th>
                <th className="px-4 py-2 hidden md:table-cell">Department</th>
                <th className="px-4 py-2">Role</th>
                <th className="px-4 py-2">Status</th>
                <th className="px-4 py-2 text-right">Face</th>
              </tr>
            </thead>
            <tbody>
              {users.map((u) => (
                <tr key={u.id} className="border-b last:border-b-0">
                  <td className="px-4 py-2">{u.employee_id}</td>
                  <td className="px-4 py-2">
                    {u.first_name} {u.last_name}
                  </td>
                  <td className="px-4 py-2 hidden md:table-cell">{u.email}</td>
                  <td className="px-4 py-2 hidden md:table-cell text-muted-foreground">
                    {u.department_id
                      ? departments.find((d) => d.id === u.department_id)?.name || '—'
                      : '—'}
                  </td>
                  <td className="px-4 py-2">{u.role}</td>
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
                  <td className="px-4 py-2 text-right">
                    <button
                      onClick={async () => {
                        try {
                          const base = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
                          const resp = await fetch(
                            `${base}/api/v1/users/${u.id}/enroll-link`,
                            {
                              method: 'POST',
                              headers: token ? { Authorization: `Bearer ${token}` } : {},
                            }
                          )
                          if (!resp.ok) {
                            const err = await resp.json().catch(() => ({}))
                            throw new Error(err.error || 'Failed to create enrollment link')
                          }
                          const data = await resp.json()
                          const url = `${window.location.origin}/enroll/${data.token}`
                          await navigator.clipboard.writeText(url)
                          toast.success('Enrollment link copied to clipboard')
                        } catch (err: any) {
                          toast.error(err.message || 'Failed to copy link')
                        }
                      }}
                      className="text-xs text-primary hover:underline"
                    >
                      Copy enroll link
                    </button>
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

