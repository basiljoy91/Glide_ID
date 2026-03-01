'use client'

import { ReactNode } from 'react'
import { format } from 'date-fns'

interface DataCardProps {
  title: string
  subtitle?: string
  value: string | number | ReactNode
  icon?: ReactNode
  footer?: ReactNode
  onClick?: () => void
  className?: string
}

export function DataCard({
  title,
  subtitle,
  value,
  icon,
  footer,
  onClick,
  className = '',
}: DataCardProps) {
  return (
    <div
      className={`
        bg-card border border-border rounded-lg p-4 shadow-sm
        ${onClick ? 'cursor-pointer hover:shadow-md transition-shadow' : ''}
        ${className}
      `}
      onClick={onClick}
    >
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center space-x-2 mb-1">
            {icon && <div className="text-muted-foreground">{icon}</div>}
            <h3 className="text-sm font-medium text-muted-foreground">{title}</h3>
          </div>
          <div className="text-2xl font-bold text-foreground">{value}</div>
          {subtitle && (
            <p className="text-xs text-muted-foreground mt-1">{subtitle}</p>
          )}
        </div>
      </div>
      {footer && <div className="mt-3 pt-3 border-t border-border">{footer}</div>}
    </div>
  )
}

interface AttendanceCardProps {
  userName: string
  employeeId: string
  punchTime: Date
  status: 'check_in' | 'check_out'
  method: 'biometric' | 'pin'
  className?: string
}

export function AttendanceCard({
  userName,
  employeeId,
  punchTime,
  status,
  method,
  className = '',
}: AttendanceCardProps) {
  return (
    <div className={`bg-card border border-border rounded-lg p-4 ${className}`}>
      <div className="flex items-start justify-between mb-2">
        <div>
          <h4 className="font-semibold text-foreground">{userName}</h4>
          <p className="text-sm text-muted-foreground">{employeeId}</p>
        </div>
        <div className={`
          px-2 py-1 rounded text-xs font-medium
          ${status === 'check_in' 
            ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200' 
            : 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'}
        `}>
          {status === 'check_in' ? 'Check In' : 'Check Out'}
        </div>
      </div>
      <div className="flex items-center justify-between text-sm">
        <span className="text-muted-foreground">
          {format(punchTime, 'MMM d, yyyy h:mm a')}
        </span>
        <span className="text-muted-foreground">
          {method === 'biometric' ? '🔐 Biometric' : '🔑 PIN'}
        </span>
      </div>
    </div>
  )
}

// Responsive grid for data cards
export function DataCardGrid({ children }: { children: ReactNode }) {
  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
      {children}
    </div>
  )
}

