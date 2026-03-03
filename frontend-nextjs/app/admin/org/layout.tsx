import { OrgAdminNavbar } from '@/components/layout/OrgAdminNavbar'

export default function OrgAdminLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <div className="flex min-h-screen flex-col">
      <OrgAdminNavbar />
      <main className="flex-1">{children}</main>
    </div>
  )
}

