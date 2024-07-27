# Use the official Golang image to create a build artifact
FROM golang:1.18 as builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go build -o main .

# Start a new stage from scratch
FROM golang:1.18

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/main .

# Copy templates directory if you have HTML templates
COPY --from=builder /app/templates ./templates

# Copy .env file
COPY --from=builder /app/.env ./.env

# Expose port 80 to the outside world
EXPOSE 80

# Command to run the executable
CMD ["./main"]
