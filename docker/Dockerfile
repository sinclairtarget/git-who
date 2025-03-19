FROM golang:1.23-alpine AS builder

RUN mkdir /app
ADD . /app

RUN cd /app && go build -a -o git-who -buildvcs=false -ldflags '-s -w'

FROM alpine/git
RUN mkdir /app && git config --system --add safe.directory /git && git config --system alias.who '!/app/git-who'
COPY --from=builder /app/git-who /app/git-who
