import Link from 'next/link'
import { format } from 'date-fns'
import { Button } from '@/components/ui/button'

export type AnomalyPreviewRow = {
  id: string
  punch_time: string
  employee_id: string
  first_name: string
  last_name: string
  anomaly_reason?: string | null
  kiosk_code?: string | null
}

export function AnomaliesPreviewTable({
  rows,
  emptyText = 'No anomalies found.',
  compact = false,
}: {
  rows: AnomalyPreviewRow[]
  emptyText?: string
  compact?: boolean
}) {
  if (!rows.length) {
    return <div className="text-sm text-muted-foreground">{emptyText}</div>
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead className="text-muted-foreground">
          <tr className="border-b">
            <th className="text-left font-medium py-2 pr-3">Time</th>
            <th className="text-left font-medium py-2 pr-3">Employee</th>
            {!compact && <th className="text-left font-medium py-2 pr-3">Kiosk</th>}
            <th className="text-left font-medium py-2 pr-3">Reason</th>
            <th className="text-right font-medium py-2 pl-3">Action</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.id} className="border-b last:border-b-0">
              <td className="py-2 pr-3 whitespace-nowrap">
                {format(new Date(r.punch_time), 'MMM d, h:mm a')}
              </td>
              <td className="py-2 pr-3">
                <div className="font-medium">
                  {r.first_name} {r.last_name}
                </div>
                <div className="text-xs text-muted-foreground">{r.employee_id}</div>
              </td>
              {!compact && (
                <td className="py-2 pr-3 text-muted-foreground">
                  {r.kiosk_code || '—'}
                </td>
              )}
              <td className="py-2 pr-3 text-muted-foreground">
                {r.anomaly_reason || '—'}
              </td>
              <td className="py-2 pl-3 text-right">
                <Link href={`/admin/org/reviews/anomalies/${r.id}`}>
                  <Button size="sm" variant="outline">
                    Review
                  </Button>
                </Link>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

