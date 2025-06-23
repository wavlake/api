.PHONY: build run test clean docker-build docker-run deploy

BINARY_NAME=server
DOCKER_IMAGE=wavlake-api
COMMIT_SHA=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
PROJECT_ID=$(shell gcloud config get-value project 2>/dev/null)
REGION=us-central1
REPOSITORY=api-repo

build:
	go build -ldflags="-s -w -X main.commitSHA=$(COMMIT_SHA)" -o $(BINARY_NAME) ./cmd/server

run: build
	./$(BINARY_NAME)

test:
	go test -v ./...

clean:
	go clean
	rm -f $(BINARY_NAME)

docker-build:
	docker build -t $(REGION)-docker.pkg.dev/$(PROJECT_ID)/$(REPOSITORY)/api:$(COMMIT_SHA) --build-arg COMMIT_SHA=$(COMMIT_SHA) .

docker-push: docker-build
	docker push $(REGION)-docker.pkg.dev/$(PROJECT_ID)/$(REPOSITORY)/api:$(COMMIT_SHA)

docker-run: docker-build
	docker run -p 8080:8080 \
		-e GOOGLE_CLOUD_PROJECT=$(PROJECT_ID) \
		-e COMMIT_SHA=$(COMMIT_SHA) \
		$(REGION)-docker.pkg.dev/$(PROJECT_ID)/$(REPOSITORY)/api:$(COMMIT_SHA)

deploy:
	gcloud builds submit --config cloudbuild.yaml --substitutions=COMMIT_SHA=$(COMMIT_SHA)

fmt:
	go fmt ./...

vet:
	go vet ./...

lint: fmt vet
	@echo "Linting complete"