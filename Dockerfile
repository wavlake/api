FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG COMMIT_SHA=unknown
ENV COMMIT_SHA=${COMMIT_SHA}

# Debug: List files to see what was copied
RUN ls -la /app
RUN ls -la /app/cmd || echo "cmd directory not found"
RUN ls -la /app/cmd/server || echo "cmd/server directory not found"

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.commitSHA=${COMMIT_SHA}" -o server ./cmd/server

FROM gcr.io/distroless/static-debian12

COPY --from=builder /app/server /server

ENV PORT=8080
ENV COMMIT_SHA=${COMMIT_SHA}

EXPOSE 8080

ENTRYPOINT ["/server"]