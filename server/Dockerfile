# builder stage
FROM golang:1.25 AS builder  
WORKDIR /app     
COPY go.mod go.sum ./ 
RUN go mod download
COPY . . 
RUN CGO_ENABLED=0 go build -o main .

#runtime stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .
EXPOSE 8080

CMD ["./main"]


