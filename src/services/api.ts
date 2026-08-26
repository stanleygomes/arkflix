import axios from 'axios'

export const JELLYFIN_SERVER_URL = import.meta.env.VITE_JELLYFIN_SERVER_URL || 'https://ark-flix.duckdns.org'
export const CLIENT_NAME = import.meta.env.VITE_APP_CLIENT_NAME || 'Arkflix'
export const CLIENT_VERSION = import.meta.env.VITE_APP_VERSION || '1.0.0'
export const DEVICE_ID = 'arkflix-web-client-v1'
export const DEVICE_NAME = 'Web Browser'

export const apiClient = axios.create({
  baseURL: JELLYFIN_SERVER_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor to add authorization headers
apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('arkflix_token')
  const authHeader = `MediaBrowser Client="${CLIENT_NAME}", Device="${DEVICE_NAME}", DeviceId="${DEVICE_ID}", Version="${CLIENT_VERSION}"`

  config.headers['X-Emby-Authorization'] = authHeader

  if (token) {
    config.headers['X-Emby-Token'] = token
  }

  return config
})

// Helper to construct image URLs
export function getImageUrl(
  itemId: string,
  type: 'Primary' | 'Backdrop' | 'Logo' | 'Thumb' = 'Primary',
  options: { fillWidth?: number; fillHeight?: number; quality?: number; tag?: string } = {}
): string {
  const params = new URLSearchParams()
  if (options.fillWidth) params.append('fillWidth', options.fillWidth.toString())
  if (options.fillHeight) params.append('fillHeight', options.fillHeight.toString())
  if (options.quality) params.append('quality', options.quality.toString())
  if (options.tag) params.append('tag', options.tag)

  const queryString = params.toString() ? `?${params.toString()}` : ''
  return `${JELLYFIN_SERVER_URL}/Items/${itemId}/Images/${type}${queryString}`
}
