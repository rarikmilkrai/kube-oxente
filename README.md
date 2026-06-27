# koxente

Painel web **read-only** de troubleshooting para clusters Kubernetes/EKS. Roda localmente, abre no navegador e mostra em tempo real a saúde do cluster — agrupando apps com falha, apontando a **causa provável** e sugerindo os comandos de diagnóstico certos.

Escrito em Go puro, **zero dependências externas**: um binário único que usa o `kubectl` que você já tem autenticado.

---

## Por que usar

Depois de autenticar num cluster, a triagem normal é uma sequência manual de `kubectl get pods -A`, `describe`, `logs`, `logs --previous`, cruzar eventos e lembrar o que cada erro significa. O koxente faz tudo isso de uma vez, só com leitura, e apresenta numa interface fácil de navegar.

---

## Instalação

### Opção 1 — Binário pronto (sem Go instalado)

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

### Opção 2 — Compilar com Go 1.22+

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

## Requisitos

| Requisito | Detalhe |
|---|---|
| `kubectl` | Instalado e configurado (o koxente usa seu kubeconfig) |
| Acesso ao cluster | Permissão de leitura (`get`, `list`, `watch`) nos recursos |
| Go 1.22+ | **Somente** se for compilar — não necessário para usar o binário |

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
| `--context` | (atual) | Contexto kubeconfig inicial |
| `--logs` | `40` | Linhas de log no detalhe de cada pod |

O contexto também pode ser trocado direto no seletor no topo do painel — lista todos os contextos do kubeconfig e troca na hora, sem reiniciar.

---

## O que o painel mostra

| Seção | O que exibe |
|---|---|
| **Overview** | Cards: pods totais, running, pending, com falha, namespaces, nós, apps com problema. Barra de saúde, split por ambiente, workloads prontos, tipos de nó (spot/on-demand) |
| **Problemas** | Apps com falha agrupadas por (namespace + app + motivo), com diagnóstico provável e comandos sugeridos. Clicar abre painel lateral com eventos, logs limpos e diagnóstico detalhado |
| **Pods** | Tabela completa com filtro por ambiente, status e busca por nome/namespace |
| **Workloads** | Deployments, StatefulSets e DaemonSets com contagem ready/desired |
| **Services** | ClusterIP, tipo (ClusterIP/NodePort/LoadBalancer) e portas |
| **Namespaces** | Contagem de pods por namespace com breakdown de saúde |
| **Eventos** | Warning events do cluster inteiro, ordenados por frequência |
| **Nós** | Nome, tipo de instância, capacidade (spot/on-demand), zona e status ready |
| **Ingresses** | Hosts, endereço do load balancer, classe (nginx/alb/traefik), portas e TLS |
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

## Segurança

- **Somente leitura.** O koxente executa apenas `kubectl get` e `kubectl logs`. Não cria, edita nem apaga nada.
- O servidor web fica restrito a `127.0.0.1` — não exposto na rede.
- As permissões são exatamente as do seu kubeconfig — o koxente não amplia acesso.

---

## Estrutura do código

```
main.go      servidor HTTP e handlers da API REST
collect.go   coletores de cada seção (pods, workloads, CronJobs, HPAs, PVCs, ...)
triage.go    execução do kubectl, cálculo de status dos pods e engine de diagnóstico
web.go       interface completa (HTML/CSS/JS embutidos no binário)
```

---

## Roadmap

- [ ] Gráficos de uso de CPU/memória por pod e nó (via `kubectl top`)
- [ ] Filtro global de namespace no topo da interface
- [ ] Streaming de logs em tempo real (WebSocket)
- [ ] Exportar relatório HTML/Markdown para anexar em ticket
- [ ] ConfigMaps viewer (metadados e chaves)
- [ ] Resource Quotas por namespace
- [ ] Imagem Docker oficial para rodar sem instalar nada
- [ ] Aba de comparação entre contextos (multi-cluster side-by-side)
- [ ] Histórico de restarts com mini-gráfico de linha

---

## Licença

MIT — veja o arquivo [LICENSE](LICENSE).
