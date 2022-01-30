FROM node:17-alpine
RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates

ARG KWOL_GIT_HASH=0
RUN git clone https://github.com/cloudhut/kowl /app
WORKDIR /app/
RUN git checkout ${KWOL_GIT_HASH}

WORKDIR /app/frontend

ENV PATH /app/frontend/node_modules/.bin:$PATH

RUN npm install
RUN npm run build