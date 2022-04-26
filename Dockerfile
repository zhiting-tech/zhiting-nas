FROM golang:1.16-alpine as builder
RUN apk add build-base
COPY . /app
WORKDIR /app
RUN go env -w GOPROXY="goproxy.cn,direct"
RUN apk add libheif-dev
RUN GOOS=linux go build -ldflags="-w -s" -o wangpan

FROM alpine
WORKDIR /app
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add yasm && apk add ffmpeg
RUN apk add libheif-dev
COPY --from=builder /app/wangpan /app/wangpan
COPY --from=builder /app/app.yaml.example /app/app.yaml
ENTRYPOINT ["/app/wangpan"]