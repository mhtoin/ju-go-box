# Use the official Golang image as the base image
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev pkgconfig opus-dev

# Set the working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o bot ./cmd/bot

# Use a minimal alpine image for the final stage
FROM alpine:latest

# Install required packages
RUN apk --no-cache add \
    ca-certificates \
    ffmpeg \
    python3 \
    yt-dlp

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/bot .

# Copy the .env file
COPY .env .

# Expose any necessary ports (if your bot needs them)
# EXPOSE 8080

# Run the bot
CMD ["./bot"] 