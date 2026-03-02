'use client'

import { useState } from 'react'
import { OnboardingData } from './OnboardingWizard'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { X, Plus } from 'lucide-react'
import toast from 'react-hot-toast'

interface Step3Props {
  data: OnboardingData
  updateData: (updates: Partial<OnboardingData>) => void
}

const roles = [
  { value: 'org_admin', label: 'Organization Admin', description: 'Full access to all features' },
  { value: 'hr', label: 'HR Manager', description: 'Reports, users, leave management' },
  { value: 'dept_manager', label: 'Department Manager', description: 'Access to assigned department only' },
]

export function Step3TeamSetup({ data, updateData }: Step3Props) {
  const [email, setEmail] = useState('')
  const [role, setRole] = useState<'org_admin' | 'hr' | 'dept_manager'>('hr')

  const addTeamMember = () => {
    if (!email || !email.includes('@')) {
      toast.error('Please enter a valid email address')
      return
    }

    if (data.teamMembers.some((m) => m.email === email)) {
      toast.error('This email is already added')
      return
    }

    updateData({
      teamMembers: [...data.teamMembers, { email, role }],
    })
    setEmail('')
    setRole('hr')
    toast.success('Team member added')
  }

  const removeTeamMember = (index: number) => {
    updateData({
      teamMembers: data.teamMembers.filter((_, i) => i !== index),
    })
    toast.success('Team member removed')
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold mb-2">Team Setup</h2>
        <p className="text-muted-foreground">
          Invite other administrators to your organization (optional)
        </p>
      </div>

      <div className="space-y-4">
        {/* Add Team Member Form */}
        <div className="border rounded-lg p-4 space-y-4">
          <h3 className="font-semibold">Add Team Member</h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="md:col-span-2">
              <Label htmlFor="teamEmail">Email Address</Label>
              <Input
                id="teamEmail"
                type="email"
                placeholder="colleague@company.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="mt-1"
                onKeyPress={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault()
                    addTeamMember()
                  }
                }}
              />
            </div>
            <div>
              <Label htmlFor="teamRole">Role</Label>
              <select
                id="teamRole"
                value={role}
                onChange={(e) =>
                  setRole(e.target.value as 'org_admin' | 'hr' | 'dept_manager')
                }
                className="mt-1 flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              >
                {roles.map((r) => (
                  <option key={r.value} value={r.value}>
                    {r.label}
                  </option>
                ))}
              </select>
            </div>
          </div>
          <Button onClick={addTeamMember} type="button" className="w-full md:w-auto">
            <Plus className="h-4 w-4 mr-2" />
            Add Team Member
          </Button>
        </div>

        {/* Team Members List */}
        {data.teamMembers.length > 0 && (
          <div className="space-y-2">
            <h3 className="font-semibold">Team Members ({data.teamMembers.length})</h3>
            <div className="space-y-2">
              {data.teamMembers.map((member, index) => {
                const roleLabel = roles.find((r) => r.value === member.role)?.label
                return (
                  <div
                    key={index}
                    className="flex items-center justify-between p-3 border rounded-lg"
                  >
                    <div>
                      <p className="font-medium">{member.email}</p>
                      <p className="text-sm text-muted-foreground">{roleLabel}</p>
                    </div>
                    <button
                      onClick={() => removeTeamMember(index)}
                      className="text-destructive hover:text-destructive/80"
                      type="button"
                    >
                      <X className="h-5 w-5" />
                    </button>
                  </div>
                )
              })}
            </div>
          </div>
        )}

        {/* Role Descriptions */}
        <div className="border-t pt-4">
          <h3 className="font-semibold mb-3">Role Permissions</h3>
          <div className="space-y-2">
            {roles.map((r) => (
              <div key={r.value} className="text-sm">
                <span className="font-medium">{r.label}:</span>{' '}
                <span className="text-muted-foreground">{r.description}</span>
              </div>
            ))}
          </div>
        </div>

        {data.teamMembers.length === 0 && (
          <div className="text-center py-8 text-muted-foreground">
            <p>No team members added yet.</p>
            <p className="text-sm mt-2">
              You can skip this step and add team members later from the admin dashboard.
            </p>
          </div>
        )}
      </div>
    </div>
  )
}

