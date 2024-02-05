
FROM node:14.17.1 as frontendBuilder

WORKDIR /dashboard
COPY dashboard/ .
# COPY init.lock /app
#RUN npm install -g yarn
RUN yarn
RUN yarn build


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
#RUN echo $(ls -1 dist)


FROM alpine:3.16.0
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
RUN apk add ca-certificates tzdata curl
#ARG MODULE_NAME
RUN addgroup -S deploy && adduser -S deploy -G deploy
ARG ROOT_DIR=/yatai
WORKDIR ${ROOT_DIR}
RUN chown deploy:deploy ${ROOT_DIR}
RUN true
COPY --from=frontendBuilder --chown=deploy:deploy /dashboard/build ./dashboard/build
COPY --from=compiler --chown=deploy:deploy /src/api-server/dist ./api-server
#COPY internal/first/app.env .
#COPY --chown=deploy:deploy start.sh .  COPY ./api-server/db /app/db
COPY --chown=deploy:deploy api-server/db ./db
COPY --chown=deploy:deploy statics/ ./statics/
RUN true
#EXPOSE 8080
#ENV MODULE_NAME=first
USER deploy

# ENTRYPOINT /yatai/api-server/main


