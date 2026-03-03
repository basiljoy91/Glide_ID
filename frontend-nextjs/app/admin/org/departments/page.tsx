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
  created_at: string
  updated_at: string
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
      await fetchDepartments()
    } catch (e: any) {
      toast.error(e.message || 'Failed to create department')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this department?')) return
    try {
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
    } catch (e: any) {
      toast.error(e.message || 'Failed to delete department')
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
          <div className="p-4 text-sm text-muted-foreground">Loading...</div>
        ) : departments.length === 0 ? (
          <div className="p-4 text-sm text-muted-foreground">
            No departments yet. Create your first department above.
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left">
                <th className="px-4 py-2">Name</th>
                <th className="px-4 py-2">Code</th>
                <th className="px-4 py-2 hidden md:table-cell">Description</th>
                <th className="px-4 py-2 text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {departments.map((d) => (
                <tr key={d.id} className="border-b last:border-b-0">
                  <td className="px-4 py-2">{d.name}</td>
                  <td className="px-4 py-2">{d.code || '-'}</td>
                  <td className="px-4 py-2 hidden md:table-cell">
                    {d.description || '-'}
                  </td>
                  <td className="px-4 py-2 text-right">
                    <button
                      onClick={() => handleDelete(d.id)}
                      className="text-xs text-destructive hover:underline"
                    >
                      Delete
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

