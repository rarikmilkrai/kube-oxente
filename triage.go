// triage.go — coleta (read-only) e diagnóstico. Sem dependência externa.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// ---------------- contexto ativo ----------------

var kubectlTimeout = 30 * time.Second

var (
	ctxMu     sync.RWMutex
	activeCtx string // contexto kubeconfig em uso (vazio = o atual do kubeconfig)
)

func setActiveContext(c string) {
	ctxMu.Lock()
	activeCtx = c
	ctxMu.Unlock()
	invalidateListCache()
}
func getActiveContext() string { ctxMu.RLock(); defer ctxMu.RUnlock(); return activeCtx }

// kubectl executa kubectl já injetando --context se houver um ativo.
func kubectl(args ...string) ([]byte, error) {
	if c := getActiveContext(); c != "" {
		args = append([]string{"--context", c}, args...)
	}
	ctx, cancel := context.WithTimeout(context.Background(), kubectlTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return out.Bytes(), fmt.Errorf("kubectl não respondeu em %s (VPN/credenciais?)", kubectlTimeout)
		}
		return out.Bytes(), fmt.Errorf("%s", strings.TrimSpace(errb.String()))
	}
	return out.Bytes(), nil
}

func listContexts() (current string, all []string) {
	out, _ := exec.Command("kubectl", "config", "current-context").Output()
	current = strings.TrimSpace(string(out))
	o2, _ := exec.Command("kubectl", "config", "get-contexts", "-o", "name").Output()
	for _, l := range strings.Split(strings.TrimSpace(string(o2)), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			all = append(all, l)
		}
	}
	sort.Strings(all)
	return
}

// ---------------- ambiente (prod/staging/dev) ----------------

var (
	reProd = regexp.MustCompile(`(?i)(prod|prd|production|live)`)
	reStg  = regexp.MustCompile(`(?i)(stg|staging|homolog|hml|hmg|uat|qa)`)
	reDev  = regexp.MustCompile(`(?i)(dev|develop|sandbox|sbx|test)`)
)

func classifyEnv(s string) string {
	switch {
	case reProd.MatchString(s):
		return "prod"
	case reStg.MatchString(s):
		return "staging"
	case reDev.MatchString(s):
		return "dev"
	default:
		return "other"
	}
}

var reEKSArn = regexp.MustCompile(`^arn:aws:eks:[^:]+:[0-9]+:cluster/(.+)$`)

// clusterShortName extrai o nome do cluster de um contexto kubeconfig. Em EKS o contexto
// costuma ser o ARN completo (arn:aws:eks:REGION:ACCOUNT:cluster/NOME) — isso é ilegível
// no header; aqui pegamos só o NOME. Fora do padrão EKS, devolve o contexto como veio.
func clusterShortName(ctx string) string {
	if m := reEKSArn.FindStringSubmatch(ctx); len(m) == 2 {
		return m[1]
	}
	return ctx
}

// ---------------- structs do kubectl -o json ----------------

type listResp struct {
	Items []json.RawMessage `json:"items"`
}

type pod struct {
	Metadata struct {
		Name              string            `json:"name"`
		Namespace         string            `json:"namespace"`
		CreationTimestamp time.Time         `json:"creationTimestamp"`
		DeletionTimestamp *time.Time        `json:"deletionTimestamp"`
		Labels            map[string]string `json:"labels"`
		OwnerReferences   []struct {
			Kind string `json:"kind"`
			Name string `json:"name"`
		} `json:"ownerReferences"`
	} `json:"metadata"`
	Spec struct {
		NodeName       string `json:"nodeName"`
		InitContainers []struct {
			Name string `json:"name"`
		} `json:"initContainers"`
		Containers []struct {
			Name  string `json:"name"`
			Ports []struct {
				Name          string `json:"name"`
				ContainerPort int    `json:"containerPort"`
			} `json:"ports"`
		} `json:"containers"`
	} `json:"spec"`
	Status struct {
		Phase                 string            `json:"phase"`
		Reason                string            `json:"reason"`
		ContainerStatuses     []containerStatus `json:"containerStatuses"`
		InitContainerStatuses []containerStatus `json:"initContainerStatuses"`
	} `json:"status"`
}

type containerStatus struct {
	Name         string `json:"name"`
	Image        string `json:"image"`
	Ready        bool   `json:"ready"`
	RestartCount int    `json:"restartCount"`
	State        struct {
		Waiting    *stateWaiting    `json:"waiting"`
		Terminated *stateTerminated `json:"terminated"`
		Running    *struct {
			StartedAt time.Time `json:"startedAt"`
		} `json:"running"`
	} `json:"state"`
	LastState struct {
		Terminated *stateTerminated `json:"terminated"`
	} `json:"lastState"`
}

type stateWaiting struct {
	Reason, Message string
}
type stateTerminated struct {
	Reason     string    `json:"reason"`
	Message    string    `json:"message"`
	ExitCode   int       `json:"exitCode"`
	Signal     int       `json:"signal"`
	FinishedAt time.Time `json:"finishedAt"`
}

func (w *stateWaiting) UnmarshalJSON(b []byte) error {
	var x struct {
		Reason  string `json:"reason"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(b, &x); err != nil {
		return err
	}
	w.Reason, w.Message = x.Reason, x.Message
	return nil
}

// ---------------- status (lógica do printer do kubectl) ----------------

type podView struct {
	pod
	reason   string
	restarts int
	ready    int
	total    int
	image    string
	lastExit time.Time
	severity int // 0 ok, 1 warn, 2 problema
}

func computeStatus(p pod) podView {
	pv := podView{pod: p}
	reason := p.Status.Phase
	if p.Status.Reason != "" {
		reason = p.Status.Reason
	}
	initializing := false
	for i, cs := range p.Status.InitContainerStatuses {
		switch {
		case cs.State.Terminated != nil && cs.State.Terminated.ExitCode == 0:
			continue
		case cs.State.Terminated != nil:
			if cs.State.Terminated.Reason != "" {
				reason = "Init:" + cs.State.Terminated.Reason
			} else if cs.State.Terminated.Signal != 0 {
				reason = fmt.Sprintf("Init:Signal:%d", cs.State.Terminated.Signal)
			} else {
				reason = fmt.Sprintf("Init:ExitCode:%d", cs.State.Terminated.ExitCode)
			}
			initializing = true
		case cs.State.Waiting != nil && cs.State.Waiting.Reason != "" && cs.State.Waiting.Reason != "PodInitializing":
			reason = "Init:" + cs.State.Waiting.Reason
			initializing = true
		default:
			reason = fmt.Sprintf("Init:%d/%d", i, len(p.Spec.InitContainers))
			initializing = true
		}
		break
	}
	pv.total = len(p.Status.ContainerStatuses)
	if !initializing {
		for i := len(p.Status.ContainerStatuses) - 1; i >= 0; i-- {
			cs := p.Status.ContainerStatuses[i]
			pv.restarts += cs.RestartCount
			if pv.image == "" {
				pv.image = cs.Image
			}
			if cs.LastState.Terminated != nil && cs.LastState.Terminated.FinishedAt.After(pv.lastExit) {
				pv.lastExit = cs.LastState.Terminated.FinishedAt
			}
			switch {
			case cs.State.Waiting != nil && cs.State.Waiting.Reason != "":
				reason = cs.State.Waiting.Reason
			case cs.State.Terminated != nil && cs.State.Terminated.Reason != "":
				reason = cs.State.Terminated.Reason
			case cs.State.Terminated != nil:
				if cs.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Signal:%d", cs.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("ExitCode:%d", cs.State.Terminated.ExitCode)
				}
			}
			if cs.Ready {
				pv.ready++
			}
		}
	} else {
		for _, cs := range p.Status.InitContainerStatuses {
			pv.restarts += cs.RestartCount
		}
	}
	if p.Metadata.DeletionTimestamp != nil {
		reason = "Terminating"
	}
	pv.reason = reason
	pv.severity = severityOf(pv)
	return pv
}

var problemReasons = map[string]bool{
	"CrashLoopBackOff": true, "Error": true, "ImagePullBackOff": true, "ErrImagePull": true,
	"ImageInspectError": true, "InvalidImageName": true, "CreateContainerConfigError": true,
	"CreateContainerError": true, "RunContainerError": true, "OOMKilled": true,
	"Evicted": true, "ContainerCannotRun": true,
}

func severityOf(pv podView) int {
	r := pv.reason
	if r == "Completed" || r == "Succeeded" {
		return 0
	}
	if problemReasons[r] || strings.HasPrefix(r, "Init:") ||
		strings.HasPrefix(r, "ExitCode:") || strings.HasPrefix(r, "Signal:") ||
		r == "Pending" || r == "Unknown" || r == "NodeLost" {
		return 2
	}
	if r == "Terminating" {
		return 1
	}
	if pv.total > 0 && pv.ready < pv.total {
		return 1
	}
	if pv.restarts >= 5 && (pv.lastExit.IsZero() || time.Since(pv.lastExit) < time.Hour) {
		return 1
	}
	return 0
}

// app = nome do workload dono (sem o hash do ReplicaSet), senão labels comuns.
func appName(p pod) string {
	for _, o := range p.Metadata.OwnerReferences {
		n := o.Name
		if o.Kind == "ReplicaSet" {
			if i := strings.LastIndex(n, "-"); i > 0 {
				n = n[:i]
			}
		}
		return n
	}
	for _, k := range []string{"app.kubernetes.io/name", "app", "k8s-app"} {
		if v := p.Metadata.Labels[k]; v != "" {
			return v
		}
	}
	return p.Metadata.Name
}

// ---------------- diagnóstico (com foco em banco) ----------------

type diag struct {
	Title string   `json:"title"`
	Cmds  []string `json:"cmds"`
}

var (
	reImagePull = regexp.MustCompile(`(?i)(imagepullbackoff|errimagepull|403 forbidden|pull access denied|manifest unknown|repository .* not found|invalidimagename|not found: name)`)
	rePg        = regexp.MustCompile(`(?i)(:5432|postgres|sqlstate\[08|pg_hba|fe_sendauth)`)
	reMy        = regexp.MustCompile(`(?i)(:3306|mysql|sqlstate\[hy000\]|sqlstate\[28000\])`)
	rePwFail    = regexp.MustCompile(`(?i)(password authentication failed|access denied for user|authentication failed|fe_sendauth: no password|invalid credentials)`)
	reRoleDB    = regexp.MustCompile(`(?i)(role ".*" does not exist|database ".*" does not exist|unknown database)`)
	reMaxConn   = regexp.MustCompile(`(?i)(too many clients|remaining connection slots|sorry, too many clients|max_connections|too many connections)`)
	rePgHba     = regexp.MustCompile(`(?i)(no pg_hba.conf entry|ssl (is )?required|ssl off|server does not support ssl)`)
	reDNS       = regexp.MustCompile(`(?i)(no such host|name resolution|could not translate host name|getaddrinfo|name or service not known|cluster-ro-)`)
	reExtConn   = regexp.MustCompile(`(?i)(connection refused.*:5432|connection refused.*:3306|connection timed out|i/o timeout|read timeout|could not connect to server|timeout expired)`)
	reProbe     = regexp.MustCompile(`(?i)probe failed.*dial tcp.*connect: connection refused`)
	reOOM       = regexp.MustCompile(`(?i)(oomkilled|out of memory|cannot allocate memory)`)
	reIRSA      = regexp.MustCompile(`(?i)(assumerolewithwebidentity|accessdenied.*sts|could not (retrieve|load) credentials|failedmount|secrets-store)`)
	reRedis     = regexp.MustCompile(`(?i)(redis.*(connection|refused|timeout)|NOAUTH|WRONGPASS|max number of clients reached|READONLY You can't write)`)
	reHyperf    = regexp.MustCompile(`(?i)(terminate_all_worker|errno 9101|wait timeout, all worker)`)
	rePanic     = regexp.MustCompile(`(?i)(panic:|fatal error:|uncaught|unhandledrejection|segmentation fault)`)
)

func engineName(text string) string {
	if rePg.MatchString(text) {
		return "PostgreSQL"
	}
	if reMy.MatchString(text) {
		return "MySQL"
	}
	return "banco"
}

func diagnose(reason, evText, logs string) []diag {
	text := reason + "\n" + evText + "\n" + logs
	var out []diag
	add := func(t string, cmds ...string) { out = append(out, diag{t, cmds}) }

	if reImagePull.MatchString(text) {
		add("Imagem ausente ou sem permissão no ECR. Causa comum: a tag (ex.: :latest) não foi publicada, ou a node role não tem pull.",
			"aws ecr describe-images --repository-name <repo> --region <região> --query 'imageDetails[].imageTags'",
			"kubectl get pod <pod> -n <ns> -o jsonpath='{.spec.containers[*].image}'")
		return out
	}

	eng := engineName(text)
	switch {
	case rePwFail.MatchString(text):
		add("Credencial do "+eng+" incorreta: senha no Secret errada/desatualizada (não é rede). Confirme e reaplique o Secret.",
			"kubectl get secret <secret> -n <ns> -o jsonpath='{.data.DB_PASSWORD}' | base64 -d; echo",
			"kubectl rollout restart deployment <deploy> -n <ns>")
	case reRoleDB.MatchString(text):
		add("Usuário (role) ou database inexistente no "+eng+". DB_USERNAME/DB_DATABASE apontam para algo que não existe nesse ambiente.",
			"# confira as vars do app:",
			"kubectl set env deployment/<deploy> -n <ns> --list | grep -i db")
	case reMaxConn.MatchString(text):
		add("Conexões esgotadas no "+eng+" (max_connections). Pool do app alto demais ou conexões vazando — comum quando KEDA escala muitos pods.",
			"# no banco: SELECT count(*) FROM pg_stat_activity;  /  SHOW max_connections;",
			"# considere PgBouncer/ProxySQL ou reduzir pool por pod")
	case rePgHba.MatchString(text):
		add("Política de conexão/SSL do "+eng+" (pg_hba/SSL). O servidor exige SSL ou rejeita a origem.",
			"# garanta sslmode=require na connection string do app")
	case reDNS.MatchString(text):
		add("Falha de DNS / read replica. Endpoint cluster-ro-... provavelmente não existe nesse ambiente. Aponte DB_READ_HOST para o endpoint de escrita.",
			"kubectl get configmap,secret -n <ns> | grep -i <app>")
	case reExtConn.MatchString(text):
		add("Conectividade com o "+eng+"/host externo. Verifique Security Group do RDS e se EKS e RDS estão na mesma VPC/subnet.",
			"kubectl exec <pod> -n <ns> -- nc -zv <host> <porta>")
	}

	if reRedis.MatchString(text) {
		add("Problema com Redis/ElastiCache: auth (NOAUTH/WRONGPASS), réplica em readonly, ou limite de clientes. Cheque o Secret do Redis e o endpoint (writer vs reader).")
	}
	if reProbe.MatchString(text) {
		add("App não está escutando na porta do probe — provavelmente trava/crasha no boot. O 'connection refused' é o probe batendo no próprio container, não rede externa. Veja a causa real nos logs.")
	}
	if reHyperf.MatchString(text) {
		add("Worker do Hyperf estourou timeout ao iniciar (ERRNO 9101): o boot trava numa dependência (DB/Redis/Secret) e o liveness derruba o pod.")
	}
	if reOOM.MatchString(text) {
		add("Estouro de memória (OOMKilled). Aumente limits/requests ou investigue vazamento.",
			"kubectl describe pod <pod> -n <ns> | grep -A3 -i limits")
	}
	if reIRSA.MatchString(text) {
		add("IRSA/Pod Identity ou montagem de Secret. Cheque a trust policy da role e a annotation eks.amazonaws.com/role-arn na ServiceAccount.",
			"kubectl describe sa <sa> -n <ns>")
	}
	if len(out) == 0 && rePanic.MatchString(text) {
		add("App crashou no boot (panic/fatal). Veja o stacktrace nos logs.")
	}
	return out
}

// ---------------- helpers ----------------

func age(t time.Time) string {
	if t.IsZero() {
		return "?"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func shortNode(n string) string {
	if n == "" {
		return "-"
	}
	return strings.Split(n, ".")[0]
}

func oneLine(s string) string { return strings.ReplaceAll(strings.TrimSpace(s), "\n", " ") }

var (
	reLogNoise = regexp.MustCompile(`(?i)(\[debug\]|afterworkerstart|beforeprocesshandle|initprocesstitle|worker#\d+ started|heartbeat ping|received pong|no tickets found|processed settlement|received message|processing settlement|received settlement|onmanagerstart|process\[)`)
	reLogErr   = regexp.MustCompile(`(?i)(error|fatal|panic|exception|fail|denied|refused|timeout|timed out|unavailable|cannot|unable|no such|sqlstate|errno|warning)`)
)

type logLine struct {
	Text  string `json:"text"`
	IsErr bool   `json:"isErr"`
}

func cleanLogLines(text string) []logLine {
	var lines []string
	for _, ln := range strings.Split(text, "\n") {
		t := strings.TrimSpace(ln)
		if t == "" || reLogNoise.MatchString(t) {
			continue
		}
		lines = append(lines, t)
	}
	type ce struct {
		s string
		n int
	}
	var dd []ce
	for _, ln := range lines {
		if len(dd) > 0 && dd[len(dd)-1].s == ln {
			dd[len(dd)-1].n++
			continue
		}
		dd = append(dd, ce{ln, 1})
	}
	var errL, allL []logLine
	for _, d := range dd {
		s := d.s
		if d.n > 1 {
			s += fmt.Sprintf("  (x%d)", d.n)
		}
		e := reLogErr.MatchString(d.s)
		allL = append(allL, logLine{s, e})
		if e {
			errL = append(errL, logLine{s, e})
		}
	}
	pick := errL
	if len(pick) < 3 {
		pick = allL
	}
	if len(pick) > 30 {
		pick = pick[len(pick)-30:]
	}
	return pick
}
