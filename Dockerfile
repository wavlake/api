FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Copy source code explicitly
COPY cmd/ cmd/
COPY internal/ internal/
COPY pkg/ pkg/

ARG COMMIT_SHA=unknown
ENV COMMIT_SHA=${COMMIT_SHA}

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.commitSHA=${COMMIT_SHA}" -o server ./cmd/server

FROM alpine:3.19

# Install ffmpeg and ca-certificates
RUN apk add --no-cache ffmpeg ca-certificates

COPY --from=builder /app/server /server

ENV PORT=8080
ENV COMMIT_SHA=${COMMIT_SHA}

EXPOSE 8080

ENTRYPOINT ["/server"]