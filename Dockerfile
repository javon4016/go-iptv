# ================================
# Builder - 只负责准备不同架构的可执行文件
# ================================
FROM --platform=$BUILDPLATFORM alpine:latest AS builder

ARG TARGETARCH
WORKDIR /app

# 只复制实际需要用于选择架构的文件
COPY license_all/ license_all/
COPY iptv_dist/ iptv_dist/
COPY start_all/ start_all/

RUN case "$TARGETARCH" in \
      amd64) cp ./license_all/license_amd64 ./license \
          && cp ./iptv_dist/iptv_amd64 ./iptv \
          && cp ./start_all/start_amd64 ./start ;; \
      arm64) cp ./license_all/license_arm64 ./license \
          && cp ./iptv_dist/iptv_arm64 ./iptv \
          && cp ./start_all/start_arm64 ./start ;; \
      arm) cp ./license_all/license_arm ./license \
          && cp ./iptv_dist/iptv_arm ./iptv \
          && cp ./start_all/start_arm ./start ;; \
      *) echo "未知架构: $TARGETARCH" && exit 1 ;; \
    esac

RUN chmod +x iptv license start apktool/apktool*


# ================================
# Final Image
# ================================
FROM alpine:latest

ENV TZ=Asia/Shanghai
ENV ANDROID_HOME=/opt/android-sdk
ENV ANDROID_SDK_ROOT=/opt/android-sdk
ENV PATH=$PATH:/opt/android-sdk/build-tools


WORKDIR /app
VOLUME /config
EXPOSE 80 8080

# 应用资源
COPY apktool/apktool apktool/apktool.jar /usr/bin/
COPY client /client
COPY static database logo /app/

# 基础依赖
RUN apk add --no-cache \
    openjdk17 \
    bash \
    curl \
    wget \
    unzip \
    zip \
    ffmpeg \
    sqlite \
    libwebp-tools \
    libc6-compat \
    libstdc++ \
    tzdata \
    && cp /usr/share/zoneinfo/${TZ} /etc/localtime \
    && echo ${TZ} > /etc/timezone \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p ${ANDROID_HOME}/build-tools \
    && curl -L -o /tmp/build-tools.zip \
        https://dl.google.com/android/repository/build-tools_r33.0.2-linux.zip \
    && unzip /tmp/build-tools.zip -d  /tmp/build-tools \
    && mv /tmp/build-tools/android-13/* \
        ${ANDROID_HOME}/build-tools \
    && rm -rf /tmp/build-tools*

COPY config.yml README.md dictionary.txt alias.json ChangeLog.md Version MyTV.apk keystore.p12 /app/
COPY license_all/Version_lic /app/Version_lic

# Go 程序
COPY --from=builder /app/iptv /app/license /app/start /app/


CMD ["./start"]
