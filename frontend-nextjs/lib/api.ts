import axios from 'axios'
import { config } from './config'
import { useAuthStore } from '@/store/useStore'

const api = axios.create({
  baseURL: config.apiUrl,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Add auth token to requests
api.interceptors.request.use((request) => {
  const token = useAuthStore.getState().token
  if (token) {
    request.headers.Authorization = `Bearer ${token}`
  }
  return request
})

// Handle auth errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      useAuthStore.getState().logout()
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export default api

