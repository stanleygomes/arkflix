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
    <div className="min-h-screen pt-24 pb-20 px-6 md:px-14 max-w-7xl mx-auto space-y-10 animate-fadeIn">
      {/* Header com Navegação Breadcrumb quando uma pasta estiver aberta */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-black/10 dark:border-white/[0.08] pb-6">
        <div className="flex items-center gap-3">
          {selectedLibrary ? (
            <button
              onClick={() => setSelectedLibrary(null)}
              className="w-10 h-10 rounded-squircle bg-black/5 dark:bg-white/10 hover:bg-black/10 dark:hover:bg-white/20 text-apple-text flex items-center justify-center border border-black/10 dark:border-white/15 transition-all active:scale-95"
              aria-label="Voltar para todas as pastas"
            >
              <ArrowLeft className="w-5 h-5" />
            </button>
          ) : (
            <div className="w-10 h-10 rounded-squircle bg-blue-500/15 text-apple-accent flex items-center justify-center border border-blue-500/25">
              <Library className="w-5 h-5" />
            </div>
          )}

          <div>
            <h1 className="text-2xl md:text-3xl font-extrabold text-apple-text tracking-tight flex items-center gap-2">
              {selectedLibrary ? (
                <>
                  <span
                    onClick={() => setSelectedLibrary(null)}
                    className="text-apple-subtext hover:text-apple-text cursor-pointer transition-colors"
                  >
                    {t.nav.library}
                  </span>
                  <span className="text-apple-subtext font-normal">/</span>
                  <span>{selectedLibrary.name}</span>
                </>
              ) : (
                t.nav.library
              )}
            </h1>
            <p className="text-xs text-apple-subtext mt-0.5">
              {selectedLibrary
                ? `${folderItems.length} títulos encontrados nesta pasta`
                : 'Suas pastas, coleções e títulos em andamento'}
            </p>
          </div>
        </div>

        {/* Filtros dentro da pasta selecionada */}
        {selectedLibrary && (
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-2 text-xs text-apple-subtext">
              <Filter className="w-3.5 h-3.5" />
              <span>Ordenar:</span>
            </div>

            <div className="w-44">
              <Select
                value={sortBy}
                onChange={(e) => setSortBy(e.target.value as any)}
                options={[
                  { value: 'DateCreated', label: 'Adicionados Recentemente' },
                  { value: 'CommunityRating', label: 'Melhor Avaliados' },
                  { value: 'SortName', label: 'Ordem Alfabética (A-Z)' },
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
              <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4 sm:gap-6">
                {Array.from({ length: 18 }).map((_, i) => (
                  <MediaCardSkeleton key={i} />
                ))}
              </div>
            ) : folderItems.length === 0 ? (
              <div className="py-24 text-center space-y-4">
                <FolderOpen className="w-12 h-12 text-apple-subtext mx-auto opacity-50" />
                <p className="text-sm font-medium text-apple-text">Nenhum item encontrado nesta pasta.</p>
                <Button variant="glass" size="sm" onClick={() => setSelectedLibrary(null)}>
                  <ArrowLeft className="w-3.5 h-3.5 mr-1" /> Voltar para Bibliotecas
                </Button>
              </div>
            ) : (
              <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4 sm:gap-6">
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
            className="space-y-12"
          >
            {/* Section 1: Pastas da Biblioteca Clicáveis */}
            <div className="space-y-4">
              <h2 className="text-lg font-bold text-apple-text tracking-tight flex items-center gap-2">
                <FolderOpen className="w-4 h-4 text-apple-accent" /> Pastas da Biblioteca
              </h2>

              {loadingLibraries ? (
                <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-4">
                  {Array.from({ length: 8 }).map((_, i) => (
                    <div key={i} className="h-20 rounded-squircle-lg bg-black/5 dark:bg-white/[0.04] animate-pulse" />
                  ))}
                </div>
              ) : (
                <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-4">
                  {libraries?.map((lib) => (
                    <motion.div
                      key={lib.Id}
                      onClick={() => setSelectedLibrary({ id: lib.Id, name: lib.Name, type: lib.CollectionType })}
                      whileHover={{ scale: 1.025, y: -2 }}
                      whileTap={{ scale: 0.97 }}
                      transition={{ type: 'spring', stiffness: 450, damping: 25 }}
                      className="p-5 rounded-squircle-lg bg-black/5 dark:bg-white/[0.04] hover:bg-black/10 dark:hover:bg-white/[0.08] border border-black/10 dark:border-white/10 transition-colors duration-200 flex items-center gap-3.5 shadow-sm cursor-pointer group will-change-transform"
                    >
                      <div className="w-10 h-10 rounded-squircle bg-white dark:bg-white/10 text-black dark:text-white flex items-center justify-center shadow-sm group-hover:scale-105 transition-transform">
                        <Library className="w-5 h-5 text-apple-accent" />
                      </div>
                      <div className="min-w-0 flex-grow">
                        <h3 className="text-sm font-bold text-apple-text truncate group-hover:text-apple-accent transition-colors">
                          {lib.Name}
                        </h3>
                        <span className="text-[10px] text-apple-subtext uppercase tracking-wider font-semibold">
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
                <h2 className="text-lg font-bold text-apple-text tracking-tight">Em Andamento</h2>
              </div>

              {loadingResume ? (
                <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4">
                  {Array.from({ length: 6 }).map((_, i) => (
                    <MediaCardSkeleton key={i} />
                  ))}
                </div>
              ) : !resumeItems || resumeItems.length === 0 ? (
                <div className="p-8 rounded-squircle bg-black/5 dark:bg-white/[0.03] border border-black/10 dark:border-white/10 text-center">
                  <p className="text-xs text-apple-subtext">Você não possui títulos em andamento no momento.</p>
                </div>
              ) : (
                <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4 sm:gap-6">
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
