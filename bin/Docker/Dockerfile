FROM ubuntu:16.04

ENV GOROOT=/usr/local/go
ENV GOPATH=/go
ENV PATH=$GOPATH/bin:$GOROOT/bin:$PATH

WORKDIR /

RUN apt-get update

RUN apt-get install curl wget build-essential autoconf libtool pkg-config unzip git libboost* libsodium-dev libgmp3-dev -y

# install golang
RUN wget https://dl.google.com/go/go1.10.2.linux-amd64.tar.gz
RUN tar -xvf go1.10.2.linux-amd64.tar.gz
RUN mv go /usr/local

RUN echo 'export GOROOT=$GOROOT' >> ~/.bashrc
RUN echo 'export GOPATH=$GOPATH' >> ~/.bashrc
RUN echo 'export PATH=$PATH' >> ~/.bashrc

RUN echo $GOROOT
RUN echo $GOPATH
RUN echo $PATH

RUN mkdir /go/bin -p
RUN mkdir /go/src/github.com/ninjadotorg -p

# install glide
RUN curl https://glide.sh/get | sh
# clone source from github.com/ninjadotorg/cash-prototype
RUN cd $GOPATH/src/github.com/ninjadotorg && git clone https://github.com/ninjadotorg/cash-prototype -b Parallel-PoS-Privacy
# install cash-prototype packages
RUN cd $GOPATH/src/github.com/ninjadotorg/cash-prototype && glide install

# install grpc
RUN git clone -b $(curl -L http://grpc.io/release) https://github.com/grpc/grpc
RUN cd grpc && git submodule update --init && make -j4 && make install

# install lib protobuf
RUN curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v3.6.1/protobuf-cpp-3.6.1.zip
RUN unzip protobuf-cpp-3.6.1.zip
RUN cd protobuf-3.6.1 && ./configure && make -j4 && make check && make install && ldconfig

# install lib cmake
RUN wget http://www.cmake.org/files/v3.5/cmake-3.5.2.tar.gz
RUN tar xf cmake-3.5.2.tar.gz
RUN cd cmake-3.5.2 && ./configure && make -j4 && make install

RUN cd $GOPATH/src/github.com/ninjadotorg/cash-prototype/privacy/server && mkdir build && cd build && cmake .. && \
    make -j4 && wget https://github.com/ninjadotorg/cash-prototype/releases/download/zkpp-v0.0.3/proving.key && \
    wget https://github.com/ninjadotorg/cash-prototype/releases/download/zkpp-v0.0.3/verifying.key

CMD ["top"]