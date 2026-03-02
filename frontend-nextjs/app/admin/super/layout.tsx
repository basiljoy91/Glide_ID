import { SuperAdminNavbar } from '@/components/layout/SuperAdminNavbar'

export default function SuperAdminLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <div className="flex min-h-screen flex-col">
      <SuperAdminNavbar />
      <main className="flex-1">{children}</main>
    </div>
  )
}


