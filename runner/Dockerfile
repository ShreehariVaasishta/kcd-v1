# Use a Go base image
FROM golang:1.20

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files and download the dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the Go program source code
COPY . .

# Build the Go program
RUN go build -o main

# Set the entrypoint command to run the Go program
CMD ["./main"]
