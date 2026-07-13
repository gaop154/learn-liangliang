FROM golang:1.23-alpine AS build

ARG GOPROXY=https://proxy.golang.org,direct
ARG GOSUMDB=sum.golang.org

WORKDIR /src

COPY go.mod go.sum* ./
RUN GOPROXY="$GOPROXY" GOSUMDB="$GOSUMDB" go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/learn-api ./cmd/api \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/content-sync ./cmd/content-sync

FROM alpine:3.20

WORKDIR /app

COPY --from=build /out/learn-api /app/learn-api
COPY --from=build /out/content-sync /app/content-sync
COPY migrations /app/migrations

EXPOSE 8080

CMD ["/app/learn-api"]
