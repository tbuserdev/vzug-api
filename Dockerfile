# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/vzug-ha ./cmd/vzug-ha

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/vzug-ha /vzug-ha
EXPOSE 3000
USER nonroot:nonroot
ENTRYPOINT ["/vzug-ha"]
