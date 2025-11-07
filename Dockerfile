FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

ARG TARGETARCH

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 根据架构复制对应 license 文件
RUN case "$TARGETARCH" in \
      amd64) cp ./license_all/license_amd64 ./license ;; \
      arm64) cp ./license_all/license_arm64 ./license ;; \
      arm)  cp ./license_all/license_armv7 ./license ;; \
      *) echo "未知架构: $TARGETARCH" && exit 1 ;; \
    esac

# 根据架构交叉编译
RUN if [ "$TARGETARCH" = "arm" ]; then \
        GOOS=linux GOARCH=arm GOARM=7 go build -o iptv main.go ; \
    else \
        GOOS=linux GOARCH=$TARGETARCH go build -o iptv main.go ; \
    fi

RUN chmod +x /app/iptv

FROM alpine:latest

VOLUME /config
WORKDIR /app
EXPOSE 80 8080

ENV TZ=Asia/Shanghai
RUN apk add --no-cache openjdk8 bash curl tzdata sqlite ffmpeg ;\
    cp /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone
    
COPY ./client /client
COPY ./apktool/* /usr/bin/
COPY ./static /app/static
COPY ./database /app/database
COPY ./config.yml /app/config.yml
COPY ./README.md  /app/README.md
COPY ./logo /app/logo
COPY ./ChangeLog.md /app/ChangeLog.md
COPY ./Version /app/Version

RUN chmod 777 -R /usr/bin/apktool* 

COPY --from=builder /app/iptv .
COPY --from=builder /app/license .

CMD ["./iptv"]