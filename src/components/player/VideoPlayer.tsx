import React, { useEffect, useRef, useState, useCallback } from 'react'
import { ArrowLeft } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import Hls from 'hls.js'
import { MediaItem, PlaybackInfo } from '@/types/jellyfin'
import { jellyfinService } from '@/services/jellyfin'
import { useAuthStore } from '@/stores/authStore'
import { useChromecast } from '@/hooks/useChromecast'
import { VideoControls } from './VideoControls'
import { CastRemoteView } from './CastRemoteView'

interface VideoPlayerProps {
  item: MediaItem
  playbackInfo?: PlaybackInfo
  onBack?: () => void
}

export const VideoPlayer: React.FC<VideoPlayerProps> = ({ item, playbackInfo, onBack }) => {
  const { token, user } = useAuthStore()
  const navigate = useNavigate()

  const videoRef = useRef<HTMLVideoElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const hlsRef = useRef<Hls | null>(null)

  const [isPlaying, setIsPlaying] = useState(true)
  const [isMuted, setIsMuted] = useState(false)
  const [volume, setVolume] = useState(1)
  const [currentTime, setCurrentTime] = useState(0)
  const [duration, setDuration] = useState(0)
  const [showControls, setShowControls] = useState(true)
  const [isFullscreen, setIsFullscreen] = useState(false)
  const [selectedAudioIndex, setSelectedAudioIndex] = useState<number | undefined>()
  const [selectedSubtitleIndex, setSelectedSubtitleIndex] = useState<number | undefined>(-1)

  // Chromecast integration
  const chromecast = useChromecast()

  // Streams list
  const mediaSource = playbackInfo?.MediaSources?.[0]
  const audioStreams = mediaSource?.MediaStreams?.filter((s) => s.Type === 'Audio') || []
  const subtitleStreams = mediaSource?.MediaStreams?.filter((s) => s.Type === 'Subtitle') || []

  const streamUrl = token ? jellyfinService.getStreamUrl(item.Id, token) : ''

  // Initialize HLS / HTML5 Video
  useEffect(() => {
    const video = videoRef.current
    if (!video || !streamUrl || chromecast.isConnected) return

    if (Hls.isSupported() && streamUrl.endsWith('.m3u8')) {
      const hls = new Hls({
        enableWorker: true,
        lowLatencyMode: true,
      })
      hls.loadSource(streamUrl)
      hls.attachMedia(video)
      hls.on(Hls.Events.MANIFEST_PARSED, () => {
        video.play().catch(() => setIsPlaying(false))
      })
      hlsRef.current = hls

      return () => {
        hls.destroy()
        hlsRef.current = null
      }
    } else {
      video.src = streamUrl
      video.play().catch(() => setIsPlaying(false))
    }
  }, [streamUrl, chromecast.isConnected])

  // Cast media automatically if already connected to Chromecast
  useEffect(() => {
    if (chromecast.isConnected && item) {
      chromecast.castMedia(item, currentTime)
      if (videoRef.current) {
        videoRef.current.pause()
      }
    }
  }, [chromecast.isConnected, item])

  // Auto hide controls after inactivity
  useEffect(() => {
    let timeout: NodeJS.Timeout
    const handleMouseMove = () => {
      setShowControls(true)
      clearTimeout(timeout)
      timeout = setTimeout(() => {
        if (isPlaying && !chromecast.isConnected) setShowControls(false)
      }, 3500)
    }

    const container = containerRef.current
    if (container) {
      container.addEventListener('mousemove', handleMouseMove)
    }
    return () => {
      if (container) {
        container.removeEventListener('mousemove', handleMouseMove)
      }
      clearTimeout(timeout)
    }
  }, [isPlaying, chromecast.isConnected])

  // Keyboard Shortcuts (Space, Arrow keys, F, M)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.target instanceof HTMLInputElement) return

      switch (e.key.toLowerCase()) {
        case ' ':
        case 'k':
          e.preventDefault()
          handleTogglePlay()
          break
        case 'f':
          e.preventDefault()
          handleToggleFullscreen()
          break
        case 'm':
          e.preventDefault()
          handleToggleMute()
          break
        case 'arrowleft':
          e.preventDefault()
          handleSkip(-10)
          break
        case 'arrowright':
          e.preventDefault()
          handleSkip(10)
          break
        case 'arrowup':
          e.preventDefault()
          handleVolumeChange(Math.min(volume + 0.1, 1))
          break
        case 'arrowdown':
          e.preventDefault()
          handleVolumeChange(Math.max(volume - 0.1, 0))
          break
        case 'escape':
          if (isFullscreen) {
            document.exitFullscreen?.()
          }
          break
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [isPlaying, volume, isFullscreen, currentTime, chromecast.isConnected])

  // Playback reporting to Jellyfin Server
  useEffect(() => {
    if (!user) return
    const sessionId = playbackInfo?.PlaySessionId || 'session-1'

    const interval = setInterval(() => {
      if (isPlaying && currentTime > 0) {
        const positionTicks = Math.floor(currentTime * 10000000)
        jellyfinService.reportPlaybackProgress(item.Id, sessionId, positionTicks, !isPlaying)
      }
    }, 10000)

    return () => {
      clearInterval(interval)
      if (currentTime > 0) {
        const positionTicks = Math.floor(currentTime * 10000000)
        jellyfinService.reportPlaybackStopped(item.Id, sessionId, positionTicks)
      }
    }
  }, [item.Id, currentTime, isPlaying, user, playbackInfo])

  const handleTogglePlay = useCallback(() => {
    if (chromecast.isConnected) {
      if (chromecast.isRemotePlaying) {
        chromecast.remotePause()
      } else {
        chromecast.remotePlay()
      }
      return
    }

    if (videoRef.current) {
      if (isPlaying) {
        videoRef.current.pause()
        setIsPlaying(false)
      } else {
        videoRef.current.play()
        setIsPlaying(true)
      }
    }
  }, [chromecast, isPlaying])

  const handleToggleMute = useCallback(() => {
    if (videoRef.current) {
      const nextMuted = !isMuted
      videoRef.current.muted = nextMuted
      setIsMuted(nextMuted)
    }
  }, [isMuted])

  const handleVolumeChange = useCallback((newVolume: number) => {
    if (videoRef.current) {
      videoRef.current.volume = newVolume
      setVolume(newVolume)
      setIsMuted(newVolume === 0)
    }
  }, [])

  const handleSeek = useCallback(
    (seconds: number) => {
      if (chromecast.isConnected) {
        chromecast.remoteSeek(seconds)
        return
      }

      if (videoRef.current) {
        videoRef.current.currentTime = seconds
        setCurrentTime(seconds)
      }
    },
    [chromecast]
  )

  const handleSkip = useCallback(
    (offsetSeconds: number) => {
      const activeTime = chromecast.isConnected ? chromecast.remoteCurrentTime : currentTime
      const targetTime = Math.max(0, Math.min(activeTime + offsetSeconds, duration))
      handleSeek(targetTime)
    },
    [chromecast, currentTime, duration, handleSeek]
  )

  const handleToggleFullscreen = useCallback(() => {
    const container = containerRef.current
    if (!container) return

    if (!document.fullscreenElement) {
      container.requestFullscreen().then(() => setIsFullscreen(true))
    } else {
      document.exitFullscreen().then(() => setIsFullscreen(false))
    }
  }, [])

  const handleBack = () => {
    if (onBack) {
      onBack()
    } else {
      navigate(-1)
    }
  }

  return (
    <div
      ref={containerRef}
      className="relative w-screen h-screen bg-black overflow-hidden select-none flex items-center justify-center"
    >
      {/* View 1: Remote Chromecast Control View */}
      {chromecast.isConnected ? (
        <CastRemoteView
          item={item}
          deviceName={chromecast.deviceName}
          isPlaying={chromecast.isRemotePlaying}
          currentTime={chromecast.remoteCurrentTime}
          duration={chromecast.remoteDuration || duration}
          onTogglePlay={handleTogglePlay}
          onSeek={handleSeek}
          onSkip={handleSkip}
          onDisconnect={chromecast.disconnect}
        />
      ) : (
        /* View 2: Local Video Element */
        <>
          <video
            ref={videoRef}
            className="w-full h-full object-contain cursor-pointer"
            onClick={handleTogglePlay}
            onTimeUpdate={() => {
              if (videoRef.current) {
                setCurrentTime(videoRef.current.currentTime)
                setDuration(videoRef.current.duration || 0)
              }
            }}
            onEnded={() => setIsPlaying(false)}
          />

          {/* Top Header Bar */}
          <div
            className={`absolute top-0 inset-x-0 p-6 md:p-8 flex items-center gap-4 bg-gradient-to-b from-black/90 via-black/40 to-transparent transition-opacity duration-300 z-30 ${
              showControls ? 'opacity-100' : 'opacity-0 pointer-events-none'
            }`}
          >
            <button
              onClick={handleBack}
              className="text-white hover:text-gray-300 p-2 rounded-full hover:bg-white/10 transition-colors"
              title="Voltar"
            >
              <ArrowLeft className="w-6 h-6" />
            </button>
            <div className="flex flex-col">
              <h1 className="text-base md:text-lg font-bold text-white truncate max-w-xl">{item.Name}</h1>
              {item.SeriesName && (
                <span className="text-xs text-netflix-gray truncate">{item.SeriesName}</span>
              )}
            </div>
          </div>

          {/* Bottom Modular Controls */}
          <VideoControls
            isPlaying={isPlaying}
            isMuted={isMuted}
            volume={volume}
            currentTime={currentTime}
            duration={duration}
            showControls={showControls}
            title={item.Name}
            audioStreams={audioStreams}
            subtitleStreams={subtitleStreams}
            selectedAudioIndex={selectedAudioIndex}
            selectedSubtitleIndex={selectedSubtitleIndex}
            onTogglePlay={handleTogglePlay}
            onToggleMute={handleToggleMute}
            onVolumeChange={handleVolumeChange}
            onSeek={handleSeek}
            onSkip={handleSkip}
            onToggleFullscreen={handleToggleFullscreen}
            isFullscreen={isFullscreen}
            onSelectAudio={setSelectedAudioIndex}
            onSelectSubtitle={setSelectedSubtitleIndex}
          />
        </>
      )}
    </div>
  )
}
