TARGET = fifu-gateway
IMAGE_NAME = fifu-gateway
CONTAINER_NAME = fifu-gateway
VOLUME_NAME = fifu-gateway

# ========== 镜像源配置 ==========
GO_IMAGE_CN = docker.1ms.run/golang:1.25-alpine
ALPINE_IMAGE_CN = docker.1ms.run/alpine:latest

build-image-cn:
	podman build \
		--build-arg GO_IMAGE=$(GO_IMAGE_CN) \
		--build-arg ALPINE_IMAGE=$(ALPINE_IMAGE_CN) \
		-t $(IMAGE_NAME) --format docker .

build-image:
	podman build -t $(IMAGE_NAME) --format docker .\

clean:
	podman stop $(CONTAINER_NAME) || true
	podman rm -f $(CONTAINER_NAME) || true
	podman rmi -f $(IMAGE_NAME) || true

run-container:
	podman run -d -v $(VOLUME_NAME):/app/data --network=host --name $(CONTAINER_NAME) --replace $(IMAGE_NAME)

.PHONY: build-image build-image-cn clean run-container
