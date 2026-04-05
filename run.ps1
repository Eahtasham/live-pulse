#!/usr/bin/env pwsh
# ──────────────────────────────────────────────
# LivePulse — Task Runner (run.ps1)
# Usage: .\run.ps1 <command>
# ──────────────────────────────────────────────

param(
    [Parameter(Position = 0)]
    [string]$Command = "help"
)

# Load .env if it exists
if (Test-Path ".env") {
    Get-Content ".env" | ForEach-Object {
        if ($_ -match '^\s*([^#][^=]+)=(.*)$') {
            $key = $matches[1].Trim()
            $val = $matches[2].Trim()
            [System.Environment]::SetEnvironmentVariable($key, $val, "Process")
        }
    }
}

$DATABASE_URL = if ($env:DATABASE_URL) { $env:DATABASE_URL } else { "postgres://livepulse:livepulse_dev@localhost:5432/livepulse?sslmode=disable" }

switch ($Command) {

    # ─── Docker ──────────────────────────────
    "docker-up" {
        Write-Host "Starting Postgres + Redis containers..." -ForegroundColor Cyan
        docker compose up -d
    }
    "docker-down" {
        Write-Host "Stopping containers..." -ForegroundColor Cyan
        docker compose down
    }
    "docker-ps" {
        docker compose ps
    }

    # ─── Database Migrations ─────────────────
    "migrate-up" {
        Write-Host "Running all pending migrations..." -ForegroundColor Cyan
        migrate -path db/migrations -database $DATABASE_URL up
    }
    "migrate-down" {
        Write-Host "Rolling back last migration..." -ForegroundColor Cyan
        migrate -path db/migrations -database $DATABASE_URL down 1
    }
    "migrate-status" {
        migrate -path db/migrations -database $DATABASE_URL version
    }

    # ─── Go Services ─────────────────────────
    "api" {
        Write-Host "Starting API service..." -ForegroundColor Cyan
        Push-Location apps/api
        go run main.go
        Pop-Location
    }
    "realtime" {
        Write-Host "Starting Realtime service..." -ForegroundColor Cyan
        Push-Location apps/realtime
        go run main.go
        Pop-Location
    }
    "api-build" {
        Push-Location apps/api
        go build -o bin/api.exe main.go
        Pop-Location
    }
    "realtime-build" {
        Push-Location apps/realtime
        go build -o bin/realtime.exe main.go
        Pop-Location
    }

    # ─── Next.js ─────────────────────────────
    "web" {
        Write-Host "Starting Next.js dev server..." -ForegroundColor Cyan
        Push-Location apps/web
        pnpm dev
        Pop-Location
    }

    # ─── Turbo ───────────────────────────────
    "dev" {
        Write-Host "Starting all services via Turborepo..." -ForegroundColor Cyan
        pnpm turbo dev
    }
    "build" {
        pnpm turbo build
    }
    "lint" {
        pnpm turbo lint
    }

    # ─── Setup ───────────────────────────────
    "setup" {
        Write-Host "Installing Node dependencies..." -ForegroundColor Cyan
        pnpm install

        Write-Host "Starting Docker containers..." -ForegroundColor Cyan
        docker compose up -d

        Write-Host "Waiting for Postgres to be ready..." -ForegroundColor Yellow
        Start-Sleep -Seconds 5

        Write-Host "Running database migrations..." -ForegroundColor Cyan
        migrate -path db/migrations -database $DATABASE_URL up

        Write-Host "`nSetup complete! Run '.\run.ps1 dev' to start all services." -ForegroundColor Green
    }

    # ─── Help ────────────────────────────────
    "help" {
        Write-Host ""
        Write-Host "LivePulse Task Runner" -ForegroundColor Cyan
        Write-Host "Usage: .\run.ps1 <command>" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  Docker:" -ForegroundColor Yellow
        Write-Host "    docker-up       Start Postgres + Redis containers"
        Write-Host "    docker-down     Stop and remove containers"
        Write-Host "    docker-ps       Show container status"
        Write-Host ""
        Write-Host "  Database:" -ForegroundColor Yellow
        Write-Host "    migrate-up      Run all pending migrations"
        Write-Host "    migrate-down    Rollback last migration"
        Write-Host "    migrate-status  Show current migration version"
        Write-Host ""
        Write-Host "  Services:" -ForegroundColor Yellow
        Write-Host "    api             Run the Go API service"
        Write-Host "    realtime        Run the Go Realtime service"
        Write-Host "    web             Run the Next.js dev server"
        Write-Host "    api-build       Build API binary"
        Write-Host "    realtime-build  Build Realtime binary"
        Write-Host ""
        Write-Host "  Turbo:" -ForegroundColor Yellow
        Write-Host "    dev             Start all services via Turborepo"
        Write-Host "    build           Build all apps"
        Write-Host "    lint            Lint all apps"
        Write-Host ""
        Write-Host "  Setup:" -ForegroundColor Yellow
        Write-Host "    setup           First-time project setup"
        Write-Host ""
    }

    default {
        Write-Host "Unknown command: $Command" -ForegroundColor Red
        Write-Host "Run '.\run.ps1 help' for available commands."
    }
}
