FROM node:20-alpine AS base

# Build context is the monorepo root (set in docker-compose.prod.yml)
# so we can access pnpm-lock.yaml and pnpm-workspace.yaml

FROM base AS deps
WORKDIR /app
COPY pnpm-lock.yaml pnpm-workspace.yaml package.json ./
COPY apps/web/package.json ./apps/web/package.json
RUN corepack enable pnpm && pnpm install --frozen-lockfile --filter web

FROM base AS builder
WORKDIR /app/apps/web
COPY --from=deps /app/node_modules /app/node_modules
COPY --from=deps /app/apps/web/node_modules ./node_modules
COPY apps/web/ .
ARG NEXT_PUBLIC_WS_URL
ENV NEXT_PUBLIC_WS_URL=$NEXT_PUBLIC_WS_URL
RUN corepack enable pnpm && pnpm run build

FROM base AS runner
WORKDIR /app
ENV NODE_ENV=production

RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

COPY --from=builder /app/apps/web/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/apps/web/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/apps/web/.next/static ./.next/static

USER nextjs
EXPOSE 3000
ENV PORT=3000

CMD ["node", "server.js"]
