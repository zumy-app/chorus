# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Accept build arguments
ARG NGINX_CONFIG=nginx.conf

# Set as environment variable
ENV NODE_ENV=production

# Copy package files
COPY package*.json ./

# Install dependencies (include dev deps needed for the build)
RUN npm ci --include=dev

# Copy source code
COPY . .

# Build the application
RUN npm run build

# Production stage
FROM nginx:alpine

# Accept NGINX_CONFIG arg again (ARG doesn't carry between multi-stage builds)
ARG NGINX_CONFIG=nginx.conf

# Copy built files
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy nginx configuration
COPY $NGINX_CONFIG /etc/nginx/conf.d/default.conf

# Expose port
EXPOSE 3000

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
