export const config = {
  apiUrl: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080',
  aiServiceUrl: process.env.NEXT_PUBLIC_AI_SERVICE_URL || 'http://localhost:8000',
  enableOfflineMode: process.env.NEXT_PUBLIC_ENABLE_OFFLINE_MODE === 'true',
  publicKey: process.env.NEXT_PUBLIC_ENCRYPTION_PUBLIC_KEY || '',
}

