FROM node:16.13-alpine
RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates

ARG CONSOLE_GIT_HASH=0
RUN git clone https://github.com/redpanda-data/console /app
WORKDIR /app/
RUN git checkout ${CONSOLE_GIT_HASH}

WORKDIR /app/frontend

ENV PATH /app/frontend/node_modules/.bin:$PATH

RUN npm install
RUN npm run build