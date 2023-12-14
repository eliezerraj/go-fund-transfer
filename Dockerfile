#docker build -t go-fund-transfer .
#docker run -dit --name go-fund-transfer -p 5000:5000 go-fund-transfer

FROM golang:1.21 As builder

WORKDIR /app
COPY . .

WORKDIR /app/cmd
RUN go build -o go-fund-transfer -ldflags '-linkmode external -w -extldflags "-static"'

FROM alpine

WORKDIR /app
COPY --from=builder /app/cmd/go-fund-transfer .

CMD ["/app/go-fund-transfer"]