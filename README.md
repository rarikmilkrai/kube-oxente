# koxente

Ferramenta de troubleshooting **read-only** para clusters Kubernetes/EKS. Sobe um painel web local (estilo Lens) que mostra a saúde do cluster, agrupa as aplicações com falha e já aponta a **causa provável** — DNS de read replica, credencial de banco errada, imagem ausente no ECR, OOM, probe falhando, conexões esgotadas, e mais.

Escrito em Go puro, **sem nenhuma dependência externa**: um binário único que usa o `kubectl` que você já tem autenticado.

## Por que

Depois de autenticar num cluster, normalmente a triagem é uma sequência manual de `kubectl get pods -A`, `describe`, `logs`, `logs --previous`, cruzar com eventos e lembrar o que cada erro significa. O koxente faz tudo isso de uma vez, só com `GET`, e mostra numa interface fácil de ler.

## Requisitos

- `kubectl` instalado e configurado (o koxente usa o seu kubeconfig).
- Go 1.21+ apenas para compilar (não é necessário se usar um binário pronto).
- Acesso de leitura ao cluster.

## Build

```bash
git clone <url-do-repo> koxente && cd koxente
go build -o koxente .
```

Binário estático e enxuto (bom para copiar entre máquinas):

```bash
CGO_ENABLED=0 go build -ldflags="-s -w" -o koxente .
```

Como é tudo stdlib, não há `go mod tidy` obrigatório nem download de pacotes.

## Uso

Autentique no cluster como de costume e rode:

```bash
./koxente                          # abre em http://127.0.0.1:8200
./koxente --context cluster-eks  # já inicia neste contexto
./koxente --port 9000              # troca a porta
./koxente --logs 80                # mais linhas de log no detalhe do pod
```

Abra o endereço no navegador. O servidor escuta apenas em `127.0.0.1` (ninguém na rede acessa).

| Flag        | Padrão | Descrição                                        |
|-------------|--------|--------------------------------------------------|
| `--port`    | `8200` | Porta do painel web.                             |
| `--context` | (atual)| Contexto kubeconfig inicial.                     |
| `--logs`    | `40`   | Linhas de log mostradas no detalhe de cada pod.  |

O contexto também pode ser trocado direto no seletor no topo do painel — ele lista todos os contextos do kubeconfig e troca na hora.

## O que o painel mostra

- **Overview** — contexto e ambiente (prod/staging/dev, detectado pelo nome), cards de pods (total, running, pending, com falha), nós, namespaces e apps com falha; barra de saúde; pods por ambiente; workloads prontos; tipos de nó (com spot/on-demand).
- **Problemas** — aplicações com falha agrupadas, com ambiente, motivo e diagnóstico provável. Clicar abre o painel lateral com eventos, diagnóstico, comandos sugeridos e logs limpos.
- **Pods** — tabela com filtro por ambiente, status e busca.
- **Workloads / Services / Namespaces / Eventos / Nós** — visão de cada recurso.
- Auto-atualização a cada poucos segundos (pode desligar).

## Diagnósticos cobertos

A engine cruza `reason` do pod + eventos + logs e sugere a causa provável:

- Banco: senha errada no Secret (Postgres/MySQL detectado automaticamente), role/database inexistente, conexões esgotadas (`max_connections`), pg_hba/SSL, DNS de read replica (`cluster-ro-...`), conectividade/Security Group.
- Redis/ElastiCache: `NOAUTH`/`WRONGPASS`, réplica em readonly, limite de clientes.
- Imagem: pull negado ou tag inexistente no ECR.
- Runtime: OOMKilled, probe local falhando (app não sobe), worker timeout (Hyperf), panic/fatal no boot.
- AWS: IRSA/Pod Identity, montagem de Secret via CSI.

## Segurança

- **Somente leitura.** O koxente apenas executa comandos `kubectl get`/`logs`. Não cria, edita nem apaga nada.
- O servidor web fica restrito a `127.0.0.1`.
- As permissões são as do seu kubeconfig — o koxente não amplia acesso.

## Estrutura

```
main.go       servidor HTTP + handlers da API
web.go        interface (HTML/CSS/JS embutidos)
collect.go    coletores de cada seção (overview, pods, workloads, ...)
triage.go     execução do kubectl, status dos pods e engine de diagnóstico
```

## Roadmap

- Filtro global de namespace no topo.
- Coluna de uso de CPU/memória (via `kubectl top`).
- Aba de CronJobs/Jobs.
- Exportar relatório (HTML/markdown) para anexar em ticket.

## Licença

Defina a licença do repositório (ex.: MIT) e adicione um arquivo `LICENSE`.