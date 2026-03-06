import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { createJSONStorage } from 'zustand/middleware'

interface User {
  id: string
  email: string
  firstName: string
  lastName: string
  role: string
  tenantId: string
}

interface AuthState {
  user: User | null
  token: string | null
  isAuthenticated: boolean
  setUser: (user: User | null) => void
  setToken: (token: string | null) => void
  logout: () => void
}

interface ThemeState {
  theme: 'light' | 'dark'
  toggleTheme: () => void
  setTheme: (theme: 'light' | 'dark') => void
}

interface KioskState {
  kioskCode: string | null
  kioskHmacSecret: string | null
  kioskName: string | null
  organizationName: string | null
  credentialsVerifiedAt: string | null
  isOnline: boolean
  lastSyncTime: Date | null
  setKioskCode: (code: string | null) => void
  setKioskHmacSecret: (secret: string | null) => void
  setKioskConnectionMeta: (kioskName: string | null, organizationName: string | null, verifiedAt: string | null) => void
  clearKioskConnectionMeta: () => void
  setOnlineStatus: (isOnline: boolean) => void
  setLastSyncTime: (time: Date | null) => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      token: null,
      isAuthenticated: false,
      setUser: (user) => set({ user, isAuthenticated: !!user }),
      setToken: (token) => set({ token }),
      logout: () => set({ user: null, token: null, isAuthenticated: false }),
    }),
    {
      name: 'auth-storage',
      storage: createJSONStorage(() => localStorage),
    }
  )
)

export const useThemeStore = create<ThemeState>()(
  persist(
    (set) => ({
      theme: 'light',
      toggleTheme: () => set((state) => ({ theme: state.theme === 'light' ? 'dark' : 'light' })),
      setTheme: (theme) => set({ theme }),
    }),
    {
      name: 'theme-storage',
      storage: createJSONStorage(() => localStorage),
    }
  )
)

export const useKioskStore = create<KioskState>()(
  persist(
    (set) => ({
      kioskCode: null,
      kioskHmacSecret: null,
      kioskName: null,
      organizationName: null,
      credentialsVerifiedAt: null,
      isOnline: true,
      lastSyncTime: null,
      setKioskCode: (code) => set({ kioskCode: code }),
      setKioskHmacSecret: (secret) => set({ kioskHmacSecret: secret }),
      setKioskConnectionMeta: (kioskName, organizationName, verifiedAt) =>
        set({ kioskName, organizationName, credentialsVerifiedAt: verifiedAt }),
      clearKioskConnectionMeta: () =>
        set({ kioskName: null, organizationName: null, credentialsVerifiedAt: null }),
      setOnlineStatus: (isOnline) => set({ isOnline }),
      setLastSyncTime: (time) => set({ lastSyncTime: time }),
    }),
    {
      name: 'kiosk-storage',
      storage: createJSONStorage(() => localStorage),
    }
  )
)
