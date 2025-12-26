# Development Environment Setup

This guide will help you set up your development environment for Zagalin plugin development.

## System Requirements

### Operating System
- **Linux** (Recommended)
- **macOS** (10.14 Mojave or later)
- **Windows** with WSL 2 (Windows Subsystem for Linux)

### Minimum Grafana Version
- Grafana v10.0.0 or later

## Required Tools

### 1. Node.js (LTS)
Zagalin requires Node.js 20 or later.

```bash
# Check your Node.js version
node --version

# Install via nvm (recommended)
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash
nvm install 20
nvm use 20
```

### 2. Go
Required for backend plugin development.

```bash
# Check your Go version
go version

# Install Go 1.21 or later
# Visit: https://go.dev/dl/
```

### 3. Mage
Build tool for Go projects.

```bash
# Install Mage
go install github.com/magefile/mage@latest

# Verify installation
mage --version
```

### 4. Docker & Docker Compose
For running local Grafana instance.

```bash
# Check Docker installation
docker --version
docker compose version

# Install Docker Desktop or Docker Engine
# Visit: https://docs.docker.com/get-docker/
```

### 5. Package Manager (Optional)
Choose one:
- **npm** (comes with Node.js)
- **yarn** (`npm install -g yarn`)
- **pnpm** (`npm install -g pnpm`)

## Project Setup

### 1. Clone the Repository

```bash
git clone https://github.com/jorgesnotebook/jorgeancal-zagalin-app.git
cd jorgeancal-zagalin-app
```

### 2. Install Dependencies

```bash
# Install frontend dependencies
npm install

# Or with yarn
yarn install

# Or with pnpm
pnpm install
```

### 3. Install Grafana LLM App

Zagalin depends on the Grafana LLM App plugin for LLM integration.

```bash
# Using Grafana CLI
grafana-cli plugins install grafana-llm-app

# Or via Docker (see provisioning/plugins/)
```

## Build the Plugin

### Frontend Build

```bash
# Development build
npm run dev

# Production build
npm run build

# Watch mode for development
npm run watch
```

### Backend Build

```bash
# Build backend with Mage
mage -v

# Or manually
cd pkg
go build -o ../dist/gpx_zagalin_linux_amd64 ./main.go
```

### Full Build

```bash
# Build both frontend and backend
npm run build
mage -v
```

## Run Development Server

### Using Docker Compose (Recommended)

The project includes a complete Docker setup with Grafana and all dependencies.

```bash
# Start Grafana with the plugin
docker compose up

# Or in detached mode
docker compose up -d

# View logs
docker compose logs -f

# Stop services
docker compose down
```

Access Grafana at: http://localhost:3000
- **Username**: `admin`
- **Password**: `admin`

### Manual Setup

If not using Docker:

1. Install Grafana locally
2. Configure plugin path in `grafana.ini`:
   ```ini
   [paths]
   plugins = /path/to/jorgeancal-zagalin-app
   ```
3. Restart Grafana
4. Enable the plugin in Grafana UI

## Verify Installation

### 1. Check Plugin is Loaded

```bash
# In Grafana UI:
# 1. Navigate to Configuration → Plugins
# 2. Search for "Zagalin"
# 3. Verify plugin is installed and enabled
```

### 2. Check Backend Health

```bash
# The backend should start automatically
# Check logs for:
# "Zagalin plugin started"
```

### 3. Test Frontend

```bash
# Navigate to Apps → Zagalin
# You should see the configuration page
```

## Development Workflow

### 1. Frontend Development

```bash
# Terminal 1: Watch and rebuild on changes
npm run watch

# Terminal 2: Run Grafana
docker compose up
```

Changes to frontend code will automatically rebuild. Refresh your browser to see changes.

### 2. Backend Development

```bash
# Rebuild backend
mage -v

# Restart Grafana to load new backend
docker compose restart grafana
```

### 3. Hot Reload (Frontend Only)

The `npm run dev` command includes source maps for easier debugging. Use browser DevTools to debug React components.

## IDE Setup

### Visual Studio Code (Recommended)

#### Required Extensions
- ESLint
- Prettier
- TypeScript and JavaScript Language Features
- Go (if editing backend)

#### Workspace Settings

Create `.vscode/settings.json`:

```json
{
  "typescript.tsdk": "node_modules/typescript/lib",
  "typescript.enablePromptUseWorkspaceTsdk": true,
  "editor.formatOnSave": true,
  "editor.codeActionsOnSave": {
    "source.fixAll.eslint": true
  },
  "files.eol": "\n"
}
```

#### Recommended Launch Configuration

Create `.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "type": "chrome",
      "request": "launch",
      "name": "Launch Chrome against localhost",
      "url": "http://localhost:3000",
      "webRoot": "${workspaceFolder}/src"
    }
  ]
}
```

## Environment Variables

### Development .env File

Create `.env.development`:

```bash
# API endpoints (if needed)
GRAFANA_API_URL=http://localhost:3000

# Plugin settings
ENABLE_DEBUG_LOGGING=true
```

### Docker Environment

Configure in `docker-compose.yml` or `.env`:

```bash
# Grafana settings
GF_DEFAULT_APP_MODE=development
GF_LOG_LEVEL=debug
GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=jorgeancal-zagalin-app
```

## Troubleshooting

### Plugin Not Loading

1. **Check plugin.json** - Ensure `id` matches folder name
2. **Verify build** - Run `npm run build` and check `dist/` folder
3. **Check Grafana logs** - Look for plugin loading errors
4. **Unsigned plugin** - Add to `GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS`

### Backend Not Starting

1. **Check Go version** - Must be 1.21+
2. **Rebuild backend** - Run `mage -v`
3. **Check executable** - Verify `dist/gpx_zagalin_*` exists
4. **Permissions** - Ensure binary is executable (`chmod +x`)

### Frontend Build Errors

1. **Clear node_modules** - `rm -rf node_modules && npm install`
2. **Clear cache** - `npm run clean` (if available)
3. **Check Node version** - Must be 20+
4. **Update dependencies** - `npm update`

### Docker Issues

1. **Port conflicts** - Ensure port 3000 is available
2. **Volume permissions** - Check Docker volume permissions
3. **Restart services** - `docker compose down && docker compose up`
4. **Check logs** - `docker compose logs grafana`

## Next Steps

- Read [Plugin Architecture](../development/architecture.md)
- Learn about [Frontend Development](../development/frontend.md)
- Set up [Testing](../testing/overview.md)
- Review [Contributing Guidelines](../contributing/guidelines.md)

## Additional Resources

- [Grafana Plugin Tools Documentation](https://grafana.com/developers/plugin-tools/)
- [Grafana Plugin Examples](https://github.com/grafana/grafana-plugin-examples)
- [Grafana Community Forum](https://community.grafana.com/)
- [React Documentation](https://react.dev/)
- [TypeScript Handbook](https://www.typescriptlang.org/docs/)
