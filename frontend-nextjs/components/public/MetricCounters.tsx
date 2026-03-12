'use client'

import { useEffect, useMemo, useState } from 'react'

type Metric = {
  label: string
  target: number
  suffix?: string
  prefix?: string
  decimals?: number
}

function useCountUp(target: number, durationMs = 1200) {
  const [value, setValue] = useState(0)

  useEffect(() => {
    let frame: number
    let start: number | null = null

    const tick = (ts: number) => {
      if (start === null) start = ts
      const progress = Math.min((ts - start) / durationMs, 1)
      const eased = 1 - Math.pow(1 - progress, 3)
      setValue(target * eased)
      if (progress < 1) {
        frame = window.requestAnimationFrame(tick)
      }
    }

    frame = window.requestAnimationFrame(tick)
    return () => window.cancelAnimationFrame(frame)
  }, [target, durationMs])

  return value
}

function MetricCard({ metric }: { metric: Metric }) {
  const value = useCountUp(metric.target)
  const formatted =
    metric.decimals !== undefined
      ? value.toFixed(metric.decimals)
      : Math.round(value).toLocaleString('en-US')

  return (
    <div className="rounded-2xl border bg-background/80 p-6 shadow-sm backdrop-blur">
      <div className="text-2xl font-semibold font-display text-foreground">
        {metric.prefix || ''}
        {formatted}
        {metric.suffix || ''}
      </div>
      <div className="mt-2 text-sm text-muted-foreground">{metric.label}</div>
    </div>
  )
}

export function MetricCounters() {
  const metrics: Metric[] = useMemo(
    () => [
      { label: 'Daily check-ins processed', target: 1200000, suffix: '+' },
      { label: 'Average verification time', target: 1.2, suffix: 's', decimals: 1 },
      { label: 'Global uptime', target: 99.98, suffix: '%', decimals: 2 },
      { label: 'Organizations onboarded', target: 380, suffix: '+' },
    ],
    []
  )

  return (
    <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
      {metrics.map((metric) => (
        <MetricCard key={metric.label} metric={metric} />
      ))}
    </div>
  )
}
