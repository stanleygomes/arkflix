import React from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { ArrowLeft, Play, Pause, Volume2, VolumeX, Maximize } from 'lucide-react'
import Hls from 'hls.js'
import { useAuthStore } from '@/stores/authStore'
import { jellyfinService } from '@/services/jellyfin'
import { useQuery } from '@tanstack/react-query'

export const WatchPage: React.FC = () => {
  const { id } = useParams<{ id: string }>()
  const { user, token } = useAuthStore()
  const navigate = useNavigate()

  const videoRef = React.useRef<HTMLVideoElement>(null)
  const [isPlaying, setIsPlaying] = React.useState(true)
  const [isMuted, setIsMuted] = React.useState(false)
  const [showControls, setShowControls] = React.useState(true)
  const [currentTime, setCurrentTime] = React.useState(0)
  const [duration, setDuration] = React.useState(0)

  // Fetch item info
  const { data: item } = useQuery({
    queryKey: ['item', id],
    queryFn: () => jellyfinService.getItemDetails(user!.Id, id!),
    enabled: !!user && !!id,
  })

  // Stream URL
  const streamUrl = id && token ? jellyfinService.getStreamUrl(id, token) : ''

  // Initialize Video & HLS
  React.useEffect(() => {
    const video = videoRef.current
    if (!video || !streamUrl) return

    if (Hls.isSupported() && streamUrl.endsWith('.m3u8')) {
      const hls = new Hls()
      hls.loadSource(streamUrl)
      hls.attachMedia(video)
      hls.on(Hls.Events.MANIFEST_PARSED, () => {
        video.play().catch(() => setIsPlaying(false))
      })
      return () => hls.destroy()
    } else {
      video.src = streamUrl
      video.play().catch(() => setIsPlaying(false))
    }
  }, [streamUrl])

  // Controls hide timer
  React.useEffect(() => {
    let timeout: NodeJS.Timeout
    const handleMouseMove = () => {
      setShowControls(true)
      clearTimeout(timeout)
      timeout = setTimeout(() => {
        if (isPlaying) setShowControls(false)
      }, 3500)
    }

    window.addEventListener('mousemove', handleMouseMove)
    return () => {
      window.removeEventListener('mousemove', handleMouseMove)
      clearTimeout(timeout)
    }
  }, [isPlaying])

  const togglePlay = () => {
    if (videoRef.current) {
      if (isPlaying) {
        videoRef.current.pause()
      } else {
        videoRef.current.play()
      }
      setIsPlaying(!isPlaying)
    }
  }

  const handleTimeUpdate = () => {
    if (videoRef.current) {
      setCurrentTime(videoRef.current.currentTime)
      setDuration(videoRef.current.duration || 0)
    }
  }

  const handleSeek = (e: React.ChangeEvent<HTMLInputElement>) => {
    const seekTime = Number(e.target.value)
    if (videoRef.current) {
      videoRef.current.currentTime = seekTime
      setCurrentTime(seekTime)
    }
  }

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60)
    const secs = Math.floor(seconds % 60)
    return `${mins}:${secs < 10 ? '0' : ''}${secs}`
  }

  return (
    <div className="relative w-screen h-screen bg-black overflow-hidden select-none">
      <video
        ref={videoRef}
        className="w-full h-full object-contain cursor-pointer"
        onClick={togglePlay}
        onTimeUpdate={handleTimeUpdate}
        onEnded={() => setIsPlaying(false)}
      />

      {/* Top bar back button */}
      <div
        className={`absolute top-0 left-0 right-0 p-6 flex items-center gap-4 bg-gradient-to-b from-black/80 to-transparent transition-opacity duration-300 ${
          showControls ? 'opacity-100' : 'opacity-0 pointer-events-none'
        }`}
      >
        <button
          onClick={() => navigate(-1)}
          className="text-white hover:text-gray-300 p-2 rounded-full hover:bg-white/10 transition-colors"
        >
          <ArrowLeft className="w-6 h-6" />
        </button>
        <h1 className="text-lg font-bold text-white truncate">{item?.Name || 'Reproduzindo'}</h1>
      </div>

      {/* Bottom Controls Bar */}
      <div
        className={`absolute bottom-0 left-0 right-0 px-8 py-6 bg-gradient-to-t from-black/90 via-black/40 to-transparent transition-opacity duration-300 flex flex-col gap-3 ${
          showControls ? 'opacity-100' : 'opacity-0 pointer-events-none'
        }`}
      >
        {/* Progress Bar */}
        <input
          type="range"
          min={0}
          max={duration || 100}
          value={currentTime}
          onChange={handleSeek}
          className="w-full h-1.5 bg-white/20 accent-netflix-red rounded-lg appearance-none cursor-pointer hover:h-2 transition-all"
        />

        <div className="flex items-center justify-between text-white">
          <div className="flex items-center gap-4">
            <button onClick={togglePlay} className="p-1 hover:text-netflix-red transition-colors">
              {isPlaying ? <Pause className="w-6 h-6" /> : <Play className="w-6 h-6" />}
            </button>

            <button
              onClick={() => {
                if (videoRef.current) {
                  videoRef.current.muted = !isMuted
                  setIsMuted(!isMuted)
                }
              }}
              className="p-1 hover:text-gray-300 transition-colors"
            >
              {isMuted ? <VolumeX className="w-6 h-6" /> : <Volume2 className="w-6 h-6" />}
            </button>

            <span className="text-xs text-gray-300">
              {formatTime(currentTime)} / {formatTime(duration)}
            </span>
          </div>

          <div className="flex items-center gap-4">
            {/* Google Cast Button Launcher inside player */}
            <div className="w-6 h-6 flex items-center justify-center">
              {React.createElement('google-cast-launcher', {
                class: 'w-6 h-6 cursor-pointer opacity-80 hover:opacity-100',
              })}
            </div>

            <button
              onClick={() => {
                if (videoRef.current) {
                  if (document.fullscreenElement) {
                    document.exitFullscreen()
                  } else {
                    videoRef.current.requestFullscreen()
                  }
                }
              }}
              className="p-1 hover:text-gray-300 transition-colors"
            >
              <Maximize className="w-6 h-6" />
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
