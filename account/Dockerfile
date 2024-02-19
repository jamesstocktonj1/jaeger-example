# Use the official Golang image as the base image
FROM golang:latest

# Set the working directory inside the container
WORKDIR /app

# Copy the source code from the current directory to the working directory inside the container
COPY . .

# Download and install the Go dependencies
RUN go mod download

# Build the Go application
RUN go build -o main ./main.go

# Expose port 8080
EXPOSE 8080

# Set the entry point of the container to the built Go application
ENTRYPOINT ["./main"]
