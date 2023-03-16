IMAGE_NAME := ezkonnect-server
IMAGE_TAG := 0.0.1
DOCKER_REPO := yotamloe/$(IMAGE_NAME):$(IMAGE_TAG)
K8S_NAMESPACE := default

.PHONY: docker-build
docker-build:
	docker build -t $(DOCKER_REPO) .

.PHONY: docker-push
docker-push:
	docker push $(DOCKER_REPO)

.PHONY: deploy-kubectl
deploy-kubectl:
	kubectl apply -f deploy/k8s-manifest.yaml -n $(K8S_NAMESPACE)

.PHONY: clean-kubectl
clean-kubectl:
	kubectl delete -f deploy/k8s-manifest.yaml -n $(K8S_NAMESPACE)

.PHONY: local-server
local-server:
	go run main.go