import React from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { VideoPlayer } from '@/components/player'
import { useItemDetails, usePlaybackInfo, useTranslation } from '@/hooks'
import { AppleSpinner } from '@/components/ui'

export const WatchPage: React.FC = () => {
  const { id } = useParams<{ id: string }>()
  const { t } = useTranslation()
  const navigate = useNavigate()

  // Chamadas desacopladas via Hooks
  const { data: item, isLoading: loadingItem } = useItemDetails(id)
  const { data: playbackInfo } = usePlaybackInfo(id)

  if (loadingItem) {
    return (
      <div className="w-screen h-screen bg-black flex items-center justify-center">
        <AppleSpinner size="lg" color="white" />
      </div>
    )
  }

  if (!item) {
    return (
      <div className="w-screen h-screen bg-black flex flex-col items-center justify-center text-white space-y-4">
        <p className="text-sm text-apple-subtext">{t.common.unavailable}</p>
        <button
          onClick={() => navigate('/')}
          className="text-apple-accent hover:underline text-xs"
        >
          {t.common.backToHome}
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
