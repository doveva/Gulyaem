/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_URL?: string
  readonly VITE_MAP_STYLE_URL?: string
  readonly VITE_CITY_ID?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
