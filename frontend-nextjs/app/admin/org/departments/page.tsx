'use client'

import { useEffect, useState } from 'react'
import { useAuthStore } from '@/store/useStore'
import { useRouter } from 'next/navigation'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import toast from 'react-hot-toast'
import { DataCard } from '@/components/data/DataCard'

interface Department {
  id: string
  name: string
  code?: string | null
  description?: string | null
  manager_id?: string | null
  manager_name?: string | null
  employee_count?: number
  created_at: string
  updated_at: string
}

interface User {
  id: string
  first_name: string
  last_name: string
  email: string
  role?: string
}

export default function OrgDepartmentsPage() {
  const { isAuthenticated, user, token } = useAuthStore()
  const router = useRouter()
  const [departments, setDepartments] = useState<Department[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [name, setName] = useState('')
  const [code, setCode] = useState('')
  const [description, setDescription] = useState('')
  const [saving, setSaving] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editing, setEditing] = useState<Partial<Department>>({})
  const [managerId, setManagerId] = useState<string>('')
  const [pendingDeleteId, setPendingDeleteId] = useState<string | null>(null)
  const [busyDeleteId, setBusyDeleteId] = useState<string | null>(null)
  const [users, setUsers] = useState<User[]>([])

  useEffect(() => {
    if (!isAuthenticated || !user) {
      router.push('/admin/login')
      return
    }
    if (!['org_admin', 'hr'].includes(user.role)) {
      router.push('/dashboard')
      return
    }
    fetchDepartments()
    fetchUsers()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, user?.role])

  const fetchDepartments = async () => {
    try {
      setIsLoading(true)
      const resp = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/departments`,
        {
          headers: token ? { Authorization: `Bearer ${token}` } : {},
        }
      )
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to load departments')
      }
      const data = await resp.json()
      setDepartments(Array.isArray(data) ? data : [])
    } catch (e: any) {
      toast.error(e.message || 'Failed to load departments')
    } finally {
      setIsLoading(false)
    }
  }

  const fetchUsers = async () => {
    try {
      const params = new URLSearchParams()
      params.set('limit', '500')
      params.set('page', '1')
      const resp = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/users?${params.toString()}`,
        {
          headers: token ? { Authorization: `Bearer ${token}` } : {},
        }
      )
      if (resp.ok) {
        const data = await resp.json()
        if (Array.isArray(data)) {
          setUsers(data)
        } else {
          setUsers(Array.isArray(data.data) ? data.data : [])
        }
      }
    } catch (e: any) {
      console.error('Failed to load users for managers list', e)
    }
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) {
      toast.error('Department name is required')
      return
    }
    try {
      setSaving(true)
      const resp = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/departments`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            ...(token ? { Authorization: `Bearer ${token}` } : {}),
          },
          body: JSON.stringify({
            name,
            code: code || undefined,
            description: description || undefined,
            manager_id: managerId || undefined,
          }),
        }
      )
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to create department')
      }
      toast.success('Department created')
      setName('')
      setCode('')
      setDescription('')
      setManagerId('')
      await fetchDepartments()
    } catch (e: any) {
      toast.error(e.message || 'Failed to create department')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (id: string) => {
    try {
      setBusyDeleteId(id)
      const resp = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/departments/${id}`,
        {
          method: 'DELETE',
          headers: token ? { Authorization: `Bearer ${token}` } : {},
        }
      )
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to delete department')
      }
      toast.success('Department deleted')
      setDepartments((prev) => prev.filter((d) => d.id !== id))
      setPendingDeleteId(null)
    } catch (e: any) {
      toast.error(e.message || 'Failed to delete department')
    } finally {
      setBusyDeleteId(null)
    }
  }

  const beginEdit = (d: Department) => {
    setEditingId(d.id)
    setEditing({
      name: d.name,
      code: d.code || '',
      description: d.description || '',
      manager_id: d.manager_id || '',
    })
  }

  const cancelEdit = () => {
    setEditingId(null)
    setEditing({})
  }

  const saveEdit = async (id: string) => {
    if (!editing.name?.trim()) {
      toast.error('Department name is required')
      return
    }
    try {
      const resp = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/departments/${id}`,
        {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/json',
            ...(token ? { Authorization: `Bearer ${token}` } : {}),
          },
          body: JSON.stringify({
            name: editing.name,
            code: editing.code || null,
            description: editing.description || null,
            manager_id: editing.manager_id || null,
          }),
        }
      )
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({}))
        throw new Error(err.error || 'Failed to update department')
      }
      toast.success('Department updated')
      setEditingId(null)
      setEditing({})
      await fetchDepartments()
    } catch (e: any) {
      toast.error(e.message || 'Failed to update department')
    }
  }

  if (!isAuthenticated || !user) return null

  return (
    <div className="container mx-auto px-4 py-8 space-y-8">
      <div>
        <h1 className="text-2xl font-bold mb-2">Departments</h1>
        <p className="text-muted-foreground">
          Manage organizational units and reporting structures.
        </p>
      </div>

      {/* Summary card */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <DataCard
          title="Total Departments"
          value={Array.isArray(departments) ? departments.length : 0}
          subtitle="Within your organization"
        />
        <DataCard
          title="Departments with Managers"
          value={departments.filter(d => !!d.manager_id).length}
          subtitle="Departments that have assigned leads"
        />
        <DataCard
          title="Total Assigned Employees"
          value={departments.reduce((sum, d) => sum + (d.employee_count || 0), 0)}
          subtitle="Across all departments"
        />
      </div>

      {/* Create form */}
      <form
        onSubmit={handleCreate}
        className="border rounded-lg p-4 space-y-4 bg-card"
      >
        <h2 className="font-semibold">Create Department</h2>
        <div className="grid md:grid-cols-3 gap-4">
          <div>
            <Label htmlFor="dept-name">Name *</Label>
            <Input
              id="dept-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Engineering"
              className="mt-1"
            />
          </div>
          <div>
            <Label htmlFor="dept-code">Code</Label>
            <Input
              id="dept-code"
              value={code}
              onChange={(e) => setCode(e.target.value)}
              placeholder="ENG"
              className="mt-1"
            />
          </div>
          <div>
            <Label htmlFor="dept-description">Description</Label>
            <Input
              id="dept-description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Engineering & Product"
              className="mt-1"
            />
          </div>
          <div>
            <Label htmlFor="dept-manager">Manager</Label>
            <select
              id="dept-manager"
              value={managerId}
              onChange={(e) => setManagerId(e.target.value)}
              className="mt-1 flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background disabled:cursor-not-allowed disabled:opacity-50"
            >
              <option value="">-- Unassigned --</option>
              {users.map((u) => (
                <option key={u.id} value={u.id}>
                  {u.first_name} {u.last_name} ({u.email}{u.role ? ` • ${u.role}` : ''})
                </option>
              ))}
            </select>
          </div>
        </div>
        <Button type="submit" disabled={saving}>
          {saving ? 'Saving...' : 'Create Department'}
        </Button>
      </form>

      {/* List */}
      <div className="border rounded-lg bg-card">
        <div className="border-b px-4 py-2 text-sm font-semibold text-muted-foreground">
          Existing Departments
        </div>
        {isLoading ? (
          <div className="p-4 space-y-3">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="skeleton h-14 w-full" />
            ))}
          </div>
        ) : departments.length === 0 ? (
          <div className="p-4 text-sm text-muted-foreground">
            No departments yet. Create your first department above.
          </div>
        ) : (
          <>
            <div className="md:hidden divide-y">
              {departments.map((d) => (
                <div key={d.id} className="p-4 space-y-3">
                  <div>
                    <div className="font-medium">{d.name}</div>
                    <div className="text-xs text-muted-foreground">Code: {d.code || '—'}</div>
                    <div className="text-xs text-muted-foreground">Description: {d.description || '—'}</div>
                    <div className="text-xs text-muted-foreground mt-1">
                      Manager: {d.manager_name || <span className="text-amber-600">Unassigned</span>}
                    </div>
                    <div className="text-xs font-semibold mt-1">
                      Employees: {d.employee_count || 0}
                    </div>
                  </div>
                  {editingId === d.id ? (
                    <div className="space-y-2">
                      <Input
                        value={editing.name || ''}
                        onChange={(e) => setEditing((prev) => ({ ...prev, name: e.target.value }))}
                        placeholder="Name"
                      />
                      <Input
                        value={editing.code || ''}
                        onChange={(e) => setEditing((prev) => ({ ...prev, code: e.target.value }))}
                        placeholder="Code"
                      />
                      <Input
                        value={editing.description || ''}
                        onChange={(e) => setEditing((prev) => ({ ...prev, description: e.target.value }))}
                        placeholder="Description"
                      />
                      <select
                        value={editing.manager_id || ''}
                        onChange={(e) => setEditing((prev) => ({ ...prev, manager_id: e.target.value }))}
                        className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background disabled:cursor-not-allowed disabled:opacity-50"
                      >
                        <option value="">-- Unassigned --</option>
                        {users.map((u) => (
                          <option key={u.id} value={u.id}>
                            {u.first_name} {u.last_name}{u.role ? ` • ${u.role}` : ''}
                          </option>
                        ))}
                      </select>
                      <div className="flex gap-2">
                        <Button size="sm" onClick={() => void saveEdit(d.id)}>
                          Save
                        </Button>
                        <Button size="sm" variant="outline" onClick={cancelEdit}>
                          Cancel
                        </Button>
                      </div>
                    </div>
                  ) : (
                    <div className="flex flex-wrap gap-2">
                      <Button size="sm" variant="outline" onClick={() => beginEdit(d)}>
                        Edit
                      </Button>
                      {pendingDeleteId === d.id ? (
                        <>
                          <Button
                            size="sm"
                            onClick={() => void handleDelete(d.id)}
                            disabled={busyDeleteId === d.id}
                            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                          >
                            Confirm Delete
                          </Button>
                          <Button size="sm" variant="outline" onClick={() => setPendingDeleteId(null)} disabled={busyDeleteId === d.id}>
                            Cancel
                          </Button>
                        </>
                      ) : (
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => setPendingDeleteId(d.id)}
                          className="text-destructive"
                        >
                          Delete
                        </Button>
                      )}
                    </div>
                  )}
                </div>
              ))}
            </div>

            <div className="hidden md:block overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b text-left">
                    <th className="px-4 py-2">Name</th>
                    <th className="px-4 py-2">Code</th>
                    <th className="px-4 py-2">Description</th>
                    <th className="px-4 py-2">Manager</th>
                    <th className="px-4 py-2 text-right">Employees</th>
                    <th className="px-4 py-2 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {departments.map((d) => (
                    <tr key={d.id} className="border-b last:border-b-0">
                      <td className="px-4 py-2">
                        {editingId === d.id ? (
                          <Input
                            value={editing.name || ''}
                            onChange={(e) => setEditing((prev) => ({ ...prev, name: e.target.value }))}
                          />
                        ) : (
                          d.name
                        )}
                      </td>
                      <td className="px-4 py-2">
                        {editingId === d.id ? (
                          <Input
                            value={editing.code || ''}
                            onChange={(e) => setEditing((prev) => ({ ...prev, code: e.target.value }))}
                          />
                        ) : (
                          d.code || '-'
                        )}
                      </td>
                      <td className="px-4 py-2">
                        {editingId === d.id ? (
                          <Input
                            value={editing.description || ''}
                            onChange={(e) =>
                              setEditing((prev) => ({ ...prev, description: e.target.value }))
                            }
                          />
                        ) : (
                          d.description || '-'
                        )}
                      </td>
                      <td className="px-4 py-2">
                        {editingId === d.id ? (
                          <select
                            value={editing.manager_id || ''}
                            onChange={(e) => setEditing((prev) => ({ ...prev, manager_id: e.target.value }))}
                            className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background"
                          >
                            <option value="">-- Unassigned --</option>
                            {users.map((u) => (
                              <option key={u.id} value={u.id}>
                                {u.first_name} {u.last_name}{u.role ? ` • ${u.role}` : ''}
                              </option>
                            ))}
                          </select>
                        ) : (
                          d.manager_name || <span className="text-amber-600 text-xs">Unassigned</span>
                        )}
                      </td>
                      <td className="px-4 py-2 text-right font-semibold">
                        {d.employee_count || 0}
                      </td>
                      <td className="px-4 py-2 text-right">
                        <div className="flex justify-end gap-2">
                          {editingId === d.id ? (
                            <>
                              <Button size="sm" onClick={() => void saveEdit(d.id)}>
                                Save
                              </Button>
                              <Button size="sm" variant="outline" onClick={cancelEdit}>
                                Cancel
                              </Button>
                            </>
                          ) : (
                            <>
                              <Button size="sm" variant="outline" onClick={() => beginEdit(d)}>
                                Edit
                              </Button>
                              {pendingDeleteId === d.id ? (
                                <>
                                  <Button
                                    size="sm"
                                    onClick={() => void handleDelete(d.id)}
                                    disabled={busyDeleteId === d.id}
                                    className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                                  >
                                    Confirm Delete
                                  </Button>
                                  <Button
                                    size="sm"
                                    variant="outline"
                                    onClick={() => setPendingDeleteId(null)}
                                    disabled={busyDeleteId === d.id}
                                  >
                                    Cancel
                                  </Button>
                                </>
                              ) : (
                                <Button
                                  size="sm"
                                  variant="outline"
                                  onClick={() => setPendingDeleteId(d.id)}
                                  className="text-destructive"
                                >
                                  Delete
                                </Button>
                              )}
                            </>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </>
        )}
      </div>
    </div>
  )
}
