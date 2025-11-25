FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

ARG TARGETARCH

WORKDIR /app

COPY . .

# 根据架构复制对应 license 文件
RUN case "$TARGETARCH" in \
      amd64) cp ./license_all/license_amd64 ./license && cp ./iptv_dist/iptv_amd64 /app/iptv && cp ./start_all/start_amd64 /app/start ;; \
      arm64) cp ./license_all/license_arm64 ./license && cp ./iptv_dist/iptv_arm64 /app/iptv && cp ./start_all/start_arm64 /app/start  ;; \
      arm)  cp ./license_all/license_arm ./license && cp ./iptv_dist/iptv_arm /app/iptv  && cp ./start_all/start_arm /app/start  ;; \
      *) echo "未知架构: $TARGETARCH" && exit 1 ;; \
    esac

RUN chmod +x /app/iptv /app/license /app/start

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
COPY ./alias.json /app/alias.json
COPY ./dictionary.txt /app/dictionary.txt

RUN chmod 777 -R /usr/bin/apktool*

COPY --from=builder /app/iptv .
COPY --from=builder /app/license .
COPY --from=builder /app/start .

CMD ["./start"]