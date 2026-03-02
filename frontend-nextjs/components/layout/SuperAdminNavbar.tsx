'use client'

import Link from 'next/link'
import { useAuthStore } from '@/store/useStore'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { 
  LayoutDashboard, 
  Building2, 
  Users, 
  CreditCard, 
  Settings,
  LogOut,
  User
} from 'lucide-react'
import toast from 'react-hot-toast'

export function SuperAdminNavbar() {
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
          <Link href="/admin/super" className="flex items-center space-x-2">
            <span className="text-xl font-bold">Glide ID</span>
            <span className="text-sm text-muted-foreground">Super Admin</span>
          </Link>

          {/* Navigation Links */}
          <div className="hidden md:flex items-center space-x-1">
            <Link href="/admin/super">
              <Button variant="ghost" className="flex items-center space-x-2">
                <LayoutDashboard className="h-4 w-4" />
                <span>Dashboard</span>
              </Button>
            </Link>
            <Link href="/admin/super/organizations">
              <Button variant="ghost" className="flex items-center space-x-2">
                <Building2 className="h-4 w-4" />
                <span>Organizations</span>
              </Button>
            </Link>
            <Link href="/admin/super/billing">
              <Button variant="ghost" className="flex items-center space-x-2">
                <CreditCard className="h-4 w-4" />
                <span>Billing</span>
              </Button>
            </Link>
            <Link href="/admin/super/settings">
              <Button variant="ghost" className="flex items-center space-x-2">
                <Settings className="h-4 w-4" />
                <span>Settings</span>
              </Button>
            </Link>
          </div>

          {/* User Menu */}
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

