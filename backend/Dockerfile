# Stage 1: Build Frontend
FROM node:23-alpine AS frontend
WORKDIR /app
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build
# → /app/dist/

# Stage 2: Build Backend
FROM golang:1.25-alpine AS backend
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

# Stage 3: Runtime
FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=backend /server .
COPY backend/db/migrations ./db/migrations
COPY --from=frontend /app/dist ./public
EXPOSE 8080
CMD ["./server"]
