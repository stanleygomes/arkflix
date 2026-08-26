import React from 'react'
import { Play, Pause, Volume2, VolumeX, Maximize, Minimize, RotateCcw, RotateCw, Subtitles } from 'lucide-react'
import { Slider } from '@/components/ui'
import { MediaStream } from '@/types/jellyfin'
import { useTranslation } from '@/hooks'

interface VideoControlsProps {
  isPlaying: boolean
  isMuted: boolean
  volume: number
  currentTime: number
  duration: number
  showControls: boolean
  title: string
  audioStreams?: MediaStream[]
  subtitleStreams?: MediaStream[]
  selectedAudioIndex?: number
  selectedSubtitleIndex?: number
  onTogglePlay: () => void
  onToggleMute: () => void
  onVolumeChange: (vol: number) => void
  onSeek: (seconds: number) => void
  onSkip: (seconds: number) => void
  onToggleFullscreen: () => void
  isFullscreen: boolean
  onSelectAudio?: (index: number) => void
  onSelectSubtitle?: (index: number) => void
}

export const VideoControls: React.FC<VideoControlsProps> = ({
  isPlaying,
  isMuted,
  volume,
  currentTime,
  duration,
  showControls,
  title,
  audioStreams = [],
  subtitleStreams = [],
  selectedAudioIndex,
  selectedSubtitleIndex,
  onTogglePlay,
  onToggleMute,
  onVolumeChange,
  onSeek,
  onSkip,
  onToggleFullscreen,
  isFullscreen,
  onSelectAudio,
  onSelectSubtitle,
}) => {
  const { t } = useTranslation()
  const [showAudioMenu, setShowAudioMenu] = React.useState(false)

  const formatTime = (seconds: number) => {
    if (isNaN(seconds)) return '00:00'
    const hrs = Math.floor(seconds / 3600)
    const mins = Math.floor((seconds % 3600) / 60)
    const secs = Math.floor(seconds % 60)
    if (hrs > 0) {
      return `${hrs}:${mins < 10 ? '0' : ''}${mins}:${secs < 10 ? '0' : ''}${secs}`
    }
    return `${mins}:${secs < 10 ? '0' : ''}${secs}`
  }

  return (
    <div
      className={`absolute inset-x-0 bottom-0 px-6 md:px-14 py-8 bg-gradient-to-t from-black/95 via-black/40 to-transparent transition-all duration-300 flex flex-col gap-4 z-30 select-none ${
        showControls ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-4 pointer-events-none'
      }`}
      onClick={(e) => e.stopPropagation()}
    >
      {/* Apple Minimalist Progress Slider */}
      <Slider
        min={0}
        max={duration || 100}
        value={currentTime}
        onChange={(e) => onSeek(Number(e.target.value))}
      />

      <div className="flex items-center justify-between text-white">
        {/* Left: Play/Pause, Skip 10s, Volume, Time */}
        <div className="flex items-center gap-5">
          <button
            onClick={onTogglePlay}
            className="w-10 h-10 rounded-full bg-white text-black flex items-center justify-center hover:bg-white/90 active:scale-90 transition-all shadow-md"
            title={isPlaying ? t.player.pauseTooltip : t.player.playTooltip}
          >
            {isPlaying ? (
              <Pause className="w-5 h-5 fill-black text-black" />
            ) : (
              <Play className="w-5 h-5 fill-black text-black ml-0.5" />
            )}
          </button>

          <button
            onClick={() => onSkip(-10)}
            className="p-2 rounded-full text-white/80 hover:text-white hover:bg-white/10 active:scale-95 transition-all"
            title={t.player.rewindTooltip}
          >
            <RotateCcw className="w-4 h-4" />
          </button>

          <button
            onClick={() => onSkip(10)}
            className="p-2 rounded-full text-white/80 hover:text-white hover:bg-white/10 active:scale-95 transition-all"
            title={t.player.forwardTooltip}
          >
            <RotateCw className="w-4 h-4" />
          </button>

          {/* Volume Control */}
          <div className="flex items-center gap-2 group/volume">
            <button
              onClick={onToggleMute}
              className="p-2 rounded-full text-white/80 hover:text-white hover:bg-white/10 transition-colors"
            >
              {isMuted || volume === 0 ? <VolumeX className="w-5 h-5" /> : <Volume2 className="w-5 h-5" />}
            </button>
            <div className="w-0 overflow-hidden group-hover/volume:w-20 transition-all duration-300">
              <Slider
                min={0}
                max={1}
                step={0.05}
                value={isMuted ? 0 : volume}
                onChange={(e) => onVolumeChange(Number(e.target.value))}
              />
            </div>
          </div>

          <span className="text-xs font-medium text-apple-subtext">
            {formatTime(currentTime)} / {formatTime(duration)}
          </span>
        </div>

        {/* Center: Title */}
        <div className="hidden lg:block text-xs font-semibold text-white/90 tracking-tight truncate max-w-md">
          {title}
        </div>

        {/* Right: Subtitles/Audio Menu, Cast Launcher, Fullscreen */}
        <div className="flex items-center gap-3 relative">
          {/* Audio & Subtitles Selector */}
          {(audioStreams.length > 0 || subtitleStreams.length > 0) && (
            <div className="relative">
              <button
                onClick={() => setShowAudioMenu(!showAudioMenu)}
                className={`p-2 rounded-full transition-all ${
                  showAudioMenu
                    ? 'bg-white/20 text-white'
                    : 'text-white/80 hover:text-white hover:bg-white/10'
                }`}
                title={t.player.audioAndSubs}
              >
                <Subtitles className="w-5 h-5" />
              </button>

              {showAudioMenu && (
                <div className="absolute right-0 bottom-full mb-3 w-64 glass-panel rounded-squircle-lg p-4 shadow-apple backdrop-blur-2xl text-xs space-y-4 animate-fadeIn">
                  {/* Audio tracks */}
                  {audioStreams.length > 0 && (
                    <div>
                      <h5 className="font-semibold text-apple-subtext mb-2 text-[11px]">{t.player.audio}</h5>
                      <div className="space-y-1">
                        {audioStreams.map((audio) => (
                          <button
                            key={audio.Index}
                            onClick={() => {
                              onSelectAudio?.(audio.Index)
                              setShowAudioMenu(false)
                            }}
                            className={`w-full text-left px-2.5 py-1.5 rounded-squircle-sm transition-all ${
                              selectedAudioIndex === audio.Index
                                ? 'bg-white text-black font-semibold shadow-sm'
                                : 'hover:bg-white/10 text-white/80'
                            }`}
                          >
                            {audio.DisplayTitle || audio.Language || `Faixa ${audio.Index}`}
                          </button>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Subtitle tracks */}
                  {subtitleStreams.length > 0 && (
                    <div>
                      <h5 className="font-semibold text-apple-subtext mb-2 text-[11px]">{t.player.subtitles}</h5>
                      <div className="space-y-1 max-h-36 overflow-y-auto">
                        <button
                          onClick={() => {
                            onSelectSubtitle?.(-1)
                            setShowAudioMenu(false)
                          }}
                          className={`w-full text-left px-2.5 py-1.5 rounded-squircle-sm transition-all ${
                            selectedSubtitleIndex === -1
                              ? 'bg-white text-black font-semibold shadow-sm'
                              : 'hover:bg-white/10 text-white/80'
                          }`}
                        >
                          {t.player.subtitleDisabled}
                        </button>
                        {subtitleStreams.map((sub) => (
                          <button
                            key={sub.Index}
                            onClick={() => {
                              onSelectSubtitle?.(sub.Index)
                              setShowAudioMenu(false)
                            }}
                            className={`w-full text-left px-2.5 py-1.5 rounded-squircle-sm transition-all ${
                              selectedSubtitleIndex === sub.Index
                                ? 'bg-white text-black font-semibold shadow-sm'
                                : 'hover:bg-white/10 text-white/80'
                            }`}
                          >
                            {sub.DisplayTitle || sub.Language || `Legenda ${sub.Index}`}
                          </button>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
          )}

          {/* Google Cast Launcher */}
          <div className="w-8 h-8 flex items-center justify-center rounded-full hover:bg-white/10 transition-colors">
            {React.createElement('google-cast-launcher', {
              class: 'w-4 h-4 cursor-pointer opacity-70 hover:opacity-100 transition-opacity',
            })}
          </div>

          {/* Fullscreen Button */}
          <button
            onClick={onToggleFullscreen}
            className="p-2 rounded-full text-white/80 hover:text-white hover:bg-white/10 transition-all"
            title={isFullscreen ? t.player.exitFullscreen : t.player.fullscreen}
          >
            {isFullscreen ? <Minimize className="w-5 h-5" /> : <Maximize className="w-5 h-5" />}
          </button>
        </div>
      </div>
    </div>
  )
}
