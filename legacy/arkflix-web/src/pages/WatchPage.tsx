import React from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useAuthStore } from '@/stores/authStore'
import { jellyfinService } from '@/services/jellyfin'
import { VideoPlayer } from '@/components/player'
import { useQuery } from '@tanstack/react-query'

export const WatchPage: React.FC = () => {
  const { id } = useParams<{ id: string }>()
  const { user } = useAuthStore()
  const navigate = useNavigate()

  // Fetch item details
  const { data: item, isLoading: loadingItem } = useQuery({
    queryKey: ['item', id],
    queryFn: () => jellyfinService.getItemDetails(user!.Id, id!),
    enabled: !!user && !!id,
  })

  // Fetch playback info (codecs, audio/subtitle streams)
  const { data: playbackInfo } = useQuery({
    queryKey: ['playbackInfo', id],
    queryFn: () => jellyfinService.getPlaybackInfo(user!.Id, id!),
    enabled: !!user && !!id,
  })

  if (loadingItem) {
    return (
      <div className="w-screen h-screen bg-black flex items-center justify-center">
        <div className="w-12 h-12 border-4 border-netflix-red border-t-transparent rounded-full animate-spin" />
      </div>
    )
  }

  if (!item) {
    return (
      <div className="w-screen h-screen bg-black flex flex-col items-center justify-center text-white space-y-4">
        <p className="text-lg">Mídia não encontrada ou indisponível.</p>
        <button
          onClick={() => navigate('/')}
          className="text-netflix-red underline text-sm hover:text-red-400"
        >
          Voltar para o início
        </button>
      </div>
    )
  }

  return (
    <VideoPlayer
      item={item}
      playbackInfo={playbackInfo}
      onBack={() => navigate(-1)}
    />
  )
}
