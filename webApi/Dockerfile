# Use an official Golang runtime as a parent image
FROM golang:1.21

# Set the working directory to /app
WORKDIR /app

# Copy the current directory contents into the container at /app
COPY . /app

RUN go mod tidy

# Build the Go app
RUN go build -o main .

# Expose port 8080 to the outside world
EXPOSE 8000

# Command to run the executable
CMD ["./main"]