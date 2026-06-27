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
