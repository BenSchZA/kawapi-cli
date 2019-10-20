############################
# STEP 1 build executable binary
############################
FROM golang:alpine AS builder
# Install git.
# Git is required for fetching the dependencies.
RUN apk update && apk add --no-cache make git gcc libc-dev
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
FROM scratch
COPY --from=builder /build/static/ /static/
# Copy our static executable.
COPY --from=builder /build/bin/main /app/bin/main
# Run the hello binary.
EXPOSE 8080
CMD ["/app/bin/main"]