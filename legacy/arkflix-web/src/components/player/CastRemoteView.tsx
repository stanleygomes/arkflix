import React from 'react'
import { Cast, Play, Pause, RotateCcw, RotateCw, LogOut } from 'lucide-react'
import { Slider, Button } from '@/components/ui'
import { MediaItem } from '@/types/jellyfin'
import { getImageUrl } from '@/services/api'

interface CastRemoteViewProps {
  item?: MediaItem
  deviceName: string | null
  isPlaying: boolean
  currentTime: number
  duration: number
  onTogglePlay: () => void
  onSeek: (seconds: number) => void
  onSkip: (seconds: number) => void
  onDisconnect: () => void
}

export const CastRemoteView: React.FC<CastRemoteViewProps> = ({
  item,
  deviceName,
  isPlaying,
  currentTime,
  duration,
  onTogglePlay,
  onSeek,
  onSkip,
  onDisconnect,
}) => {
  const backdropUrl = item ? getImageUrl(item.Id, 'Backdrop', { fillWidth: 1920, quality: 75 }) : ''

  const formatTime = (seconds: number) => {
    if (isNaN(seconds)) return '00:00'
    const mins = Math.floor(seconds / 60)
    const secs = Math.floor(seconds % 60)
    return `${mins}:${secs < 10 ? '0' : ''}${secs}`
  }

  return (
    <div className="relative w-full h-full flex flex-col items-center justify-between p-8 md:p-16 bg-netflix-black text-white select-none">
      {/* Background with blur */}
      {backdropUrl && (
        <div
          className="absolute inset-0 bg-cover bg-center opacity-20 filter blur-xl scale-105"
          style={{ backgroundImage: `url(${backdropUrl})` }}
        />
      )}
      <div className="absolute inset-0 bg-black/60" />

      {/* Top Header: Cast Status */}
      <div className="relative z-10 flex items-center justify-between w-full max-w-3xl">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-full bg-netflix-red/20 text-netflix-red flex items-center justify-center animate-pulse">
            <Cast className="w-5 h-5" />
          </div>
          <div>
            <h4 className="text-sm font-semibold text-white">Transmitindo no Chromecast</h4>
            <p className="text-xs text-netflix-gray">{deviceName || 'Dispositivo Google Cast'}</p>
          </div>
        </div>

        <Button
          variant="secondary"
          size="sm"
          onClick={onDisconnect}
          className="text-xs border border-white/20"
        >
          <LogOut className="w-4 h-4 mr-1" /> Desconectar
        </Button>
      </div>

      {/* Center Poster & Title */}
      <div className="relative z-10 flex flex-col items-center text-center space-y-4 my-auto">
        {item && (
          <img
            src={getImageUrl(item.Id, 'Primary', { fillWidth: 300, quality: 85 })}
            alt={item.Name}
            className="w-40 md:w-56 aspect-[2/3] object-cover rounded-lg shadow-2xl border border-white/10"
          />
        )}
        <h2 className="text-xl md:text-3xl font-black text-white">{item?.Name || 'Mídia em Reprodução'}</h2>
        {item?.SeriesName && <p className="text-sm text-netflix-gray">{item.SeriesName}</p>}
      </div>

      {/* Bottom Controls */}
      <div className="relative z-10 w-full max-w-3xl space-y-4 bg-netflix-dark/80 p-6 rounded-xl border border-white/10 backdrop-blur-md">
        <Slider
          min={0}
          max={duration || 100}
          value={currentTime}
          onChange={(e) => onSeek(Number(e.target.value))}
        />

        <div className="flex items-center justify-between text-xs text-gray-400">
          <span>{formatTime(currentTime)}</span>
          <span>{formatTime(duration)}</span>
        </div>

        <div className="flex items-center justify-center gap-6 pt-2">
          <button
            onClick={() => onSkip(-10)}
            className="p-2 text-white/80 hover:text-white transition-colors"
            title="Voltar 10s"
          >
            <RotateCcw className="w-6 h-6" />
          </button>

          <button
            onClick={onTogglePlay}
            className="w-14 h-14 rounded-full bg-netflix-red text-white flex items-center justify-center hover:bg-red-700 transition-transform active:scale-95 shadow-lg"
          >
            {isPlaying ? <Pause className="w-6 h-6 fill-white" /> : <Play className="w-6 h-6 fill-white ml-0.5" />}
          </button>

          <button
            onClick={() => onSkip(10)}
            className="p-2 text-white/80 hover:text-white transition-colors"
            title="Avançar 10s"
          >
            <RotateCw className="w-6 h-6" />
          </button>
        </div>
      </div>
    </div>
  )
}
