'use client'

import { DataCard, DataCardGrid } from '@/components/data/DataCard'
import { useAuthStore } from '@/store/useStore'

export default function DashboardPage() {
  const { user } = useAuthStore()

  return (
    <div className="container mx-auto p-6">
      <h1 className="text-3xl font-bold mb-6">Dashboard</h1>
      
      {user && (
        <div className="mb-6">
          <p className="text-muted-foreground">
            Welcome, {user.firstName} {user.lastName}
          </p>
        </div>
      )}

      <DataCardGrid>
        <DataCard
          title="Total Employees"
          value="125"
          icon="👥"
          subtitle="Active users"
        />
        <DataCard
          title="Today's Check-Ins"
          value="89"
          icon="✅"
          subtitle="As of now"
        />
        <DataCard
          title="Pending Reviews"
          value="3"
          icon="⚠️"
          subtitle="Anomalies detected"
        />
        <DataCard
          title="Kiosks Active"
          value="5"
          icon="🖥️"
          subtitle="All systems operational"
        />
      </DataCardGrid>
    </div>
  )
}

