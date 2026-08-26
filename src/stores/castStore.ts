import { create } from 'zustand'

interface CastState {
  isAvailable: boolean
  isConnected: boolean
  deviceName: string | null
  setAvailable: (available: boolean) => void
  setConnected: (connected: boolean, deviceName?: string | null) => void
}

export const useCastStore = create<CastState>((set) => ({
  isAvailable: false,
  isConnected: false,
  deviceName: null,
  setAvailable: (available) => set({ isAvailable: available }),
  setConnected: (connected, deviceName = null) => set({ isConnected: connected, deviceName }),
}))
