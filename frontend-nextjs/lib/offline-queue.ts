import { openDB, DBSchema, IDBPDatabase } from 'idb'
import { config } from './config'
import { aesGcmEncryptBase64, hmacSha256Hex, importRsaOaepPublicKey, rsaOaepEncryptBase64 } from './crypto'

interface OfflineQueueItem {
  id: string
  type: 'check_in' | 'check_out'
  imageData: string
  encryptedPayload: string
  timestamp: number
  monotonicOffset: number
  kioskCode: string
  synced: 0 | 1
  retryCount: number
}

interface AttendanceDB extends DBSchema {
  queue: {
    key: string
    value: OfflineQueueItem
    indexes: { 'by-synced': 0 | 1; 'by-timestamp': number }
  }
}

class OfflineQueue {
  private db: IDBPDatabase<AttendanceDB> | null = null
  private publicKey: string
  private cachedCryptoKey: CryptoKey | null = null

  constructor() {
    this.publicKey = (config.publicKey || '').replace(/\\n/g, '\n').trim()
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

  // Encrypt payload (envelope: RSA-OAEP-256 + AES-256-GCM)
  private async encryptPayload(data: any): Promise<string> {
    if (!this.publicKey || !globalThis.crypto?.subtle) {
      throw new Error('Offline encryption public key not configured')
    }

    if (!this.cachedCryptoKey) {
      this.cachedCryptoKey = await importRsaOaepPublicKey(this.publicKey)
    }

    const enc = new TextEncoder()
    const plaintext = enc.encode(JSON.stringify(data))
    const { keyRaw, ivB64, ctB64 } = await aesGcmEncryptBase64(plaintext)
    const ekB64 = await rsaOaepEncryptBase64(this.cachedCryptoKey, keyRaw)

    return JSON.stringify({
      alg: 'RSA-OAEP-256+A256GCM',
      ek: ekB64,
      iv: ivB64,
      ct: ctB64,
    })
  }

  // Add item to offline queue
  async addToQueue(
    type: 'check_in' | 'check_out',
    imageData: string,
    monotonicOffset: number = 0,
    kioskCode: string,
    hasConsented: boolean = true
  ): Promise<string> {
    const db = await this.init()
    
    const timestamp = Date.now()
    const item: OfflineQueueItem = {
      id: `queue-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      type,
      imageData,
      encryptedPayload: await this.encryptPayload({
        type,
        imageData,
        timestamp,
        monotonicOffset,
        has_consented: hasConsented,
      }),
      timestamp,
      monotonicOffset,
      kioskCode,
      synced: 0,
      retryCount: 0,
    }

    await db.add('queue', item)
    return item.id
  }

  // Get all unsynced items
  async getUnsyncedItems(): Promise<OfflineQueueItem[]> {
    const db = await this.init()
    const index = db.transaction('queue').store.index('by-synced')
    return index.getAll(0)
  }

  // Mark item as synced
  async markAsSynced(id: string) {
    const db = await this.init()
    const item = await db.get('queue', id)
    if (item) {
      item.synced = 1
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
      if (item.synced === 1 && item.timestamp < sevenDaysAgo) {
        await db.delete('queue', item.id)
      }
    }
  }

  // Sync queue with backend
  async syncQueue(apiUrl: string, kioskHmacSecret: string): Promise<{ success: number; failed: number }> {
    const unsynced = await this.getUnsyncedItems()
    let success = 0
    let failed = 0

    for (const item of unsynced) {
      try {
        const timestamp = Math.floor(Date.now() / 1000).toString()
        const body = JSON.stringify({ encrypted_payload: item.encryptedPayload })
        const msg = `${body}${timestamp}${item.kioskCode}`
        const signature = await hmacSha256Hex(kioskHmacSecret, msg)

        const response = await fetch(`${apiUrl}/api/v1/kiosk/offline/sync`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Kiosk-Code': item.kioskCode,
            'X-Timestamp': timestamp,
            'X-HMAC-Signature': signature,
          },
          body,
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
    const unsynced = allItems.filter(item => item.synced === 0)
    
    return {
      total: allItems.length,
      unsynced: unsynced.length,
      synced: allItems.length - unsynced.length,
    }
  }
}

export const offlineQueue = new OfflineQueue()

