'use client'

import Link from 'next/link'
import { useState } from 'react'
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
  CalendarRange,
  ShieldCheck,
  Settings,
  Shield,
  KeyRound,
  Laptop,
  CreditCard,
  Bell,
  LifeBuoy,
  LogOut,
  Menu,
  User,
  X,
} from 'lucide-react'
import toast from 'react-hot-toast'

export function OrgAdminNavbar() {
  const { user, logout } = useAuthStore()
  const router = useRouter()
  const [isMobileOpen, setIsMobileOpen] = useState(false)

  const navItems = [
    { href: '/admin/org', label: 'Dashboard', icon: LayoutDashboard },
    { href: '/admin/org/users', label: 'Employees', icon: Users },
    { href: '/admin/org/departments', label: 'Departments', icon: Building2 },
    { href: '/admin/org/integrations', label: 'Integrations', icon: Cable },
    { href: '/admin/org/kiosks', label: 'Kiosks', icon: MonitorSmartphone },
    { href: '/admin/org/reviews/anomalies', label: 'Reviews', icon: AlertTriangle },
    { href: '/admin/org/operations', label: 'Operations', icon: CalendarRange },
    { href: '/admin/org/reports/attendance', label: 'Reports', icon: BarChart3 },
    { href: '/admin/org/finance', label: 'Finance', icon: CreditCard },
    { href: '/admin/org/alerts', label: 'Alerts', icon: Bell },
    { href: '/admin/org/support', label: 'Support', icon: LifeBuoy },
    { href: '/admin/org/audit', label: 'Audit', icon: ShieldCheck },
    { href: '/admin/org/settings', label: 'Settings', icon: Settings, permission: 'settings.manage' },
    { href: '/admin/org/security', label: 'Security', icon: Shield, permission: 'security.manage' },
    { href: '/admin/org/access', label: 'Roles', icon: KeyRound, permission: 'roles.manage' },
    { href: '/admin/org/sessions', label: 'Sessions', icon: Laptop, permission: 'sessions.manage' },
  ].filter((item) => {
    if (!item.permission) return true
    if (user?.role === 'org_admin') return true
    return !!user?.permissions?.includes(item.permission)
  })

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
            {navItems.map((item) => {
              const Icon = item.icon
              return (
                <Link key={item.href} href={item.href}>
                  <Button variant="ghost" className="flex items-center space-x-2">
                    <Icon className="h-4 w-4" />
                    <span>{item.label}</span>
                  </Button>
                </Link>
              )
            })}
          </div>

          {/* User Menu + Logout at top-right (spec-compliant) */}
          <div className="flex items-center space-x-4">
            <Button
              variant="ghost"
              size="icon"
              className="md:hidden"
              onClick={() => setIsMobileOpen((value) => !value)}
            >
              {isMobileOpen ? <X className="h-4 w-4" /> : <Menu className="h-4 w-4" />}
            </Button>
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
        {isMobileOpen && (
          <div className="border-t py-3 md:hidden">
            <div className="grid gap-2">
              {navItems.map((item) => {
                const Icon = item.icon
                return (
                  <Link key={item.href} href={item.href} onClick={() => setIsMobileOpen(false)}>
                    <Button variant="ghost" className="w-full justify-start space-x-2">
                      <Icon className="h-4 w-4" />
                      <span>{item.label}</span>
                    </Button>
                  </Link>
                )
              })}
            </div>
          </div>
        )}
      </div>
    </nav>
  )
}
