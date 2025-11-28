ins:
	go mod tidy && go mod vendor

up:
	docker compose up -d

build:
	docker compose up --build -d

run:
	go run cmd/travel/main.go

swag:
	swag init -g /cmd/travel/main.go -o api

IMAGE ?= my-app
VERSION ?= latest
DOCKER_USER ?= c1789

.PHONY: docker-push
docker-push:
	docker build -t $(IMAGE):$(VERSION) .
	docker tag $(IMAGE):$(VERSION) $(DOCKER_USER)/$(IMAGE):$(VERSION)
	docker push $(DOCKER_USER)/$(IMAGE):$(VERSION)