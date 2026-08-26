import React from 'react'
import { useUserLibraries, useResumeItems, useTranslation } from '@/hooks'
import { MediaCard } from '@/components/media/MediaCard'
import { DetailModal } from '@/components/media/DetailModal'
import { MediaCardSkeleton } from '@/components/ui'
import { Library, BookmarkCheck } from 'lucide-react'

export const MyListPage: React.FC = () => {
  const { t } = useTranslation()
  const { data: resumeItems, isLoading: loadingResume } = useResumeItems(24)
  const { data: libraries } = useUserLibraries()

  return (
    <div className="min-h-screen pt-24 pb-20 px-6 md:px-14 max-w-7xl mx-auto space-y-10 animate-fadeIn">
      {/* Page Header */}
      <div className="flex items-center gap-3 border-b border-white/[0.08] pb-6">
        <div className="w-10 h-10 rounded-squircle bg-blue-500/15 text-apple-accent flex items-center justify-center border border-blue-500/25">
          <Library className="w-5 h-5" />
        </div>
        <div>
          <h1 className="text-2xl md:text-3xl font-extrabold text-white tracking-tight">{t.nav.library}</h1>
          <p className="text-xs text-apple-subtext mt-0.5">Suas pastas, coleções e títulos em andamento</p>
        </div>
      </div>

      {/* Section 1: Continuar Assistindo */}
      <div className="space-y-4">
        <div className="flex items-center gap-2">
          <BookmarkCheck className="w-4 h-4 text-emerald-400" />
          <h2 className="text-lg font-bold text-white tracking-tight">Em Andamento</h2>
        </div>

        {loadingResume ? (
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4">
            {Array.from({ length: 6 }).map((_, i) => (
              <MediaCardSkeleton key={i} />
            ))}
          </div>
        ) : !resumeItems || resumeItems.length === 0 ? (
          <div className="p-8 rounded-squircle bg-white/[0.03] border border-white/10 text-center">
            <p className="text-xs text-apple-subtext">Você não possui títulos em andamento no momento.</p>
          </div>
        ) : (
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4 sm:gap-6">
            {resumeItems.map((item) => (
              <MediaCard key={item.Id} item={item} />
            ))}
          </div>
        )}
      </div>

      {/* Section 2: Bibliotecas do Servidor */}
      {libraries && libraries.length > 0 && (
        <div className="space-y-4 pt-4">
          <h2 className="text-lg font-bold text-white tracking-tight">Pastas da Biblioteca</h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-4">
            {libraries.map((lib) => (
              <div
                key={lib.Id}
                className="p-5 rounded-squircle-lg bg-white/[0.04] hover:bg-white/[0.08] border border-white/10 transition-all duration-300 flex items-center gap-3.5 shadow-sm"
              >
                <div className="w-10 h-10 rounded-squircle bg-white/10 flex items-center justify-center text-white">
                  <Library className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="text-sm font-semibold text-white truncate">{lib.Name}</h3>
                  <span className="text-[11px] text-apple-subtext uppercase tracking-wider">
                    {lib.CollectionType || 'Coleção'}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Details Modal */}
      <DetailModal />
    </div>
  )
}
