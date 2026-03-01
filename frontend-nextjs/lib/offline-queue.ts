import { openDB, DBSchema, IDBPDatabase } from 'idb'
import CryptoJS from 'crypto-js'
import { config } from './config'

interface OfflineQueueItem {
  id: string
  type: 'check_in' | 'check_out'
  imageData: string
  encryptedPayload: string
  timestamp: number
  monotonicOffset: number
  synced: boolean
  retryCount: number
}

interface AttendanceDB extends DBSchema {
  queue: {
    key: string
    value: OfflineQueueItem
    indexes: { 'by-synced': boolean; 'by-timestamp': number }
  }
}

class OfflineQueue {
  private db: IDBPDatabase<AttendanceDB> | null = null
  private publicKey: string

  constructor() {
    this.publicKey = config.publicKey || ''
  }

  async init() {
    if (!this.db) {
      this.db = await openDB<AttendanceDB>('attendance-offline', 1, {
        upgrade(db) {
          const queueStore = db.createObjectStore('queue', { keyPath: 'id' })
          queueStore.createIndex('by-synced', 'synced')
          queueStore.createIndex('by-timestamp', 'timestamp')
        },
      })
    }
    return this.db
  }

  // Encrypt payload using public key (asymmetric encryption)
  private encryptPayload(data: any): string {
    if (!this.publicKey) {
      // Fallback to AES if no public key (for development)
      return CryptoJS.AES.encrypt(JSON.stringify(data), 'dev-key').toString()
    }

    // In production, use RSA encryption with public key
    // For now, using AES with public key as salt
    return CryptoJS.AES.encrypt(JSON.stringify(data), this.publicKey).toString()
  }

  // Add item to offline queue
  async addToQueue(
    type: 'check_in' | 'check_out',
    imageData: string,
    monotonicOffset: number = 0
  ): Promise<string> {
    const db = await this.init()
    
    const item: OfflineQueueItem = {
      id: `queue-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      type,
      imageData,
      encryptedPayload: this.encryptPayload({
        type,
        imageData,
        timestamp: Date.now(),
        monotonicOffset,
      }),
      timestamp: Date.now(),
      monotonicOffset,
      synced: false,
      retryCount: 0,
    }

    await db.add('queue', item)
    return item.id
  }

  // Get all unsynced items
  async getUnsyncedItems(): Promise<OfflineQueueItem[]> {
    const db = await this.init()
    const index = db.transaction('queue').store.index('by-synced')
    return index.getAll(false)
  }

  // Mark item as synced
  async markAsSynced(id: string) {
    const db = await this.init()
    const item = await db.get('queue', id)
    if (item) {
      item.synced = true
      await db.put('queue', item)
    }
  }

  // Increment retry count
  async incrementRetry(id: string) {
    const db = await this.init()
    const item = await db.get('queue', id)
    if (item) {
      item.retryCount += 1
      await db.put('queue', item)
    }
  }

  // Remove synced items older than 7 days
  async cleanup() {
    const db = await this.init()
    const sevenDaysAgo = Date.now() - 7 * 24 * 60 * 60 * 1000
    
    const index = db.transaction('queue').store.index('by-timestamp')
    const allItems = await index.getAll()
    
    for (const item of allItems) {
      if (item.synced && item.timestamp < sevenDaysAgo) {
        await db.delete('queue', item.id)
      }
    }
  }

  // Sync queue with backend
  async syncQueue(apiUrl: string, apiKey: string): Promise<{ success: number; failed: number }> {
    const unsynced = await this.getUnsyncedItems()
    let success = 0
    let failed = 0

    for (const item of unsynced) {
      try {
        // Decrypt payload (in real implementation, backend would decrypt)
        const decrypted = CryptoJS.AES.decrypt(item.encryptedPayload, this.publicKey || 'dev-key').toString(CryptoJS.enc.Utf8)
        const payload = JSON.parse(decrypted)

        // Send to backend
        const response = await fetch(`${apiUrl}/api/v1/kiosk/check-in`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-API-Key': apiKey,
          },
          body: JSON.stringify({
            image_base64: payload.imageData.split(',')[1], // Remove data:image/jpeg;base64, prefix
            local_time: new Date(payload.timestamp).toISOString(),
            monotonic_offset_ms: payload.monotonicOffset,
            verification_method: 'biometric',
          }),
        })

        if (response.ok) {
          await this.markAsSynced(item.id)
          success++
        } else {
          await this.incrementRetry(item.id)
          failed++
        }
      } catch (error) {
        console.error('Sync error:', error)
        await this.incrementRetry(item.id)
        failed++
      }
    }

    return { success, failed }
  }

  // Get queue statistics
  async getStats() {
    const db = await this.init()
    const allItems = await db.getAll('queue')
    const unsynced = allItems.filter(item => !item.synced)
    
    return {
      total: allItems.length,
      unsynced: unsynced.length,
      synced: allItems.length - unsynced.length,
    }
  }
}

export const offlineQueue = new OfflineQueue()

