import { format } from 'date-fns'

export type ChartPoint7d = {
  date: string // YYYY-MM-DD
  count: number
}

export function MiniBarChart7d({ points }: { points: ChartPoint7d[] }) {
  const max = Math.max(1, ...points.map((p) => p.count))

  return (
    <div className="flex items-end gap-2 h-28">
      {points.map((p) => {
        const heightPct = Math.max(2, Math.round((p.count / max) * 100))
        const dayLabel = format(new Date(`${p.date}T00:00:00`), 'EEE')
        return (
          <div key={p.date} className="flex flex-1 flex-col items-center gap-2">
            <div className="w-full h-20 rounded bg-muted relative overflow-hidden">
              <div
                className="absolute bottom-0 left-0 right-0 bg-primary"
                style={{ height: `${heightPct}%` }}
                title={`${p.date}: ${p.count}`}
              />
            </div>
            <div className="text-[10px] text-muted-foreground">{dayLabel}</div>
          </div>
        )
      })}
    </div>
  )
}

