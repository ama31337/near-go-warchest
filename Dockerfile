FROM ubuntu:18.04

RUN apt-get update -y && apt-get install -y wget libcurl4-openssl-dev libusb-1.0-0-dev libcurl3-gnutls libicu60 

SHELL ["/bin/bash", "--login", "-c"]

ENV NODE_ENV="guildnet"

ENV NVM_DIR /usr/local/nvm
ENV NODE_VERSION 14.0.0

WORKDIR $NVM_DIR

# install nvm
RUN wget -qO- https://raw.githubusercontent.com/nvm-sh/nvm/v0.35.3/install.sh | bash \
    && . $NVM_DIR/nvm.sh \
    && nvm install $NODE_VERSION \
    && nvm alias default $NODE_VERSION \
    && nvm use default

ENV NODE_PATH $NVM_DIR/versions/node/v$NODE_VERSION/lib/node_modules
ENV PATH      $NVM_DIR/versions/node/v$NODE_VERSION/bin:$PATH

# install near-shell
RUN npm install -g near-shell

# install go
RUN wget https://dl.google.com/go/go1.13.3.linux-amd64.tar.gz
RUN tar -xvf go1.13.3.linux-amd64.tar.gz
RUN mv go /usr/local
RUN echo 'export GOROOT=/usr/local/go' >> ~/.profile
RUN echo 'export GOPATH=$HOME/go' >> ~/.profile
RUN echo 'export PATH=$GOPATH/bin:$GOROOT/bin:$PATH' >> ~/.profile
RUN source ~/.profile

# build go-warchest
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /build

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -a -installsuffix cgo -ldflags="-w -s" -o go-warchest .

WORKDIR /dist

RUN cp /build/go-warchest .

EXPOSE 9444

CMD ["/dist/go-warchest"]