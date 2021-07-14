FROM alpine:3.12

# This makes it easy to build tagged images with different `kubectl` versions.

# docker-prod-registry.cn-hangzhou.cr.aliyuncs.com/cloudnative/starship-webhook:202107141919
# Fixes https://snyk.io/vuln/SNYK-LINUX-MUSL-458116
RUN apk upgrade musl

RUN apk add --update openssl
RUN wget http://dmp-test.oss-cn-shenzhen.aliyuncs.com/webhook/kubectl \
    && chmod +x ./kubectl && mv ./kubectl /usr/local/bin/kubectl && wget http://dmp-test.oss-cn-shenzhen.aliyuncs.com/webhook/kustomize \
    && chmod +x ./kustomize && mv ./kustomize /usr/local/bin/kustomize

COPY ./generate_mutating.sh /app/generate_mutating.sh
COPY ./k8s /app/
COPY ./k8s /app/k8s
RUN chmod +x /app/generate_mutating.sh

WORKDIR /app
USER 1000
CMD ["./generate_mutating.sh"]