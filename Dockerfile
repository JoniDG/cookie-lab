# 1. Frontend build. VITE_API_URL stays empty: the image serves the SPA and the
#    API from the same origin (mode A).
FROM node:24-alpine AS web
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# 2. Go build. go:embed cannot read outside backend/, so the build is copied
#    into backend/web/dist before compiling.
FROM golang:1.27-alpine AS api
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=web /src/frontend/dist/ ./web/dist/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/cookie-lab .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=api /out/cookie-lab /cookie-lab
EXPOSE 8080
ENTRYPOINT ["/cookie-lab"]
