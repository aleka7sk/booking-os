FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/booking-os ./cmd/server

FROM alpine:3.21
RUN addgroup -S booking && adduser -S booking -G booking
WORKDIR /app
COPY --from=build /out/booking-os /usr/local/bin/booking-os
RUN mkdir -p /app/data && chown -R booking:booking /app
USER booking
ENV BOOKING_OS_ADDR=:8080 \
    BOOKING_OS_DATA_PATH=/app/data/booking-os.json
EXPOSE 8080
VOLUME ["/app/data"]
ENTRYPOINT ["/usr/local/bin/booking-os"]
