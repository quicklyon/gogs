FROM ysicing/goa AS binarybuilder
RUN sed -i 's#https://mirrors.aliyun.com#http://mirrors.tencent.com#g' /etc/apk/repositories \
  && apk update \
  && apk --no-cache --no-progress add --virtual \
  build-deps \
  build-base \
  git \
  linux-pam-dev

WORKDIR /gogs.io/gogs
COPY . .

RUN ./docker/build/install-task.sh
RUN TAGS="cert pam" task build

FROM alpine:3.14
RUN sed -i 's#dl-cdn.alpinelinux.org#mirrors.tencent.com#g' /etc/apk/repositories \
  && apk update \
  && apk --no-cache --no-progress add \
  bash \
  ca-certificates \
  curl \
  git \
  linux-pam \
  openssh \
  s6 \
  shadow \
  socat \
  tzdata \
  rsync

ENV GOGS_CUSTOM /data/gogs

# Configure LibC Name Service
COPY docker/nsswitch.conf /etc/nsswitch.conf

WORKDIR /app/gogs
COPY docker ./docker
COPY --from=binarybuilder /gogs.io/gogs/gogs .

RUN ./docker/build/finalize.sh

# Configure Docker Container
VOLUME ["/data", "/backup"]
EXPOSE 22 3000
HEALTHCHECK CMD (curl -o /dev/null -sS http://localhost:3000/healthcheck) || exit 1
ENTRYPOINT ["/app/gogs/docker/start.sh"]
CMD ["/bin/s6-svscan", "/app/gogs/docker/s6/"]
