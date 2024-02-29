FROM alpine:3.19

COPY --from=golang:1.22-alpine /usr/local/go/ /usr/local/go/
 
ENV PATH="/usr/local/go/bin:${PATH}"

RUN apk add --no-cache git

RUN git clone https://github.com/hexahigh/yapc.git /source

WORKDIR /source

RUN go build -o server ./backend/main.go

ENTRYPOINT [ "./server -d /data -db:file /data/yapc.db" ]