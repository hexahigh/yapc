docker-build:
	echo "Building docker images"
	docker build --pull --no-cache --progress plain --rm -f "frontend/dockerfile" -t yapc-frontend:latest "frontend" &
	docker build --pull --no-cache --progress plain --rm -f "backend/dockerfile" -t yapc:latest "backend" &
	wait

docker-push:
	echo "Pushing docker images"
	docker image push docker.io/hexahigh/yapc:latest &
	docker image push docker.io/hexahigh/yapc-frontend:latest &
	wait

docker: docker-build docker-push