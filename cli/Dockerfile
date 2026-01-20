FROM golang:1.24-alpine AS builder

RUN mkdir /usr/local/frags
WORKDIR /usr/local/frags

COPY . .

RUN go get
RUN go build -o frags *.go


FROM alpine
RUN mkdir /usr/local/frags
WORKDIR /usr/local/frags
COPY --from=builder /usr/local/frags/frags .

RUN addgroup -g 1000 frags && \
    adduser -D \
        -u 1000 \
        -G frags \
        -h /usr/local/frags \
        -s /bin/sh \
        frags

USER frags
WORKDIR /usr/local/frags
ENTRYPOINT [ "/usr/local/frags/frags" ]