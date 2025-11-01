FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

ARG TARGETARCH
ARG TARGETVARIANT
ARG TARGETPLATFORM
ARG BUILDPLATFORM

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 打印调试信息（第一次构建建议保留，确认后可删）
RUN echo "TARGETARCH=$TARGETARCH TARGETVARIANT=$TARGETVARIANT TARGETPLATFORM=$TARGETPLATFORM"

# 根据架构复制对应 license 文件
RUN case "$TARGETARCH/$TARGETVARIANT" in \
      amd64/*) cp ./license/license_amd64 ./license ;; \
      arm64/*) cp ./license/license_arm64 ./license ;; \
      arm/v7)  cp ./license/license_armv7 ./license ;; \
      arm/*)   cp ./license/license_armv7 ./license ;; \
      *) echo "未知架构: $TARGETARCH/$TARGETVARIANT" && exit 1 ;; \
    esac

RUN  go build -o iptv main.go
RUN chmod +x /app/iptv

FROM alpine:latest

VOLUME /config
WORKDIR /app
EXPOSE 80 8080

ENV TZ=Asia/Shanghai
RUN apk add --no-cache openjdk8 bash curl tzdata sqlite;\
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