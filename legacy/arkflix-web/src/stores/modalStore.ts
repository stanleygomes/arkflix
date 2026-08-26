import { create } from 'zustand'
import { MediaItem } from '@/types/jellyfin'

interface ModalState {
  isOpen: boolean
  selectedItem: MediaItem | null
  openModal: (item: MediaItem) => void
  closeModal: () => void
}

export const useModalStore = create<ModalState>((set) => ({
  isOpen: false,
  selectedItem: null,
  openModal: (item) => set({ isOpen: true, selectedItem: item }),
  closeModal: () => set({ isOpen: false, selectedItem: null }),
}))
