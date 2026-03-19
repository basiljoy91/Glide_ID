'use client'

import Link from 'next/link'
import { useAuthStore } from '@/store/useStore'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import {
  LayoutDashboard,
  Users,
  Building2,
  Cable,
  MonitorSmartphone,
  AlertTriangle,
  BarChart3,
  ShieldCheck,
  LogOut,
  User,
} from 'lucide-react'
import toast from 'react-hot-toast'

export function OrgAdminNavbar() {
  const { user, logout } = useAuthStore()
  const router = useRouter()

  const handleLogout = () => {
    logout()
    toast.success('Logged out successfully')
    router.push('/admin/login')
  }

  return (
    <nav className="border-b bg-background">
      <div className="container mx-auto px-4">
        <div className="flex h-16 items-center justify-between">
          {/* Logo */}
          <Link href="/admin/org" className="flex items-center space-x-2">
            <span className="text-xl font-bold">Glide ID</span>
            <span className="text-sm text-muted-foreground">Org Admin</span>
          </Link>

          {/* Navigation Links */}
          <div className="hidden md:flex items-center space-x-1">
            <Link href="/admin/org">
              <Button variant="ghost" className="flex items-center space-x-2">
                <LayoutDashboard className="h-4 w-4" />
                <span>Dashboard</span>
              </Button>
            </Link>
            <Link href="/admin/org/users">
              <Button variant="ghost" className="flex items-center space-x-2">
                <Users className="h-4 w-4" />
                <span>Employees</span>
              </Button>
            </Link>
            <Link href="/admin/org/departments">
              <Button variant="ghost" className="flex items-center space-x-2">
                <Building2 className="h-4 w-4" />
                <span>Departments</span>
              </Button>
            </Link>
            <Link href="/admin/org/integrations">
              <Button variant="ghost" className="flex items-center space-x-2">
                <Cable className="h-4 w-4" />
                <span>Integrations</span>
              </Button>
            </Link>
            <Link href="/admin/org/kiosks">
              <Button variant="ghost" className="flex items-center space-x-2">
                <MonitorSmartphone className="h-4 w-4" />
                <span>Kiosks</span>
              </Button>
            </Link>
            <Link href="/admin/org/reviews/anomalies">
              <Button variant="ghost" className="flex items-center space-x-2">
                <AlertTriangle className="h-4 w-4" />
                <span>Reviews</span>
              </Button>
            </Link>
            <Link href="/admin/org/reports/attendance">
              <Button variant="ghost" className="flex items-center space-x-2">
                <BarChart3 className="h-4 w-4" />
                <span>Reports</span>
              </Button>
            </Link>
            <Link href="/admin/org/audit">
              <Button variant="ghost" className="flex items-center space-x-2">
                <ShieldCheck className="h-4 w-4" />
                <span>Audit</span>
              </Button>
            </Link>
          </div>

          {/* User Menu + Logout at top-right (spec-compliant) */}
          <div className="flex items-center space-x-4">
            {user && (
              <div className="flex items-center space-x-2 text-sm">
                <User className="h-4 w-4 text-muted-foreground" />
                <span className="hidden sm:inline">{user.email}</span>
              </div>
            )}
            <Button variant="ghost" onClick={handleLogout} size="icon">
              <LogOut className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </div>
    </nav>
  )
}
