import React from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { VideoPlayer } from '@/components/player'
import { useItemDetails, usePlaybackInfo } from '@/hooks'

export const WatchPage: React.FC = () => {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  // Chamadas desacopladas via Hooks
  const { data: item, isLoading: loadingItem } = useItemDetails(id)
  const { data: playbackInfo } = usePlaybackInfo(id)

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
