FROM golang:1.23-alpine AS build

WORKDIR /app

# Only re-runs if go.mod or go.sum changes
COPY go.mod ./
RUN go mod download

COPY . ./
RUN go build -o /bin/api ./cmd/api

FROM alpine:3.20
WORKDIR /app

# Copy the built binary from the build stage
COPY --from=build /bin/api /usr/local/bin/api

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
