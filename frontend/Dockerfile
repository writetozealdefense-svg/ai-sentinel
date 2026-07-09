# ---- Build stage ----
FROM node:20-alpine AS builder
WORKDIR /app

# Install deps (use lockfile when present for reproducible builds)
COPY package.json package-lock.json* ./
RUN npm ci || npm install

# Copy source and build the standalone output
COPY . .
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

# ---- Runtime stage ----
FROM node:20-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3000
ENV HOSTNAME=0.0.0.0

RUN addgroup --system --gid 1001 nodejs \
  && adduser --system --uid 1001 nextjs

# Standalone server + static assets
COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

# Bundled demo CSV must exist in the runtime image — the scenario1 BFF reads it
# from process.cwd()/src/data at request time.
COPY --from=builder --chown=nextjs:nodejs /app/src/data ./src/data

USER nextjs
EXPOSE 3000

CMD ["node", "server.js"]
