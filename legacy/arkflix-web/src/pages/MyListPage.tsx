import React, { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useUserLibraries, useResumeItems, useLibraryItems, useTranslation } from '@/hooks'
import { MediaCard } from '@/components/media/MediaCard'
import { DetailModal } from '@/components/media/DetailModal'
import { MediaCardSkeleton, Button, Select } from '@/components/ui'
import { Library, BookmarkCheck, ArrowLeft, FolderOpen, Filter } from 'lucide-react'

export const MyListPage: React.FC = () => {
  const { t } = useTranslation()
  const [selectedLibrary, setSelectedLibrary] = useState<{ id: string; name: string; type?: string } | null>(null)
  const [sortBy, setSortBy] = useState<'DateCreated' | 'CommunityRating' | 'SortName' | 'PremiereDate'>('DateCreated')

  // Dados das bibliotecas e itens em andamento
  const { data: resumeItems, isLoading: loadingResume } = useResumeItems(24)
  const { data: libraries, isLoading: loadingLibraries } = useUserLibraries()

  // Itens da biblioteca selecionada
  const { data: libraryItemsData, isLoading: loadingFolderItems } = useLibraryItems(
    selectedLibrary?.id,
    { sortBy, sortOrder: 'Descending', limit: 80 }
  )

  const folderItems = libraryItemsData?.Items || []

  return (
    <div className="min-h-screen pt-20 sm:pt-24 pb-20 px-4 sm:px-6 md:px-14 max-w-7xl mx-auto space-y-8 sm:space-y-10 animate-fadeIn">
      {/* Header com Navegação Breadcrumb quando uma pasta estiver aberta */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 sm:gap-4 border-b border-white/[0.08] pb-4 sm:pb-6">
        <div className="flex items-center gap-3">
          {selectedLibrary ? (
            <button
              onClick={() => setSelectedLibrary(null)}
              className="w-9 h-9 sm:w-10 sm:h-10 rounded-squircle bg-white/10 hover:bg-white/20 text-white flex items-center justify-center border border-white/15 transition-all active:scale-95"
              aria-label="Voltar para todas as pastas"
            >
              <ArrowLeft className="w-4 h-4 sm:w-5 sm:h-5" />
            </button>
          ) : (
            <div className="w-9 h-9 sm:w-10 sm:h-10 rounded-squircle bg-blue-500/15 text-apple-accent flex items-center justify-center border border-blue-500/25">
              <Library className="w-4 h-4 sm:w-5 sm:h-5" />
            </div>
          )}

          <div>
            <h1 className="text-xl sm:text-2xl md:text-3xl font-extrabold text-white tracking-tight flex items-center gap-2">
              {selectedLibrary ? (
                <>
                  <span
                    onClick={() => setSelectedLibrary(null)}
                    className="text-apple-subtext hover:text-white cursor-pointer transition-colors"
                  >
                    {t.nav.library}
                  </span>
                  <span className="text-apple-subtext font-normal">/</span>
                  <span className="truncate max-w-[180px] sm:max-w-none">{selectedLibrary.name}</span>
                </>
              ) : (
                t.nav.library
              )}
            </h1>
            <p className="text-[11px] sm:text-xs text-apple-subtext mt-0.5">
              {selectedLibrary
                ? `${folderItems.length} títulos nesta pasta`
                : 'Suas pastas, coleções e títulos em andamento'}
            </p>
          </div>
        </div>

        {/* Filtros dentro da pasta selecionada */}
        {selectedLibrary && (
          <div className="flex items-center gap-2.5 sm:gap-3 self-end sm:self-auto w-full sm:w-auto">
            <div className="flex items-center gap-1.5 text-xs text-apple-subtext whitespace-nowrap">
              <Filter className="w-3.5 h-3.5" />
              <span>Ordenar:</span>
            </div>

            <div className="w-full sm:w-48">
              <Select
                value={sortBy}
                onChange={(e) => setSortBy(e.target.value as any)}
                options={[
                  { value: 'DateCreated', label: 'Recentes' },
                  { value: 'CommunityRating', label: 'Melhor Avaliados' },
                  { value: 'SortName', label: 'Ordem A-Z' },
                  { value: 'PremiereDate', label: 'Ano de Lançamento' },
                ]}
              />
            </div>
          </div>
        )}
      </div>

      <AnimatePresence mode="wait">
        {selectedLibrary ? (
          /* ================= VIEW: ITENS DA PASTA SELECIONADA ================= */
          <motion.div
            key="library-contents"
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -12 }}
            transition={{ duration: 0.25 }}
            className="space-y-6"
          >
            {loadingFolderItems ? (
              <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3 sm:gap-6">
                {Array.from({ length: 12 }).map((_, i) => (
                  <MediaCardSkeleton key={i} />
                ))}
              </div>
            ) : folderItems.length === 0 ? (
              <div className="py-20 text-center space-y-4">
                <FolderOpen className="w-10 h-10 text-apple-subtext mx-auto opacity-50" />
                <p className="text-sm font-medium text-white">Nenhum item encontrado nesta pasta.</p>
                <Button variant="glass" size="sm" onClick={() => setSelectedLibrary(null)}>
                  <ArrowLeft className="w-3.5 h-3.5 mr-1" /> Voltar para Bibliotecas
                </Button>
              </div>
            ) : (
              <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3 sm:gap-6">
                {folderItems.map((item) => (
                  <MediaCard key={item.Id} item={item} layout="grid" />
                ))}
              </div>
            )}
          </motion.div>
        ) : (
          /* ================= VIEW: VISÃO GERAL (PASTAS & CONTINUAR ASSISTINDO) ================= */
          <motion.div
            key="library-overview"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="space-y-10 sm:space-y-12"
          >
            {/* Section 1: Pastas da Biblioteca Clicáveis */}
            <div className="space-y-4">
              <h2 className="text-base sm:text-lg font-bold text-white tracking-tight flex items-center gap-2">
                <FolderOpen className="w-4 h-4 text-apple-accent" /> Pastas da Biblioteca
              </h2>

              {loadingLibraries ? (
                <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3 sm:gap-4">
                  {Array.from({ length: 8 }).map((_, i) => (
                    <div key={i} className="h-16 sm:h-20 rounded-squircle-lg bg-white/[0.04] animate-pulse" />
                  ))}
                </div>
              ) : (
                <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3 sm:gap-4">
                  {libraries?.map((lib) => (
                    <motion.div
                      key={lib.Id}
                      onClick={() => setSelectedLibrary({ id: lib.Id, name: lib.Name, type: lib.CollectionType })}
                      whileHover={{ scale: 1.025, y: -2 }}
                      whileTap={{ scale: 0.97 }}
                      transition={{ type: 'spring', stiffness: 450, damping: 25 }}
                      className="p-3.5 sm:p-5 rounded-squircle-lg bg-white/[0.04] hover:bg-white/[0.08] border border-white/10 transition-colors duration-200 flex items-center gap-3 sm:gap-3.5 shadow-sm cursor-pointer group will-change-transform"
                    >
                      <div className="w-9 h-9 sm:w-10 sm:h-10 rounded-squircle bg-white/10 text-white flex items-center justify-center shadow-sm group-hover:scale-105 transition-transform flex-none">
                        <Library className="w-4 h-4 sm:w-5 sm:h-5 text-apple-accent" />
                      </div>
                      <div className="min-w-0 flex-grow">
                        <h3 className="text-xs sm:text-sm font-bold text-white truncate group-hover:text-apple-accent transition-colors">
                          {lib.Name}
                        </h3>
                        <span className="text-[9px] sm:text-[10px] text-apple-subtext uppercase tracking-wider font-semibold">
                          {lib.CollectionType || 'Coleção'}
                        </span>
                      </div>
                    </motion.div>
                  ))}
                </div>
              )}
            </div>

            {/* Section 2: Continuar Assistindo */}
            <div className="space-y-4 pt-2">
              <div className="flex items-center gap-2">
                <BookmarkCheck className="w-4 h-4 text-emerald-400" />
                <h2 className="text-base sm:text-lg font-bold text-white tracking-tight">Em Andamento</h2>
              </div>

              {loadingResume ? (
                <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3 sm:gap-4">
                  {Array.from({ length: 6 }).map((_, i) => (
                    <MediaCardSkeleton key={i} />
                  ))}
                </div>
              ) : !resumeItems || resumeItems.length === 0 ? (
                <div className="p-6 sm:p-8 rounded-squircle bg-white/[0.03] border border-white/10 text-center">
                  <p className="text-xs text-apple-subtext">Você não possui títulos em andamento no momento.</p>
                </div>
              ) : (
                <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3 sm:gap-6">
                  {resumeItems.map((item) => (
                    <MediaCard key={item.Id} item={item} layout="grid" />
                  ))}
                </div>
              )}
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Details Modal */}
      <DetailModal />
    </div>
  )
}
