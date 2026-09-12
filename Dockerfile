# syntax=docker/dockerfile:1

FROM golang:1.27.1-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/demoapi ./cmd/demoapi \
    && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/demoapi /usr/local/bin/demoapi
COPY --from=build /out/migrate /usr/local/bin/migrate
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/demoapi"]
