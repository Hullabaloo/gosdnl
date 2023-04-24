FROM golang:1.19

#RUN apk add --no-cache git

WORKDIR /app
copy go.mod .
copy go.sum .
RUN go mod download

COPY . .

CMD ["go", "run", "main.go"]