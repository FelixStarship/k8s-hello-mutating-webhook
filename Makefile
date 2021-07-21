WEBHOOK_SERVICE?=starship-webhook-service
NAMESPACE?=starship-mutating-webhook
CONTAINER_REPO?=docker-prod-registry.cn-hangzhou.cr.aliyuncs.com/cloudnative/starship-mutating-webhook
CONTAINER_VERSION?=3.1.9
CONTAINER_IMAGE=$(CONTAINER_REPO):$(CONTAINER_VERSION)


.PHONY: docker-build
docker-build:
	docker build -t $(CONTAINER_IMAGE) webhook


.PHONY: docker-push
docker-push:
	docker push $(CONTAINER_IMAGE) 


.PHONY: k8s-deploy
k8s-deploy: k8s-deploy-other  k8s-deploy-deployment


.PHONY: k8s-deploy-other
k8s-deploy-other:
	#kubectl create ns starship-mutating-webhook
	kustomize build k8s/other | kubectl apply -f -
	kustomize build k8s/csr | kubectl apply -f -
	@echo Waiting for cert creation ...
	@sleep 20
	kubectl certificate approve $(WEBHOOK_SERVICE).$(NAMESPACE)


.PHONY: k8s-deploy-deployment
k8s-deploy-deployment:
	(cd k8s/deployment && \
	kustomize edit set image CONTAINER_IMAGE=$(CONTAINER_IMAGE))
	kustomize build k8s/deployment | kubectl apply -f -


.PHONY: k8s-delete-all
k8s-delete-all:
	kustomize build k8s/other | kubectl delete --ignore-not-found=true -f  - 
	kustomize build k8s/csr | kubectl delete --ignore-not-found=true -f  - 
	kustomize build k8s/deployment | kubectl delete --ignore-not-found=true -f  - 
	kubectl delete --ignore-not-found=true csr $(WEBHOOK_SERVICE).$(NAMESPACE)
	kubectl delete --ignore-not-found=true secret starship-mutating-tls-secret
	kubectl delete MutatingWebhookConfiguration/mutating-webhook.starship.com
	#kubectl delete ns starship-mutating-webhook

.PHONY: test
test:
	cd webhook && go test ./...