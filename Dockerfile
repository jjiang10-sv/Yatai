
FROM node:14.17.1 as frontendBuilder

WORKDIR /dashboard
COPY dashboard/ .
# COPY init.lock /app
#RUN npm install -g yarn
RUN yarn
RUN yarn build --silent


FROM golang:1.20 as compiler
USER root
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn/,https://goproxy.io/,direct
WORKDIR /src
COPY . .
WORKDIR /src/api-server
#-ldflags "-X google.golang.org/protobuf/reflect/protoregistry.conflictPolicy=warn -s -w"
#RUN GOOS=linux GOARCH=amd64 go build -tags musl  -o main ./cmd/*
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./main.go
RUN mkdir -p dist && \
    cp main dist 

FROM alpine:3.16.0
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
RUN apk add ca-certificates tzdata curl

WORKDIR /app

COPY --from=frontendBuilder  /dashboard/build ./dashboard/build
COPY --from=compiler  /src/api-server/dist/main ./api-server
COPY  api-server/db ./db
COPY  statics/ ./statics
COPY  scripts/ ./scripts

RUN chmod a+x ./api-server


################
# RUN mkdir /app

# RUN mkdir -p /app/dashboard
# RUN mkdir -p /app/scripts
# COPY ./statics /app/statics
# COPY ./dashboard/build /app/dashboard/build
# COPY ./api-server/db /app/db
# RUN echo $(ls -1 ./bin)
# #COPY ./bin/api-server /app/api-server
# COPY ./bin/bin /app/api-server
# RUN chmod a+x /app/api-server

# WORKDIR /app


