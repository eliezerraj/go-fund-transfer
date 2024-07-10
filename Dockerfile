#docker build -t go-fund-transfer .
#docker run -dit --name go-fund-transfer -p 5000:5000 go-fund-transfer

FROM golang:1.22.4 As builder

RUN apt-get update && apt-get install bash && apt-get install -y --no-install-recommends ca-certificates

WORKDIR /app
COPY . .

WORKDIR /app/cmd
RUN go build -o go-fund-transfer -ldflags '-linkmode external -w -extldflags "-static"'

FROM alpine

WORKDIR /app
COPY --from=builder /app/cmd/go-fund-transfer .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

CMD ["/app/go-fund-transfer"]