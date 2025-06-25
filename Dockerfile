# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o valhalla .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN adduser -D -s /bin/sh valhalla

# Set working directory
WORKDIR /home/valhalla

# Copy binary from builder stage
COPY --from=builder /app/valhalla .

# Copy configuration files
COPY --from=builder /app/README.md .
COPY --from=builder /app/LICENSE .

# Create directories for output and config
RUN mkdir -p output config && \
    chown -R valhalla:valhalla /home/valhalla

# Switch to non-root user
USER valhalla

# Expose any necessary ports (if adding web interface later)
# EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["./valhalla"]

# Default command
CMD ["--help"]

# Labels
LABEL org.opencontainers.image.title="Valhalla"
LABEL org.opencontainers.image.description="Hypervisor Infrastructure Discovery and IaC Generation Tool"
LABEL org.opencontainers.image.vendor="BigChiefRick"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.source="https://github.com/BigChiefRick/valhalla"
