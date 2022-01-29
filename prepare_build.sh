docker build -t waldkauz-builder:latest .
docker run --rm  waldkauz-desktop-builder bash
docker cp  waldkauz-desktop-builder:/app/bin/ bin
docker cp  waldkauz-desktop-builder:/app/waldkauz-data-template/frontend waldkauz-data-template/frontend