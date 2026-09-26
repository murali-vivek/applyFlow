# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG SERVICE=api
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/applyflow ./cmd/${SERVICE}

# Fargate runs x86_64. Compile on the host, then pack a static binary so the
# final image does not need an emulated apk install.
FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/applyflow /app/applyflow
ENTRYPOINT ["/app/applyflow"]
