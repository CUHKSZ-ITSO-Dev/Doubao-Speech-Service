FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata ffmpeg

COPY --chmod=0755 doubao-speech-service-linux_amd64 .

CMD ["./doubao-speech-service-linux_amd64"]