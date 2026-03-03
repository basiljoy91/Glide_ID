'use client'

import { useEffect, useState, useCallback } from 'react'
import { offlineQueue } from '@/lib/offline-queue'
import { config } from '@/lib/config'
import { useKioskStore } from '@/store/useStore'

export function useOfflineQueue() {
  const [isOnline, setIsOnline] = useState(true)
  const [queueStats, setQueueStats] = useState({ total: 0, unsynced: 0, synced: 0 })
  const { setOnlineStatus, setLastSyncTime, kioskCode, kioskHmacSecret } = useKioskStore()

  useEffect(() => {
    // Initialize offline queue
    offlineQueue.init()

    // Check online status
    const handleOnline = () => {
      setIsOnline(true)
      setOnlineStatus(true)
      syncQueue()
    }

    const handleOffline = () => {
      setIsOnline(false)
      setOnlineStatus(false)
    }

    window.addEventListener('online', handleOnline)
    window.addEventListener('offline', handleOffline)

    // Initial sync
    syncQueue()

    // Periodic sync (every 30 seconds)
    const syncInterval = setInterval(syncQueue, 30000)

    // Update stats periodically
    const statsInterval = setInterval(updateStats, 5000)

    return () => {
      window.removeEventListener('online', handleOnline)
      window.removeEventListener('offline', handleOffline)
      clearInterval(syncInterval)
      clearInterval(statsInterval)
    }
  }, [])

  const updateStats = async () => {
    const stats = await offlineQueue.getStats()
    setQueueStats(stats)
  }

  const syncQueue = useCallback(async () => {
    if (!navigator.onLine) return
    if (!kioskCode || !kioskHmacSecret) return

    try {
      const result = await offlineQueue.syncQueue(
        config.apiUrl,
        kioskHmacSecret
      )
      
      if (result.success > 0) {
        setLastSyncTime(new Date())
        await updateStats()
      }
    } catch (error) {
      console.error('Queue sync failed:', error)
    }
  }, [kioskCode, kioskHmacSecret, setLastSyncTime, setOnlineStatus])

  const addToQueue = useCallback(async (
    type: 'check_in' | 'check_out',
    imageData: string,
    monotonicOffset: number = 0
  ) => {
    if (!kioskCode) throw new Error('Kiosk code not set')
    const id = await offlineQueue.addToQueue(type, imageData, monotonicOffset, kioskCode, true)
    await updateStats()
    return id
  }, [kioskCode])

  return {
    isOnline,
    queueStats,
    addToQueue,
    syncQueue,
  }
}

