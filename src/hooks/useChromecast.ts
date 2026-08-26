import { useEffect } from 'react'
import { useCastStore } from '@/stores/castStore'
import { MediaItem } from '@/types/jellyfin'
import { getImageUrl } from '@/services/api'
import { jellyfinService } from '@/services/jellyfin'
import { useAuthStore } from '@/stores/authStore'

const CAST_APP_ID = import.meta.env.VITE_JELLYFIN_CAST_APP_ID || 'F007D354'

export function useChromecast() {
  const {
    isAvailable,
    isConnected,
    deviceName,
    castSession,
    mediaSession,
    isRemotePlaying,
    remoteCurrentTime,
    remoteDuration,
    setAvailable,
    setCastSession,
    setMediaSession,
    updateRemoteState,
    disconnect,
  } = useCastStore()

  const { token } = useAuthStore()

  // Initialize Cast SDK Framework
  useEffect(() => {
    window.__onGCastApiAvailable = (available: boolean) => {
      if (available && window.cast?.framework) {
        setAvailable(true)
        const context = window.cast.framework.CastContext.getInstance()

        context.setOptions({
          receiverApplicationId: CAST_APP_ID,
          autoJoinPolicy: window.chrome?.cast?.AutoJoinPolicy.ORIGIN_SCOPED,
        })

        // Listen for session state changes
        context.addEventListener(
          window.cast.framework.CastContextEventType.SESSION_STATE_CHANGED,
          (event: any) => {
            const SessionState = window.cast?.framework.SessionState
            if (!SessionState) return

            if (
              event.sessionState === SessionState.SESSION_STARTED ||
              event.sessionState === SessionState.SESSION_RESUMED
            ) {
              const session = context.getCurrentSession()
              setCastSession(session)
              if (session) {
                const currentMedia = session.getMediaSession()
                if (currentMedia) {
                  setMediaSession(currentMedia)
                  attachMediaListeners(currentMedia)
                }
              }
            } else if (event.sessionState === SessionState.SESSION_ENDED) {
              setCastSession(null)
            }
          }
        )

        // Check if session is already active
        const existingSession = context.getCurrentSession()
        if (existingSession) {
          setCastSession(existingSession)
        }
      }
    }

    if (window.cast?.framework) {
      window.__onGCastApiAvailable(true)
    }
  }, [setAvailable, setCastSession, setMediaSession])

  // Attach status listener to active media session
  const attachMediaListeners = (media: any) => {
    media.addUpdateListener((isAlive: boolean) => {
      if (isAlive) {
        const isPlaying = media.playerState === 'PLAYING'
        updateRemoteState(isPlaying, media.currentTime, media.media?.duration || 0)
      }
    })
  }

  // Cast a Jellyfin media item to Chromecast
  const castMedia = async (item: MediaItem, startPosition = 0) => {
    if (!castSession || !token) return

    const streamUrl = jellyfinService.getStreamUrl(item.Id, token)
    const backdropUrl = getImageUrl(item.Id, 'Backdrop', { fillWidth: 1280, quality: 80 })

    const loadRequest = {
      autoplay: true,
      currentTime: startPosition,
      media: {
        contentId: streamUrl,
        contentType: 'video/mp4',
        streamType: 'BUFFERED',
        metadata: {
          metadataType: item.Type === 'Series' || item.Type === 'Episode' ? 2 : 1,
          title: item.Name,
          subtitle: item.SeriesName || item.Overview,
          images: [{ url: backdropUrl }],
        },
      },
    }

    castSession.loadMedia(
      loadRequest,
      (newMediaSession) => {
        setMediaSession(newMediaSession)
        attachMediaListeners(newMediaSession)
      },
      (error) => {
        console.error('Failed to load media on Chromecast:', error)
      }
    )
  }

  // Remote Control methods
  const remotePlay = () => {
    mediaSession?.play()
  }

  const remotePause = () => {
    mediaSession?.pause()
  }

  const remoteSeek = (positionSeconds: number) => {
    mediaSession?.seek({ currentTime: positionSeconds })
  }

  return {
    isAvailable,
    isConnected,
    deviceName,
    isRemotePlaying,
    remoteCurrentTime,
    remoteDuration,
    castMedia,
    remotePlay,
    remotePause,
    remoteSeek,
    disconnect,
  }
}
