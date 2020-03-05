FROM golang:1.14.0-alpine AS builder
LABEL maintainer="patrick.bucher@stud.hslu.ch"
RUN apk add --no-cache ca-certificates git
COPY firstlink.go /src/
COPY go.mod /src/
WORKDIR /src
RUN go build -o /app/firstlink firstlink.go

FROM alpine:latest
LABEL maintainer="patrick.bucher@stud.hslu.ch"
RUN apk add --no-cache ca-certificates
RUN addgroup -g 1001 gophers && adduser -D -G gophers -u 1001 gopher
USER gopher
WORKDIR /home/gopher
COPY --from=builder /app/firstlink /home/gopher/firstlink
COPY assets /home/gopher/assets
ENV PORT=8080
EXPOSE $PORT
CMD ["/home/gopher/firstlink"]
