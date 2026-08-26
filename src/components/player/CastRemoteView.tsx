import React from 'react'
import { Slider, Button } from '@/components/ui'
import { MediaItem } from '@/types/jellyfin'
import { getImageUrl } from '@/services/api'
import { useTranslation } from '@/hooks'
import { Play, Pause, RotateCcw, RotateCw, LogOut } from 'lucide-react'

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
  const { t } = useTranslation()
  const backdropUrl = item ? getImageUrl(item.Id, 'Backdrop', { fillWidth: 1920, quality: 75 }) : ''

  const formatTime = (seconds: number) => {
    if (isNaN(seconds)) return '00:00'
    const mins = Math.floor(seconds / 60)
    const secs = Math.floor(seconds % 60)
    return `${mins}:${secs < 10 ? '0' : ''}${secs}`
  }

  return (
    <div className="relative w-full h-full flex flex-col items-center justify-between p-8 md:p-14 bg-apple-bg text-white select-none overflow-hidden">
      {/* Background Ambient Blur */}
      {backdropUrl && (
        <div
          className="absolute inset-0 bg-cover bg-center opacity-30 filter blur-2xl scale-110 pointer-events-none"
          style={{ backgroundImage: `url(${backdropUrl})` }}
        />
      )}
      <div className="absolute inset-0 bg-black/65 pointer-events-none" />

      {/* Top Header: Cast Status Badge */}
      <div className="relative z-10 flex items-center justify-between w-full max-w-2xl">
        <div className="flex items-center gap-3 bg-white/10 backdrop-blur-xl px-4 py-2 rounded-full border border-white/10">
          <div className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
          <div>
            <h4 className="text-xs font-semibold text-white">{t.player.castingAirplay}</h4>
            <p className="text-[10px] text-apple-subtext">{deviceName || t.player.castDevice}</p>
          </div>
        </div>

        <Button
          variant="glass"
          size="sm"
          onClick={onDisconnect}
          className="text-xs text-red-300 hover:text-red-200"
        >
          <LogOut className="w-3.5 h-3.5 mr-1" /> {t.common.disconnect}
        </Button>
      </div>

      {/* Center Poster & Title */}
      <div className="relative z-10 flex flex-col items-center text-center space-y-4 my-auto">
        {item && (
          <img
            src={getImageUrl(item.Id, 'Primary', { fillWidth: 300, quality: 85 })}
            alt={item.Name}
            className="w-40 md:w-52 aspect-[2/3] object-cover rounded-squircle-lg shadow-apple border border-white/15"
          />
        )}
        <div className="space-y-1">
          <h2 className="text-2xl md:text-3xl font-extrabold text-white tracking-tight">{item?.Name || t.player.playing}</h2>
          {item?.SeriesName && <p className="text-xs text-apple-subtext">{item.SeriesName}</p>}
        </div>
      </div>

      {/* Bottom Apple Glass Control Card */}
      <div className="relative z-10 w-full max-w-2xl space-y-4 glass-panel p-6 rounded-squircle-xl border border-white/15 shadow-apple">
        <Slider
          min={0}
          max={duration || 100}
          value={currentTime}
          onChange={(e) => onSeek(Number(e.target.value))}
        />

        <div className="flex items-center justify-between text-[11px] text-apple-subtext font-medium">
          <span>{formatTime(currentTime)}</span>
          <span>{formatTime(duration)}</span>
        </div>

        <div className="flex items-center justify-center gap-8 pt-1">
          <button
            onClick={() => onSkip(-10)}
            className="p-2.5 text-white/80 hover:text-white hover:bg-white/10 rounded-full active:scale-95 transition-all"
            title={t.player.rewindTooltip}
          >
            <RotateCcw className="w-5 h-5" />
          </button>

          <button
            onClick={onTogglePlay}
            className="w-14 h-14 rounded-full bg-white text-black flex items-center justify-center hover:bg-white/90 active:scale-95 transition-all shadow-md"
          >
            {isPlaying ? (
              <Pause className="w-6 h-6 fill-black text-black" />
            ) : (
              <Play className="w-6 h-6 fill-black text-black ml-0.5" />
            )}
          </button>

          <button
            onClick={() => onSkip(10)}
            className="p-2.5 text-white/80 hover:text-white hover:bg-white/10 rounded-full active:scale-95 transition-all"
            title={t.player.forwardTooltip}
          >
            <RotateCw className="w-5 h-5" />
          </button>
        </div>
      </div>
    </div>
  )
}
