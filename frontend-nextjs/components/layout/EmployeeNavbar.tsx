'use client'

import Link from 'next/link'
import { useAuthStore } from '@/store/useStore'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import {
  LayoutDashboard,
  LogOut,
  User,
} from 'lucide-react'
import toast from 'react-hot-toast'

export function EmployeeNavbar() {
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
          <Link href="/dashboard" className="flex items-center space-x-2">
            <span className="text-xl font-bold">Glide ID</span>
            <span className="text-sm text-muted-foreground">Employee Portal</span>
          </Link>

          {/* Navigation Links */}
          <div className="hidden md:flex items-center space-x-1">
            <Link href="/dashboard">
              <Button variant="ghost" className="flex items-center space-x-2">
                <LayoutDashboard className="h-4 w-4" />
                <span>My Dashboard</span>
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
