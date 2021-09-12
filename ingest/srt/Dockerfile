FROM golang:1.17-bullseye

RUN apt-get update && apt-get install -y openssl libssl-dev pkg-config cmake tclsh git && rm -rf /var/lib/apt/lists/*

WORKDIR /srt
RUN git clone https://github.com/Haivision/srt
RUN cd srt && ./configure && make && make install
RUN rm -rf /srt

WORKDIR /rist
RUN git clone https://github.com/Haivision/srt
RUN cd rist && ./configure && make && make install
RUN rm -rf /srt

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...
RUN go build cmd/main.go

CMD ["/go/src/app/main"]