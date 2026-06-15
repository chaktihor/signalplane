FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
COPY cmd ./cmd
COPY internal ./internal
RUN go build -trimpath -ldflags="-s -w" -o /out/signalplane ./cmd/signalplane

FROM alpine:3.22
RUN adduser -D -H -u 10001 signalplane
RUN mkdir -p /data && chown signalplane:signalplane /data
USER signalplane
COPY --from=build /out/signalplane /usr/local/bin/signalplane
EXPOSE 4318
ENV SIGNALPLANE_ADDR=0.0.0.0:4318
ENV SIGNALPLANE_DATA_PATH=/data/signalplane.json
ENTRYPOINT ["signalplane"]
