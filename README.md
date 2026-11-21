# koxente

Painel web **read-only** de troubleshooting para clusters Kubernetes/EKS. Roda localmente, abre no navegador e mostra em tempo real a saúde do cluster — agrupando apps com falha, apontando a **causa provável** e sugerindo os comandos de diagnóstico certos.

Escrito em Go puro: um binário único que usa o `kubectl` que você já tem autenticado. Disponível como binário nativo **ou imagem Docker** — sem instalar nada além do Docker.

---

## Por que usar

Depois de autenticar num cluster, a triagem normal é uma sequência manual de `kubectl get pods -A`, `describe`, `logs`, `logs --previous`, cruzar eventos e lembrar o que cada erro significa. O koxente faz tudo isso de uma vez, só com leitura, e apresenta numa interface fácil de navegar.

---

## Instalação

### Opção 1 — Docker (sem instalar nada)

> Inclui `kubectl` e AWS CLI v2 na imagem — funciona com EKS sem configuração extra.

**Pré-requisitos:** Docker instalado e autenticado no cluster (`~/.kube` e `~/.aws` já configurados).

```bash
docker run -p 8200:8200 \
  -v ~/.kube:/root/.kube:ro \
  -v ~/.aws:/root/.aws:ro \
  rarikmilkrai/koxente:latest
```

Acesse **http://localhost:8200** no navegador.

> ⚠️ **Cluster local (minikube, kind, k3d)?** O Docker acima só funciona bem com cluster **remoto** (EKS, GKE, AKS, etc). Se o seu cluster roda na sua própria máquina, veja a seção [Usando com cluster local](#usando-com-cluster-local-minikube-kind-k3d) antes de continuar — tem um detalhe de rede que trava bastante gente iniciante.

**Variáveis de ambiente úteis:**

```bash
docker run -p 8200:8200 \
  -v ~/.kube:/root/.kube:ro \
  -v ~/.aws:/root/.aws:ro \
  -e AWS_PROFILE=meu-perfil \
  -e AWS_DEFAULT_REGION=sa-east-1 \
  rarikmilkrai/koxente:latest --context meu-cluster
```

**Com docker-compose** (salve como `docker-compose.yml`):

```yaml
services:
  koxente:
    image: rarikmilkrai/koxente:latest
    ports:
      - "8200:8200"
    volumes:
      - ${HOME}/.kube:/root/.kube:ro
      - ${HOME}/.aws:/root/.aws:ro
    environment:
      - AWS_PROFILE=${AWS_PROFILE:-default}
      - AWS_DEFAULT_REGION=${AWS_DEFAULT_REGION:-}
    restart: unless-stopped
```

```bash
docker-compose up -d
```

---

### Opção 2 — Binário pronto (sem Go instalado)

Baixe o binário da sua plataforma na [página de releases](../../releases):

| Plataforma | Arquivo |
|---|---|
| Linux x86-64 | `koxente-linux-amd64` |
| macOS Intel | `koxente-darwin-amd64` |
| macOS Apple Silicon (M1/M2/M3) | `koxente-darwin-arm64` |
| Windows x86-64 | `koxente-windows-amd64.exe` |

```bash
# Linux / macOS — após baixar:
chmod +x koxente-linux-amd64
mv koxente-linux-amd64 /usr/local/bin/koxente

# Windows — mova o .exe para uma pasta no PATH ou execute direto:
.\koxente-windows-amd64.exe
```

### Opção 3 — Compilar com Go 1.24+

```bash
git clone <url-do-repo> koxente
cd koxente
go build -o koxente .
```

Binário estático e enxuto (ideal para copiar entre máquinas):

```bash
CGO_ENABLED=0 go build -ldflags="-s -w" -o koxente .
```

#### Cross-compile para outras plataformas

```bash
# Linux
GOOS=linux  GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o koxente-linux-amd64 .

# macOS Intel
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o koxente-darwin-amd64 .

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o koxente-darwin-arm64 .

# Windows
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o koxente-windows-amd64.exe .
```

Como usa apenas a stdlib do Go, não há `go mod tidy` nem download de pacotes.

---

## Usando com cluster local (minikube, kind, k3d)

Se você tá começando, é bem comum testar tudo primeiro num cluster que roda na sua própria máquina (minikube, kind, k3d) antes de mexer num cluster de verdade. Só que rodar o koxente **via Docker** contra um cluster desses tem uma pegadinha — e ela não tem nada a ver com o koxente quebrado, é só como containers e rede funcionam.

**O que acontece:** o kubeconfig do minikube normalmente aponta o servidor da API pra `https://127.0.0.1:PORTA` — um endereço que só faz sentido *na sua máquina*. Só que um container Docker tem sua própria "rede isolada": o `127.0.0.1` de dentro do container `rarikmilkrai/koxente` aponta pro próprio container, **não** pro seu computador. Resultado: o koxente sobe, abre a tela, mas não consegue falar com o cluster — parece erro de conexão, mas na real é só esse desencontro de endereço.

**Como saber se é isso:** se o painel abre mas fica travado em "carregando" ou mostra erro tipo `connection refused` logo de cara, e seu cluster é minikube/kind/k3d, é praticamente certeza que é esse o motivo.

### Qual caminho escolher

| Seu cenário | O que usar |
|---|---|
| Cluster remoto (EKS, GKE, AKS, on-prem com IP de verdade) | Docker normal, [Opção 1](#opção-1--docker-sem-instalar-nada) lá em cima — funciona direto |
| Cluster local (minikube, kind, k3d) | **Binário nativo** (mais simples, sem pegadinha) — veja abaixo |

### Opção A — rodar sem Docker (recomendado pra cluster local)

Como o koxente é só um binário Go, sem containers e sem dependências, ele enxerga o `127.0.0.1` do seu kubeconfig igualzinho um `kubectl` normal — porque tecnicamente é a mesma coisa: um programa rodando direto na sua máquina.

```bash
git clone https://github.com/rarikmilkrai/kube-oxente.git koxente
cd koxente
go build -o koxente .
./koxente
```

Só precisa ter o [Go](https://go.dev/dl/) instalado (versão 1.24 ou mais nova — confira com `go version`). Depois é só abrir `http://127.0.0.1:8200` no navegador. Nenhuma outra configuração é necessária: se o `kubectl get pods` já funciona no seu terminal, o koxente também vai funcionar.

### Opção B — insistir no Docker, emprestando a rede do host

Se por algum motivo você quer usar Docker mesmo assim, no **Linux** (inclusive WSL2) dá pra fazer o container "pegar emprestada" a rede da sua máquina em vez de ter a dele própria, com a flag `--network host`:

```bash
docker run --network host \
  -v ~/.kube:/root/.kube:ro \
  rarikmilkrai/koxente:latest --port 8200
```

Assim o `127.0.0.1` de dentro do container passa a ser o mesmo `127.0.0.1` da sua máquina, e o kubeconfig do minikube funciona normalmente.

> ⚠️ **macOS (Docker Desktop):** essa opção não é confiável por lá — o suporte a `--network host` ainda é experimental no Docker Desktop pra Mac. Se você está num Mac, vá direto de Opção A.

---

## Requisitos

| Requisito | Detalhe |
|---|---|
| `kubectl` | Instalado e configurado (o koxente usa seu kubeconfig) |
| Acesso ao cluster | Permissão de leitura (`get`, `list`, `watch`) nos recursos |
| Go 1.24+ | **Somente** se for compilar — não necessário para usar o binário |

---

## Autenticação

O koxente **não gerencia credenciais** — ele usa o `kubectl` que já está autenticado na sua máquina. A forma de autenticar depende do seu ambiente:

### AWS EKS com credenciais temporárias (Control Tower / SSO)

1. Copie as credenciais do Control Tower e exporte no terminal:

```bash
export AWS_ACCESS_KEY_ID="ASIA..."
export AWS_SECRET_ACCESS_KEY="..."
export AWS_SESSION_TOKEN="..."
```

2. Atualize o kubeconfig:

```bash
aws eks update-kubeconfig --name meu-cluster --region sa-east-1
```

3. Rode o koxente normalmente.

> Se você usa um script de login (como `login-eks.sh`) que já faz esses passos, basta autenticar como de costume — o koxente funciona automaticamente após isso.

### AWS EKS com credenciais permanentes (chave IAM)

```bash
aws configure  # configura ~/.aws/credentials
aws eks update-kubeconfig --name meu-cluster --region sa-east-1
```

### Outros provedores (GKE, AKS, on-premise)

O koxente funciona com qualquer cluster acessível pelo `kubectl`. Configure o contexto normalmente (`gcloud container clusters get-credentials`, `az aks get-credentials`, etc.) e rode.

### EC2 / ambiente com IAM Role

Nenhuma configuração necessária. A role do servidor é usada automaticamente pelo AWS CLI/SDK.

---

## Uso

```bash
./koxente                             # abre em http://127.0.0.1:8200
./koxente --context betgo-prd         # inicia já neste contexto
./koxente --port 9000                 # troca a porta
./koxente --logs 80                   # mais linhas de log no detalhe do pod
```

Abra o endereço no navegador. O servidor escuta apenas em `127.0.0.1` — ninguém na rede acessa.

| Flag | Padrão | Descrição |
|---|---|---|
| `--port` | `8200` | Porta do painel web |
| `--host` | `127.0.0.1` | Endereço de escuta (`0.0.0.0` para Docker/rede) |
| `--context` | (atual) | Contexto kubeconfig inicial |
| `--logs` | `40` | Linhas de log no detalhe de cada pod |
| `--timeout` | `30s` | Tempo máximo de cada chamada ao `kubectl` (evita o painel travar sem VPN/credencial) |
| `--arch-secrets` | `false` | Na aba Arquitetura, lê os valores dos Secrets **só no servidor** para descobrir hosts de banco/cache/fila. Nenhum valor sai na API nem no Terraform |

O contexto também pode ser trocado direto no seletor no topo do painel — lista todos os contextos do kubeconfig e troca na hora, sem reiniciar.

---

## O que o painel mostra

| Seção | O que exibe |
|---|---|
| **Overview** | Cards: pods totais, running, pending, com falha, namespaces, nós, apps com problema. Barra de saúde, split por ambiente, workloads prontos, tipos de nó (spot/on-demand) |
| **Problemas** | Apps com falha agrupadas por (namespace + app + motivo), com diagnóstico provável e comandos sugeridos. Clicar abre painel lateral com eventos, logs limpos e diagnóstico detalhado |
| **Pods** | Tabela completa com filtro por ambiente, status, aplicação (dentro do namespace selecionado) e busca por nome/namespace. Clicar num pod abre painel com métricas de CPU/Mem em tempo real, eventos, logs (com "ao vivo") e as chaves de Secret que aquele pod referencia (nunca o valor) |
| **Workloads** | Deployments, StatefulSets e DaemonSets com contagem ready/desired |
| **Services** | ClusterIP, tipo (ClusterIP/NodePort/LoadBalancer), portas e contagem de endpoints prontos/não-prontos |
| **Namespaces** | Contagem de pods por namespace com breakdown de saúde |
| **Eventos** | Warning events do cluster inteiro, ordenados por frequência |
| **Nós** | Nome, tipo de instância, capacidade (spot/on-demand), zona e status ready |
| **Rede** | Services órfãos (sem endpoint pronto) ou com mismatch de porta, cobertura de NetworkPolicy por namespace, saúde do VPC CNI (aws-node) e config de SNAT/IP, réplicas e forwarders do CoreDNS, TargetGroupBinding/SecurityGroupPolicy (quando instalados) e Ingress — hosts, load balancer, classe, portas e TLS |
| **Arquitetura** | Desenho de quem fala com quem: Internet → LB → Ingress → Service → Workload, mais ConfigMaps/Secrets, PVCs, HPA, NetworkPolicy e dependências externas (RDS, ElastiCache, SQS, MongoDB Atlas…) descobertas nas variáveis de ambiente e ConfigMaps. Canvas interativo: arrastar, zoom, adicionar nós e conexões manuais, ocultar, anotar e exportar PNG/SVG (layout salvo no navegador por contexto). Cada workload mostra pods prontos, CPU, memória e restarts (via metrics-server); CrashLoop pisca em vermelho e o painel lateral leva direto ao diagnóstico com eventos e logs. **Raio de impacto** (tecla `i`): destaca tudo que para se aquele recurso cair. Em cluster grande os namespaces já abrem recolhidos (duplo clique expande). Exporta para **Mermaid** (cola direto em README/GitHub) e **draw.io**, além de PNG/SVG. Modo **Terraform**: gera o HCL (`hashicorp/kubernetes`) com blocos `import {}` e mostra o diagrama a partir do próprio HCL — edite o código e o desenho acompanha |
| **CronJobs** | Schedule, status de execução ativa, última execução, último sucesso — com detecção de falha (sem sucesso, último run falhou, suspenso) |
| **HPA** | Autoscalers com min/max/atual/desejado, destaque quando fora do range |
| **PVCs** | Status (Bound/Pending/Lost), capacidade, access modes e StorageClass |

Auto-atualização a cada 7 segundos (pode desligar com o botão "auto: on").

---

## Diagnósticos automáticos

A engine cruza o `reason` do pod + eventos + logs e aponta a causa provável com comandos de investigação:

**Banco de dados (PostgreSQL / MySQL — detectado automaticamente)**
- Senha errada no Secret (`password authentication failed`)
- Role ou database inexistente
- Conexões esgotadas (`max_connections`) — aponta PgBouncer como solução
- Política pg_hba / SSL mal configurado
- DNS de read replica (`cluster-ro-...`) apontando para ambiente errado
- Conectividade / Security Group

**Redis / ElastiCache**
- `NOAUTH` / `WRONGPASS`
- Réplica em readonly
- Limite de clientes atingido

**Imagem**
- Pull negado ou tag inexistente no ECR

**Runtime**
- OOMKilled (sugere aumentar limits)
- Probe falhando localmente (app não sobe, não é rede externa)
- Worker timeout do Hyperf (ERRNO 9101)
- Panic / fatal no boot

**AWS**
- IRSA / Pod Identity mal configurado
- Falha na montagem de Secret via CSI

---

## Publicar nova versão no Docker Hub

```bash
# 1. Login (uma vez)
docker login

# 2. Criar o builder multi-arch (uma vez)
docker buildx create --use --name multiarch

# 3. Build + push para linux/amd64 e linux/arm64
make docker-release VERSION=1.0.0

# ou apenas latest:
make docker-push
```

---

## Segurança

- **Somente leitura.** O koxente executa apenas `kubectl get` e `kubectl logs`. Não cria, edita nem apaga nada. O Terraform gerado é só texto — quem aplica é você.
- O servidor web fica restrito a `127.0.0.1` — não exposto na rede — e recusa requisições com `Host` diferente de localhost (proteção contra DNS rebinding).
- As permissões são exatamente as do seu kubeconfig — o koxente não amplia acesso.

---

## Estrutura do código

```
main.go      servidor HTTP e handlers da API REST
collect.go   coletores de cada seção (pods, workloads, CronJobs, HPAs, PVCs, rede, ...)
triage.go    execução do kubectl, cálculo de status dos pods e engine de diagnóstico
stream.go    streaming de logs em tempo real (SSE)
web.go       interface completa (HTML/CSS/JS embutidos no binário)
arch.go      aba Arquitetura: leitura do cluster e inferência de conexões
arch_tf.go   geração do Terraform a partir do cluster
static.go    arquivos de ui/ embutidos (go:embed) e proteção de Host
ui/          canvas da aba Arquitetura (arch.js / arch.css)
*_test.go    testes unitários (rodar com `go test ./...`)
```

---

## Roadmap

- [x] Gráficos de uso de CPU/memória por pod e nó (via `kubectl top`)
- [x] Filtro global de namespace no topo da interface
- [x] Streaming de logs em tempo real (SSE)
- [x] Exportar relatório HTML para anexar em ticket
- [x] ConfigMaps viewer (metadados e chaves)
- [x] Resource Quotas por namespace
- [x] Imagem Docker oficial (`rarikmilkrai/koxente`)
- [x] Histórico de restarts com mini-gráfico de linha
- [x] Filtro por aplicação dentro do namespace, na aba Pods
- [x] Métricas de CPU/Mem em tempo real no painel de detalhe do pod
- [x] Visualização das chaves de Secret referenciadas por um pod (sem expor valores)
- [x] Aba Rede: endpoints órfãos/mismatch de porta, cobertura de NetworkPolicy, saúde do VPC CNI, CoreDNS, TargetGroupBinding/SecurityGroupPolicy
- [x] Aba Arquitetura: diagrama interativo, saúde por workload, raio de impacto, recolher namespaces, exportar Mermaid/draw.io e gerar Terraform com import
- [ ] Aba de comparação entre contextos (multi-cluster side-by-side)
- [ ] Alertas configuráveis (threshold de restarts, pods pending há N min)

---

## Licença

MIT — veja o arquivo [LICENSE](LICENSE).
