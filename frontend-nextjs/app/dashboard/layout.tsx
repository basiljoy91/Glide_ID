export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  // We can render EmployeeNavbar here, but to avoid hydration mismatches 
  // with dynamic auth state, we usually render it as a client component 
  // or let the page handle it. Since we want layout.tsx, we can safely 
  // import EmployeeNavbar and render it on the client. Wait, no, Next layout
  // imports it dynamically? No, just import the client component.
  
  // Actually, wait, let's look at `app/(admin)/org/layout.tsx` for reference?
  // I will just import and use it here.
  return (
    <div className="min-h-screen bg-background">
      <EmployeeNavbarWrapper />
      <main>{children}</main>
    </div>
  )
}

function EmployeeNavbarWrapper() {
  const { EmployeeNavbar } = require('@/components/layout/EmployeeNavbar')
  return <EmployeeNavbar />
}
