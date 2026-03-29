'use client'

import Link from 'next/link'
import { useAuthStore } from '@/store/useStore'
import { usePathname, useRouter } from 'next/navigation'
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { 
  LayoutDashboard, 
  Building2, 
  CreditCard, 
  Settings,
  LogOut,
  User,
  Menu,
  X
} from 'lucide-react'
import toast from 'react-hot-toast'

export function SuperAdminNavbar() {
  const { user, logout } = useAuthStore()
  const router = useRouter()
  const pathname = usePathname()
  const [mobileOpen, setMobileOpen] = useState(false)

  const navItems = [
    { href: '/admin/super', label: 'Dashboard', icon: LayoutDashboard },
    { href: '/admin/super/organizations', label: 'Organizations', icon: Building2 },
    { href: '/admin/super/billing', label: 'Billing', icon: CreditCard },
    { href: '/admin/super/settings', label: 'Settings', icon: Settings },
  ]

  const isActive = (href: string) => pathname === href
  const navButtonClass = (href: string) =>
    isActive(href)
      ? 'bg-primary text-primary-foreground hover:bg-primary/90 hover:text-primary-foreground'
      : 'text-muted-foreground hover:text-foreground'

  const handleLogout = () => {
    logout()
    toast.success('Logged out successfully')
    router.push('/admin/login')
  }

  return (
    <>
      <nav className="border-b bg-background">
      <div className="container mx-auto px-4">
        <div className="flex h-16 items-center justify-between">
          <Link href="/admin/super" className="flex items-center space-x-2">
            <span className="text-xl font-bold">Glide ID</span>
            <span className="text-sm text-muted-foreground">Super Admin</span>
          </Link>

          <div className="hidden md:flex items-center space-x-1">
            {navItems.map((item) => {
              const Icon = item.icon
              return (
                <Link key={item.href} href={item.href}>
                  <Button variant="ghost" className={`flex items-center space-x-2 ${navButtonClass(item.href)}`}>
                    <Icon className="h-4 w-4" />
                    <span>{item.label}</span>
                  </Button>
                </Link>
              )
            })}
          </div>

          <div className="flex items-center space-x-4">
            {user && (
              <div className="flex items-center space-x-2 text-sm">
                <User className="h-4 w-4 text-muted-foreground" />
                <span className="hidden sm:inline">{user.email}</span>
              </div>
            )}
            <Button
              variant="ghost"
              size="icon"
              className="md:hidden"
              onClick={() => setMobileOpen((prev) => !prev)}
              aria-label={mobileOpen ? 'Close navigation menu' : 'Open navigation menu'}
            >
              {mobileOpen ? <X className="h-4 w-4" /> : <Menu className="h-4 w-4" />}
            </Button>
            <Button variant="ghost" onClick={handleLogout} size="icon">
              <LogOut className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </div>
      </nav>

      {mobileOpen ? (
        <div className="md:hidden border-b bg-background">
          <div className="container mx-auto px-4 py-4 space-y-3">
            <div className="rounded-lg border border-border bg-card p-3 text-sm">
              <div className="font-medium">{user?.email || 'Super admin'}</div>
              <div className="text-xs text-muted-foreground">Platform controls</div>
            </div>
            <div className="grid gap-2">
              {navItems.map((item) => {
                const Icon = item.icon
                return (
                  <Link key={item.href} href={item.href} onClick={() => setMobileOpen(false)}>
                    <div
                      className={`flex items-center gap-3 rounded-lg border px-4 py-3 text-sm transition-colors ${
                        isActive(item.href)
                          ? 'border-primary bg-primary/10 text-primary'
                          : 'border-border bg-card text-foreground'
                      }`}
                    >
                      <Icon className="h-4 w-4" />
                      <span>{item.label}</span>
                    </div>
                  </Link>
                )
              })}
            </div>
          </div>
        </div>
      ) : null}
    </>
  )
}
