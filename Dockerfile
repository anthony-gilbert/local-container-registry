FROM golang:1.24

# WORKDIR /usr/src/app
WORKDIR /app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY . .
RUN go mod download

# COPY . .

# Download and Install the dependencies
RUN go get -d -v ./...

# RUN go build -v -o /usr/local/bin/app ./...
RUN go build -o local-container-registry .

EXPOSE 8081

CMD ["./local-container-registry"]
