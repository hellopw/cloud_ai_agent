# Combined backend + frontend
FROM golang:1.25-alpine AS go-builder
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG NO_PROXY
ENV HTTP_PROXY=$HTTP_PROXY
ENV HTTPS_PROXY=$HTTPS_PROXY
ENV NO_PROXY=$NO_PROXY
RUN apk add --no-cache git
WORKDIR /app
COPY backend/ .
RUN CGO_ENABLED=0 go build -mod=vendor -o server ./cmd/server/

FROM node:22-alpine AS frontend-builder
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG NO_PROXY
ENV HTTP_PROXY=$HTTP_PROXY
ENV HTTPS_PROXY=$HTTPS_PROXY
ENV NO_PROXY=$NO_PROXY
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

FROM nginx:alpine
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG NO_PROXY
ENV HTTP_PROXY=$HTTP_PROXY
ENV HTTPS_PROXY=$HTTPS_PROXY
ENV NO_PROXY=$NO_PROXY
RUN apk add --no-cache ca-certificates docker git
COPY --from=go-builder /app/server /server
COPY --from=frontend-builder /app/dist /usr/share/nginx/html
COPY frontend/nginx-combined.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["sh", "-c", "nginx && /server"]
