#!/usr/bin/env bash
# ──────────────────────────────────────────────
# LivePulse — Task Runner (run.sh)
# Usage: ./run.sh <command>
# ──────────────────────────────────────────────

COMMAND="${1:-help}"

# Load .env if it exists
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
fi

DATABASE_URL="${DATABASE_URL:-postgres://livepulse:livepulse_dev@localhost:5432/livepulse?sslmode=disable}"

case "$COMMAND" in

    # Docker
    docker-up)
        echo "Starting Postgres + Redis containers..."
        docker compose up -d
        ;;
    docker-down)
        echo "Stopping containers..."
        docker compose down
        ;;
    docker-ps)
        docker compose ps
        ;;

    # Database Migrations
    migrate-up)
        echo "Running all pending migrations..."
        migrate -path db/migrations -database "$DATABASE_URL" up
        ;;
    migrate-down)
        echo "Rolling back last migration..."
        migrate -path db/migrations -database "$DATABASE_URL" down 1
        ;;
    migrate-status)
        migrate -path db/migrations -database "$DATABASE_URL" version
        ;;

    # Go Services
    api)
        echo "Starting API service..."
        cd apps/api && go run main.go
        ;;
    realtime)
        echo "Starting Realtime service..."
        cd apps/realtime && go run main.go
        ;;
    api-build)
        cd apps/api && go build -o bin/api main.go
        ;;
    realtime-build)
        cd apps/realtime && go build -o bin/realtime main.go
        ;;

    # Next.js
    web)
        echo "Starting Next.js dev server..."
        cd apps/web && pnpm dev
        ;;

    # Turbo
    dev)
        echo "Starting all services via Turborepo..."
        pnpm turbo dev
        ;;
    build)
        pnpm turbo build
        ;;
    lint)
        pnpm turbo lint
        ;;

    # Setup
    setup)
        echo "Installing Node dependencies..."
        pnpm install

        echo "Starting Docker containers..."
        docker compose up -d

        echo "Waiting for Postgres to be ready..."
        sleep 5

        echo "Running database migrations..."
        migrate -path db/migrations -database "$DATABASE_URL" up

        echo "Setup complete! Run './run.sh dev' to start all services."
        ;;

    # Help
    help)
        echo ""
        echo "LivePulse Task Runner"
        echo "Usage: ./run.sh <command>"
        echo ""
        echo "  Docker:"
        echo "    docker-up       Start Postgres + Redis containers"
        echo "    docker-down     Stop and remove containers"
        echo "    docker-ps       Show container status"
        echo ""
        echo "  Database:"
        echo "    migrate-up      Run all pending migrations"
        echo "    migrate-down    Rollback last migration"
        echo "    migrate-status  Show current migration version"
        echo ""
        echo "  Services:"
        echo "    api             Run the Go API service"
        echo "    realtime        Run the Go Realtime service"
        echo "    web             Run the Next.js dev server"
        echo "    api-build       Build API binary"
        echo "    realtime-build  Build Realtime binary"
        echo ""
        echo "  Turbo:"
        echo "    dev             Start all services via Turborepo"
        echo "    build           Build all apps"
        echo "    lint            Lint all apps"
        echo ""
        echo "  Setup:"
        echo "    setup           First-time project setup"
        echo ""
        ;;

    *)
        echo "Unknown command: $COMMAND"
        echo "Run './run.sh help' for available commands."
        ;;
esac
