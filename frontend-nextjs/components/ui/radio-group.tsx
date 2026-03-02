'use client'

import * as React from "react"
import { cn } from "@/lib/utils"

interface RadioGroupContextValue {
  value: string
  onValueChange: (value: string) => void
}

const RadioGroupContext = React.createContext<RadioGroupContextValue | undefined>(undefined)

export interface RadioGroupProps {
  value: string
  onValueChange: (value: string) => void
  children: React.ReactNode
  className?: string
}

export function RadioGroup({ value, onValueChange, children, className }: RadioGroupProps) {
  return (
    <RadioGroupContext.Provider value={{ value, onValueChange }}>
      <div className={cn("space-y-2", className)}>{children}</div>
    </RadioGroupContext.Provider>
  )
}

export interface RadioGroupItemProps extends React.InputHTMLAttributes<HTMLInputElement> {
  value: string
}

export const RadioGroupItem = React.forwardRef<HTMLInputElement, RadioGroupItemProps>(
  ({ className, value, id, ...props }, ref) => {
    const context = React.useContext(RadioGroupContext)
    if (!context) throw new Error('RadioGroupItem must be used within RadioGroup')

    return (
      <input
        ref={ref}
        type="radio"
        id={id || value}
        value={value}
        checked={context.value === value}
        onChange={() => context.onValueChange(value)}
        className={cn(
          "h-4 w-4 border-primary text-primary focus:ring-2 focus:ring-ring focus:ring-offset-2",
          className
        )}
        {...props}
      />
    )
  }
)
RadioGroupItem.displayName = "RadioGroupItem"

