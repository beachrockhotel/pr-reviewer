FROM golang:1.25.4-alpine AS build

WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/pr-reviewer ./cmd/pr-reviewer

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/pr-reviewer /usr/local/bin/pr-reviewer
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/pr-reviewer"]
