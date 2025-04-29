FROM golang:1.24 AS builder

ENV CGO_ENABLED=0

WORKDIR /go/src/app

ADD . .

RUN go build -o /cert2cloud

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

COPY --from=builder /cert2cloud /cert2cloud

ENTRYPOINT ["/cert2cloud"]
