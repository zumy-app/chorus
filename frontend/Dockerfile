# Build stage.
# NOTE: this Dockerfile expects the MONOREPO ROOT as build context
# (docker-compose.yml and CI pass `context: .`), because the app imports
# @chorus/shared via a tsconfig path alias to ../packages/shared — that
# directory is invisible when the context is ./frontend alone, and tsc
# fails with "Cannot find module '@chorus/shared'".
FROM node:20-alpine AS builder

WORKDIR /app

# Accept build arguments
ARG NGINX_CONFIG=nginx.conf

# Set as environment variable
ENV NODE_ENV=production

# Copy manifests first for layer caching
COPY frontend/package*.json ./frontend/
COPY packages/shared/package*.json ./packages/shared/

# Install dependencies (include dev deps needed for the build)
RUN npm ci --prefix packages/shared --no-audit --no-fund
RUN npm ci --prefix frontend --include=dev --no-audit --no-fund

# Copy source code
COPY frontend ./frontend
COPY packages/shared ./packages/shared

# Build the application
WORKDIR /app/frontend
RUN npm run build

# Production stage
FROM nginx:alpine

# Accept NGINX_CONFIG arg again (ARG doesn't carry between multi-stage builds)
ARG NGINX_CONFIG=nginx.conf

# Copy built files
COPY --from=builder /app/frontend/dist /usr/share/nginx/html

# Copy nginx configuration (path is relative to the monorepo-root context)
COPY frontend/$NGINX_CONFIG /etc/nginx/conf.d/default.conf

# Expose port
EXPOSE 3000

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
