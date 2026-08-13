FROM ghcr.io/nubjs/nub:alpine AS frontend-builder
WORKDIR /app
COPY --chown=node:node package.json nub.lock ./
COPY --chown=node:node frontend/package.json ./frontend/
RUN nub ci
COPY  --chown=node:node frontend/ ./frontend/
RUN nub run build:frontend

FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY --from=frontend-builder /app/frontend/dist /app/frontend/dist
COPY . .
RUN CGO_ENABLED=0 go build -gcflags='all=-N -l' -o wago .

FROM alpine:3.20 AS runner
WORKDIR /app
COPY --from=builder /app/wago /app/wago
EXPOSE 8090
CMD ["/app/wago", "serve", "--http=0.0.0.0:8090"]