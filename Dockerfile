FROM busybox
MAINTAINER zhuangshuo <sosozhuang@163.com>

ADD gocomponent /usr/bin/gocomponent
ENTRYPOINT ["/usr/bin/gocomponent"]