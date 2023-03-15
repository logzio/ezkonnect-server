IMAGE_NAME := ezkonnect-server
IMAGE_TAG := 0.0.1
DOCKER_REPO := yotamloe/$(IMAGE_NAME):$(IMAGE_TAG)
K8S_NAMESPACE := default

.PHONY: build
build:
	docker build -t $(DOCKER_REPO) .

.PHONY: push
push:
	docker push $(DOCKER_REPO)

.PHONY: deploy
deploy:
	kubectl apply -f deploy/k8s-manifest.yaml -n $(K8S_NAMESPACE)

.PHONY: clean
clean:
	kubectl delete -f deploy/k8s-manifest.yaml -n $(K8S_NAMESPACE)
