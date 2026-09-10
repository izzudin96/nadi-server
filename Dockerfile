# --- Frontend build ---
FROM node:22-alpine AS frontend
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ .
RUN npm run build

# --- Go build (embeds the frontend) ---
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /web/dist ./internal/webui/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o /out/nadi-server ./cmd/nadi-server

# --- Runtime ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/nadi-server /usr/local/bin/nadi-server
EXPOSE 8080
ENTRYPOINT ["nadi-server"]
