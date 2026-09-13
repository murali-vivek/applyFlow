# syntax=docker/dockerfile:1
FROM golang:1.24-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG SERVICE=api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/applyflow ./cmd/${SERVICE}

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/applyflow /app/applyflow
USER nobody
ENTRYPOINT ["/app/applyflow"]
