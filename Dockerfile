# Start from a Golang base image
FROM golang:1.19-alpine AS build

# Set working directory
WORKDIR /app

# Copy API code to container
COPY . .

# Build the Go binary
RUN go build -o main .

# Create a new image from scratch
FROM alpine:3.14

# Copy the Go binary from the previous stage
COPY --from=build /app/main /app/main

# Expose port 8080
EXPOSE 5050

# Start the API server
CMD ["/app/main"]