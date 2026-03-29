'use client'

import Link from 'next/link'
import { useMemo, useState } from 'react'
import { useAuthStore } from '@/store/useStore'
import { usePathname, useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
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
  User,
  X,
  PanelLeft,
  ChevronRight,
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

type OrgAdminShellProps = {
  children: React.ReactNode
}

function isItemActive(pathname: string, href: string) {
  if (href === '/admin/org') return pathname === href
  return pathname === href || pathname.startsWith(`${href}/`)
}

function filterSections(userRole?: string, permissions?: string[]) {
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
      title: 'Administration',
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
        if (userRole === 'org_admin') return true
        return !!permissions?.includes(item.permission)
      }),
    }))
    .filter((section) => section.items.length > 0)
}

function getActiveContext(sections: NavSection[], pathname: string) {
  for (const section of sections) {
    const item = section.items.find((candidate) => isItemActive(pathname, candidate.href))
    if (item) {
      return { section, item }
    }
  }
  return {
    section: sections[0],
    item: sections[0]?.items[0],
  }
}

export function OrgAdminNavbar({ children }: OrgAdminShellProps) {
  const { user, logout } = useAuthStore()
  const router = useRouter()
  const pathname = usePathname()
  const [isMobileSidebarOpen, setIsMobileSidebarOpen] = useState(false)

  const sections = useMemo(() => filterSections(user?.role, user?.permissions), [user?.permissions, user?.role])
  const activeContext = useMemo(() => getActiveContext(sections, pathname), [sections, pathname])

  const handleLogout = () => {
    logout()
    toast.success('Logged out successfully')
    router.push('/admin/login')
  }

  const renderSidebarItem = (item: NavItem, mobile = false) => {
    const Icon = item.icon
    const active = isItemActive(pathname, item.href)

    return (
      <Link key={item.href} href={item.href} onClick={() => setIsMobileSidebarOpen(false)}>
        <div
          className={cn(
            'flex items-center justify-between rounded-xl px-3 py-2.5 text-sm transition-colors',
            active
              ? 'bg-primary text-primary-foreground shadow-sm'
              : 'text-muted-foreground hover:bg-muted hover:text-foreground'
          )}
        >
          <div className="flex items-center gap-3">
            <Icon className="h-4 w-4" />
            <span className="font-medium">{item.label}</span>
          </div>
          {mobile ? <ChevronRight className="h-4 w-4 opacity-60" /> : null}
        </div>
      </Link>
    )
  }

  const renderSectionTabs = () => {
    if (!activeContext.section) return null
    return (
      <div className="flex gap-2 overflow-x-auto px-4 py-3 md:px-6">
        {activeContext.section.items.map((item) => {
          const Icon = item.icon
          const active = isItemActive(pathname, item.href)
          return (
            <Link key={item.href} href={item.href}>
              <Button
                variant={active ? 'secondary' : 'ghost'}
                className={cn('gap-2 whitespace-nowrap rounded-full', active ? 'font-semibold' : 'text-muted-foreground')}
              >
                <Icon className="h-4 w-4" />
                <span>{item.label}</span>
              </Button>
            </Link>
          )
        })}
      </div>
    )
  }

  return (
    <div className="flex min-h-screen bg-background">
      <aside className="hidden w-72 shrink-0 border-r bg-muted/20 lg:flex lg:flex-col">
        <div className="border-b px-6 py-5">
          <Link href="/admin/org" className="block">
            <div className="text-2xl font-bold leading-none">Glide ID</div>
            <div className="mt-2 text-xs uppercase tracking-[0.18em] text-muted-foreground">Org Admin Panel</div>
          </Link>
        </div>

        <div className="flex-1 overflow-y-auto px-4 py-5">
          <div className="space-y-6">
            {sections.map((section) => (
              <div key={section.title}>
                <div className="mb-2 px-2 text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                  {section.title}
                </div>
                <div className="space-y-1">{section.items.map((item) => renderSidebarItem(item))}</div>
              </div>
            ))}
          </div>
        </div>

        <div className="border-t px-4 py-4">
          <div className="rounded-2xl border bg-background px-4 py-3">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-muted">
                <User className="h-4 w-4 text-muted-foreground" />
              </div>
              <div className="min-w-0">
                <div className="truncate text-sm font-medium">{user?.firstName || 'Admin'} {user?.lastName || ''}</div>
                <div className="truncate text-xs text-muted-foreground">{user?.email}</div>
              </div>
            </div>
            <Button variant="outline" className="mt-3 w-full justify-start gap-2" onClick={handleLogout}>
              <LogOut className="h-4 w-4" />
              <span>Log out</span>
            </Button>
          </div>
        </div>
      </aside>

      {isMobileSidebarOpen ? (
        <div className="fixed inset-0 z-40 bg-black/40 lg:hidden" onClick={() => setIsMobileSidebarOpen(false)}>
          <div
            className="h-full w-[86%] max-w-sm bg-background shadow-xl"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="flex items-center justify-between border-b px-4 py-4">
              <div>
                <div className="text-lg font-bold">Glide ID</div>
                <div className="text-xs uppercase tracking-[0.16em] text-muted-foreground">Org Admin Panel</div>
              </div>
              <Button variant="ghost" size="icon" onClick={() => setIsMobileSidebarOpen(false)}>
                <X className="h-4 w-4" />
              </Button>
            </div>

            <div className="space-y-5 overflow-y-auto px-4 py-4">
              <div className="rounded-2xl border bg-muted/20 px-4 py-3">
                <div className="text-sm font-medium">{user?.firstName || 'Admin'} {user?.lastName || ''}</div>
                <div className="mt-1 text-xs text-muted-foreground">{user?.email}</div>
              </div>

              {sections.map((section) => (
                <div key={section.title}>
                  <div className="mb-2 px-2 text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                    {section.title}
                  </div>
                  <div className="space-y-1">{section.items.map((item) => renderSidebarItem(item, true))}</div>
                </div>
              ))}
            </div>

            <div className="border-t px-4 py-4">
              <Button variant="outline" className="w-full justify-start gap-2" onClick={handleLogout}>
                <LogOut className="h-4 w-4" />
                <span>Log out</span>
              </Button>
            </div>
          </div>
        </div>
      ) : null}

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="sticky top-0 z-30 border-b bg-background/95 backdrop-blur">
          <div className="flex min-h-16 items-center justify-between gap-4 px-4 py-3 md:px-6">
            <div className="flex min-w-0 items-center gap-3">
              <Button
                variant="ghost"
                size="icon"
                className="lg:hidden"
                onClick={() => setIsMobileSidebarOpen(true)}
                aria-label="Open sidebar"
              >
                <PanelLeft className="h-4 w-4" />
              </Button>
              <div className="min-w-0">
                <div className="text-xs uppercase tracking-[0.16em] text-muted-foreground">
                  {activeContext.section?.title || 'Org Admin'}
                </div>
                <div className="truncate text-lg font-semibold">
                  {activeContext.item?.label || 'Dashboard'}
                </div>
              </div>
            </div>

            <div className="flex items-center gap-2 md:gap-4">
              <div className="hidden rounded-full border px-3 py-1 text-sm text-muted-foreground md:block">
                {user?.email}
              </div>
              <Button variant="outline" size="sm" onClick={handleLogout} className="hidden md:inline-flex">
                Log out
              </Button>
            </div>
          </div>
          <div className="border-t">{renderSectionTabs()}</div>
        </header>

        <main className="min-w-0 flex-1">{children}</main>
      </div>
    </div>
  )
}
