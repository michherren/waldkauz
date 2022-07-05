CONSOLE_GIT_HASH=f95af628521e75c1edc7470dbf3fb79bda5bc26b

go get "github.com/redpanda-data/console@${CONSOLE_GIT_HASH}"
go mod tidy

docker build -t waldkauz-frontend-builder:latest --build-arg CONSOLE_GIT_HASH=$CONSOLE_GIT_HASH .
docker create --name waldkauz-frontend-builder waldkauz-frontend-builder
if [ -d "frontend_previous" ] ; then
    rm -rf "frontend_previous"
fi
if [ -d "frontend" ] ; then
    mv frontend frontend_previous
fi
docker cp  waldkauz-frontend-builder:/app/frontend/build frontend
docker rm waldkauz-frontend-builder 