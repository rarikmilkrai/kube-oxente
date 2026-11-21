package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type hclWriter struct {
	b      strings.Builder
	indent int
}

func (w *hclWriter) line(s string) {
	w.b.WriteString(strings.Repeat("  ", w.indent))
	w.b.WriteString(s)
	w.b.WriteByte('\n')
}

func (w *hclWriter) blank() { w.b.WriteByte('\n') }

func (w *hclWriter) open(s string) {
	w.line(s + " {")
	w.indent++
}

func (w *hclWriter) close() {
	w.indent--
	w.line("}")
}

func (w *hclWriter) attr(k, expr string) { w.line(k + " = " + expr) }

func (w *hclWriter) str(k, v string) {
	if v != "" {
		w.attr(k, hclString(v))
	}
}

func (w *hclWriter) num(k string, v int) { w.attr(k, itoa(v)) }

func (w *hclWriter) strMap(k string, m map[string]string) {
	if len(m) == 0 {
		return
	}
	keys := sortedKeys(m)
	w.line(k + " = {")
	w.indent++
	width := 0
	for _, kk := range keys {
		if l := len(hclKey(kk)); l > width {
			width = l
		}
	}
	for _, kk := range keys {
		hk := hclKey(kk)
		w.line(hk + strings.Repeat(" ", width-len(hk)) + " = " + hclValue(m[kk]))
	}
	w.indent--
	w.line("}")
}

func (w *hclWriter) strList(k string, v []string) {
	if len(v) == 0 {
		return
	}
	q := make([]string, len(v))
	for i, s := range v {
		q[i] = hclString(s)
	}
	w.attr(k, "["+strings.Join(q, ", ")+"]")
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

var reIdent = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

func hclKey(k string) string {
	if reIdent.MatchString(k) {
		return k
	}
	return hclString(k)
}

func hclEscape(s string) string {
	s = strings.ReplaceAll(s, "${", "$${")
	return strings.ReplaceAll(s, "%{", "%%{")
}

func hclString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range hclEscape(s) {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func hclValue(s string) string {
	if strings.Contains(s, "\n") && !strings.Contains(s, "\nEOT") {
		body := hclEscape(s)
		if !strings.HasSuffix(body, "\n") {
			body += "\n"
		}
		return "<<-EOT\n" + body + "EOT"
	}
	return hclString(s)
}

var reTFName = regexp.MustCompile(`[^a-z0-9_]+`)

func tfName(parts ...string) string {
	n := reTFName.ReplaceAllString(strings.ToLower(strings.Join(parts, "_")), "_")
	n = strings.Trim(n, "_")
	if n == "" || (n[0] >= '0' && n[0] <= '9') {
		n = "r_" + n
	}
	return n
}

type tfRegistry struct {
	names map[string]string
	used  map[string]bool
}

func (r *tfRegistry) name(kind, ns, name string) string {
	k := kind + "|" + ns + "|" + name
	if n, ok := r.names[k]; ok {
		return n
	}
	base := tfName(ns, name)
	if ns == "" {
		base = tfName(name)
	}
	n := base
	for i := 2; r.used[kind+"."+n]; i++ {
		n = fmt.Sprintf("%s_%d", base, i)
	}
	r.used[kind+"."+n] = true
	r.names[k] = n
	return n
}

func (r *tfRegistry) has(kind, ns, name string) (string, bool) {
	n, ok := r.names[kind+"|"+ns+"|"+name]
	return n, ok
}

var managedLabels = regexp.MustCompile(`^(pod-template-hash|controller-revision-hash|statefulset\.kubernetes\.io/pod-name)$`)

var noisyAnnotations = regexp.MustCompile(`^(kubectl\.kubernetes\.io/last-applied-configuration|deployment\.kubernetes\.io/revision|meta\.helm\.sh/.*|argocd\.argoproj\.io/.*|kubectl\.kubernetes\.io/restartedAt)$`)

func cleanLabels(m map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range m {
		if !managedLabels.MatchString(k) {
			out[k] = v
		}
	}
	return out
}

func cleanAnnotations(m map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range m {
		if !noisyAnnotations.MatchString(k) {
			out[k] = v
		}
	}
	return out
}

type tfGen struct {
	w       *hclWriter
	reg     *tfRegistry
	d       *archData
	imports []string
}

func (t *tfGen) nsRef(ns string) string {
	if n, ok := t.reg.has("kubernetes_namespace_v1", "", ns); ok {
		return "kubernetes_namespace_v1." + n + ".metadata[0].name"
	}
	return hclString(ns)
}

func (t *tfGen) ref(kind, ns, name string) string {
	if n, ok := t.reg.has(kind, ns, name); ok {
		return kind + "." + n + ".metadata[0].name"
	}
	return hclString(name)
}

func (t *tfGen) metadata(m k8sMeta, withNS bool) {
	t.w.open("metadata")
	t.w.str("name", m.Name)
	if withNS {
		t.w.attr("namespace", t.nsRef(m.Namespace))
	}
	t.w.strMap("labels", cleanLabels(m.Labels))
	t.w.strMap("annotations", cleanAnnotations(m.Annotations))
	t.w.close()
}

func (t *tfGen) importBlock(kind, name, id string) {
	t.imports = append(t.imports, "import {\n  to = "+kind+"."+name+"\n  id = "+hclString(id)+"\n}")
}

func (t *tfGen) container(block string, ns string, c containerSpec) {
	w := t.w
	w.open(block)
	w.str("name", c.Name)
	w.str("image", c.Image)
	w.strList("command", c.Command)
	w.strList("args", c.Args)
	for _, p := range c.Ports {
		w.open("port")
		w.str("name", p.Name)
		w.num("container_port", p.ContainerPort)
		if p.Protocol != "" && p.Protocol != "TCP" {
			w.str("protocol", p.Protocol)
		}
		w.close()
	}
	for _, ef := range c.EnvFrom {
		w.open("env_from")
		if ef.ConfigMapRef != nil {
			w.open("config_map_ref")
			w.attr("name", t.ref("kubernetes_config_map_v1", ns, ef.ConfigMapRef.Name))
			w.close()
		}
		if ef.SecretRef != nil {
			w.open("secret_ref")
			w.attr("name", t.ref("kubernetes_secret_v1", ns, ef.SecretRef.Name))
			w.close()
		}
		w.close()
	}
	for _, e := range c.Env {
		w.open("env")
		w.str("name", e.Name)
		switch {
		case e.ValueFrom == nil:
			w.attr("value", hclString(e.Value))
		case e.ValueFrom.ConfigMapKeyRef != nil:
			w.open("value_from")
			w.open("config_map_key_ref")
			w.attr("name", t.ref("kubernetes_config_map_v1", ns, e.ValueFrom.ConfigMapKeyRef.Name))
			w.str("key", e.ValueFrom.ConfigMapKeyRef.Key)
			w.close()
			w.close()
		case e.ValueFrom.SecretKeyRef != nil:
			w.open("value_from")
			w.open("secret_key_ref")
			w.attr("name", t.ref("kubernetes_secret_v1", ns, e.ValueFrom.SecretKeyRef.Name))
			w.str("key", e.ValueFrom.SecretKeyRef.Key)
			w.close()
			w.close()
		}
		w.close()
	}
	if len(c.Resources.Limits)+len(c.Resources.Requests) > 0 {
		w.open("resources")
		w.strMap("limits", c.Resources.Limits)
		w.strMap("requests", c.Resources.Requests)
		w.close()
	}
	for _, vm := range c.VolumeMounts {
		w.open("volume_mount")
		w.str("name", vm.Name)
		w.str("mount_path", vm.MountPath)
		w.str("sub_path", vm.SubPath)
		if vm.ReadOnly {
			w.attr("read_only", "true")
		}
		w.close()
	}
	w.close()
}

func (t *tfGen) podSpec(ns string, tpl podTemplate) {
	w := t.w
	w.open("spec")
	w.str("service_account_name", tpl.Spec.ServiceAccountName)
	if tpl.Spec.RestartPolicy != "" && tpl.Spec.RestartPolicy != "Always" {
		w.str("restart_policy", tpl.Spec.RestartPolicy)
	}
	w.attr("automount_service_account_token", boolStr(tpl.Spec.AutomountSAToken == nil || *tpl.Spec.AutomountSAToken))
	w.attr("enable_service_links", boolStr(tpl.Spec.EnableServiceLinks == nil || *tpl.Spec.EnableServiceLinks))
	w.strMap("node_selector", tpl.Spec.NodeSelector)
	for _, c := range tpl.Spec.InitContainers {
		t.container("init_container", ns, c)
	}
	for _, c := range tpl.Spec.Containers {
		t.container("container", ns, c)
	}
	for _, v := range tpl.Spec.Volumes {
		w.open("volume")
		w.str("name", v.Name)
		switch {
		case v.ConfigMap != nil:
			w.open("config_map")
			w.attr("name", t.ref("kubernetes_config_map_v1", ns, v.ConfigMap.Name))
			w.close()
		case v.Secret != nil:
			w.open("secret")
			w.attr("secret_name", t.ref("kubernetes_secret_v1", ns, v.Secret.SecretName))
			w.close()
		case v.PersistentVolumeClaim != nil:
			w.open("persistent_volume_claim")
			w.attr("claim_name", t.ref("kubernetes_persistent_volume_claim_v1", ns, v.PersistentVolumeClaim.ClaimName))
			w.close()
		case v.EmptyDir != nil:
			w.line("empty_dir {}")
		case v.Projected != nil:
			w.open("projected")
			for _, s := range v.Projected.Sources {
				if s.ConfigMap != nil {
					w.open("sources")
					w.open("config_map")
					w.attr("name", t.ref("kubernetes_config_map_v1", ns, s.ConfigMap.Name))
					w.close()
					w.close()
				}
				if s.Secret != nil {
					w.open("sources")
					w.open("secret")
					w.attr("name", t.ref("kubernetes_secret_v1", ns, s.Secret.Name))
					w.close()
					w.close()
				}
			}
			w.close()
		}
		w.close()
	}
	w.close()
}

func (t *tfGen) template(ns string, tpl podTemplate) {
	t.w.open("template")
	t.w.open("metadata")
	t.w.strMap("labels", cleanLabels(tpl.Metadata.Labels))
	t.w.strMap("annotations", cleanAnnotations(tpl.Metadata.Annotations))
	t.w.close()
	t.podSpec(ns, tpl)
	t.w.close()
}

func generateTerraform(nsFilter string, includeSystem bool) string {
	d := loadArchData()
	reg := &tfRegistry{names: map[string]string{}, used: map[string]bool{}}
	w := &hclWriter{}
	t := &tfGen{w: w, reg: reg, d: d}

	var nsList []string
	nsSeen := map[string]bool{}
	addNS := func(ns string) {
		if ns != "" && !nsSeen[ns] && inScope(ns, nsFilter, includeSystem) && ns != "default" {
			nsSeen[ns] = true
			nsList = append(nsList, ns)
		}
	}
	for _, wl := range d.Workloads {
		addNS(wl.Meta.Namespace)
	}
	for _, s := range d.Services {
		addNS(s.Meta.Namespace)
	}
	sort.Strings(nsList)

	usedCM := map[string]bool{}
	usedSecret := map[string]bool{}
	var workloads []*archWorkload
	for _, wl := range d.Workloads {
		if !inScope(wl.Meta.Namespace, nsFilter, includeSystem) {
			continue
		}
		workloads = append(workloads, wl)
		cms, secs, _ := workloadRefs(wl)
		for _, c := range cms {
			usedCM[nsKey(wl.Meta.Namespace, c)] = true
		}
		for _, s := range secs {
			usedSecret[nsKey(wl.Meta.Namespace, s)] = true
		}
	}
	for _, ing := range d.Ingresses {
		if !inScope(ing.Meta.Namespace, nsFilter, includeSystem) {
			continue
		}
		for _, tls := range ing.TLS {
			if tls.SecretName != "" {
				usedSecret[nsKey(ing.Meta.Namespace, tls.SecretName)] = true
			}
		}
	}

	for _, ns := range nsList {
		reg.name("kubernetes_namespace_v1", "", ns)
	}
	cmKeys := make([]string, 0)
	for k := range usedCM {
		if _, ok := d.ConfigMaps[k]; ok {
			cmKeys = append(cmKeys, k)
		}
	}
	sort.Strings(cmKeys)
	for _, k := range cmKeys {
		p := strings.SplitN(k, "/", 2)
		reg.name("kubernetes_config_map_v1", p[0], p[1])
	}
	secKeys := make([]string, 0)
	for k := range usedSecret {
		secKeys = append(secKeys, k)
	}
	sort.Strings(secKeys)
	for _, k := range secKeys {
		p := strings.SplitN(k, "/", 2)
		reg.name("kubernetes_secret_v1", p[0], p[1])
	}
	pvcKeys := make([]string, 0)
	for k, p := range d.PVCs {
		if inScope(p.Meta.Namespace, nsFilter, includeSystem) {
			pvcKeys = append(pvcKeys, k)
		}
	}
	sort.Strings(pvcKeys)
	for _, k := range pvcKeys {
		p := d.PVCs[k]
		if strings.Contains(p.Meta.Name, "-") && isClaimTemplatePVC(d, p) {
			continue
		}
		reg.name("kubernetes_persistent_volume_claim_v1", p.Meta.Namespace, p.Meta.Name)
	}
	var services []*archService
	for _, s := range d.Services {
		if !inScope(s.Meta.Namespace, nsFilter, includeSystem) {
			continue
		}
		if s.Meta.Namespace == "default" && s.Meta.Name == "kubernetes" {
			continue
		}
		services = append(services, s)
		reg.name("kubernetes_service_v1", s.Meta.Namespace, s.Meta.Name)
	}

	w.open("terraform")
	w.open("required_providers")
	w.line("kubernetes = {")
	w.indent++
	w.str("source", "hashicorp/kubernetes")
	w.str("version", "~> 2.35")
	w.close()
	w.close()
	w.close()
	w.blank()
	w.open(`provider "kubernetes"`)
	w.attr("config_path", "var.kubeconfig_path")
	w.attr("config_context", "var.kube_context")
	w.close()
	w.blank()
	w.open(`variable "kubeconfig_path"`)
	w.attr("type", "string")
	w.str("default", "~/.kube/config")
	w.close()
	w.blank()
	w.open(`variable "kube_context"`)
	w.attr("type", "string")
	w.str("default", effectiveContext())
	w.close()
	if len(secKeys) > 0 {
		w.blank()
		w.open(`variable "secret_data"`)
		w.attr("type", "map(map(string))")
		w.attr("sensitive", "true")
		w.attr("default", "{}")
		w.close()
	}

	for _, ns := range nsList {
		n, _ := reg.has("kubernetes_namespace_v1", "", ns)
		w.blank()
		w.open(`resource "kubernetes_namespace_v1" "` + n + `"`)
		w.open("metadata")
		w.str("name", ns)
		labels := map[string]string{}
		for k, v := range d.Namespaces[ns] {
			if k != "kubernetes.io/metadata.name" {
				labels[k] = v
			}
		}
		w.strMap("labels", labels)
		w.close()
		w.close()
		t.importBlock("kubernetes_namespace_v1", n, ns)
	}

	for _, k := range cmKeys {
		p := strings.SplitN(k, "/", 2)
		n, _ := reg.has("kubernetes_config_map_v1", p[0], p[1])
		w.blank()
		w.open(`resource "kubernetes_config_map_v1" "` + n + `"`)
		t.metadata(k8sMeta{Name: p[1], Namespace: p[0]}, true)
		w.strMap("data", d.ConfigMaps[k])
		w.close()
		t.importBlock("kubernetes_config_map_v1", n, k)
	}

	for _, k := range secKeys {
		p := strings.SplitN(k, "/", 2)
		n, _ := reg.has("kubernetes_secret_v1", p[0], p[1])
		w.blank()
		w.open(`resource "kubernetes_secret_v1" "` + n + `"`)
		t.metadata(k8sMeta{Name: p[1], Namespace: p[0]}, true)
		if s, ok := d.Secrets[k]; ok && s.Type != "" && s.Type != "Opaque" {
			w.str("type", s.Type)
		}
		w.attr("data", "lookup(var.secret_data, "+hclString(k)+", {})")
		w.open("lifecycle")
		w.attr("ignore_changes", "[data]")
		w.close()
		w.close()
		t.importBlock("kubernetes_secret_v1", n, k)
	}

	for _, k := range pvcKeys {
		p := d.PVCs[k]
		n, ok := reg.has("kubernetes_persistent_volume_claim_v1", p.Meta.Namespace, p.Meta.Name)
		if !ok {
			continue
		}
		w.blank()
		w.open(`resource "kubernetes_persistent_volume_claim_v1" "` + n + `"`)
		t.metadata(k8sMeta{Name: p.Meta.Name, Namespace: p.Meta.Namespace, Labels: p.Meta.Labels}, true)
		w.open("spec")
		w.strList("access_modes", p.AccessModes)
		w.str("storage_class_name", p.StorageClass)
		w.open("resources")
		w.strMap("requests", map[string]string{"storage": firstNonEmpty(p.Size, "1Gi")})
		w.close()
		w.close()
		w.attr("wait_until_bound", "false")
		w.close()
		t.importBlock("kubernetes_persistent_volume_claim_v1", n, k)
	}

	for _, wl := range workloads {
		ns := wl.Meta.Namespace
		var kind string
		switch wl.Kind {
		case "Deployment":
			kind = "kubernetes_deployment_v1"
		case "StatefulSet":
			kind = "kubernetes_stateful_set_v1"
		case "DaemonSet":
			kind = "kubernetes_daemon_set_v1"
		case "CronJob":
			kind = "kubernetes_cron_job_v1"
		default:
			continue
		}
		n := reg.name(kind, ns, wl.Meta.Name)
		w.blank()
		w.open(`resource "` + kind + `" "` + n + `"`)
		t.metadata(wl.Meta, true)
		w.open("spec")
		switch wl.Kind {
		case "CronJob":
			cs := wl.CronSpec
			w.str("schedule", wl.Schedule)
			w.str("timezone", cs.TimeZone)
			if wl.Suspend {
				w.attr("suspend", "true")
			}
			w.str("concurrency_policy", cs.ConcurrencyPolicy)
			optNum(w, "successful_jobs_history_limit", cs.SuccessfulJobsHistoryLimit)
			optNum(w, "failed_jobs_history_limit", cs.FailedJobsHistoryLimit)
			optNum(w, "starting_deadline_seconds", cs.StartingDeadlineSeconds)
			js := cs.JobTemplate.Spec
			w.open("job_template")
			w.line("metadata {}")
			w.open("spec")
			w.num("backoff_limit", intOr(js.BackoffLimit, 6))
			w.num("completions", intOr(js.Completions, 1))
			w.num("parallelism", intOr(js.Parallelism, 1))
			optNum(w, "active_deadline_seconds", js.ActiveDeadlineSeconds)
			optNum(w, "ttl_seconds_after_finished", js.TTLSecondsAfterFinished)
			t.template(ns, wl.Template)
			w.close()
			w.close()
		default:
			if wl.Replicas != nil && wl.Kind != "DaemonSet" {
				w.num("replicas", *wl.Replicas)
			}
			if wl.Kind == "StatefulSet" {
				w.str("service_name", wl.ServiceName)
			}
			w.open("selector")
			w.strMap("match_labels", wl.Selector)
			w.close()
			t.template(ns, wl.Template)
			for _, ct := range wl.ClaimTpls {
				w.open("volume_claim_template")
				w.open("metadata")
				w.str("name", ct.Meta.Name)
				w.close()
				w.open("spec")
				w.strList("access_modes", ct.AccessModes)
				w.str("storage_class_name", ct.StorageClass)
				w.open("resources")
				w.strMap("requests", map[string]string{"storage": firstNonEmpty(ct.Size, "1Gi")})
				w.close()
				w.close()
				w.close()
			}
		}
		w.close()
		if ig := ignoreFor(d, wl); len(ig) > 0 {
			w.open("lifecycle")
			if len(ig) == 1 {
				w.attr("ignore_changes", "["+ig[0]+"]")
			} else {
				w.line("ignore_changes = [")
				w.indent++
				for _, x := range ig {
					w.line(x + ",")
				}
				w.indent--
				w.line("]")
			}
			w.close()
		}
		w.close()
		t.importBlock(kind, n, nsKey(ns, wl.Meta.Name))
	}

	for _, s := range services {
		ns := s.Meta.Namespace
		n, _ := reg.has("kubernetes_service_v1", ns, s.Meta.Name)
		w.blank()
		w.open(`resource "kubernetes_service_v1" "` + n + `"`)
		t.metadata(s.Meta, true)
		w.open("spec")
		if s.Type != "" && s.Type != "ClusterIP" {
			w.str("type", s.Type)
		}
		w.str("external_name", s.ExternalName)
		w.strMap("selector", s.Selector)
		for _, p := range s.Ports {
			w.open("port")
			w.str("name", p.Name)
			w.num("port", p.Port)
			if p.TargetPort != nil {
				if p.TargetPort.IsStr {
					w.str("target_port", p.TargetPort.StrVal)
				} else if p.TargetPort.IntVal != p.Port {
					w.str("target_port", itoa(p.TargetPort.IntVal))
				}
			}
			if p.Protocol != "" && p.Protocol != "TCP" {
				w.str("protocol", p.Protocol)
			}
			w.close()
		}
		w.close()
		if s.Type == "LoadBalancer" {
			w.attr("wait_for_load_balancer", "false")
		}
		w.close()
		t.importBlock("kubernetes_service_v1", n, nsKey(ns, s.Meta.Name))
	}

	for _, ing := range d.Ingresses {
		ns := ing.Meta.Namespace
		if !inScope(ns, nsFilter, includeSystem) {
			continue
		}
		n := reg.name("kubernetes_ingress_v1", ns, ing.Meta.Name)
		w.blank()
		w.open(`resource "kubernetes_ingress_v1" "` + n + `"`)
		t.metadata(ing.Meta, true)
		w.open("spec")
		w.str("ingress_class_name", ing.Class)
		if ing.DefaultService != "" {
			w.open("default_backend")
			t.backend(ns, ing.DefaultService, ing.DefaultPort, "")
			w.close()
		}
		for _, r := range ing.Rules {
			w.open("rule")
			w.str("host", r.Host)
			if len(r.Paths) > 0 {
				w.open("http")
				for _, p := range r.Paths {
					w.open("path")
					w.str("path", p.Path)
					w.str("path_type", p.PathType)
					w.open("backend")
					t.backend(ns, p.Service, p.Port, p.PortName)
					w.close()
					w.close()
				}
				w.close()
			}
			w.close()
		}
		for _, tls := range ing.TLS {
			w.open("tls")
			w.strList("hosts", tls.Hosts)
			if tls.SecretName != "" {
				w.attr("secret_name", t.ref("kubernetes_secret_v1", ns, tls.SecretName))
			}
			w.close()
		}
		w.close()
		w.attr("wait_for_load_balancer", "false")
		w.close()
		t.importBlock("kubernetes_ingress_v1", n, nsKey(ns, ing.Meta.Name))
	}

	for _, h := range d.HPAs {
		ns := h.Meta.Namespace
		if !inScope(ns, nsFilter, includeSystem) {
			continue
		}
		n := reg.name("kubernetes_horizontal_pod_autoscaler_v2", ns, h.Meta.Name)
		target := hclString(h.TargetName)
		switch h.TargetKind {
		case "Deployment":
			target = t.ref("kubernetes_deployment_v1", ns, h.TargetName)
		case "StatefulSet":
			target = t.ref("kubernetes_stateful_set_v1", ns, h.TargetName)
		}
		w.blank()
		w.open(`resource "kubernetes_horizontal_pod_autoscaler_v2" "` + n + `"`)
		t.metadata(h.Meta, true)
		w.open("spec")
		w.num("min_replicas", h.Min)
		w.num("max_replicas", h.Max)
		w.open("scale_target_ref")
		w.str("api_version", firstNonEmpty(h.TargetAPI, "apps/v1"))
		w.str("kind", h.TargetKind)
		w.attr("name", target)
		w.close()
		for _, m := range []struct {
			name string
			v    int
		}{{"cpu", h.CPUTarget}, {"memory", h.MemTarget}} {
			if m.v == 0 {
				continue
			}
			w.open("metric")
			w.str("type", "Resource")
			w.open("resource")
			w.str("name", m.name)
			w.open("target")
			w.str("type", "Utilization")
			w.num("average_utilization", m.v)
			w.close()
			w.close()
			w.close()
		}
		w.close()
		w.close()
		t.importBlock("kubernetes_horizontal_pod_autoscaler_v2", n, nsKey(ns, h.Meta.Name))
	}

	if len(t.imports) > 0 {
		w.blank()
		w.b.WriteString(strings.Join(t.imports, "\n\n"))
		w.b.WriteByte('\n')
	}

	header := "# Gerado pelo Kube Oxente a partir do contexto " + effectiveContext() + "\n" +
		"# Revise antes de aplicar: rode terraform plan e leia cada mudança.\n" +
		"# No primeiro plan (import) o provider kubernetes 2.x mostra diferenças esperadas:\n" +
		"#   - wait_for_rollout / wait_for_load_balancer / wait_until_bound: só no state, não muda o cluster\n" +
		"#   - automount_service_account_token / enable_service_links = true: mesmo comportamento do\n" +
		"#     default do Kubernetes, mas grava o campo no template e causa um rollout\n" +
		"#   - CronJob: o provider força recriar por causa de completions (perde só o histórico de Jobs)\n" +
		"# Não troque esses valores por false nem use ignore_changes neles: o provider passa a enviar\n" +
		"# false/0 ao cluster na próxima alteração (desliga o token da ServiceAccount e zera parallelism).\n\n"
	return header + alignHCL(w.b.String())
}

var reAttrLine = regexp.MustCompile(`^(\s*)([A-Za-z_][A-Za-z0-9_-]*|"[^"]*")\s=\s(.*)$`)

func alignHCL(src string) string {
	lines := strings.Split(src, "\n")
	inHeredoc := ""
	start := -1
	flush := func(end int) {
		if start < 0 {
			return
		}
		width := 0
		for i := start; i < end; i++ {
			if m := reAttrLine.FindStringSubmatch(lines[i]); m != nil && len(m[2]) > width {
				width = len(m[2])
			}
		}
		for i := start; i < end; i++ {
			if m := reAttrLine.FindStringSubmatch(lines[i]); m != nil {
				lines[i] = m[1] + m[2] + strings.Repeat(" ", width-len(m[2])) + " = " + m[3]
			}
		}
		start = -1
	}
	indentOf := func(s string) int { return len(s) - len(strings.TrimLeft(s, " ")) }
	for i, l := range lines {
		if inHeredoc != "" {
			if strings.TrimSpace(l) == inHeredoc {
				inHeredoc = ""
			}
			continue
		}
		m := reAttrLine.FindStringSubmatch(l)
		single := m != nil && !strings.HasSuffix(m[3], "{") && !strings.HasSuffix(m[3], "[") && !strings.HasPrefix(m[3], "<<")
		if single && (start < 0 || indentOf(lines[start]) == indentOf(l)) {
			if start < 0 {
				start = i
			}
			continue
		}
		flush(i)
		if single {
			start = i
			continue
		}
		if m != nil {
			start = i
			flush(i + 1)
		}
		if m != nil && strings.HasPrefix(m[3], "<<") {
			inHeredoc = strings.TrimLeft(strings.TrimPrefix(m[3], "<<"), "-")
		}
	}
	flush(len(lines))
	return strings.Join(lines, "\n")
}

func (t *tfGen) backend(ns, svc string, port int, portName string) {
	w := t.w
	w.open("service")
	w.attr("name", t.ref("kubernetes_service_v1", ns, svc))
	w.open("port")
	if port > 0 {
		w.num("number", port)
	} else {
		w.str("name", portName)
	}
	w.close()
	w.close()
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func intOr(v *int, def int) int {
	if v != nil {
		return *v
	}
	return def
}

func optNum(w *hclWriter, k string, v *int) {
	if v != nil {
		w.num(k, *v)
	}
}

func ignoreFor(d *archData, wl *archWorkload) []string {
	var out []string
	if (wl.Kind == "Deployment" || wl.Kind == "StatefulSet") && hasHPA(d, wl) {
		out = append(out, "spec[0].replicas")
	}
	if wl.Kind == "StatefulSet" && len(wl.ClaimTpls) > 0 {
		out = append(out, "spec[0].volume_claim_template")
	}
	return out
}

func hasHPA(d *archData, wl *archWorkload) bool {
	for _, h := range d.HPAs {
		if h.Meta.Namespace == wl.Meta.Namespace && h.TargetName == wl.Meta.Name && strings.EqualFold(h.TargetKind, wl.Kind) {
			return true
		}
	}
	return false
}

func isClaimTemplatePVC(d *archData, p *archPVC) bool {
	for _, wl := range d.Workloads {
		if wl.Meta.Namespace != p.Meta.Namespace {
			continue
		}
		for _, ct := range wl.ClaimTpls {
			if strings.HasPrefix(p.Meta.Name, ct.Meta.Name+"-"+wl.Meta.Name+"-") {
				return true
			}
		}
	}
	return false
}
