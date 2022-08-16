CONSOLE_GIT_HASH=1faa0ba497b27df13809cf107a565f127b6bd453

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