FROM golang:1.23-alpine AS build

WORKDIR /src

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/learn-api ./cmd/api

FROM alpine:3.20

WORKDIR /app

COPY --from=build /out/learn-api /app/learn-api
COPY migrations /app/migrations

EXPOSE 8080

CMD ["/app/learn-api"]
