# Lotto Manager

Sistema de gestión de rifas/loterías con integración de Telegram Mini App.

## Características

- Panel de administración integrado en Telegram
- Gestión de rifas (terminal 00-99 o triple 000-999)
- Reserva y venta de boletos
- Registro de pagos y abonos
- Búsqueda de clientes
- Base de datos Turso (SQLite distribuido)

## Requisitos

- Go 1.21+
- Cuenta en Turso (base de datos)
- Bot de Telegram

## Configuración

Crear archivo `.env`:

```env
TELEGRAM_TOKEN=tu_bot_token
PORT=8085

# Turso Database
TURSO_DATABASE_URL=libsql://tu-db.turso.io
TURSO_AUTH_TOKEN=tu_token

# Telegram Admin IDs (separados por coma)
ADMIN_TELEGRAM_IDS=123456789
```

## Desarrollo

```bash
# Compilar
./build.sh

# Ejecutar
./run.sh
```

## Estructura

```
├── cmd/server/         # Punto de entrada
├── internal/
│   ├── db/             # Conexión a base de datos
│   ├── handlers/       # Controladores HTTP
│   ├── middleware/     # Autenticación Telegram
│   ├── models/         # Modelos de datos
│   └── services/       # Bot de Telegram
└── web/templates/      # Plantillas HTML
```

## Configurar Bot en Telegram

1. Abrir `@BotFather`
2. `/mybots` → seleccionar tu bot
3. `Bot Settings` → `Menu Button` → `Configure menu button`
4. Agregar URL del admin: `https://tu-dominio.com/admin`

## Despliegue

Compatible con:
- DigitalOcean App Platform
- Railway
- Fly.io
- Cualquier servidor con Go

Variables de entorno requeridas en producción:
- `TELEGRAM_TOKEN`
- `TURSO_DATABASE_URL`
- `TURSO_AUTH_TOKEN`
- `ADMIN_TELEGRAM_IDS`
- `PORT`
