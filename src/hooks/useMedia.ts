import { useQuery } from '@tanstack/react-query'
import { jellyfinService } from '@/services/jellyfin'
import { useAuthStore } from '@/stores/authStore'

// Hook: User libraries / views
export function useUserLibraries() {
  const { user } = useAuthStore()
  const userId = user?.Id || ''

  return useQuery({
    queryKey: ['libraries', userId],
    queryFn: () => jellyfinService.getUserLibraries(userId),
    enabled: !!userId,
  })
}

// Hook: Continue Watching (Resume Items)
export function useResumeItems(limit = 12) {
  const { user } = useAuthStore()
  const userId = user?.Id || ''

  return useQuery({
    queryKey: ['resumeItems', userId, limit],
    queryFn: () => jellyfinService.getResumeItems(userId, limit),
    enabled: !!userId,
  })
}

// Hook: Latest items
export function useLatestItems(parentId?: string, limit = 16) {
  const { user } = useAuthStore()
  const userId = user?.Id || ''

  return useQuery({
    queryKey: ['latestItems', userId, parentId, limit],
    queryFn: () => jellyfinService.getLatestItems(userId, parentId, limit),
    enabled: !!userId,
  })
}

// Hook: Movies with pagination and filtering
export function useMovies(params: { limit?: number; sortBy?: string; sortOrder?: 'Ascending' | 'Descending' } = {}) {
  const { user } = useAuthStore()
  const userId = user?.Id || ''

  return useQuery({
    queryKey: ['movies', userId, params],
    queryFn: () =>
      jellyfinService.getItems(userId, {
        includeItemTypes: 'Movie',
        sortBy: params.sortBy || 'DateCreated',
        sortOrder: params.sortOrder || 'Descending',
        limit: params.limit || 16,
      }),
    enabled: !!userId,
  })
}

// Hook: Series with pagination and filtering
export function useSeries(params: { limit?: number; sortBy?: string; sortOrder?: 'Ascending' | 'Descending' } = {}) {
  const { user } = useAuthStore()
  const userId = user?.Id || ''

  return useQuery({
    queryKey: ['series', userId, params],
    queryFn: () =>
      jellyfinService.getItems(userId, {
        includeItemTypes: 'Series',
        sortBy: params.sortBy || 'DateCreated',
        sortOrder: params.sortOrder || 'Descending',
        limit: params.limit || 16,
      }),
    enabled: !!userId,
  })
}

// Hook: Item Details
export function useItemDetails(itemId?: string) {
  const { user } = useAuthStore()
  const userId = user?.Id || ''

  return useQuery({
    queryKey: ['itemDetails', userId, itemId],
    queryFn: () => jellyfinService.getItemDetails(userId, itemId!),
    enabled: !!userId && !!itemId,
  })
}

// Hook: Series Seasons
export function useSeasons(seriesId?: string, enabled = true) {
  const { user } = useAuthStore()
  const userId = user?.Id || ''

  return useQuery({
    queryKey: ['seasons', userId, seriesId],
    queryFn: () => jellyfinService.getSeasons(userId, seriesId!),
    enabled: !!userId && !!seriesId && enabled,
  })
}

// Hook: Season Episodes
export function useEpisodes(seriesId?: string, seasonId?: string, enabled = true) {
  const { user } = useAuthStore()
  const userId = user?.Id || ''

  return useQuery({
    queryKey: ['episodes', userId, seriesId, seasonId],
    queryFn: () => jellyfinService.getEpisodes(userId, seriesId!, seasonId!),
    enabled: !!userId && !!seriesId && !!seasonId && enabled,
  })
}

// Hook: Item Playback Info
export function usePlaybackInfo(itemId?: string) {
  const { user } = useAuthStore()
  const userId = user?.Id || ''

  return useQuery({
    queryKey: ['playbackInfo', userId, itemId],
    queryFn: () => jellyfinService.getPlaybackInfo(userId, itemId!),
    enabled: !!userId && !!itemId,
  })
}

// Hook: Search Media Items
export function useSearchMedia(searchTerm: string) {
  const { user } = useAuthStore()
  const userId = user?.Id || ''

  return useQuery({
    queryKey: ['searchMedia', userId, searchTerm],
    queryFn: () =>
      jellyfinService.getItems(userId, {
        searchTerm,
        includeItemTypes: 'Movie,Series,Episode',
        limit: 24,
      }),
    enabled: !!userId && searchTerm.trim().length > 1,
  })
}
