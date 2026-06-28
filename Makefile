IMAGE   ?= rarikmilkrai/koxente
VERSION ?= latest

# ── build local ─────────────────────────────────────────────────────────────
.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o koxente .

# binários para distribuição (todas as plataformas)
.PHONY: release
release:
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o koxente-linux-amd64 .
	GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o koxente-linux-arm64 .
	GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o koxente-darwin-amd64 .
	GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o koxente-darwin-arm64 .
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o koxente-windows-amd64.exe .

# ── Docker ───────────────────────────────────────────────────────────────────

# build local (somente a arquitetura da máquina atual, sem push)
.PHONY: docker-build
docker-build:
	docker build -t $(IMAGE):$(VERSION) .

# push multi-arch para o Docker Hub (requer: docker buildx + docker login)
# uso: make docker-push
#      make docker-push VERSION=1.2.0
.PHONY: docker-push
docker-push:
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--tag $(IMAGE):$(VERSION) \
		--push \
		.

# push com tag de versão + latest simultaneamente
# uso: make docker-release VERSION=1.2.0
.PHONY: docker-release
docker-release:
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--tag $(IMAGE):$(VERSION) \
		--tag $(IMAGE):latest \
		--push \
		.

# ── utilidades ───────────────────────────────────────────────────────────────
.PHONY: clean
clean:
	rm -f koxente koxente-linux-* koxente-darwin-* koxente-windows-*
