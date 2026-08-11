/// <reference types="vite/client" />

interface Window {
  __GULYAEM_DEBUG_MAP__?: import('maplibre-gl').Map
}

interface ImportMetaEnv {
  readonly VITE_API_URL?: string
  readonly VITE_MAP_STYLE_URL?: string
  readonly VITE_CITY_ID?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
