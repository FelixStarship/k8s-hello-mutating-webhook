#!/usr/bin/env sh
echo starship......
kubectl delete ns starship-mutating-webhook
sleep 30
kubectl create ns starship-mutating-webhook


kustomize build /app/k8s/other | kubectl apply -f -
kustomize build /app/k8s/csr | kubectl apply -f -
echo Waiting for cert creation ...
sleep 20
kubectl certificate approve starship-webhook-service.starship-mutating-webhook


(cd /app/k8s/deployment && \
	kustomize edit set image CONTAINER_IMAGE=docker-prod-registry.cn-hangzhou.cr.aliyuncs.com/cloudnative/starship-mutating-webhook:3.1.3)
	kustomize build /app/k8s/deployment | kubectl apply -f -




