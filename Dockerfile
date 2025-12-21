FROM golang:1.20-alpine AS build
WORKDIR /src
COPY . .
RUN go build -o /bin/creditengine ./cmd/creditengine

FROM alpine:3.18
COPY --from=build /bin/creditengine /bin/creditengine
CMD ["/bin/creditengine"]
