import axios from 'axios'

export const DEFAULT_SERVER_URL = import.meta.env.VITE_JELLYFIN_SERVER_URL || ''
export const CLIENT_NAME = import.meta.env.VITE_APP_CLIENT_NAME || 'Arkflix'
export const CLIENT_VERSION = import.meta.env.VITE_APP_VERSION || '1.0.0'
export const DEVICE_ID = 'arkflix-web-client-v1'
export const DEVICE_NAME = 'Web Browser'

// Helper to get active server URL from localStorage
export function getServerUrl(): string {
  return localStorage.getItem('arkflix_server_url') || DEFAULT_SERVER_URL
}

// Helper to set active server URL
export function setServerUrl(url: string) {
  let cleanUrl = url.trim().replace(/\/+$/, '')
  if (cleanUrl && !cleanUrl.startsWith('http://') && !cleanUrl.startsWith('https://')) {
    cleanUrl = `https://${cleanUrl}`
  }
  if (cleanUrl) {
    localStorage.setItem('arkflix_server_url', cleanUrl)
    apiClient.defaults.baseURL = cleanUrl
  } else {
    localStorage.removeItem('arkflix_server_url')
    apiClient.defaults.baseURL = ''
  }
}

export const apiClient = axios.create({
  baseURL: getServerUrl(),
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor to add authorization headers and ensure dynamic baseURL
apiClient.interceptors.request.use((config) => {
  const activeUrl = getServerUrl()
  if (activeUrl) {
    config.baseURL = activeUrl
  }
  const token = localStorage.getItem('arkflix_token')
  const authHeader = `MediaBrowser Client="${CLIENT_NAME}", Device="${DEVICE_NAME}", DeviceId="${DEVICE_ID}", Version="${CLIENT_VERSION}"`

  config.headers['X-Emby-Authorization'] = authHeader

  if (token) {
    config.headers['X-Emby-Token'] = token
  }

  return config
})

// Helper to construct user avatar URL
export function getUserAvatarUrl(userId: string, tag?: string): string {
  const base = getServerUrl()
  if (!base) return ''
  const params = new URLSearchParams()
  params.append('fillWidth', '200')
  params.append('quality', '85')
  if (tag) params.append('tag', tag)
  return `${base}/Users/${userId}/Images/Primary?${params.toString()}`
}

// Helper to construct image URLs with dynamic server url
export function getImageUrl(
  itemId: string,
  type: 'Primary' | 'Backdrop' | 'Logo' | 'Thumb' = 'Primary',
  options: { fillWidth?: number; fillHeight?: number; quality?: number; tag?: string } = {}
): string {
  const base = getServerUrl()
  if (!base) return ''
  const params = new URLSearchParams()
  if (options.fillWidth) params.append('fillWidth', options.fillWidth.toString())
  if (options.fillHeight) params.append('fillHeight', options.fillHeight.toString())
  if (options.quality) params.append('quality', options.quality.toString())
  if (options.tag) params.append('tag', options.tag)

  const queryString = params.toString() ? `?${params.toString()}` : ''
  return `${base}/Items/${itemId}/Images/${type}${queryString}`
}
