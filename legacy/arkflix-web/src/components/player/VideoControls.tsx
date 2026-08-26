import React from 'react'
import { Play, Pause, Volume2, VolumeX, Maximize, Minimize, RotateCcw, RotateCw, Subtitles } from 'lucide-react'
import { Slider } from '@/components/ui'
import { MediaStream } from '@/types/jellyfin'

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
      className={`absolute inset-x-0 bottom-0 px-6 md:px-12 py-6 bg-gradient-to-t from-black/95 via-black/60 to-transparent transition-opacity duration-300 flex flex-col gap-3 z-30 select-none ${
        showControls ? 'opacity-100' : 'opacity-0 pointer-events-none'
      }`}
      onClick={(e) => e.stopPropagation()}
    >
      {/* Progress Bar Slider */}
      <Slider
        min={0}
        max={duration || 100}
        value={currentTime}
        onChange={(e) => onSeek(Number(e.target.value))}
      />

      <div className="flex items-center justify-between text-white">
        {/* Left: Play/Pause, Skip, Volume, Time */}
        <div className="flex items-center gap-4">
          <button
            onClick={onTogglePlay}
            className="p-1 hover:text-netflix-red transition-transform active:scale-90"
            title={isPlaying ? 'Pausar (Espaço)' : 'Reproduzir (Espaço)'}
          >
            {isPlaying ? <Pause className="w-7 h-7 fill-white" /> : <Play className="w-7 h-7 fill-white" />}
          </button>

          <button
            onClick={() => onSkip(-10)}
            className="p-1 hover:text-white/80 transition-transform active:scale-90"
            title="Voltar 10s"
          >
            <RotateCcw className="w-5 h-5" />
          </button>

          <button
            onClick={() => onSkip(10)}
            className="p-1 hover:text-white/80 transition-transform active:scale-90"
            title="Avançar 10s"
          >
            <RotateCw className="w-5 h-5" />
          </button>

          {/* Volume Control */}
          <div className="flex items-center gap-2 group/volume">
            <button onClick={onToggleMute} className="p-1 hover:text-white/80">
              {isMuted || volume === 0 ? <VolumeX className="w-6 h-6" /> : <Volume2 className="w-6 h-6" />}
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

          <span className="text-xs font-medium text-gray-300">
            {formatTime(currentTime)} / {formatTime(duration)}
          </span>
        </div>

        {/* Center: Title */}
        <div className="hidden lg:block text-sm font-semibold text-gray-200 truncate max-w-md">
          {title}
        </div>

        {/* Right: Subtitles/Audio Menu, Cast Launcher, Fullscreen */}
        <div className="flex items-center gap-4 relative">
          {/* Audio & Subtitles Selector */}
          {(audioStreams.length > 0 || subtitleStreams.length > 0) && (
            <div className="relative">
              <button
                onClick={() => setShowAudioMenu(!showAudioMenu)}
                className={`p-1 transition-colors ${showAudioMenu ? 'text-netflix-red' : 'hover:text-gray-300'}`}
                title="Áudio e Legendas"
              >
                <Subtitles className="w-6 h-6" />
              </button>

              {showAudioMenu && (
                <div className="absolute right-0 bottom-full mb-3 w-64 bg-netflix-dark/95 border border-white/20 rounded-lg p-4 shadow-2xl backdrop-blur-md text-xs space-y-4">
                  {/* Audio tracks */}
                  {audioStreams.length > 0 && (
                    <div>
                      <h5 className="font-bold text-gray-400 mb-2 uppercase tracking-wider">Áudio</h5>
                      <div className="space-y-1">
                        {audioStreams.map((audio) => (
                          <button
                            key={audio.Index}
                            onClick={() => {
                              onSelectAudio?.(audio.Index)
                              setShowAudioMenu(false)
                            }}
                            className={`w-full text-left px-2 py-1.5 rounded transition-colors ${
                              selectedAudioIndex === audio.Index
                                ? 'bg-netflix-red text-white font-bold'
                                : 'hover:bg-white/10 text-gray-300'
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
                      <h5 className="font-bold text-gray-400 mb-2 uppercase tracking-wider">Legendas</h5>
                      <div className="space-y-1 max-h-36 overflow-y-auto">
                        <button
                          onClick={() => {
                            onSelectSubtitle?.(-1)
                            setShowAudioMenu(false)
                          }}
                          className={`w-full text-left px-2 py-1.5 rounded transition-colors ${
                            selectedSubtitleIndex === -1
                              ? 'bg-netflix-red text-white font-bold'
                              : 'hover:bg-white/10 text-gray-300'
                          }`}
                        >
                          Desativada
                        </button>
                        {subtitleStreams.map((sub) => (
                          <button
                            key={sub.Index}
                            onClick={() => {
                              onSelectSubtitle?.(sub.Index)
                              setShowAudioMenu(false)
                            }}
                            className={`w-full text-left px-2 py-1.5 rounded transition-colors ${
                              selectedSubtitleIndex === sub.Index
                                ? 'bg-netflix-red text-white font-bold'
                                : 'hover:bg-white/10 text-gray-300'
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
          <div className="w-6 h-6 flex items-center justify-center">
            {React.createElement('google-cast-launcher', {
              class: 'w-6 h-6 cursor-pointer opacity-80 hover:opacity-100 transition-opacity',
            })}
          </div>

          {/* Fullscreen Button */}
          <button
            onClick={onToggleFullscreen}
            className="p-1 hover:text-gray-300 transition-colors"
            title={isFullscreen ? 'Sair da tela cheia (F)' : 'Tela cheia (F)'}
          >
            {isFullscreen ? <Minimize className="w-6 h-6" /> : <Maximize className="w-6 h-6" />}
          </button>
        </div>
      </div>
    </div>
  )
}
