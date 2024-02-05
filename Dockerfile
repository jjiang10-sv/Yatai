FROM alpine:3.16.2

RUN mkdir /app

RUN mkdir -p /app/dashboard
RUN mkdir -p /app/scripts
COPY ./statics /app/statics
COPY ./dashboard/build /app/dashboard/build
COPY ./api-server/db /app/db
RUN echo $(ls -1 ./bin)
#COPY ./bin/api-server /app/api-server
COPY ./bin/bin /app/api-server
RUN chmod a+x /app/api-server

WORKDIR /app
