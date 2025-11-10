FROM golang:alpine as build
# LABEL stage=builder
WORKDIR /root
# RUN apk add --no-cache git
# ARG GITHUB_USER
# ARG GITHUB_PAT
# RUN go env -w GOPRIVATE=github.com/PanQL &&\
#   go env -w GOPROXY=https://goproxy.cn,direct &&\
#   go env -w GOSUMDB=off &&\
#   git config --global --add url."https://${GITHUB_USER}:${GITHUB_PAT}@github.com/".insteadOf "https://github.com/"
# ADD go.mod go.sum /root/
# RUN go mod download -x
COPY . .
RUN go build -o tape ./cmd/tape


FROM alpine
RUN sed -i 's#https\?://dl-cdn.alpinelinux.org/alpine#https://mirrors.tuna.tsinghua.edu.cn/alpine#g' /etc/apk/repositories
RUN apk update
RUN apk add --no-cache yq
RUN mkdir -p /config
# COPY --from=build /root/tape_benchmark /usr/local/bin
COPY --from=build /root/tape /usr/local/bin
CMD ["tape"]
