FROM golang:latest

COPY main.go /go/src

ENV PATH=$PATH:/usr/local/go/bin:$GOPATH/bin

RUN apt-get update && apt-get install git -y 

WORKDIR /go/src
RUN git clone https://github.com/ipfs/go-ipfs.git

WORKDIR /go/src/go-ipfs
RUN make install

CMD go run main.go
