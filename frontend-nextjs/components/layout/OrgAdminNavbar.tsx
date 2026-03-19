'use client'

import Link from 'next/link'
import { useMemo, useState } from 'react'
import { useAuthStore } from '@/store/useStore'
import { usePathname, useRouter } from 'next/navigation'
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
  type LucideIcon,
} from 'lucide-react'
import toast from 'react-hot-toast'

type NavItem = {
  href: string
  label: string
  icon: LucideIcon
  permission?: string
}

type NavSection = {
  title: string
  items: NavItem[]
}

function isItemActive(pathname: string, href: string) {
  if (href === '/admin/org') {
    return pathname === href
  }
  return pathname === href || pathname.startsWith(`${href}/`)
}

export function OrgAdminNavbar() {
  const { user, logout } = useAuthStore()
  const router = useRouter()
  const pathname = usePathname()
  const [isMobileOpen, setIsMobileOpen] = useState(false)

  const sections = useMemo<NavSection[]>(() => {
    const rawSections: NavSection[] = [
      {
        title: 'Core',
        items: [
          { href: '/admin/org', label: 'Dashboard', icon: LayoutDashboard },
          { href: '/admin/org/users', label: 'Employees', icon: Users },
          { href: '/admin/org/departments', label: 'Departments', icon: Building2 },
          { href: '/admin/org/reviews/anomalies', label: 'Reviews', icon: AlertTriangle },
          { href: '/admin/org/reports/attendance', label: 'Reports', icon: BarChart3 },
        ],
      },
      {
        title: 'Operations',
        items: [
          { href: '/admin/org/operations', label: 'Operations', icon: CalendarRange },
          { href: '/admin/org/kiosks', label: 'Kiosks', icon: MonitorSmartphone },
          { href: '/admin/org/integrations', label: 'Integrations', icon: Cable },
          { href: '/admin/org/finance', label: 'Finance', icon: CreditCard },
          { href: '/admin/org/alerts', label: 'Alerts', icon: Bell },
          { href: '/admin/org/support', label: 'Support', icon: LifeBuoy },
        ],
      },
      {
        title: 'Admin',
        items: [
          { href: '/admin/org/audit', label: 'Audit', icon: ShieldCheck },
          { href: '/admin/org/settings', label: 'Settings', icon: Settings, permission: 'settings.manage' },
          { href: '/admin/org/security', label: 'Security', icon: Shield, permission: 'security.manage' },
          { href: '/admin/org/access', label: 'Roles', icon: KeyRound, permission: 'roles.manage' },
          { href: '/admin/org/sessions', label: 'Sessions', icon: Laptop, permission: 'sessions.manage' },
        ],
      },
    ]

    return rawSections
      .map((section) => ({
        ...section,
        items: section.items.filter((item) => {
          if (!item.permission) return true
          if (user?.role === 'org_admin') return true
          return !!user?.permissions?.includes(item.permission)
        }),
      }))
      .filter((section) => section.items.length > 0)
  }, [user?.permissions, user?.role])

  const handleLogout = () => {
    logout()
    toast.success('Logged out successfully')
    router.push('/admin/login')
  }

  const renderNavButton = (item: NavItem, compact = false) => {
    const Icon = item.icon
    const active = isItemActive(pathname, item.href)
    return (
      <Link key={item.href} href={item.href} onClick={() => setIsMobileOpen(false)}>
        <Button
          variant={active ? 'secondary' : 'ghost'}
          className={[
            compact ? 'w-full justify-start gap-2' : 'gap-2 whitespace-nowrap',
            active ? 'font-semibold' : 'text-muted-foreground',
          ].join(' ')}
        >
          <Icon className="h-4 w-4" />
          <span>{item.label}</span>
        </Button>
      </Link>
    )
  }

  return (
    <nav className="border-b bg-background">
      <div className="container mx-auto px-4">
        <div className="flex min-h-16 items-center justify-between gap-4 py-3">
          <Link href="/admin/org" className="flex min-w-0 items-center gap-3">
            <div className="min-w-0">
              <div className="text-xl font-bold leading-none">Glide ID</div>
              <div className="text-xs uppercase tracking-[0.18em] text-muted-foreground">Org Admin</div>
            </div>
          </Link>

          <div className="flex items-center gap-2 md:gap-4">
            {user && (
              <div className="hidden items-center gap-2 rounded-full border px-3 py-1 text-sm md:flex">
                <User className="h-4 w-4 text-muted-foreground" />
                <span className="max-w-[220px] truncate">{user.email}</span>
              </div>
            )}
            <Button
              variant="ghost"
              size="icon"
              className="md:hidden"
              onClick={() => setIsMobileOpen((value) => !value)}
              aria-label="Toggle navigation"
            >
              {isMobileOpen ? <X className="h-4 w-4" /> : <Menu className="h-4 w-4" />}
            </Button>
            <Button variant="ghost" onClick={handleLogout} size="icon" aria-label="Log out">
              <LogOut className="h-4 w-4" />
            </Button>
          </div>
        </div>

        <div className="hidden border-t py-3 md:block">
          <div className="flex flex-wrap gap-6">
            {sections.map((section) => (
              <div key={section.title} className="min-w-0">
                <div className="mb-2 text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                  {section.title}
                </div>
                <div className="flex flex-wrap gap-1">
                  {section.items.map((item) => renderNavButton(item))}
                </div>
              </div>
            ))}
          </div>
        </div>

        {isMobileOpen && (
          <div className="border-t py-3 md:hidden">
            <div className="mb-3 flex items-center gap-2 rounded-lg border px-3 py-2 text-sm">
              <User className="h-4 w-4 text-muted-foreground" />
              <span className="truncate">{user?.email}</span>
            </div>
            <div className="space-y-4">
              {sections.map((section) => (
                <div key={section.title}>
                  <div className="mb-2 text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                    {section.title}
                  </div>
                  <div className="grid gap-1">
                    {section.items.map((item) => renderNavButton(item, true))}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </nav>
  )
}
