FROM golang:1.19.5-buster

WORKDIR /build
ADD https://raw.githubusercontent.com/KM8Oz/unofficial-gpt3-whatsapp-bot/main/go.mod go.mod
ADD https://raw.githubusercontent.com/KM8Oz/unofficial-gpt3-whatsapp-bot/main/go.sum go.sum
ADD https://raw.githubusercontent.com/KM8Oz/unofficial-gpt3-whatsapp-bot/main/main.go main.go

RUN go mod download
RUN go build -o ../main .

WORKDIR /
RUN rm -rf build
RUN go clean -modcache

ENTRYPOINT ["/main"]