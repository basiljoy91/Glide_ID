'use client'

import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Shield, Mail, Lock } from 'lucide-react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import toast from 'react-hot-toast'
import { useAuthStore } from '@/store/useStore'

export default function AdminLoginPage() {
  const router = useRouter()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [authMethod, setAuthMethod] = useState<'sso' | 'password'>('password')
  const { setUser, setToken } = useAuthStore()

  const routeForRole = (role: string) => {
    if (role === 'super_admin') return '/admin/super'
    if (['org_admin', 'hr', 'dept_manager'].includes(role)) return '/admin/org'
    return '/dashboard'
  }

  const handlePasswordLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!email || !password) {
      toast.error('Please enter both email and password')
      return
    }

    setIsLoading(true)
    try {
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/public/auth/login`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ email, password }),
        }
      )

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.error || 'Login failed')
      }

      const data = await response.json()
      
      // Store auth data
      setToken(data.token)
      setUser({
        id: data.user.id,
        email: data.user.email,
        firstName: data.user.first_name,
        lastName: data.user.last_name,
        role: data.user.role,
        tenantId: data.user.tenant_id,
      })

      toast.success('Login successful!')
      router.push(routeForRole(data.user.role))
    } catch (error: any) {
      toast.error(error.message || 'Login failed')
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="container mx-auto px-4 py-12 max-w-md">
      <div className="border rounded-lg p-8 bg-card">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold mb-2">Admin Login</h1>
          <p className="text-muted-foreground">
            Access your organization&apos;s dashboard
          </p>
        </div>

        {/* Auth Method Toggle */}
        <div className="flex gap-2 mb-6 p-1 bg-muted rounded-lg">
          <button
            type="button"
            onClick={() => setAuthMethod('sso')}
            className={`flex-1 py-2 px-4 rounded-md text-sm font-medium transition-colors ${
              authMethod === 'sso'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            <Shield className="h-4 w-4 inline mr-2" />
            SSO Later
          </button>
          <button
            type="button"
            onClick={() => setAuthMethod('password')}
            className={`flex-1 py-2 px-4 rounded-md text-sm font-medium transition-colors ${
              authMethod === 'password'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            <Lock className="h-4 w-4 inline mr-2" />
            Password
          </button>
        </div>

        {/* SSO Login Form */}
        {authMethod === 'sso' && (
          <div className="space-y-4">
            <div className="rounded-lg border border-amber-300 bg-amber-50 p-4 text-sm text-amber-900">
              Enterprise SSO is not enabled yet in this build. Use password login for now.
            </div>
            <Button type="button" onClick={() => setAuthMethod('password')} className="w-full" size="lg">
              <Lock className="h-4 w-4 mr-2" />
              Use Password Login
            </Button>
          </div>
        )}

        {/* Password Login Form */}
        {authMethod === 'password' && (
          <form onSubmit={handlePasswordLogin} className="space-y-4">
            <div>
              <Label htmlFor="password-email">Email Address</Label>
              <div className="relative mt-1">
                <Mail className="absolute left-3 top-1/2 transform -translate-y-1/2 h-5 w-5 text-muted-foreground" />
                <Input
                  id="password-email"
                  type="email"
                  placeholder="admin@company.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="pl-10"
                  required
                />
              </div>
            </div>

            <div>
              <Label htmlFor="password">Password</Label>
              <div className="relative mt-1">
                <Lock className="absolute left-3 top-1/2 transform -translate-y-1/2 h-5 w-5 text-muted-foreground" />
                <Input
                  id="password"
                  type="password"
                  placeholder="Enter your password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="pl-10"
                  required
                />
              </div>
            </div>

            <div className="flex items-center justify-between text-sm">
              <label className="flex items-center">
                <input type="checkbox" className="mr-2" />
                <span className="text-muted-foreground">Remember me</span>
              </label>
              <span className="text-muted-foreground">Password reset is managed by your admin.</span>
            </div>

            <Button
              type="submit"
              disabled={isLoading}
              className="w-full"
              size="lg"
            >
              {isLoading ? (
                <>
                  <div className="w-4 h-4 border-2 border-primary-foreground border-t-transparent rounded-full animate-spin mr-2" />
                  Signing in...
                </>
              ) : (
                'Sign In'
              )}
            </Button>
          </form>
        )}

        <div className="mt-6 text-center text-sm">
          <p className="text-muted-foreground">
            Don&apos;t have an account?{' '}
            <Link href="/onboarding" className="text-primary hover:underline">
              Get Started
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}
