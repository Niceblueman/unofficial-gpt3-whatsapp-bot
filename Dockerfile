FROM golang:1.19.5-buster

WORKDIR /build
ADD https://raw.githubusercontent.com/Niceblueman/unofficial-gpt3-whatsapp-bot/main/go.mod go.mod
ADD https://raw.githubusercontent.com/Niceblueman/unofficial-gpt3-whatsapp-bot/main/go.sum go.sum
ADD https://raw.githubusercontent.com/Niceblueman/unofficial-gpt3-whatsapp-bot/main/main.go main.go
ADD https://raw.githubusercontent.com/Niceblueman/unofficial-gpt3-whatsapp-bot/main/api.go api.go
ADD https://raw.githubusercontent.com/Niceblueman/unofficial-gpt3-whatsapp-bot/main/send_qr_email.go send_qr_email.go
ADD https://raw.githubusercontent.com/Niceblueman/unofficial-gpt3-whatsapp-bot/main/admin.go admin.go
ADD https://raw.githubusercontent.com/Niceblueman/unofficial-gpt3-whatsapp-bot/main/apikey-manager.go apikey-manager.go
ADD https://raw.githubusercontent.com/Niceblueman/unofficial-gpt3-whatsapp-bot/main/genkey.pb.go genkey.pb.go
ADD https://raw.githubusercontent.com/Niceblueman/unofficial-gpt3-whatsapp-bot/main/genkey.proto genkey.proto
COPY store.db .
COPY .env .
COPY doc.md .
RUN go mod download
RUN go build -o ../main .

WORKDIR /
RUN rm -rf build
RUN go clean -modcache
EXPOSE 8385
ENTRYPOINT ["/main"]