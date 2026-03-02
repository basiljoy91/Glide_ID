import { SuperAdminNavbar } from '@/components/layout/SuperAdminNavbar'
import { ThemeProvider } from '@/components/theme-provider'
import { Toaster } from 'react-hot-toast'

export default function SuperAdminLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <ThemeProvider>
      <div className="flex min-h-screen flex-col">
        <SuperAdminNavbar />
        <main className="flex-1">{children}</main>
        <Toaster position="top-center" />
      </div>
    </ThemeProvider>
  )
}

