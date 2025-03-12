FROM golang:1.24

# WORKDIR /usr/src/app
WORKDIR /app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# RUN go build -v -o /usr/local/bin/app ./...
RUN go build -o main .

EXPOSE 8080

CMD ["./main"]
