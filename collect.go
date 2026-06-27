// collect.go — monta os payloads JSON de cada seção do painel.
package main

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"
)

func getList(kind string) []json.RawMessage {
	args := []string{"get", kind, "-A", "-o", "json"}
	if kind == "nodes" { // recurso cluster-scoped: -A gera warning
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
	Ctx        string   `json:"ctx"`
	Env        string   `json:"env"`
	Nodes      []string `json:"nodes"`
	NodesTotal int      `json:"nodesTotal"`
	NodesDown  int      `json:"nodesDown"`
	Pods       struct {
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

func collectOverview() overview {
	var o overview
	o.Generated = time.Now().Format("15:04:05")
	o.Ctx = effectiveContext()
	o.Env = classifyEnv(o.Ctx)
	o.Nodes, o.NodesTotal, o.NodesDown = nodeLines()

	pods := allPods()
	nsSet := map[string]bool{}
	appsBad := map[string]bool{}
	envc := map[string]int{"prod": 0, "staging": 0, "dev": 0, "other": 0}
	for _, pv := range pods {
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
		wlReady("deployments"), wlReady("statefulsets"), wlReady("daemonsets"),
	}
	return o
}

func wlReady(kind string) workloadKind {
	w := workloadKind{Kind: strings.TrimSuffix(kind, "s")}
	for _, raw := range getList(kind) {
		var d struct {
			Status struct {
				ReadyReplicas        int `json:"readyReplicas"`
				Replicas             int `json:"replicas"`
				NumberReady          int `json:"numberReady"`
				DesiredNumberSched   int `json:"desiredNumberScheduled"`
			} `json:"status"`
		}
		if json.Unmarshal(raw, &d) != nil {
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
	App         string   `json:"app"`
	NS          string   `json:"ns"`
	Env         string   `json:"env"`
	Reason      string   `json:"reason"`
	Severity    int      `json:"severity"`
	PodCount    int      `json:"podCount"`
	Restarts    int      `json:"restarts"`
	Ready       int      `json:"ready"`
	Total       int      `json:"total"`
	Age         string   `json:"age"`
	Node        string   `json:"node"`
	LastExit    string   `json:"lastExit"`
	SamplePod   string   `json:"samplePod"`
	SampleReason string  `json:"sampleReason"`
	Pods        []string `json:"pods"`
	PrimaryDiag string   `json:"primaryDiag"`
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
			NS: pv.Metadata.Namespace, Name: pv.Metadata.Name, Reason: pv.reason,
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
	NS    string `json:"ns"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	IP    string `json:"ip"`
	Ports string `json:"ports"`
	Env   string `json:"env"`
}

func collectServices() []svcRow {
	var out []svcRow
	for _, raw := range getList("services") {
		var s struct {
			Metadata struct{ Name, Namespace string } `json:"metadata"`
			Spec     struct {
				Type      string `json:"type"`
				ClusterIP string `json:"clusterIP"`
				Ports     []struct {
					Port     int    `json:"port"`
					Protocol string `json:"protocol"`
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
		out = append(out, svcRow{
			NS: s.Metadata.Namespace, Name: s.Metadata.Name, Type: s.Spec.Type,
			IP: s.Spec.ClusterIP, Ports: strings.Join(ports, ","), Env: classifyEnv(s.Metadata.Namespace),
		})
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
			Metadata struct{ Namespace string } `json:"metadata"`
			Type     string                     `json:"type"`
			Reason   string                     `json:"reason"`
			Message  string                     `json:"message"`
			Count    int                        `json:"count"`
			LastTimestamp time.Time             `json:"lastTimestamp"`
			InvolvedObject struct{ Kind, Name string } `json:"involvedObject"`
		}
		if json.Unmarshal(raw, &e) != nil || e.Type != "Warning" {
			continue
		}
		out = append(out, evRow{
			NS: e.Metadata.Namespace, Type: e.Type, Reason: e.Reason,
			Object: e.InvolvedObject.Kind + "/" + e.InvolvedObject.Name,
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
	Name    string `json:"name"`
	Type    string `json:"type"`
	Cap     string `json:"cap"`
	Zone    string `json:"zone"`
	Ready   bool   `json:"ready"`
	Age     string `json:"age"`
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
	Events []evRow   `json:"events"`
	Diags  []diag    `json:"diags"`
	Logs   []logLine `json:"logs"`
}

func collectDetail(ns, name, reason string, logLines int) detail {
	var d detail
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
