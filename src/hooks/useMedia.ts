import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { jellyfinService } from '@/services/jellyfin'
import { useAuthStore } from '@/stores/authStore'

// Hook: User Libraries (Views)
export function useLibraries() {
  const { user } = useAuthStore()
  const userId = user?.Id || ''

  return useQuery({
    queryKey: ['libraries', userId],
    queryFn: () => jellyfinService.getUserLibraries(userId),
    enabled: !!userId,
  })
}

// Hook: Resume / Continue Watching
export function useResumeItems(limit = 12) {
  const { user } = useAuthStore()
  const userId = user?.Id || ''

  return useQuery({
    queryKey: ['resumeItems', userId, limit],
    queryFn: () => jellyfinService.getResumeItems(userId, limit),
    enabled: !!userId,
  })
}

// Hook: Latest Items
export function useLatestItems(parentId?: string, limit = 16) {
  const { user } = useAuthStore()
  const userId = user?.Id || ''

  return useQuery({
    queryKey: ['latestItems', userId, parentId, limit],
    queryFn: () => jellyfinService.getLatestItems(userId, parentId, limit),
    enabled: !!userId,
  })
}

// Hook: Movies Library Items
export function useMovies(params: {
  sortBy?: string
  sortOrder?: 'Ascending' | 'Descending'
  limit?: number
  startIndex?: number
  genres?: string
} = {}) {
  const { user } = useAuthStore()
  const userId = user?.Id || ''

  return useQuery({
    queryKey: ['movies', userId, params],
    queryFn: () =>
      jellyfinService.getItems(userId, {
        includeItemTypes: 'Movie',
        sortBy: params.sortBy || 'DateCreated',
        sortOrder: params.sortOrder || 'Descending',
        limit: params.limit || 50,
        startIndex: params.startIndex || 0,
        genres: params.genres,
      }),
    enabled: !!userId,
  })
}

// Hook: Series Library Items
export function useSeries(params: {
  sortBy?: string
  sortOrder?: 'Ascending' | 'Descending'
  limit?: number
  startIndex?: number
  genres?: string
} = {}) {
  const { user } = useAuthStore()
  const userId = user?.Id || ''

  return useQuery({
    queryKey: ['series', userId, params],
    queryFn: () =>
      jellyfinService.getItems(userId, {
        includeItemTypes: 'Series',
        sortBy: params.sortBy || 'DateCreated',
        sortOrder: params.sortOrder || 'Descending',
        limit: params.limit || 50,
        startIndex: params.startIndex || 0,
        genres: params.genres,
      }),
    enabled: !!userId,
  })
}

// Hook: Favorite Items (Minha Lista)
export function useFavorites(limit = 80) {
  const { user } = useAuthStore()
  const userId = user?.Id || ''

  return useQuery({
    queryKey: ['favorites', userId, limit],
    queryFn: () => jellyfinService.getFavoriteItems(userId, limit),
    enabled: !!userId,
  })
}

// Hook: Toggle Favorite Mutation (Optimistic update & cache invalidation)
export function useToggleFavorite() {
  const { user } = useAuthStore()
  const userId = user?.Id || ''
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({ itemId, isFavorite }: { itemId: string; isFavorite: boolean }) => {
      if (!userId) return
      if (isFavorite) {
        return jellyfinService.unmarkFavorite(userId, itemId)
      } else {
        return jellyfinService.markFavorite(userId, itemId)
      }
    },
    onMutate: async ({ itemId, isFavorite }) => {
      // Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: ['favorites'] })

      // Optimistically update favorites cache
      queryClient.setQueriesData({ queryKey: ['favorites'] }, (old: any) => {
        if (!old || !old.Items) return old
        if (isFavorite) {
          // Remove from list immediately
          return {
            ...old,
            Items: old.Items.filter((i: any) => i.Id !== itemId),
            TotalRecordCount: Math.max(0, (old.TotalRecordCount || 1) - 1),
          }
        }
        return old
      })
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ['favorites'] })
      queryClient.invalidateQueries({ queryKey: ['itemDetails'] })
      queryClient.invalidateQueries({ queryKey: ['movies'] })
      queryClient.invalidateQueries({ queryKey: ['series'] })
      queryClient.invalidateQueries({ queryKey: ['latestItems'] })
      queryClient.invalidateQueries({ queryKey: ['resumeItems'] })
    },
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
