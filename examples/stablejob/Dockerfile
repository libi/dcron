
FROM golang:alpine AS builder
LABEL stage=gobuilder
ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOPROXY https://goproxy.cn,direct
WORKDIR /build
ADD go.mod .
ADD go.sum .
RUN go mod download
COPY . .
RUN go build -o /app/stablejob examples/stablejob/stablejob.go

FROM alpine:latest
RUN apk update --no-cache
RUN apk add --no-cache bash
WORKDIR /app
COPY --from=builder /app/stablejob /app/stablejob
CMD [ "./stablejob" ]