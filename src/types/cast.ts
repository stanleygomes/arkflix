// Google Cast Web SDK Types Definitions

export interface CastSession {
  getSessionId: () => string
  getMediaSession: () => CastMediaSession | null
  loadMedia: (
    loadRequest: CastLoadRequest,
    onSuccess?: (mediaSession: CastMediaSession) => void,
    onError?: (error: any) => void
  ) => void
  endSession: (stopCasting: boolean) => void
  getCastDevice: () => { friendlyName: string }
}

export interface CastMediaSession {
  media: {
    contentId: string
    contentType: string
    metadata?: any
  }
  currentTime: number
  playerState: 'PLAYING' | 'PAUSED' | 'BUFFERING' | 'IDLE'
  play: (successCallback?: () => void, errorCallback?: () => void) => void
  pause: (successCallback?: () => void, errorCallback?: () => void) => void
  seek: (seekRequest: { currentTime: number }, successCallback?: () => void, errorCallback?: () => void) => void
  stop: (successCallback?: () => void, errorCallback?: () => void) => void
  addUpdateListener: (listener: (isAlive: boolean) => void) => void
  removeUpdateListener: (listener: (isAlive: boolean) => void) => void
}

export interface CastLoadRequest {
  autoplay: boolean
  currentTime: number
  media: {
    contentId: string
    contentType: string
    metadata: {
      metadataType: number
      title: string
      subtitle?: string
      images?: { url: string }[]
    }
    streamType: string
  }
}

declare global {
  interface Window {
    __onGCastApiAvailable?: (isAvailable: boolean) => void
    cast?: {
      framework: {
        CastContext: {
          getInstance: () => {
            setOptions: (options: { receiverApplicationId: string; autoJoinPolicy: any }) => void
            getCurrentSession: () => CastSession | null
            requestSession: () => Promise<any>
            addEventListener: (eventType: string, eventHandler: (event: any) => void) => void
            removeEventListener: (eventType: string, eventHandler: (event: any) => void) => void
          }
        }
        CastContextEventType: {
          CAST_STATE_CHANGED: string
          SESSION_STATE_CHANGED: string
        }
        SessionState: {
          SESSION_STARTED: string
          SESSION_RESUMED: string
          SESSION_ENDED: string
        }
        CastState: {
          NO_DEVICES_AVAILABLE: string
          NOT_CONNECTED: string
          CONNECTING: string
          CONNECTED: string
        }
        CastReceiverContext?: any
      }
    }
    chrome?: {
      cast?: {
        AutoJoinPolicy: {
          ORIGIN_SCOPED: string
          TAB_AND_ORIGIN_SCOPED: string
          PAGE_SCOPED: string
        }
        media: {
          MetadataType: {
            GENERIC: number
            MOVIE: number
            TV_SHOW: number
          }
          StreamType: {
            BUFFERED: string
            LIVE: string
          }
        }
      }
    }
  }
}
