// collect.go — monta os payloads JSON de cada seção do painel.
package main

import (
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// --- cache de listas (evita múltiplos kubectl por request) ---

const listCacheTTL = 15 * time.Second

var (
	listCacheMu  sync.RWMutex
	listCacheMap = map[string]listCacheEntry{}
)

type listCacheEntry struct {
	items []json.RawMessage
	t     time.Time
	ctx   string
}

func invalidateListCache() {
	listCacheMu.Lock()
	listCacheMap = map[string]listCacheEntry{}
	listCacheMu.Unlock()
}

// clusterScoped lista recursos sem -A (evita warning do kubectl)
var clusterScoped = map[string]bool{"nodes": true}

func getList(kind string) []json.RawMessage {
	ctx := effectiveContext()

	listCacheMu.RLock()
	if e, ok := listCacheMap[kind]; ok && e.ctx == ctx && time.Since(e.t) < listCacheTTL {
		result := e.items
		listCacheMu.RUnlock()
		return result
	}
	listCacheMu.RUnlock()

	args := []string{"get", kind, "-A", "-o", "json"}
	if clusterScoped[kind] {
		args = []string{"get", kind, "-o", "json"}
	}
	out, err := kubectl(args...)
	if err != nil {
		return nil
	}
	var lr listResp
	if json.Unmarshal(out, &lr) != nil {
		return nil
	}

	listCacheMu.Lock()
	listCacheMap[kind] = listCacheEntry{items: lr.Items, t: time.Now(), ctx: ctx}
	listCacheMu.Unlock()

	return lr.Items
}

func allPods() []podView {
	var pvs []podView
	for _, raw := range getList("pods") {
		var p pod
		if json.Unmarshal(raw, &p) == nil {
			pvs = append(pvs, computeStatus(p))
		}
	}
	return pvs
}

// ---------------- overview ----------------

type kv struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

type overview struct {
	Ctx         string   `json:"ctx"`
	Env         string   `json:"env"`
	ClusterName string   `json:"clusterName"`
	Nodes       []string `json:"nodes"`
	NodesTotal  int      `json:"nodesTotal"`
	NodesDown   int      `json:"nodesDown"`
	Pods        struct {
		Total, Running, Pending, Failed, Succeeded int
		OK, Warn, Prob                             int
	} `json:"pods"`
	Workloads []workloadKind `json:"workloads"`
	NsCount   int            `json:"nsCount"`
	ProbApps  int            `json:"probApps"`
	EnvSplit  []kv           `json:"envSplit"`
	Generated string         `json:"generated"`
	Err       string         `json:"err,omitempty"`
}

type workloadKind struct {
	Kind  string `json:"kind"`
	Total int    `json:"total"`
	Ready int    `json:"ready"`
}

func nodeLines() (lines []string, total, down int) {
	types := map[string]int{}
	for _, raw := range getList("nodes") {
		var n struct {
			Metadata struct {
				Labels map[string]string `json:"labels"`
			} `json:"metadata"`
			Status struct {
				Conditions []struct{ Type, Status string } `json:"conditions"`
			} `json:"status"`
		}
		if json.Unmarshal(raw, &n) != nil {
			continue
		}
		total++
		t := n.Metadata.Labels["node.kubernetes.io/instance-type"]
		if t == "" {
			t = n.Metadata.Labels["beta.kubernetes.io/instance-type"]
		}
		if t == "" {
			t = "?"
		}
		c := n.Metadata.Labels["eks.amazonaws.com/capacityType"]
		if c == "" {
			c = n.Metadata.Labels["karpenter.sh/capacity-type"]
		}
		if c != "" {
			t += " (" + strings.ToLower(c) + ")"
		}
		types[t]++
		ready := false
		for _, cc := range n.Status.Conditions {
			if cc.Type == "Ready" && cc.Status == "True" {
				ready = true
			}
		}
		if !ready {
			down++
		}
	}
	var keys []string
	for k := range types {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		lines = append(lines, itoa(types[k])+"x "+k)
	}
	return
}

func collectOverview(ns string) overview {
	var o overview
	o.Generated = time.Now().Format("15:04:05")
	o.Ctx = effectiveContext()
	o.Env = classifyEnv(o.Ctx)
	o.ClusterName = clusterShortName(o.Ctx)
	o.Nodes, o.NodesTotal, o.NodesDown = nodeLines()

	pods := allPods()
	nsSet := map[string]bool{}
	appsBad := map[string]bool{}
	envc := map[string]int{"prod": 0, "staging": 0, "dev": 0, "other": 0}
	for _, pv := range pods {
		if ns != "" && pv.Metadata.Namespace != ns {
			continue
		}
		nsSet[pv.Metadata.Namespace] = true
		o.Pods.Total++
		switch pv.Status.Phase {
		case "Running":
			o.Pods.Running++
		case "Pending":
			o.Pods.Pending++
		case "Failed":
			o.Pods.Failed++
		case "Succeeded":
			o.Pods.Succeeded++
		}
		switch pv.severity {
		case 0:
			o.Pods.OK++
		case 1:
			o.Pods.Warn++
		case 2:
			o.Pods.Prob++
		}
		if pv.severity == 2 {
			appsBad[pv.Metadata.Namespace+"/"+appName(pv.pod)] = true
		}
		envc[classifyEnv(pv.Metadata.Namespace)]++
	}
	o.NsCount = len(nsSet)
	o.ProbApps = len(appsBad)
	for _, e := range []string{"prod", "staging", "dev", "other"} {
		if envc[e] > 0 {
			o.EnvSplit = append(o.EnvSplit, kv{e, envc[e]})
		}
	}
	o.Workloads = []workloadKind{
		wlReady("deployments", ns), wlReady("statefulsets", ns), wlReady("daemonsets", ns),
	}
	return o
}

func wlReady(kind, ns string) workloadKind {
	w := workloadKind{Kind: strings.TrimSuffix(kind, "s")}
	for _, raw := range getList(kind) {
		var d struct {
			Metadata struct {
				Namespace string `json:"namespace"`
			} `json:"metadata"`
			Status struct {
				ReadyReplicas      int `json:"readyReplicas"`
				Replicas           int `json:"replicas"`
				NumberReady        int `json:"numberReady"`
				DesiredNumberSched int `json:"desiredNumberScheduled"`
			} `json:"status"`
		}
		if json.Unmarshal(raw, &d) != nil {
			continue
		}
		if ns != "" && d.Metadata.Namespace != ns {
			continue
		}
		w.Total++
		ready := d.Status.ReadyReplicas
		desired := d.Status.Replicas
		if kind == "daemonsets" {
			ready = d.Status.NumberReady
			desired = d.Status.DesiredNumberSched
		}
		if desired > 0 && ready >= desired {
			w.Ready++
		} else if desired == 0 {
			w.Ready++
		}
	}
	return w
}

// ---------------- problemas (apps com falha) ----------------

type probApp struct {
	App          string   `json:"app"`
	NS           string   `json:"ns"`
	Env          string   `json:"env"`
	Reason       string   `json:"reason"`
	Severity     int      `json:"severity"`
	PodCount     int      `json:"podCount"`
	Restarts     int      `json:"restarts"`
	Ready        int      `json:"ready"`
	Total        int      `json:"total"`
	Age          string   `json:"age"`
	Node         string   `json:"node"`
	LastExit     string   `json:"lastExit"`
	SamplePod    string   `json:"samplePod"`
	SampleReason string   `json:"sampleReason"`
	Pods         []string `json:"pods"`
	PrimaryDiag  string   `json:"primaryDiag"`
}

func collectProblems() []probApp {
	pods := allPods()
	// agrupa por (ns + app + reason)
	type key struct{ ns, app, reason string }
	groups := map[key]*probApp{}
	var order []key
	for _, pv := range pods {
		if pv.severity == 0 {
			continue
		}
		k := key{pv.Metadata.Namespace, appName(pv.pod), pv.reason}
		g := groups[k]
		if g == nil {
			primary := pv.reason
			if ds := diagnose(pv.reason, "", ""); len(ds) > 0 {
				primary = ds[0].Title
			}
			le := ""
			if !pv.lastExit.IsZero() {
				le = age(pv.lastExit)
			}
			g = &probApp{
				App: k.app, NS: k.ns, Env: classifyEnv(k.ns), Reason: pv.reason,
				Severity: pv.severity, Restarts: pv.restarts, Ready: pv.ready, Total: pv.total,
				Age: age(pv.Metadata.CreationTimestamp), Node: shortNode(pv.Spec.NodeName),
				LastExit: le, SamplePod: pv.Metadata.Name, SampleReason: pv.reason,
				PrimaryDiag: primary,
			}
			groups[k] = g
			order = append(order, k)
		}
		g.PodCount++
		g.Pods = append(g.Pods, pv.Metadata.Name)
		if pv.restarts > g.Restarts {
			g.Restarts = pv.restarts
		}
	}
	out := make([]probApp, 0, len(order))
	for _, k := range order {
		out = append(out, *groups[k])
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Severity != out[j].Severity {
			return out[i].Severity > out[j].Severity
		}
		if out[i].Env != out[j].Env {
			return envRank(out[i].Env) < envRank(out[j].Env)
		}
		return out[i].App < out[j].App
	})
	return out
}

func envRank(e string) int {
	switch e {
	case "prod":
		return 0
	case "staging":
		return 1
	case "dev":
		return 2
	}
	return 3
}

// ---------------- namespaces ----------------

type nsRow struct {
	Namespace string `json:"namespace"`
	Env       string `json:"env"`
	Pods      int    `json:"pods"`
	Running   int    `json:"running"`
	Warn      int    `json:"warn"`
	Prob      int    `json:"prob"`
}

func collectNamespaces() []nsRow {
	pods := allPods()
	m := map[string]*nsRow{}
	var order []string
	for _, pv := range pods {
		ns := pv.Metadata.Namespace
		r := m[ns]
		if r == nil {
			r = &nsRow{Namespace: ns, Env: classifyEnv(ns)}
			m[ns] = r
			order = append(order, ns)
		}
		r.Pods++
		switch pv.severity {
		case 0:
			r.Running++
		case 1:
			r.Warn++
		case 2:
			r.Prob++
		}
	}
	sort.Strings(order)
	out := make([]nsRow, 0, len(order))
	for _, ns := range order {
		out = append(out, *m[ns])
	}
	return out
}

// ---------------- pods (tabela) ----------------

type podRow struct {
	NS       string `json:"ns"`
	Name     string `json:"name"`
	App      string `json:"app"`
	Reason   string `json:"reason"`
	Severity int    `json:"severity"`
	Ready    int    `json:"ready"`
	Total    int    `json:"total"`
	Restarts int    `json:"restarts"`
	Age      string `json:"age"`
	Node     string `json:"node"`
	Env      string `json:"env"`
}

func collectPods() []podRow {
	pods := allPods()
	out := make([]podRow, 0, len(pods))
	for _, pv := range pods {
		out = append(out, podRow{
			NS: pv.Metadata.Namespace, Name: pv.Metadata.Name, App: appName(pv.pod), Reason: pv.reason,
			Severity: pv.severity, Ready: pv.ready, Total: pv.total, Restarts: pv.restarts,
			Age: age(pv.Metadata.CreationTimestamp), Node: shortNode(pv.Spec.NodeName),
			Env: classifyEnv(pv.Metadata.Namespace),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Severity != out[j].Severity {
			return out[i].Severity > out[j].Severity
		}
		if out[i].NS != out[j].NS {
			return out[i].NS < out[j].NS
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// ---------------- workloads (tabela) ----------------

type wlRow struct {
	Kind    string `json:"kind"`
	NS      string `json:"ns"`
	Name    string `json:"name"`
	Ready   int    `json:"ready"`
	Desired int    `json:"desired"`
	Env     string `json:"env"`
	OK      bool   `json:"ok"`
}

func collectWorkloads() []wlRow {
	var out []wlRow
	for _, kind := range []string{"deployments", "statefulsets", "daemonsets"} {
		for _, raw := range getList(kind) {
			var d struct {
				Metadata struct{ Name, Namespace string } `json:"metadata"`
				Status   struct {
					ReadyReplicas      int `json:"readyReplicas"`
					Replicas           int `json:"replicas"`
					NumberReady        int `json:"numberReady"`
					DesiredNumberSched int `json:"desiredNumberScheduled"`
				} `json:"status"`
			}
			if json.Unmarshal(raw, &d) != nil {
				continue
			}
			ready, desired := d.Status.ReadyReplicas, d.Status.Replicas
			if kind == "daemonsets" {
				ready, desired = d.Status.NumberReady, d.Status.DesiredNumberSched
			}
			out = append(out, wlRow{
				Kind: strings.TrimSuffix(kind, "s"), NS: d.Metadata.Namespace, Name: d.Metadata.Name,
				Ready: ready, Desired: desired, Env: classifyEnv(d.Metadata.Namespace),
				OK: desired == 0 || ready >= desired,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].OK != out[j].OK {
			return !out[i].OK
		}
		return out[i].NS+out[i].Name < out[j].NS+out[j].Name
	})
	return out
}

// ---------------- services ----------------

type svcRow struct {
	NS           string   `json:"ns"`
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	IP           string   `json:"ip"`
	Ports        string   `json:"ports"`
	Env          string   `json:"env"`
	ReadyEps     int      `json:"readyEps"`
	NotReadyEps  int      `json:"notReadyEps"`
	Orphan       bool     `json:"orphan"`       // tem selector mas zero endpoint pronto
	PortMismatch []string `json:"portMismatch"` // portas do Service sem container correspondente
}

// intOrString cobre o campo targetPort do Service, que no JSON vem como número ou string (porta nomeada).
type intOrString struct {
	IntVal int
	StrVal string
	IsStr  bool
}

func (v *intOrString) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '"' {
		v.IsStr = true
		return json.Unmarshal(b, &v.StrVal)
	}
	return json.Unmarshal(b, &v.IntVal)
}

func matchSelector(sel, labels map[string]string) bool {
	if len(sel) == 0 {
		return false
	}
	for k, v := range sel {
		if labels[k] != v {
			return false
		}
	}
	return true
}

// anyContainerHasPort verifica se algum container dos pods casados declara a porta-alvo.
// portsDeclared indica se havia alguma porta declarada nos pods (pra não falsear "mismatch"
// quando o pod simplesmente não declara containerPort, o que é opcional e comum em k8s).
func anyContainerHasPort(pods []pod, tp *intOrString, fallback int) (found, portsDeclared bool) {
	for _, p := range pods {
		for _, c := range p.Spec.Containers {
			for _, cp := range c.Ports {
				portsDeclared = true
				if tp != nil && tp.IsStr {
					if cp.Name == tp.StrVal {
						found = true
					}
				} else {
					want := fallback
					if tp != nil {
						want = tp.IntVal
					}
					if cp.ContainerPort == want {
						found = true
					}
				}
			}
		}
	}
	return
}

func collectEndpointCounts() map[string][2]int {
	out := map[string][2]int{}
	for _, raw := range getList("endpoints") {
		var e struct {
			Metadata struct{ Name, Namespace string } `json:"metadata"`
			Subsets  []struct {
				Addresses         []struct{} `json:"addresses"`
				NotReadyAddresses []struct{} `json:"notReadyAddresses"`
			} `json:"subsets"`
		}
		if json.Unmarshal(raw, &e) != nil {
			continue
		}
		var ready, notReady int
		for _, ss := range e.Subsets {
			ready += len(ss.Addresses)
			notReady += len(ss.NotReadyAddresses)
		}
		out[e.Metadata.Namespace+"/"+e.Metadata.Name] = [2]int{ready, notReady}
	}
	return out
}

func collectServices() []svcRow {
	pods := allPods()
	epCounts := collectEndpointCounts()

	var out []svcRow
	for _, raw := range getList("services") {
		var s struct {
			Metadata struct{ Name, Namespace string } `json:"metadata"`
			Spec     struct {
				Type      string            `json:"type"`
				ClusterIP string            `json:"clusterIP"`
				Selector  map[string]string `json:"selector"`
				Ports     []struct {
					Port       int          `json:"port"`
					Protocol   string       `json:"protocol"`
					TargetPort *intOrString `json:"targetPort"`
				} `json:"ports"`
			} `json:"spec"`
		}
		if json.Unmarshal(raw, &s) != nil {
			continue
		}
		var ports []string
		for _, p := range s.Spec.Ports {
			ports = append(ports, itoa(p.Port)+"/"+p.Protocol)
		}
		eps := epCounts[s.Metadata.Namespace+"/"+s.Metadata.Name]
		row := svcRow{
			NS: s.Metadata.Namespace, Name: s.Metadata.Name, Type: s.Spec.Type,
			IP: s.Spec.ClusterIP, Ports: strings.Join(ports, ","), Env: classifyEnv(s.Metadata.Namespace),
			ReadyEps: eps[0], NotReadyEps: eps[1],
		}

		if len(s.Spec.Selector) > 0 && s.Spec.Type != "ExternalName" {
			row.Orphan = eps[0] == 0

			var matched []pod
			for _, pv := range pods {
				if pv.Metadata.Namespace == s.Metadata.Namespace && matchSelector(s.Spec.Selector, pv.Metadata.Labels) {
					matched = append(matched, pv.pod)
				}
			}
			if len(matched) > 0 {
				for _, p := range s.Spec.Ports {
					found, declared := anyContainerHasPort(matched, p.TargetPort, p.Port)
					if declared && !found {
						row.PortMismatch = append(row.PortMismatch, itoa(p.Port))
					}
				}
			}
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NS+out[i].Name < out[j].NS+out[j].Name })
	return out
}

// ---------------- events (warning, cluster-wide) ----------------

type evRow struct {
	NS      string `json:"ns"`
	Type    string `json:"type"`
	Reason  string `json:"reason"`
	Object  string `json:"object"`
	Message string `json:"message"`
	Count   int    `json:"count"`
	Age     string `json:"age"`
}

func collectEvents() []evRow {
	var out []evRow
	for _, raw := range getList("events") {
		var e struct {
			Metadata       struct{ Namespace string }  `json:"metadata"`
			Type           string                      `json:"type"`
			Reason         string                      `json:"reason"`
			Message        string                      `json:"message"`
			Count          int                         `json:"count"`
			LastTimestamp  time.Time                   `json:"lastTimestamp"`
			InvolvedObject struct{ Kind, Name string } `json:"involvedObject"`
		}
		if json.Unmarshal(raw, &e) != nil || e.Type != "Warning" {
			continue
		}
		out = append(out, evRow{
			NS: e.Metadata.Namespace, Type: e.Type, Reason: e.Reason,
			Object:  e.InvolvedObject.Kind + "/" + e.InvolvedObject.Name,
			Message: oneLine(e.Message), Count: e.Count, Age: age(e.LastTimestamp),
		})
	}
	// mais recentes primeiro não temos timestamp aqui; ordena por contagem desc
	sort.Slice(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	if len(out) > 200 {
		out = out[:200]
	}
	return out
}

// ---------------- nodes (tabela) ----------------

type ndRow struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Cap   string `json:"cap"`
	Zone  string `json:"zone"`
	Ready bool   `json:"ready"`
	Age   string `json:"age"`
}

func collectNodes() []ndRow {
	var out []ndRow
	for _, raw := range getList("nodes") {
		var n struct {
			Metadata struct {
				Name              string            `json:"name"`
				Labels            map[string]string `json:"labels"`
				CreationTimestamp time.Time         `json:"creationTimestamp"`
			} `json:"metadata"`
			Status struct {
				Conditions []struct{ Type, Status string } `json:"conditions"`
			} `json:"status"`
		}
		if json.Unmarshal(raw, &n) != nil {
			continue
		}
		t := n.Metadata.Labels["node.kubernetes.io/instance-type"]
		if t == "" {
			t = n.Metadata.Labels["beta.kubernetes.io/instance-type"]
		}
		c := n.Metadata.Labels["eks.amazonaws.com/capacityType"]
		if c == "" {
			c = n.Metadata.Labels["karpenter.sh/capacity-type"]
		}
		ready := false
		for _, cc := range n.Status.Conditions {
			if cc.Type == "Ready" && cc.Status == "True" {
				ready = true
			}
		}
		out = append(out, ndRow{
			Name: n.Metadata.Name, Type: t, Cap: strings.ToLower(c),
			Zone: n.Metadata.Labels["topology.kubernetes.io/zone"], Ready: ready,
			Age: age(n.Metadata.CreationTimestamp),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ---------------- detalhe de um pod ----------------

type detail struct {
	Events  []evRow     `json:"events"`
	Diags   []diag      `json:"diags"`
	Logs    []logLine   `json:"logs"`
	Secrets []secretRef `json:"secrets"`
}

func collectDetail(ns, name, reason string, logLines int) detail {
	var d detail
	d.Secrets = collectPodSecrets(ns, name)
	out, _ := kubectl("get", "events", "-n", ns, "--field-selector", "involvedObject.name="+name, "-o", "json")
	var lr listResp
	json.Unmarshal(out, &lr)
	type ev struct {
		Type, Reason, Message string
		Count                 int
		LastTimestamp         time.Time
	}
	var evs []ev
	for _, raw := range lr.Items {
		var e ev
		if json.Unmarshal(raw, &e) == nil {
			evs = append(evs, e)
		}
	}
	sort.Slice(evs, func(i, j int) bool { return evs[i].LastTimestamp.After(evs[j].LastTimestamp) })
	var evText strings.Builder
	for i, e := range evs {
		if i >= 8 {
			break
		}
		evText.WriteString(e.Reason + " " + e.Message + "\n")
		d.Events = append(d.Events, evRow{Type: e.Type, Reason: e.Reason, Message: oneLine(e.Message), Count: e.Count, Age: age(e.LastTimestamp)})
	}

	logs := getLogs(ns, name, logLines, false)
	if strings.Contains(reason, "CrashLoop") || strings.HasPrefix(reason, "Init:") || reason == "Error" {
		if prev := getLogs(ns, name, logLines, true); strings.TrimSpace(prev) != "" {
			logs = prev
		}
	}
	d.Diags = diagnose(reason, evText.String(), logs)
	d.Logs = cleanLogLines(logs)
	return d
}

// ---------------- secrets referenciados por um pod (só nomes de chaves) ----------------

type secretRef struct {
	Secret string   `json:"secret"`
	Keys   []string `json:"keys"`
	Denied bool     `json:"denied"`
}

type secretContainerSpec struct {
	Env []struct {
		ValueFrom *struct {
			SecretKeyRef *struct {
				Name string `json:"name"`
				Key  string `json:"key"`
			} `json:"secretKeyRef"`
		} `json:"valueFrom"`
	} `json:"env"`
	EnvFrom []struct {
		SecretRef *struct {
			Name string `json:"name"`
		} `json:"secretRef"`
	} `json:"envFrom"`
}

type podSecretSpec struct {
	Spec struct {
		Volumes []struct {
			Secret *struct {
				SecretName string `json:"secretName"`
				Items      []struct {
					Key string `json:"key"`
				} `json:"items"`
			} `json:"secret"`
		} `json:"volumes"`
		Containers     []secretContainerSpec `json:"containers"`
		InitContainers []secretContainerSpec `json:"initContainers"`
	} `json:"spec"`
}

// collectPodSecrets lista, para um pod, os Secrets que ele referencia via env/envFrom/volume —
// retorna apenas os nomes das chaves, nunca o valor.
func collectPodSecrets(ns, name string) []secretRef {
	out, err := kubectl("get", "pod", name, "-n", ns, "-o", "json")
	if err != nil {
		return nil
	}
	var ps podSecretSpec
	if json.Unmarshal(out, &ps) != nil {
		return nil
	}

	want := map[string]map[string]bool{}
	all := map[string]bool{}
	touch := func(secret string) {
		if _, ok := want[secret]; !ok {
			want[secret] = map[string]bool{}
		}
	}
	handleContainer := func(c secretContainerSpec) {
		for _, e := range c.Env {
			if e.ValueFrom != nil && e.ValueFrom.SecretKeyRef != nil {
				touch(e.ValueFrom.SecretKeyRef.Name)
				want[e.ValueFrom.SecretKeyRef.Name][e.ValueFrom.SecretKeyRef.Key] = true
			}
		}
		for _, ef := range c.EnvFrom {
			if ef.SecretRef != nil {
				touch(ef.SecretRef.Name)
				all[ef.SecretRef.Name] = true
			}
		}
	}
	for _, c := range ps.Spec.Containers {
		handleContainer(c)
	}
	for _, c := range ps.Spec.InitContainers {
		handleContainer(c)
	}
	for _, v := range ps.Spec.Volumes {
		if v.Secret == nil || v.Secret.SecretName == "" {
			continue
		}
		touch(v.Secret.SecretName)
		if len(v.Secret.Items) == 0 {
			all[v.Secret.SecretName] = true
		} else {
			for _, it := range v.Secret.Items {
				want[v.Secret.SecretName][it.Key] = true
			}
		}
	}

	names := make([]string, 0, len(want))
	for n := range want {
		names = append(names, n)
	}
	sort.Strings(names)

	out2 := make([]secretRef, 0, len(names))
	for _, sname := range names {
		sr := secretRef{Secret: sname}
		raw, err := kubectl("get", "secret", sname, "-n", ns, "-o", "json")
		if err != nil {
			sr.Denied = true
			out2 = append(out2, sr)
			continue
		}
		var sec struct {
			Data       map[string]string `json:"data"`
			StringData map[string]string `json:"stringData"`
		}
		if json.Unmarshal(raw, &sec) != nil {
			sr.Denied = true
			out2 = append(out2, sr)
			continue
		}
		keySet := map[string]bool{}
		for k := range sec.Data {
			keySet[k] = true
		}
		for k := range sec.StringData {
			keySet[k] = true
		}
		var keys []string
		if all[sname] {
			for k := range keySet {
				keys = append(keys, k)
			}
		} else {
			for k := range want[sname] {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		sr.Keys = keys
		out2 = append(out2, sr)
	}
	return out2
}

func getLogs(ns, name string, lines int, previous bool) string {
	args := []string{"logs", "-n", ns, name, "--all-containers", "--tail", itoa(lines)}
	if previous {
		args = append(args, "--previous")
	}
	out, _ := kubectl(args...)
	return string(out)
}

func effectiveContext() string {
	if c := getActiveContext(); c != "" {
		return c
	}
	cur, _ := listContexts()
	return cur
}

func itoa(i int) string {
	return strconv.Itoa(i)
}

// ---------------- metrics (kubectl top) ----------------

type podMetric struct {
	NS     string `json:"ns"`
	Name   string `json:"name"`
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

type nodeMetric struct {
	Name   string `json:"name"`
	CPU    string `json:"cpu"`
	CPUPct string `json:"cpuPct"`
	Memory string `json:"memory"`
	MemPct string `json:"memPct"`
}

type metricsResp struct {
	Pods  []podMetric  `json:"pods"`
	Nodes []nodeMetric `json:"nodes"`
	Err   string       `json:"err,omitempty"`
}

func collectMetrics() metricsResp {
	var resp metricsResp
	out, err := kubectl("top", "pods", "-A", "--no-headers")
	if err != nil {
		resp.Err = err.Error()
	} else {
		for _, line := range strings.Split(string(out), "\n") {
			f := strings.Fields(line)
			if len(f) < 4 {
				continue
			}
			resp.Pods = append(resp.Pods, podMetric{NS: f[0], Name: f[1], CPU: f[2], Memory: f[3]})
		}
	}
	out2, err2 := kubectl("top", "nodes", "--no-headers")
	if err2 == nil {
		for _, line := range strings.Split(string(out2), "\n") {
			f := strings.Fields(line)
			if len(f) < 5 {
				continue
			}
			resp.Nodes = append(resp.Nodes, nodeMetric{Name: f[0], CPU: f[1], CPUPct: f[2], Memory: f[3], MemPct: f[4]})
		}
	}
	return resp
}

// ---------------- resource quotas ----------------

type quotaResource struct {
	Hard string `json:"hard"`
	Used string `json:"used"`
}

type quotaRow struct {
	NS        string                   `json:"ns"`
	Name      string                   `json:"name"`
	Resources map[string]quotaResource `json:"resources"`
	Env       string                   `json:"env"`
}

func collectQuotas() []quotaRow {
	var out []quotaRow
	for _, raw := range getList("resourcequotas") {
		var q struct {
			Metadata struct {
				Name, Namespace string
			} `json:"metadata"`
			Status struct {
				Hard map[string]string `json:"hard"`
				Used map[string]string `json:"used"`
			} `json:"status"`
		}
		if json.Unmarshal(raw, &q) != nil {
			continue
		}
		resources := map[string]quotaResource{}
		for k, v := range q.Status.Hard {
			resources[k] = quotaResource{Hard: v, Used: q.Status.Used[k]}
		}
		out = append(out, quotaRow{
			NS: q.Metadata.Namespace, Name: q.Metadata.Name,
			Resources: resources, Env: classifyEnv(q.Metadata.Namespace),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NS < out[j].NS })
	return out
}

// ---------------- configmaps ----------------

type cmRow struct {
	NS   string `json:"ns"`
	Name string `json:"name"`
	Keys int    `json:"keys"`
	Age  string `json:"age"`
	Env  string `json:"env"`
}

var skipCMNS = map[string]bool{
	"kube-system": true, "kube-public": true, "kube-node-lease": true,
}

func collectConfigMaps() []cmRow {
	var out []cmRow
	for _, raw := range getList("configmaps") {
		var cm struct {
			Metadata struct {
				Name, Namespace   string
				CreationTimestamp time.Time
			} `json:"metadata"`
			Data map[string]string `json:"data"`
		}
		if json.Unmarshal(raw, &cm) != nil {
			continue
		}
		if skipCMNS[cm.Metadata.Namespace] || cm.Metadata.Name == "kube-root-ca.crt" {
			continue
		}
		out = append(out, cmRow{
			NS: cm.Metadata.Namespace, Name: cm.Metadata.Name,
			Keys: len(cm.Data), Age: age(cm.Metadata.CreationTimestamp),
			Env: classifyEnv(cm.Metadata.Namespace),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NS+out[i].Name < out[j].NS+out[j].Name })
	return out
}

// ---------------- ingresses ----------------

type ingressRow struct {
	NS      string `json:"ns"`
	Name    string `json:"name"`
	Class   string `json:"class"`
	Hosts   string `json:"hosts"`
	Address string `json:"address"`
	Ports   string `json:"ports"`
	Age     string `json:"age"`
	Env     string `json:"env"`
}

func collectIngresses() []ingressRow {
	var out []ingressRow
	for _, raw := range getList("ingresses") {
		var ing struct {
			Metadata struct {
				Name, Namespace   string
				CreationTimestamp time.Time
				Annotations       map[string]string
			} `json:"metadata"`
			Spec struct {
				IngressClassName *string `json:"ingressClassName"`
				Rules            []struct {
					Host string `json:"host"`
				} `json:"rules"`
				TLS []struct{} `json:"tls"`
			} `json:"spec"`
			Status struct {
				LoadBalancer struct {
					Ingress []struct {
						IP       string `json:"ip"`
						Hostname string `json:"hostname"`
					} `json:"ingress"`
				} `json:"loadBalancer"`
			} `json:"status"`
		}
		if json.Unmarshal(raw, &ing) != nil {
			continue
		}
		class := ""
		if ing.Spec.IngressClassName != nil {
			class = *ing.Spec.IngressClassName
		}
		if class == "" {
			class = ing.Metadata.Annotations["kubernetes.io/ingress.class"]
		}
		var hosts []string
		for _, r := range ing.Spec.Rules {
			if r.Host != "" {
				hosts = append(hosts, r.Host)
			}
		}
		var addrs []string
		for _, lb := range ing.Status.LoadBalancer.Ingress {
			if lb.IP != "" {
				addrs = append(addrs, lb.IP)
			} else if lb.Hostname != "" {
				addrs = append(addrs, lb.Hostname)
			}
		}
		ports := "80"
		if len(ing.Spec.TLS) > 0 {
			ports = "80,443"
		}
		out = append(out, ingressRow{
			NS: ing.Metadata.Namespace, Name: ing.Metadata.Name, Class: class,
			Hosts: strings.Join(hosts, ", "), Address: strings.Join(addrs, ","),
			Ports: ports, Age: age(ing.Metadata.CreationTimestamp),
			Env: classifyEnv(ing.Metadata.Namespace),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NS+out[i].Name < out[j].NS+out[j].Name })
	return out
}

// ---------------- cronjobs ----------------

type cronRow struct {
	NS           string `json:"ns"`
	Name         string `json:"name"`
	Schedule     string `json:"schedule"`
	Suspended    bool   `json:"suspended"`
	Active       int    `json:"active"`
	LastSchedule string `json:"lastSchedule"`
	LastSuccess  string `json:"lastSuccess"`
	Age          string `json:"age"`
	Env          string `json:"env"`
	Severity     int    `json:"severity"`
	Problem      string `json:"problem"`
}

func collectCronJobs() []cronRow {
	var out []cronRow
	for _, raw := range getList("cronjobs") {
		var cj struct {
			Metadata struct {
				Name, Namespace   string
				CreationTimestamp time.Time
			} `json:"metadata"`
			Spec struct {
				Schedule string `json:"schedule"`
				Suspend  *bool  `json:"suspend"`
			} `json:"spec"`
			Status struct {
				Active             []json.RawMessage `json:"active"`
				LastScheduleTime   *time.Time        `json:"lastScheduleTime"`
				LastSuccessfulTime *time.Time        `json:"lastSuccessfulTime"`
			} `json:"status"`
		}
		if json.Unmarshal(raw, &cj) != nil {
			continue
		}
		suspended := cj.Spec.Suspend != nil && *cj.Spec.Suspend
		lastSched := ""
		if cj.Status.LastScheduleTime != nil {
			lastSched = age(*cj.Status.LastScheduleTime)
		}
		lastSucc := ""
		if cj.Status.LastSuccessfulTime != nil {
			lastSucc = age(*cj.Status.LastSuccessfulTime)
		}
		activeCount := len(cj.Status.Active)
		severity, problem := 0, ""
		switch {
		case suspended:
			severity, problem = 1, "suspenso"
		case cj.Status.LastScheduleTime != nil && cj.Status.LastSuccessfulTime == nil && activeCount == 0:
			severity, problem = 2, "sem sucesso"
		case cj.Status.LastScheduleTime != nil && cj.Status.LastSuccessfulTime != nil &&
			activeCount == 0 &&
			cj.Status.LastSuccessfulTime.Before(*cj.Status.LastScheduleTime) &&
			time.Since(*cj.Status.LastScheduleTime) > 30*time.Minute:
			severity, problem = 2, "último run falhou"
		}
		out = append(out, cronRow{
			NS: cj.Metadata.Namespace, Name: cj.Metadata.Name,
			Schedule: cj.Spec.Schedule, Suspended: suspended,
			Active: activeCount, LastSchedule: lastSched, LastSuccess: lastSucc,
			Age: age(cj.Metadata.CreationTimestamp), Env: classifyEnv(cj.Metadata.Namespace),
			Severity: severity, Problem: problem,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Severity != out[j].Severity {
			return out[i].Severity > out[j].Severity
		}
		return out[i].NS+out[i].Name < out[j].NS+out[j].Name
	})
	return out
}

// ---------------- hpa ----------------

type hpaRow struct {
	NS          string `json:"ns"`
	Name        string `json:"name"`
	Reference   string `json:"reference"`
	MinReplicas int    `json:"minReplicas"`
	MaxReplicas int    `json:"maxReplicas"`
	Current     int    `json:"current"`
	Desired     int    `json:"desired"`
	Age         string `json:"age"`
	Env         string `json:"env"`
	OK          bool   `json:"ok"`
}

func collectHPAs() []hpaRow {
	var out []hpaRow
	for _, raw := range getList("hpa") {
		var h struct {
			Metadata struct {
				Name, Namespace   string
				CreationTimestamp time.Time
			} `json:"metadata"`
			Spec struct {
				MinReplicas    *int `json:"minReplicas"`
				MaxReplicas    int  `json:"maxReplicas"`
				ScaleTargetRef struct {
					Kind string `json:"kind"`
					Name string `json:"name"`
				} `json:"scaleTargetRef"`
			} `json:"spec"`
			Status struct {
				CurrentReplicas int `json:"currentReplicas"`
				DesiredReplicas int `json:"desiredReplicas"`
			} `json:"status"`
		}
		if json.Unmarshal(raw, &h) != nil {
			continue
		}
		minR := 1
		if h.Spec.MinReplicas != nil {
			minR = *h.Spec.MinReplicas
		}
		ref := h.Spec.ScaleTargetRef.Kind + "/" + h.Spec.ScaleTargetRef.Name
		ok := h.Status.CurrentReplicas >= minR && h.Status.CurrentReplicas <= h.Spec.MaxReplicas
		out = append(out, hpaRow{
			NS: h.Metadata.Namespace, Name: h.Metadata.Name, Reference: ref,
			MinReplicas: minR, MaxReplicas: h.Spec.MaxReplicas,
			Current: h.Status.CurrentReplicas, Desired: h.Status.DesiredReplicas,
			Age: age(h.Metadata.CreationTimestamp), Env: classifyEnv(h.Metadata.Namespace), OK: ok,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NS+out[i].Name < out[j].NS+out[j].Name })
	return out
}

// ---------------- pvcs ----------------

type pvcRow struct {
	NS           string `json:"ns"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	Volume       string `json:"volume"`
	Capacity     string `json:"capacity"`
	AccessModes  string `json:"accessModes"`
	StorageClass string `json:"storageClass"`
	Age          string `json:"age"`
	Env          string `json:"env"`
}

func collectPVCs() []pvcRow {
	var out []pvcRow
	for _, raw := range getList("persistentvolumeclaims") {
		var pvc struct {
			Metadata struct {
				Name, Namespace   string
				CreationTimestamp time.Time
			} `json:"metadata"`
			Spec struct {
				AccessModes      []string `json:"accessModes"`
				StorageClassName *string  `json:"storageClassName"`
				VolumeName       string   `json:"volumeName"`
			} `json:"spec"`
			Status struct {
				Phase    string            `json:"phase"`
				Capacity map[string]string `json:"capacity"`
			} `json:"status"`
		}
		if json.Unmarshal(raw, &pvc) != nil {
			continue
		}
		sc := ""
		if pvc.Spec.StorageClassName != nil {
			sc = *pvc.Spec.StorageClassName
		}
		cap := pvc.Status.Capacity["storage"]
		modes := strings.Join(pvc.Spec.AccessModes, ",")
		modes = strings.ReplaceAll(modes, "ReadWriteOnce", "RWO")
		modes = strings.ReplaceAll(modes, "ReadOnlyMany", "ROX")
		modes = strings.ReplaceAll(modes, "ReadWriteMany", "RWX")
		out = append(out, pvcRow{
			NS: pvc.Metadata.Namespace, Name: pvc.Metadata.Name, Status: pvc.Status.Phase,
			Volume: pvc.Spec.VolumeName, Capacity: cap, AccessModes: modes,
			StorageClass: sc, Age: age(pvc.Metadata.CreationTimestamp),
			Env: classifyEnv(pvc.Metadata.Namespace),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Status != out[j].Status {
			if out[i].Status == "Bound" {
				return false
			}
			if out[j].Status == "Bound" {
				return true
			}
		}
		return out[i].NS+out[i].Name < out[j].NS+out[j].Name
	})
	return out
}

// ---------------- rede: NetworkPolicy, CNI, CoreDNS, TargetGroupBinding/SecurityGroupPolicy ----------------

type netpolNSRow struct {
	NS          string `json:"ns"`
	Env         string `json:"env"`
	Policies    int    `json:"policies"`
	Pods        int    `json:"pods"`
	Unprotected bool   `json:"unprotected"`
}

func collectNetworkPolicyCoverage() (rows []netpolNSRow, total int) {
	podCountByNS := map[string]int{}
	for _, pv := range allPods() {
		podCountByNS[pv.Metadata.Namespace]++
	}

	polCountByNS := map[string]int{}
	for _, raw := range getList("networkpolicies") {
		var np struct {
			Metadata struct{ Namespace string } `json:"metadata"`
		}
		if json.Unmarshal(raw, &np) != nil {
			continue
		}
		polCountByNS[np.Metadata.Namespace]++
		total++
	}

	var nsNames []string
	for ns := range podCountByNS {
		nsNames = append(nsNames, ns)
	}
	sort.Strings(nsNames)

	for _, ns := range nsNames {
		rows = append(rows, netpolNSRow{
			NS: ns, Env: classifyEnv(ns), Policies: polCountByNS[ns], Pods: podCountByNS[ns],
			Unprotected: polCountByNS[ns] == 0,
		})
	}
	return
}

type cniInfo struct {
	Found           bool   `json:"found"`
	DesiredNodes    int    `json:"desiredNodes"`
	ReadyNodes      int    `json:"readyNodes"`
	ExternalSNAT    string `json:"externalSnat"`
	WarmIPTarget    string `json:"warmIpTarget"`
	MinimumIPTarget string `json:"minimumIpTarget"`
	WarmENITarget   string `json:"warmEniTarget"`
}

func collectCNI() cniInfo {
	out, err := kubectl("get", "ds", "aws-node", "-n", "kube-system", "-o", "json")
	if err != nil {
		return cniInfo{}
	}
	var ds struct {
		Status struct {
			DesiredNumberScheduled int `json:"desiredNumberScheduled"`
			NumberReady            int `json:"numberReady"`
		} `json:"status"`
		Spec struct {
			Template struct {
				Spec struct {
					Containers []struct {
						Name string `json:"name"`
						Env  []struct {
							Name  string `json:"name"`
							Value string `json:"value"`
						} `json:"env"`
					} `json:"containers"`
				} `json:"spec"`
			} `json:"template"`
		} `json:"spec"`
	}
	if json.Unmarshal(out, &ds) != nil {
		return cniInfo{}
	}
	info := cniInfo{Found: true, DesiredNodes: ds.Status.DesiredNumberScheduled, ReadyNodes: ds.Status.NumberReady}
	for _, c := range ds.Spec.Template.Spec.Containers {
		if c.Name != "aws-node" {
			continue
		}
		for _, e := range c.Env {
			switch e.Name {
			case "AWS_VPC_K8S_CNI_EXTERNALSNAT":
				info.ExternalSNAT = e.Value
			case "WARM_IP_TARGET":
				info.WarmIPTarget = e.Value
			case "MINIMUM_IP_TARGET":
				info.MinimumIPTarget = e.Value
			case "WARM_ENI_TARGET":
				info.WarmENITarget = e.Value
			}
		}
	}
	return info
}

type dnsInfo struct {
	Found      bool   `json:"found"`
	Desired    int    `json:"desired"`
	Ready      int    `json:"ready"`
	Forwarders string `json:"forwarders"`
}

var reCoreDNSForward = regexp.MustCompile(`(?m)^\s*forward\s+(\S+)\s+(.+?)\s*(?:\{|$)`)

func collectDNS() dnsInfo {
	var wg sync.WaitGroup
	var deployOut, cmOut []byte
	var deployErr error
	wg.Add(2)
	go func() {
		defer wg.Done()
		deployOut, deployErr = kubectl("get", "deploy", "coredns", "-n", "kube-system", "-o", "json")
	}()
	go func() {
		defer wg.Done()
		cmOut, _ = kubectl("get", "configmap", "coredns", "-n", "kube-system", "-o", "jsonpath={.data.Corefile}")
	}()
	wg.Wait()

	if deployErr != nil {
		return dnsInfo{}
	}
	var d struct {
		Status struct {
			Replicas      int `json:"replicas"`
			ReadyReplicas int `json:"readyReplicas"`
		} `json:"status"`
	}
	if json.Unmarshal(deployOut, &d) != nil {
		return dnsInfo{}
	}
	info := dnsInfo{Found: true, Desired: d.Status.Replicas, Ready: d.Status.ReadyReplicas}
	if m := reCoreDNSForward.FindStringSubmatch(string(cmOut)); len(m) == 3 {
		info.Forwarders = strings.TrimSpace(m[1] + " -> " + m[2])
	}
	return info
}

type tgbRow struct {
	NS             string `json:"ns"`
	Name           string `json:"name"`
	ServiceName    string `json:"serviceName"`
	TargetType     string `json:"targetType"`
	TargetGroupARN string `json:"targetGroupArn"`
}

func collectTargetGroupBindings() []tgbRow {
	out, err := kubectl("get", "targetgroupbindings", "-A", "-o", "json")
	if err != nil {
		return nil
	}
	var lr listResp
	if json.Unmarshal(out, &lr) != nil {
		return nil
	}
	var rows []tgbRow
	for _, raw := range lr.Items {
		var t struct {
			Metadata struct{ Name, Namespace string } `json:"metadata"`
			Spec     struct {
				ServiceRef struct {
					Name string `json:"name"`
				} `json:"serviceRef"`
				TargetType     string `json:"targetType"`
				TargetGroupARN string `json:"targetGroupARN"`
			} `json:"spec"`
		}
		if json.Unmarshal(raw, &t) != nil {
			continue
		}
		rows = append(rows, tgbRow{
			NS: t.Metadata.Namespace, Name: t.Metadata.Name, ServiceName: t.Spec.ServiceRef.Name,
			TargetType: t.Spec.TargetType, TargetGroupARN: t.Spec.TargetGroupARN,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].NS+rows[i].Name < rows[j].NS+rows[j].Name })
	return rows
}

type sgpRow struct {
	NS             string   `json:"ns"`
	Name           string   `json:"name"`
	SecurityGroups []string `json:"securityGroups"`
}

func collectSecurityGroupPolicies() []sgpRow {
	out, err := kubectl("get", "securitygrouppolicies", "-A", "-o", "json")
	if err != nil {
		return nil
	}
	var lr listResp
	if json.Unmarshal(out, &lr) != nil {
		return nil
	}
	var rows []sgpRow
	for _, raw := range lr.Items {
		var s struct {
			Metadata struct{ Name, Namespace string } `json:"metadata"`
			Spec     struct {
				SecurityGroups struct {
					GroupIDs []string `json:"groupIds"`
				} `json:"securityGroups"`
			} `json:"spec"`
		}
		if json.Unmarshal(raw, &s) != nil {
			continue
		}
		rows = append(rows, sgpRow{NS: s.Metadata.Namespace, Name: s.Metadata.Name, SecurityGroups: s.Spec.SecurityGroups.GroupIDs})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].NS+rows[i].Name < rows[j].NS+rows[j].Name })
	return rows
}

type networkExtra struct {
	NetPolicy        []netpolNSRow `json:"netpolicy"`
	NetPolicyTotal   int           `json:"netpolicyTotal"`
	CNI              cniInfo       `json:"cni"`
	DNS              dnsInfo       `json:"dns"`
	TargetGroupBinds []tgbRow      `json:"targetGroupBindings"`
	SecurityGroupPol []sgpRow      `json:"securityGroupPolicies"`
}

func collectNetworkExtra() networkExtra {
	var (
		wg      sync.WaitGroup
		npRows  []netpolNSRow
		npTotal int
		cni     cniInfo
		dns     dnsInfo
		tgb     []tgbRow
		sgp     []sgpRow
	)
	wg.Add(5)
	go func() { defer wg.Done(); npRows, npTotal = collectNetworkPolicyCoverage() }()
	go func() { defer wg.Done(); cni = collectCNI() }()
	go func() { defer wg.Done(); dns = collectDNS() }()
	go func() { defer wg.Done(); tgb = collectTargetGroupBindings() }()
	go func() { defer wg.Done(); sgp = collectSecurityGroupPolicies() }()
	wg.Wait()

	return networkExtra{
		NetPolicy: npRows, NetPolicyTotal: npTotal,
		CNI: cni, DNS: dns,
		TargetGroupBinds: tgb,
		SecurityGroupPol: sgp,
	}
}
