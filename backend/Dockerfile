FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api \
 && CGO_ENABLED=0 go build -o /out/seed ./cmd/seed

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/ /app/
EXPOSE 8080
CMD ["/app/api"]
