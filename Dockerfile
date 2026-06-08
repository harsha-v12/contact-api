# Build Stage
FROM golang:1.20-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -o contact-api ./cmd/main.go

# Run Stage
FROM alpine:latest  

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/contact-api .

# Expose port 8081 to the outside world
EXPOSE 8081

# Command to run the executable
CMD ["./contact-api"]
