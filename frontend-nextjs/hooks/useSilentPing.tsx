'use client'

import { useEffect, createContext, useContext, ReactNode } from 'react'
import { config } from '@/lib/config'

interface SilentPingContextType {
  pingBackend: () => Promise<void>
  pingAIService: () => Promise<void>
}

const SilentPingContext = createContext<SilentPingContextType | undefined>(undefined)

export function SilentPingProvider({ children }: { children: ReactNode }) {
  useEffect(() => {
    // Silent ping on mount to warm up backends
    const pingBackends = async () => {
      try {
        // Ping Golang API
        await fetch(`${config.apiUrl}/health`, {
          method: 'GET',
          cache: 'no-cache',
        }).catch(() => {
          // Silently fail - this is just a warm-up ping
        })

        // Ping Python AI Service
        await fetch(`${config.aiServiceUrl}/health`, {
          method: 'GET',
          cache: 'no-cache',
        }).catch(() => {
          // Silently fail - this is just a warm-up ping
        })
      } catch (error) {
        // Silently fail - this is just a warm-up ping
      }
    }

    pingBackends()

    // Continue pinging every 30 seconds to keep backends warm
    const interval = setInterval(pingBackends, 30000)

    return () => clearInterval(interval)
  }, [])

  const pingBackend = async () => {
    try {
      await fetch(`${config.apiUrl}/health`, { method: 'GET', cache: 'no-cache' })
    } catch (error) {
      // Silently fail
    }
  }

  const pingAIService = async () => {
    try {
      await fetch(`${config.aiServiceUrl}/health`, { method: 'GET', cache: 'no-cache' })
    } catch (error) {
      // Silently fail
    }
  }

  return (
    <SilentPingContext.Provider value={{ pingBackend, pingAIService }}>
      {children}
    </SilentPingContext.Provider>
  )
}

export function useSilentPing() {
  const context = useContext(SilentPingContext)
  if (!context) {
    throw new Error('useSilentPing must be used within SilentPingProvider')
  }
  return context
}

