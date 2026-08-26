/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_JELLYFIN_SERVER_URL: string
  readonly VITE_JELLYFIN_CAST_APP_ID: string
  readonly VITE_APP_CLIENT_NAME: string
  readonly VITE_APP_VERSION: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare namespace JSX {
  interface IntrinsicElements {
    'google-cast-launcher': React.DetailedHTMLProps<React.HTMLAttributes<HTMLElement>, HTMLElement> & {
      class?: string
    }
  }
}
