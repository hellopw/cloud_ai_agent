# Combined backend + frontend
FROM golang:1.22-alpine AS go-builder
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=0 go build -o server ./cmd/server/

FROM node:22-alpine AS frontend-builder
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

FROM nginx:alpine
RUN apk add --no-cache ca-certificates docker git
COPY --from=go-builder /app/server /server
COPY --from=frontend-builder /app/dist /usr/share/nginx/html
COPY frontend/nginx-combined.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["sh", "-c", "nginx && /server"]
