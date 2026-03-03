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
  const [authMethod, setAuthMethod] = useState<'sso' | 'password'>('sso')
  const { setUser, setToken } = useAuthStore()

  const handleSSOLogin = async () => {
    if (!email || !email.includes('@')) {
      toast.error('Please enter a valid corporate email address')
      return
    }

    setIsLoading(true)
    try {
      // Detect SSO provider from email domain
      const domain = email.split('@')[1]
      
      // Call backend to initiate SSO flow
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/public/auth/sso/initiate`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ email, domain }),
        }
      )

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.error || 'SSO initiation failed')
      }

      const data = await response.json()
      
      // Redirect to identity provider
      const redirectUrl = data.redirectUrl || data.redirect_url
      if (redirectUrl) {
        window.location.href = redirectUrl
      } else {
        toast.error(data.error || data.message || 'SSO configuration not found for this domain')
      }
    } catch (error: any) {
      toast.error(error.message || 'Failed to initiate SSO login')
    } finally {
      setIsLoading(false)
    }
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
      
      // Redirect based on role
      if (data.user.role === 'super_admin') {
        router.push('/admin/super')
      } else {
        // Org Admin / HR / Dept Manager / Employee
        router.push('/admin/org')
      }
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
            Access your organization's dashboard
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
            Enterprise SSO
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
            <div>
              <Label htmlFor="sso-email">Corporate Email Address</Label>
              <div className="relative mt-1">
                <Mail className="absolute left-3 top-1/2 transform -translate-y-1/2 h-5 w-5 text-muted-foreground" />
                <Input
                  id="sso-email"
                  type="email"
                  placeholder="admin@company.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="pl-10"
                  onKeyPress={(e) => {
                    if (e.key === 'Enter') {
                      handleSSOLogin()
                    }
                  }}
                />
              </div>
              <p className="text-sm text-muted-foreground mt-2">
                Enter your corporate email to be redirected to your identity provider
              </p>
            </div>

            <Button
              onClick={handleSSOLogin}
              disabled={isLoading}
              className="w-full"
              size="lg"
            >
              {isLoading ? (
                <>
                  <div className="w-4 h-4 border-2 border-primary-foreground border-t-transparent rounded-full animate-spin mr-2" />
                  Redirecting...
                </>
              ) : (
                <>
                  <Shield className="h-4 w-4 mr-2" />
                  Continue with SSO
                </>
              )}
            </Button>

            <div className="text-center text-sm text-muted-foreground">
              <p>Supported providers: Okta, Azure AD, Google Workspace, SAML 2.0, OIDC</p>
            </div>
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
              <Link href="/admin/forgot-password" className="text-primary hover:underline">
                Forgot password?
              </Link>
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
            Don't have an account?{' '}
            <Link href="/onboarding" className="text-primary hover:underline">
              Get Started
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}

