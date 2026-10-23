# --- Stage 1: build (Node, discarded afterwards) ---
FROM node:24-alpine AS build
WORKDIR /app

# Copy only dependency files first so this layer is cached until they change.
COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

# --- Stage 2: runtime (nginx serves the static files and proxies /api to the Go API) ---
FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 3000
