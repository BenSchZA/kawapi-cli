############################
# STEP 1 build executable binary
############################
FROM golang:alpine AS builder
# Install git.
# Git is required for fetching the dependencies.
RUN apk update && apk add --no-cache make git gcc libc-dev ca-certificates
# Git is required for fetching the dependencies.
RUN mkdir -p /build
WORKDIR /build
COPY . .
# Fetch dependencies.
# Using go get.
RUN make fetch
# Build the binary.
RUN make build
############################
# STEP 2 build a small image
############################
FROM alpine:3.10
COPY --from=builder /build/static/ /static/
# Copy our static executable.
COPY --from=builder /build/bin/main /app/bin/main
# Import the root ca-certificates (required for Let's Encrypt)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
VOLUME ["/cert-cache"]
# Run the hello binary.
EXPOSE 443
EXPOSE 80
CMD ["/app/bin/main"]