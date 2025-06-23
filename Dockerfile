FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG COMMIT_SHA=unknown
ENV COMMIT_SHA=${COMMIT_SHA}

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.commitSHA=${COMMIT_SHA}" -o server ./cmd/server

FROM gcr.io/distroless/static-debian12

COPY --from=builder /app/server /server

ENV PORT=8080
ENV COMMIT_SHA=${COMMIT_SHA}

EXPOSE 8080

ENTRYPOINT ["/server"]