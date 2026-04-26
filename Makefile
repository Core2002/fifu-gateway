TARGET = fifu-gateway
IMAGE_NAME = fifu-gateway
CONTAINER_NAME = fifu-gateway
VOLUME_NAME = fifu-gateway
APP_ENV ?= development 

# ========== 镜像源配置 ==========
GO_IMAGE_CN = docker.1ms.run/golang:1.25-alpine
ALPINE_IMAGE_CN = docker.1ms.run/alpine:latest

build-image-cn:
	podman build \
	 	--build-arg APP_ENV=$(APP_ENV) \
		--build-arg GO_IMAGE=$(GO_IMAGE_CN) \
		--build-arg ALPINE_IMAGE=$(ALPINE_IMAGE_CN) \
		-t $(IMAGE_NAME) --format docker .

build-image:
	podman build -t $(IMAGE_NAME) --build-arg APP_ENV=$(APP_ENV) --format docker .\

clean:
	podman stop $(CONTAINER_NAME) || true
	podman rm -f $(CONTAINER_NAME) || true
	podman rmi -f $(IMAGE_NAME) || true

# 若需添加管理员，进入容器操作数据库即可
# podman exec -it -u root fifu-gateway /bin/sh
run-container:
	podman run -d -v $(VOLUME_NAME):/app/data --network=host --name $(CONTAINER_NAME) --replace $(IMAGE_NAME)

.PHONY: build-image build-image-cn clean run-container
