KOWL_GIT_HASH=0e7d7d546ee20b5c395d1350b5244d75f478dd81

go get "github.com/cloudhut/kowl/backend@${KOWL_GIT_HASH}"
go mod tidy

docker build -t waldkauz-frontend-builder:latest --build-arg KWOL_GIT_HASH=$KOWL_GIT_HASH .
docker create --name waldkauz-frontend-builder waldkauz-frontend-builder
if [ -d "waldkauz-data-template/frontend_previous" ] ; then
    rm -rf "waldkauz-data-template/frontend_previous"
fi
if [ -d "waldkauz-data-template/frontend" ] ; then
    mv waldkauz-data-template/frontend waldkauz-data-template/frontend_previous
fi
docker cp  waldkauz-frontend-builder:/app/frontend/build waldkauz-data-template/frontend
docker rm waldkauz-frontend-builder 