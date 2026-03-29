import { OrgAdminNavbar } from '@/components/layout/OrgAdminNavbar'

export default function OrgAdminLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <OrgAdminNavbar>{children}</OrgAdminNavbar>
  )
}
