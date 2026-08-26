import { apiClient, getServerUrl } from './api'
import { AuthenticationResult, MediaItem, MediaLibrary, PlaybackInfo } from '@/types/jellyfin'

export const jellyfinService = {
  // Authentication
  async authenticate(username: string, pw: string): Promise<AuthenticationResult> {
    const response = await apiClient.post<AuthenticationResult>('/Users/AuthenticateByName', {
      Username: username,
      Pw: pw,
    })
    return response.data
  },

  // User Views / Libraries
  async getUserLibraries(userId: string): Promise<MediaLibrary[]> {
    const response = await apiClient.get<{ Items: MediaLibrary[] }>(`/Users/${userId}/Views`)
    return response.data.Items
  },

  // Resume / Continue Watching
  async getResumeItems(userId: string, limit = 12): Promise<MediaItem[]> {
    const response = await apiClient.get<{ Items: MediaItem[] }>(`/User/${userId}/Items/Resume`, {
      params: { Limit: limit },
    })
    return response.data.Items
  },

  // Latest Added Items
  async getLatestItems(userId: string, parentId?: string, limit = 16): Promise<MediaItem[]> {
    const response = await apiClient.get<MediaItem[]>(`/Users/${userId}/Items/Latest`, {
      params: {
        ParentId: parentId,
        Limit: limit,
      },
    })
    return response.data
  },

  // Favorite Items (Minha Lista)
  async getFavoriteItems(userId: string, limit = 40): Promise<{ Items: MediaItem[]; TotalRecordCount: number }> {
    const response = await apiClient.get<{ Items: MediaItem[]; TotalRecordCount: number }>(
      `/Users/${userId}/Items`,
      {
        params: {
          Recursive: true,
          Filters: 'IsFavorite',
          SortBy: 'DateCreated',
          SortOrder: 'Descending',
          Limit: limit,
        },
      }
    )
    return response.data
  },

  // Mark Item as Favorite
  async markFavorite(userId: string, itemId: string): Promise<any> {
    const response = await apiClient.post(`/User/${userId}/FavoriteItems/${itemId}`)
    return response.data
  },

  // Unmark Item as Favorite
  async unmarkFavorite(userId: string, itemId: string): Promise<any> {
    const response = await apiClient.delete(`/User/${userId}/FavoriteItems/${itemId}`)
    return response.data
  },

  // Items from Library (Movies, Series)
  async getItems(userId: string, params: {
    parentId?: string
    includeItemTypes?: string
    genres?: string
    sortBy?: string
    sortOrder?: 'Ascending' | 'Descending'
    limit?: number
    startIndex?: number
    searchTerm?: string
  }): Promise<{ Items: MediaItem[]; TotalRecordCount: number }> {
    const response = await apiClient.get<{ Items: MediaItem[]; TotalRecordCount: number }>(
      `/Users/${userId}/Items`,
      {
        params: {
          Recursive: true,
          ...params,
        },
      }
    )
    return response.data
  },

  // Item Details
  async getItemDetails(userId: string, itemId: string): Promise<MediaItem> {
    const response = await apiClient.get<MediaItem>(`/Items/${itemId}`, {
      params: { UserId: userId },
    })
    return response.data
  },

  // Seasons for a Series
  async getSeasons(userId: string, seriesId: string): Promise<MediaItem[]> {
    const response = await apiClient.get<{ Items: MediaItem[] }>(`/Shows/${seriesId}/Seasons`, {
      params: { UserId: userId },
    })
    return response.data.Items
  },

  // Episodes for a Season
  async getEpisodes(userId: string, seriesId: string, seasonId: string): Promise<MediaItem[]> {
    const response = await apiClient.get<{ Items: MediaItem[] }>(`/Shows/${seriesId}/Episodes`, {
      params: {
        UserId: userId,
        SeasonId: seasonId,
      },
    })
    return response.data.Items
  },

  // Playback Info
  async getPlaybackInfo(userId: string, itemId: string): Promise<PlaybackInfo> {
    const response = await apiClient.post<PlaybackInfo>(`/Items/${itemId}/PlaybackInfo`, null, {
      params: { UserId: userId },
    })
    return response.data
  },

  // Direct Stream URL
  getStreamUrl(itemId: string, token: string): string {
    return `${getServerUrl()}/Videos/${itemId}/stream.mp4?Static=true&MediaSourceId=${itemId}&api_key=${token}`
  },

  // Progress Reporting
  async reportPlaybackStart(itemId: string, playSessionId: string, positionTicks = 0) {
    return apiClient.post('/Sessions/Playing', {
      ItemId: itemId,
      PlaySessionId: playSessionId,
      PositionTicks: positionTicks,
      IsPaused: false,
    })
  },

  async reportPlaybackProgress(itemId: string, playSessionId: string, positionTicks: number, isPaused = false) {
    return apiClient.post('/Sessions/Playing/Progress', {
      ItemId: itemId,
      PlaySessionId: playSessionId,
      PositionTicks: positionTicks,
      IsPaused: isPaused,
    })
  },

  async reportPlaybackStopped(itemId: string, playSessionId: string, positionTicks: number) {
    return apiClient.post('/Sessions/Playing/Stopped', {
      ItemId: itemId,
      PlaySessionId: playSessionId,
      PositionTicks: positionTicks,
    })
  },
}
