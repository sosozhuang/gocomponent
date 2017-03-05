FROM busybox
MAINTAINER sosozhuang <"sosozhuang@163.com">
ADD component /opt/bin/component
ADD conf /opt/bin/conf
WORKDIR /opt/bin
EXPOSE 8086
CMD ["./component", "daemon", "start", "-p", "8086"]
