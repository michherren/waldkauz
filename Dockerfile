############################################################
# Frontend Build
############################################################
FROM node:17-buster as frontendBuilder
#RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates

RUN git clone https://github.com/cloudhut/kowl /kowl
WORKDIR /app

ENV PATH /app/node_modules/.bin:$PATH

#RUN cp /waldkauz/frontend/package.json ./package.json
#RUN cp /waldkauz/frontend/package-lock.json ./package-lock.json
#RUN cp /waldkauz/frontend/config-overrides.js ./config-overrides.js
RUN mv /kowl/frontend/* /app/
RUN ls -al 
RUN npm install

RUN npm run build

############################################################
# Backend Build
############################################################
FROM golang:1.17-alpine as builder
RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates

WORKDIR /app

COPY waldkauz-data-template ./
COPY --from=frontendBuilder /app/build/ /app/waldkauz-data-template/frontend

COPY ./go.mod .
COPY ./go.sum .
RUN go mod download

RUN go build -o ./bin/waldkauz main.go
RUN go build -ldflags "-H=windowsgui" -o ./bin/waldkauz.exe main.go

############################################################
# Final Image
############################################################
FROM alpine:3.13

WORKDIR /app

COPY --from=builder /app/bin/ /app/bin/
COPY --from=frontendBuilder COPY /app/waldkauz-data-template/ /app/waldkauz-data-template/