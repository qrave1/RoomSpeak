/// <reference types="vite/client" />

interface ImportMetaEnv {
    readonly VITE_BACKEND_URL: string
    readonly VITE_APP_ENV: 'development' | 'production'
    readonly VITE_DEBUG: string
}

interface ImportMeta {
    readonly env: ImportMetaEnv
}

