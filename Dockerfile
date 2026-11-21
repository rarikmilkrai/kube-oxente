# syntax=docker/dockerfile:1

# ── Stage 1: compilar o binário Go (estático, sem CGO) ─────────────────────
FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod .
COPY *.go ./
COPY ui ./ui
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o koxente .

# ── Stage 2: imagem de runtime ─────────────────────────────────────────────
FROM debian:bookworm-slim

# TARGETARCH é preenchido automaticamente pelo docker buildx (amd64 ou arm64)
ARG TARGETARCH=amd64

RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y --no-install-recommends \
      ca-certificates curl unzip && \
    rm -rf /var/lib/apt/lists/*

# kubectl — versão estável mais recente para a arquitetura alvo
RUN KVER=$(curl -fsSL https://dl.k8s.io/release/stable.txt) && \
    curl -fsSL "https://dl.k8s.io/release/${KVER}/bin/linux/${TARGETARCH}/kubectl" \
      -o /usr/local/bin/kubectl && \
    chmod +x /usr/local/bin/kubectl

# AWS CLI v2 — necessário para "aws eks get-token" usado nos kubeconfigs EKS
RUN AWSARCH=$([ "${TARGETARCH}" = "arm64" ] && echo "aarch64" || echo "x86_64") && \
    curl -fsSL "https://awscli.amazonaws.com/awscli-exe-linux-${AWSARCH}.zip" \
      -o /tmp/awscliv2.zip && \
    unzip -q /tmp/awscliv2.zip -d /tmp && \
    /tmp/aws/install --bin-dir /usr/local/bin --install-dir /usr/local/aws-cli && \
    rm -rf /tmp/awscliv2.zip /tmp/aws

COPY --from=builder /src/koxente /usr/local/bin/koxente

EXPOSE 8200

# --host 0.0.0.0 obrigatório para que o Docker exponha a porta corretamente
ENTRYPOINT ["koxente", "--host", "0.0.0.0"]
