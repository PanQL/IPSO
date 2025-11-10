# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0

ARG GO_VER
ARG ALPINE_VER
FROM golang:${GO_VER}-alpine${ALPINE_VER} as golang

RUN apk add --no-cache \
	bash \
	gcc \
	git \
	make \
	boost-dev \
	gflags-dev \
    g++ \
	musl-dev;


    # RUN echo 'http://nl.alpinelinux.org/alpine/edge/testing' >> /etc/apk/repositories
RUN echo 'http://nl.alpinelinux.org/alpine/v3.10/main' >> /etc/apk/repositories
RUN apk update --allow-untrusted

RUN apk add --no-cache \ 
#   crypto++-dev=8.6.0-r0 \
    protobuf-dev=3.6.1-r1 \
    czmq-dev=4.1.1-r2

ADD . $GOPATH/src/github.com/hyperledger/fabric
WORKDIR $GOPATH/src/github.com/hyperledger/fabric

ENV USTORE_HOME=$GOPATH/src/github.com/hyperledger/fabric/ustore_home
ENV LD_LIBRARY_PATH=${USTORE_HOME}/lib:${LD_LIBRARY_PATH}
ENV LIBRARY_PATH=${USTORE_HOME}/lib:${LIBRARY_PATH}
ENV CPLUS_INCLUDE_PATH=${USTORE_HOME}/include:/usr/local/include/:${CPLUS_INCLUDE_PATH}
ARG GO_TAGS


FROM golang as tools
RUN make configtxgen configtxlator cryptogen peer discover idemixgen

FROM golang:${GO_VER}-alpine${ALPINE_VER}
RUN echo 'http://nl.alpinelinux.org/alpine/v3.10/main' >> /etc/apk/repositories
RUN apk update --allow-untrusted
RUN apk add --no-cache \
	bash \
	git \
	g++ \
	gcc \
	boost-dev \
	jq \
	protobuf-dev=3.6.1-r1 \
    czmq-dev=4.1.1-r2 \
	tzdata

COPY --from=tools /go/src/github.com/hyperledger/fabric/ustore_home /usr/ustore_home
ENV USTORE_HOME=/usr/ustore_home
ENV LD_LIBRARY_PATH=${USTORE_HOME}/lib:${LD_LIBRARY_PATH}
ENV FABRIC_CFG_PATH /etc/hyperledger/fabric
VOLUME /etc/hyperledger/fabric
COPY --from=tools /go/src/github.com/hyperledger/fabric/build/bin /usr/local/bin
COPY --from=tools /go/src/github.com/hyperledger/fabric/sampleconfig ${FABRIC_CFG_PATH}
