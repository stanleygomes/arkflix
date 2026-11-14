import { create } from 'zustand'
import { CastMediaSession, CastSession } from '@/types/cast'

interface CastState {
  isAvailable: boolean
  isConnected: boolean
  deviceName: string | null
  castSession: CastSession | null
  mediaSession: CastMediaSession | null
  isRemotePlaying: boolean
  remoteCurrentTime: number
  remoteDuration: number
  setAvailable: (available: boolean) => void
  setCastSession: (session: CastSession | null) => void
  setMediaSession: (session: CastMediaSession | null) => void
  updateRemoteState: (isPlaying: boolean, currentTime: number, duration?: number) => void
  disconnect: () => void
}

export const useCastStore = create<CastState>((set, get) => ({
  isAvailable: false,
  isConnected: false,
  deviceName: null,
  castSession: null,
  mediaSession: null,
  isRemotePlaying: false,
  remoteCurrentTime: 0,
  remoteDuration: 0,

  setAvailable: (isAvailable) => set({ isAvailable }),

  setCastSession: (session) => {
    if (!session) {
      set({
        castSession: null,
        mediaSession: null,
        isConnected: false,
        deviceName: null,
        isRemotePlaying: false,
      })
      return
    }

    const deviceName = session.getCastDevice()?.friendlyName || 'Chromecast'
    set({
      castSession: session,
      isConnected: true,
      deviceName,
    })
  },

  setMediaSession: (mediaSession) => set({ mediaSession }),

  updateRemoteState: (isRemotePlaying, remoteCurrentTime, remoteDuration) =>
    set((state) => ({
      isRemotePlaying,
      remoteCurrentTime,
      remoteDuration: remoteDuration !== undefined ? remoteDuration : state.remoteDuration,
    })),

  disconnect: () => {
    const { castSession } = get()
    if (castSession) {
      try {
        castSession.endSession(true)
      } catch (err) {
        console.error('Error ending cast session:', err)
      }
    }
    set({
      castSession: null,
      mediaSession: null,
      isConnected: false,
      deviceName: null,
      isRemotePlaying: false,
    })
  },
}))
