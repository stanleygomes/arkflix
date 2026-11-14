# Arkflix — Jellyfin Web Client (Netflix UI + Google Cast)

Arkflix é uma aplicação web moderna estilo **Netflix** desenvolvida em **React 19 + TypeScript + Vite + Tailwind CSS**, integrada diretamente à API do **Jellyfin** e com suporte nativo ao **Google Cast (Chromecast)**.

---

## 🚀 Tecnologias Utilizadas

* **Framework & Build:** [React 19](https://react.dev/) + [Vite](https://vitejs.dev/) + [TypeScript](https://www.typescriptlang.org/)
* **Estilização:** [Tailwind CSS](https://tailwindcss.com/) com a paleta oficial da Netflix (`#141414`, `#E50914`, `#181818`)
* **State Management & Server Cache:** [TanStack Query v5](https://tanstack.com/query) + [Zustand](https://github.com/pmndrs/zustand)
* **Streaming & Vídeo:** [HLS.js](https://github.com/video-dev/hls.js/) + HTML5 Video API + Google Cast SDK
* **Ícones:** [Lucide React](https://lucide.dev/)
* **Roteamento:** [React Router v7](https://reactrouter.com/)

---

## 📁 Estrutura do Projeto

O projeto foi organizado com foco em **reutilização de componentes**, escalabilidade e separação clara de responsabilidades:

```text
arkflix/
├── src/
│   ├── components/            # Componentes reutilizáveis
│   │   ├── layout/            # Estrutura global (Navbar, Footer, MainLayout)
│   │   ├── media/             # Componentes de mídia (HeroBanner, MediaCard, MediaRow, DetailModal)
│   │   ├── player/            # Controles de vídeo e cast
│   │   └── ui/                # Primitivos reutilizáveis de UI (Button, Modal, Slider, etc.)
│   ├── hooks/                 # Custom React Hooks
│   ├── lib/                   # Utilitários e helpers (cn, formatters)
│   ├── pages/                 # Páginas da aplicação (HomePage, LoginPage, WatchPage, SearchPage)
│   ├── services/              # Integração de API (apiClient, jellyfinService)
│   ├── stores/                # Estados globais Zustand (authStore, modalStore, castStore)
│   ├── types/                 # Interfaces e tipos TypeScript do Jellyfin
│   ├── App.tsx                # Roteamento e Provedores globais
│   ├── main.tsx               # Entrypoint da aplicação
│   └── index.css              # Configuração global de Tailwind CSS
├── public/                    # Arquivos estáticos
├── .env                       # Variáveis de ambiente
├── tailwind.config.js         # Configuração de tema e cores
└── vite.config.ts             # Configuração do Vite com aliases (@/*)
```

---

## ⚙️ Configuração & Execução

### 1. Instalar dependências:
```bash
npm install
```

### 2. Rodar em desenvolvimento:
```bash
npm run dev
```

### 3. Build de produção:
```bash
npm run build
```

---

## 📡 Integração com Servidor Jellyfin

* **Base URL:** `https://ark-flix.duckdns.org`
* **Chromecast Receiver ID:** `F007D354`