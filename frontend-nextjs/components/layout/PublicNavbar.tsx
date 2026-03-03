'use client'

import Link from 'next/link'
import { useThemeStore } from '@/store/useStore'
import { Moon, Sun } from 'lucide-react'
import { Button } from '@/components/ui/button'

export function PublicNavbar() {
  const { theme, toggleTheme } = useThemeStore()

  return (
    <nav className="border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container mx-auto flex h-16 items-center justify-between px-4">
        {/* Logo */}
        <Link href="/" className="flex items-center space-x-2">
          <span className="text-2xl font-bold">Glide ID</span>
        </Link>

        {/* Navigation Links */}
        <div className="hidden md:flex items-center space-x-6">
          <Link
            href="/about"
            className="text-sm font-medium transition-colors hover:text-primary"
          >
            About Us
          </Link>
          <Link
            href="/blog"
            className="text-sm font-medium transition-colors hover:text-primary"
          >
            Blog
          </Link>
          <Link
            href="/features"
            className="text-sm font-medium transition-colors hover:text-primary"
          >
            Features
          </Link>
          <Link
            href="/contact"
            className="text-sm font-medium transition-colors hover:text-primary"
          >
            Contact Us
          </Link>
          <Link
            href="/pricing"
            className="text-sm font-medium transition-colors hover:text-primary"
          >
            Pricing
          </Link>
        </div>

        {/* Right Side Actions */}
        <div className="flex items-center space-x-4">
          {/* Theme Toggle */}
          <Button
            variant="ghost"
            size="icon"
            onClick={toggleTheme}
            aria-label="Toggle theme"
          >
            {theme === 'dark' ? (
              <Sun className="h-5 w-5" />
            ) : (
              <Moon className="h-5 w-5" />
            )}
          </Button>

          {/* Admin Login Button */}
          <Link href="/admin/login">
            <Button variant="outline">Admin Login</Button>
          </Link>

          {/* Kiosk */}
          <Link href="/kiosk">
            <Button variant="secondary">Kiosk</Button>
          </Link>

          {/* Get Started CTA */}
          <Link href="/onboarding">
            <Button>Get Started</Button>
          </Link>
        </div>
      </div>
    </nav>
  )
}

