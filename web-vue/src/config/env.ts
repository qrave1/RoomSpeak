/**
 * Конфигурация переменных окружения
 * Все переменные должны начинаться с VITE_ чтобы быть доступными в клиенте
 */

export const env = {
    // API URL для запросов к бэкенду
    backendUrl: import.meta.env.VITE_BACKEND_URL || 'http://localhost:3000',

    // Окружение приложения
    appEnv: import.meta.env.VITE_APP_ENV || 'development',

    // Debug режим
    debug: import.meta.env.VITE_DEBUG === 'true',

    // Проверки окружения
    isDevelopment: import.meta.env.DEV,
    isProduction: import.meta.env.PROD,

    // Vite mode
    mode: import.meta.env.MODE
}

// Логирование конфигурации в dev режиме
if (env.debug && env.isDevelopment) {
    console.log('🔧 Environment Config:', env)
}

export default env

