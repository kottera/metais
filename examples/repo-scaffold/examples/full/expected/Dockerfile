FROM golang:1.25 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/weather-cli ./cmd/weather-cli

FROM scratch
COPY --from=build /out/weather-cli /weather-cli
ENTRYPOINT ["/weather-cli"]
