'use client'

import { useEffect, useMemo, useState } from 'react'
import { useRouter } from 'next/navigation'
import toast from 'react-hot-toast'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useAuthStore } from '@/store/useStore'

type RoleRecord = {
  id: string
  name: string
  description?: string | null
  is_active: boolean
  permissions: string[]
  assigned_users: number
}

type Assignment = {
  user_id: string
  first_name: string
  last_name: string
  email: string
  base_role: string
  custom_role_id: string
  custom_role_name: string
}

type UserRow = {
  id: string
  first_name: string
  last_name: string
  email: string
  role: string
}

const emptyRole = {
  id: '',
  name: '',
  description: '',
  is_active: true,
  permissions: [] as string[],
}

export default function AccessPage() {
  const { isAuthenticated, user, token } = useAuthStore()
  const router = useRouter()
  const base = useMemo(() => process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080', [])
  const canManage = user?.role === 'org_admin' || user?.permissions?.includes('roles.manage')

  const [roles, setRoles] = useState<RoleRecord[]>([])
  const [assignments, setAssignments] = useState<Assignment[]>([])
  const [permissions, setPermissions] = useState<string[]>([])
  const [users, setUsers] = useState<UserRow[]>([])
  const [draft, setDraft] = useState(emptyRole)
  const [assignmentUserID, setAssignmentUserID] = useState('')
  const [assignmentRoleID, setAssignmentRoleID] = useState('')
  const [isLoading, setIsLoading] = useState(true)
  const [isSaving, setIsSaving] = useState(false)
  const [isAssigning, setIsAssigning] = useState(false)

  useEffect(() => {
    if (!isAuthenticated || !user) {
      router.push('/admin/login')
      return
    }
    if (!canManage) {
      router.push('/admin/org')
      return
    }
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, user?.id, canManage])

  const authHeaders = () => ({
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  })

  const load = async () => {
    try {
      setIsLoading(true)
      const [rolesResp, usersResp] = await Promise.all([
        fetch(`${base}/api/v1/org/access/roles`, { headers: authHeaders() }),
        fetch(`${base}/api/v1/users?limit=200`, { headers: authHeaders() }),
      ])
      if (!rolesResp.ok || !usersResp.ok) {
        throw new Error('Failed to load access data')
      }
      const rolesPayload = await rolesResp.json()
      const usersPayload = await usersResp.json()
      setRoles(Array.isArray(rolesPayload.roles) ? rolesPayload.roles : [])
      setAssignments(Array.isArray(rolesPayload.assignments) ? rolesPayload.assignments : [])
      setPermissions(Array.isArray(rolesPayload.permissions) ? rolesPayload.permissions : [])
      const rows = Array.isArray(usersPayload.data) ? usersPayload.data : []
      setUsers(rows.filter((entry: any) => ['org_admin', 'hr', 'dept_manager'].includes(entry.role)))
    } catch (error: any) {
      toast.error(error.message || 'Failed to load access data')
    } finally {
      setIsLoading(false)
    }
  }

  const saveRole = async () => {
    try {
      setIsSaving(true)
      const endpoint = draft.id ? `${base}/api/v1/org/access/roles/${draft.id}` : `${base}/api/v1/org/access/roles`
      const method = draft.id ? 'PUT' : 'POST'
      const resp = await fetch(endpoint, {
        method,
        headers: authHeaders(),
        body: JSON.stringify({
          name: draft.name,
          description: draft.description || null,
          is_active: draft.is_active,
          permissions: draft.permissions,
        }),
      })
      if (!resp.ok) {
        const error = await resp.json().catch(() => ({}))
        throw new Error(error.error || 'Failed to save role')
      }
      toast.success('Custom role saved')
      setDraft(emptyRole)
      await load()
    } catch (error: any) {
      toast.error(error.message || 'Failed to save role')
    } finally {
      setIsSaving(false)
    }
  }

  const deleteRole = async (id: string) => {
    try {
      const resp = await fetch(`${base}/api/v1/org/access/roles/${id}`, {
        method: 'DELETE',
        headers: authHeaders(),
      })
      if (!resp.ok) {
        const error = await resp.json().catch(() => ({}))
        throw new Error(error.error || 'Failed to delete role')
      }
      toast.success('Custom role deleted')
      await load()
    } catch (error: any) {
      toast.error(error.message || 'Failed to delete role')
    }
  }

  const saveAssignment = async () => {
    if (!assignmentUserID) {
      toast.error('Select a user')
      return
    }
    try {
      setIsAssigning(true)
      const resp = await fetch(`${base}/api/v1/org/access/assignments`, {
        method: 'POST',
        headers: authHeaders(),
        body: JSON.stringify({ user_id: assignmentUserID, custom_role_id: assignmentRoleID || null }),
      })
      if (!resp.ok) {
        const error = await resp.json().catch(() => ({}))
        throw new Error(error.error || 'Failed to update assignment')
      }
      toast.success('Assignment updated')
      setAssignmentUserID('')
      setAssignmentRoleID('')
      await load()
    } catch (error: any) {
      toast.error(error.message || 'Failed to update assignment')
    } finally {
      setIsAssigning(false)
    }
  }

  const togglePermission = (permission: string) => {
    setDraft((current) => ({
      ...current,
      permissions: current.permissions.includes(permission)
        ? current.permissions.filter((item) => item !== permission)
        : [...current.permissions, permission],
    }))
  }

  if (!isAuthenticated || !user || !canManage) return null

  return (
    <div className="container mx-auto space-y-6 p-6">
      <div>
        <h1 className="text-3xl font-bold">Roles & Permissions</h1>
        <p className="text-muted-foreground">Create custom admin roles with granular permissions and assign them to active org admin, HR, or department manager accounts.</p>
      </div>

      {isLoading ? (
        <div className="rounded-lg border p-6 text-sm text-muted-foreground">Loading roles...</div>
      ) : (
        <>
          <section className="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
            <div className="space-y-4 rounded-lg border bg-card p-6">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <h2 className="text-lg font-semibold">Custom Roles</h2>
                  <p className="text-sm text-muted-foreground">Assignments override the user&apos;s default admin permission bundle with the selected permission set.</p>
                </div>
              </div>
              <div className="space-y-3">
                {roles.length === 0 ? (
                  <div className="rounded-md border border-dashed p-4 text-sm text-muted-foreground">No custom roles created yet.</div>
                ) : (
                  roles.map((role) => (
                    <div key={role.id} className="rounded-md border p-4">
                      <div className="flex items-start justify-between gap-4">
                        <div>
                          <div className="flex items-center gap-2">
                            <h3 className="font-medium">{role.name}</h3>
                            {!role.is_active && <span className="rounded-full border px-2 py-0.5 text-xs">Inactive</span>}
                          </div>
                          <div className="text-sm text-muted-foreground">{role.description || 'No description provided.'}</div>
                          <div className="mt-2 flex flex-wrap gap-2 text-xs">
                            {role.permissions.map((permission) => (
                              <span key={permission} className="rounded-full bg-muted px-2 py-1">{permission}</span>
                            ))}
                          </div>
                          <div className="mt-2 text-xs text-muted-foreground">Assigned users: {role.assigned_users}</div>
                        </div>
                        <div className="flex gap-2">
                          <Button variant="outline" onClick={() => setDraft({ id: role.id, name: role.name, description: role.description || '', is_active: role.is_active, permissions: role.permissions })}>Edit</Button>
                          <Button variant="ghost" onClick={() => void deleteRole(role.id)}>Delete</Button>
                        </div>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>

            <div className="space-y-4 rounded-lg border bg-card p-6">
              <h2 className="text-lg font-semibold">Role Editor</h2>
              <Input placeholder="Role name" value={draft.name} onChange={(e) => setDraft((current) => ({ ...current, name: e.target.value }))} />
              <textarea className="min-h-[96px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm" placeholder="Role description" value={draft.description} onChange={(e) => setDraft((current) => ({ ...current, description: e.target.value }))} />
              <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={draft.is_active} onChange={(e) => setDraft((current) => ({ ...current, is_active: e.target.checked }))} /> Role is active</label>
              <div className="space-y-2">
                <div className="text-sm font-medium">Permissions</div>
                <div className="flex flex-wrap gap-2">
                  {permissions.map((permission) => (
                    <button
                      key={permission}
                      type="button"
                      onClick={() => togglePermission(permission)}
                      className={`rounded-full border px-3 py-1 text-sm ${draft.permissions.includes(permission) ? 'border-primary bg-primary text-primary-foreground' : 'border-border bg-background text-foreground'}`}
                    >
                      {permission}
                    </button>
                  ))}
                </div>
              </div>
              <div className="flex gap-2">
                <Button onClick={() => void saveRole()} disabled={isSaving}>{isSaving ? 'Saving...' : 'Save Role'}</Button>
                <Button variant="outline" onClick={() => setDraft(emptyRole)}>Clear</Button>
              </div>
            </div>
          </section>

          <section className="grid gap-6 lg:grid-cols-[0.9fr_1.1fr]">
            <div className="space-y-4 rounded-lg border bg-card p-6">
              <h2 className="text-lg font-semibold">Assign Custom Role</h2>
              <select className="h-10 rounded-md border border-input bg-background px-3 text-sm" value={assignmentUserID} onChange={(e) => setAssignmentUserID(e.target.value)}>
                <option value="">Select user</option>
                {users.map((entry) => (
                  <option key={entry.id} value={entry.id}>{entry.first_name} {entry.last_name} • {entry.role}</option>
                ))}
              </select>
              <select className="h-10 rounded-md border border-input bg-background px-3 text-sm" value={assignmentRoleID} onChange={(e) => setAssignmentRoleID(e.target.value)}>
                <option value="">Remove custom role</option>
                {roles.filter((role) => role.is_active).map((role) => (
                  <option key={role.id} value={role.id}>{role.name}</option>
                ))}
              </select>
              <Button onClick={() => void saveAssignment()} disabled={isAssigning}>{isAssigning ? 'Saving...' : 'Save Assignment'}</Button>
            </div>

            <div className="space-y-4 rounded-lg border bg-card p-6">
              <h2 className="text-lg font-semibold">Current Assignments</h2>
              {assignments.length === 0 ? (
                <div className="rounded-md border border-dashed p-4 text-sm text-muted-foreground">No custom role assignments yet.</div>
              ) : (
                <div className="space-y-3">
                  {assignments.map((assignment) => (
                    <div key={assignment.user_id} className="rounded-md border p-4">
                      <div className="flex items-start justify-between gap-4">
                        <div>
                          <div className="font-medium">{assignment.first_name} {assignment.last_name}</div>
                          <div className="text-sm text-muted-foreground">{assignment.email} • {assignment.base_role}</div>
                          <div className="mt-1 text-sm">{assignment.custom_role_name}</div>
                        </div>
                        <Button variant="ghost" onClick={() => {
                          setAssignmentUserID(assignment.user_id)
                          setAssignmentRoleID('')
                        }}>Prepare removal</Button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </section>
        </>
      )}
    </div>
  )
}
