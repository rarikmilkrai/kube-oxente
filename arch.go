package main

import (
	"encoding/base64"
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"sync"
)

var archScanSecrets bool

type k8sMeta struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

type envVar struct {
	Name      string `json:"name"`
	Value     string `json:"value"`
	ValueFrom *struct {
		ConfigMapKeyRef *struct{ Name, Key string } `json:"configMapKeyRef"`
		SecretKeyRef    *struct{ Name, Key string } `json:"secretKeyRef"`
	} `json:"valueFrom"`
}

type containerSpec struct {
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	Command []string `json:"command"`
	Args    []string `json:"args"`
	Env     []envVar `json:"env"`
	EnvFrom []struct {
		ConfigMapRef *struct{ Name string } `json:"configMapRef"`
		SecretRef    *struct{ Name string } `json:"secretRef"`
	} `json:"envFrom"`
	Ports []struct {
		Name          string `json:"name"`
		ContainerPort int    `json:"containerPort"`
		Protocol      string `json:"protocol"`
	} `json:"ports"`
	Resources struct {
		Limits   map[string]string `json:"limits"`
		Requests map[string]string `json:"requests"`
	} `json:"resources"`
	VolumeMounts []struct {
		Name      string `json:"name"`
		MountPath string `json:"mountPath"`
		ReadOnly  bool   `json:"readOnly"`
		SubPath   string `json:"subPath"`
	} `json:"volumeMounts"`
}

type volumeSpec struct {
	Name      string `json:"name"`
	ConfigMap *struct {
		Name string `json:"name"`
	} `json:"configMap"`
	Secret *struct {
		SecretName string `json:"secretName"`
	} `json:"secret"`
	PersistentVolumeClaim *struct {
		ClaimName string `json:"claimName"`
	} `json:"persistentVolumeClaim"`
	EmptyDir  *struct{} `json:"emptyDir"`
	Projected *struct {
		Sources []struct {
			ConfigMap *struct{ Name string } `json:"configMap"`
			Secret    *struct{ Name string } `json:"secret"`
		} `json:"sources"`
	} `json:"projected"`
}

type podTemplate struct {
	Metadata struct {
		Labels      map[string]string `json:"labels"`
		Annotations map[string]string `json:"annotations"`
	} `json:"metadata"`
	Spec struct {
		ServiceAccountName string            `json:"serviceAccountName"`
		InitContainers     []containerSpec   `json:"initContainers"`
		Containers         []containerSpec   `json:"containers"`
		Volumes            []volumeSpec      `json:"volumes"`
		NodeSelector       map[string]string `json:"nodeSelector"`
		RestartPolicy      string            `json:"restartPolicy"`
		AutomountSAToken   *bool             `json:"automountServiceAccountToken"`
		EnableServiceLinks *bool             `json:"enableServiceLinks"`
	} `json:"spec"`
}

type archWorkload struct {
	Kind        string
	Meta        k8sMeta
	Replicas    *int
	Ready       int
	Desired     int
	Selector    map[string]string
	Template    podTemplate
	Schedule    string
	Suspend     bool
	CronSpec    cronSpec
	ClaimTpls   []archPVC
	ServiceName string
}

type archService struct {
	Meta         k8sMeta
	Type         string
	Selector     map[string]string
	ExternalName string
	Ports        []struct {
		Name       string       `json:"name"`
		Port       int          `json:"port"`
		Protocol   string       `json:"protocol"`
		TargetPort *intOrString `json:"targetPort"`
		NodePort   int          `json:"nodePort"`
	}
	LBHosts []string
}

type archIngressPath struct {
	Path     string
	PathType string
	Service  string
	Port     int
	PortName string
}

type archIngressRule struct {
	Host  string
	Paths []archIngressPath
}

type archIngress struct {
	Meta           k8sMeta
	Class          string
	Rules          []archIngressRule
	DefaultService string
	DefaultPort    int
	TLS            []struct {
		Hosts      []string `json:"hosts"`
		SecretName string   `json:"secretName"`
	}
	LBHosts []string
}

type archPVC struct {
	Meta         k8sMeta
	StorageClass string
	AccessModes  []string
	Size         string
	Phase        string
}

type archHPA struct {
	Meta       k8sMeta
	TargetKind string
	TargetName string
	TargetAPI  string
	Min        int
	Max        int
	Current    int
	CPUTarget  int
	MemTarget  int
}

type archNetPol struct {
	Meta        k8sMeta
	PodSelector map[string]string
	PolicyTypes []string
	Ingress     []archNetPolPeerSet
	Egress      []archNetPolPeerSet
}

type archNetPolPeerSet struct {
	Peers []archNetPolPeer
	Ports []string
}

type archNetPolPeer struct {
	PodSelector       map[string]string
	PodSelectorSet    bool
	NamespaceSelector map[string]string
	NamespaceSet      bool
	CIDR              string
}

type archData struct {
	Namespaces map[string]map[string]string
	Workloads  []*archWorkload
	Services   []*archService
	Ingresses  []*archIngress
	ConfigMaps map[string]map[string]string
	Secrets    map[string]archSecret
	PVCs       map[string]*archPVC
	HPAs       []*archHPA
	NetPols    []*archNetPol
}

type archSecret struct {
	Type string
	Keys []string
	Data map[string]string
}

var systemNamespaces = map[string]bool{
	"kube-system": true, "kube-public": true, "kube-node-lease": true,
	"local-path-storage": true,
}

func isSystemNS(ns string) bool { return systemNamespaces[ns] }

func nsKey(ns, name string) string { return ns + "/" + name }

func loadArchData() *archData {
	kinds := []string{"namespaces", "deployments", "statefulsets", "daemonsets", "cronjobs",
		"services", "ingresses", "configmaps", "persistentvolumeclaims",
		"horizontalpodautoscalers", "networkpolicies"}
	if archScanSecrets {
		kinds = append(kinds, "secrets")
	}
	raws := make(map[string][]json.RawMessage, len(kinds))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, k := range kinds {
		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			items := getList(k)
			mu.Lock()
			raws[k] = items
			mu.Unlock()
		}(k)
	}
	wg.Wait()

	d := &archData{
		Namespaces: map[string]map[string]string{},
		ConfigMaps: map[string]map[string]string{},
		Secrets:    map[string]archSecret{},
		PVCs:       map[string]*archPVC{},
	}

	for _, raw := range raws["namespaces"] {
		var o struct{ Metadata k8sMeta }
		if json.Unmarshal(raw, &o) == nil {
			d.Namespaces[o.Metadata.Name] = o.Metadata.Labels
		}
	}

	for _, kind := range []string{"deployments", "statefulsets", "daemonsets"} {
		for _, raw := range raws[kind] {
			var o struct {
				Metadata k8sMeta `json:"metadata"`
				Spec     struct {
					Replicas    *int   `json:"replicas"`
					ServiceName string `json:"serviceName"`
					Selector    struct {
						MatchLabels map[string]string `json:"matchLabels"`
					} `json:"selector"`
					Template             podTemplate `json:"template"`
					VolumeClaimTemplates []struct {
						Metadata k8sMeta `json:"metadata"`
						Spec     struct {
							StorageClassName *string  `json:"storageClassName"`
							AccessModes      []string `json:"accessModes"`
							Resources        struct {
								Requests map[string]string `json:"requests"`
							} `json:"resources"`
						} `json:"spec"`
					} `json:"volumeClaimTemplates"`
				} `json:"spec"`
				Status struct {
					Replicas               int `json:"replicas"`
					ReadyReplicas          int `json:"readyReplicas"`
					DesiredNumberScheduled int `json:"desiredNumberScheduled"`
					NumberReady            int `json:"numberReady"`
				} `json:"status"`
			}
			if json.Unmarshal(raw, &o) != nil {
				continue
			}
			w := &archWorkload{
				Kind: map[string]string{"deployments": "Deployment", "statefulsets": "StatefulSet", "daemonsets": "DaemonSet"}[kind],
				Meta: o.Metadata, Replicas: o.Spec.Replicas, Selector: o.Spec.Selector.MatchLabels,
				Template: o.Spec.Template, ServiceName: o.Spec.ServiceName,
			}
			if kind == "daemonsets" {
				w.Desired, w.Ready = o.Status.DesiredNumberScheduled, o.Status.NumberReady
			} else {
				w.Ready = o.Status.ReadyReplicas
				w.Desired = 1
				if o.Spec.Replicas != nil {
					w.Desired = *o.Spec.Replicas
				}
			}
			for _, vct := range o.Spec.VolumeClaimTemplates {
				p := archPVC{Meta: vct.Metadata, AccessModes: vct.Spec.AccessModes, Size: vct.Spec.Resources.Requests["storage"]}
				p.Meta.Namespace = o.Metadata.Namespace
				if vct.Spec.StorageClassName != nil {
					p.StorageClass = *vct.Spec.StorageClassName
				}
				w.ClaimTpls = append(w.ClaimTpls, p)
			}
			d.Workloads = append(d.Workloads, w)
		}
	}

	for _, raw := range raws["cronjobs"] {
		var o struct {
			Metadata k8sMeta  `json:"metadata"`
			Spec     cronSpec `json:"spec"`
		}
		if json.Unmarshal(raw, &o) != nil {
			continue
		}
		d.Workloads = append(d.Workloads, &archWorkload{
			Kind: "CronJob", Meta: o.Metadata, Template: o.Spec.JobTemplate.Spec.Template,
			Schedule: o.Spec.Schedule, Suspend: o.Spec.Suspend, CronSpec: o.Spec,
		})
	}

	for _, raw := range raws["services"] {
		var o struct {
			Metadata k8sMeta `json:"metadata"`
			Spec     struct {
				Type         string            `json:"type"`
				Selector     map[string]string `json:"selector"`
				ExternalName string            `json:"externalName"`
				Ports        []struct {
					Name       string       `json:"name"`
					Port       int          `json:"port"`
					Protocol   string       `json:"protocol"`
					TargetPort *intOrString `json:"targetPort"`
					NodePort   int          `json:"nodePort"`
				} `json:"ports"`
			} `json:"spec"`
			Status struct {
				LoadBalancer struct {
					Ingress []struct{ IP, Hostname string } `json:"ingress"`
				} `json:"loadBalancer"`
			} `json:"status"`
		}
		if json.Unmarshal(raw, &o) != nil {
			continue
		}
		s := &archService{Meta: o.Metadata, Type: o.Spec.Type, Selector: o.Spec.Selector,
			ExternalName: o.Spec.ExternalName, Ports: o.Spec.Ports}
		for _, lb := range o.Status.LoadBalancer.Ingress {
			s.LBHosts = append(s.LBHosts, firstNonEmpty(lb.Hostname, lb.IP))
		}
		d.Services = append(d.Services, s)
	}

	for _, raw := range raws["ingresses"] {
		var o struct {
			Metadata k8sMeta `json:"metadata"`
			Spec     struct {
				IngressClassName *string `json:"ingressClassName"`
				DefaultBackend   *struct {
					Service *ingBackendSvc `json:"service"`
				} `json:"defaultBackend"`
				Rules []struct {
					Host string `json:"host"`
					HTTP *struct {
						Paths []struct {
							Path     string `json:"path"`
							PathType string `json:"pathType"`
							Backend  struct {
								Service *ingBackendSvc `json:"service"`
							} `json:"backend"`
						} `json:"paths"`
					} `json:"http"`
				} `json:"rules"`
				TLS []struct {
					Hosts      []string `json:"hosts"`
					SecretName string   `json:"secretName"`
				} `json:"tls"`
			} `json:"spec"`
			Status struct {
				LoadBalancer struct {
					Ingress []struct{ IP, Hostname string } `json:"ingress"`
				} `json:"loadBalancer"`
			} `json:"status"`
		}
		if json.Unmarshal(raw, &o) != nil {
			continue
		}
		ing := &archIngress{Meta: o.Metadata, TLS: o.Spec.TLS}
		if o.Spec.IngressClassName != nil {
			ing.Class = *o.Spec.IngressClassName
		}
		if ing.Class == "" {
			ing.Class = o.Metadata.Annotations["kubernetes.io/ingress.class"]
		}
		if o.Spec.DefaultBackend != nil && o.Spec.DefaultBackend.Service != nil {
			ing.DefaultService = o.Spec.DefaultBackend.Service.Name
			ing.DefaultPort = o.Spec.DefaultBackend.Service.Port.Number
		}
		for _, r := range o.Spec.Rules {
			rule := archIngressRule{Host: r.Host}
			if r.HTTP != nil {
				for _, p := range r.HTTP.Paths {
					if p.Backend.Service == nil {
						continue
					}
					rule.Paths = append(rule.Paths, archIngressPath{
						Path: p.Path, PathType: p.PathType, Service: p.Backend.Service.Name,
						Port: p.Backend.Service.Port.Number, PortName: p.Backend.Service.Port.Name,
					})
				}
			}
			ing.Rules = append(ing.Rules, rule)
		}
		for _, lb := range o.Status.LoadBalancer.Ingress {
			ing.LBHosts = append(ing.LBHosts, firstNonEmpty(lb.Hostname, lb.IP))
		}
		d.Ingresses = append(d.Ingresses, ing)
	}

	for _, raw := range raws["configmaps"] {
		var o struct {
			Metadata k8sMeta           `json:"metadata"`
			Data     map[string]string `json:"data"`
		}
		if json.Unmarshal(raw, &o) == nil {
			d.ConfigMaps[nsKey(o.Metadata.Namespace, o.Metadata.Name)] = o.Data
		}
	}

	for _, raw := range raws["secrets"] {
		var o struct {
			Metadata k8sMeta           `json:"metadata"`
			Type     string            `json:"type"`
			Data     map[string]string `json:"data"`
		}
		if json.Unmarshal(raw, &o) != nil {
			continue
		}
		s := archSecret{Type: o.Type, Data: map[string]string{}}
		for k, v := range o.Data {
			s.Keys = append(s.Keys, k)
			if b, err := base64.StdEncoding.DecodeString(v); err == nil && len(b) < 4096 {
				s.Data[k] = string(b)
			}
		}
		sort.Strings(s.Keys)
		d.Secrets[nsKey(o.Metadata.Namespace, o.Metadata.Name)] = s
	}

	for _, raw := range raws["persistentvolumeclaims"] {
		var o struct {
			Metadata k8sMeta `json:"metadata"`
			Spec     struct {
				StorageClassName *string  `json:"storageClassName"`
				AccessModes      []string `json:"accessModes"`
				Resources        struct {
					Requests map[string]string `json:"requests"`
				} `json:"resources"`
			} `json:"spec"`
			Status struct {
				Phase string `json:"phase"`
			} `json:"status"`
		}
		if json.Unmarshal(raw, &o) != nil {
			continue
		}
		p := &archPVC{Meta: o.Metadata, AccessModes: o.Spec.AccessModes,
			Size: o.Spec.Resources.Requests["storage"], Phase: o.Status.Phase}
		if o.Spec.StorageClassName != nil {
			p.StorageClass = *o.Spec.StorageClassName
		}
		d.PVCs[nsKey(o.Metadata.Namespace, o.Metadata.Name)] = p
	}

	for _, raw := range raws["horizontalpodautoscalers"] {
		var o struct {
			Metadata k8sMeta `json:"metadata"`
			Spec     struct {
				ScaleTargetRef struct {
					APIVersion string `json:"apiVersion"`
					Kind       string `json:"kind"`
					Name       string `json:"name"`
				} `json:"scaleTargetRef"`
				MinReplicas                    *int `json:"minReplicas"`
				MaxReplicas                    int  `json:"maxReplicas"`
				TargetCPUUtilizationPercentage *int `json:"targetCPUUtilizationPercentage"`
				Metrics                        []struct {
					Type     string `json:"type"`
					Resource *struct {
						Name   string `json:"name"`
						Target struct {
							AverageUtilization *int `json:"averageUtilization"`
						} `json:"target"`
					} `json:"resource"`
				} `json:"metrics"`
			} `json:"spec"`
			Status struct {
				CurrentReplicas int `json:"currentReplicas"`
			} `json:"status"`
		}
		if json.Unmarshal(raw, &o) != nil {
			continue
		}
		h := &archHPA{Meta: o.Metadata, TargetKind: o.Spec.ScaleTargetRef.Kind,
			TargetName: o.Spec.ScaleTargetRef.Name, TargetAPI: o.Spec.ScaleTargetRef.APIVersion,
			Max: o.Spec.MaxReplicas, Min: 1, Current: o.Status.CurrentReplicas}
		if o.Spec.MinReplicas != nil {
			h.Min = *o.Spec.MinReplicas
		}
		if o.Spec.TargetCPUUtilizationPercentage != nil {
			h.CPUTarget = *o.Spec.TargetCPUUtilizationPercentage
		}
		for _, m := range o.Spec.Metrics {
			if m.Resource == nil || m.Resource.Target.AverageUtilization == nil {
				continue
			}
			switch m.Resource.Name {
			case "cpu":
				h.CPUTarget = *m.Resource.Target.AverageUtilization
			case "memory":
				h.MemTarget = *m.Resource.Target.AverageUtilization
			}
		}
		d.HPAs = append(d.HPAs, h)
	}

	for _, raw := range raws["networkpolicies"] {
		var o struct {
			Metadata k8sMeta `json:"metadata"`
			Spec     struct {
				PodSelector labelSelector `json:"podSelector"`
				PolicyTypes []string      `json:"policyTypes"`
				Ingress     []struct {
					From  []netpolPeerJSON `json:"from"`
					Ports []netpolPortJSON `json:"ports"`
				} `json:"ingress"`
				Egress []struct {
					To    []netpolPeerJSON `json:"to"`
					Ports []netpolPortJSON `json:"ports"`
				} `json:"egress"`
			} `json:"spec"`
		}
		if json.Unmarshal(raw, &o) != nil {
			continue
		}
		np := &archNetPol{Meta: o.Metadata, PodSelector: o.Spec.PodSelector.MatchLabels, PolicyTypes: o.Spec.PolicyTypes}
		for _, r := range o.Spec.Ingress {
			np.Ingress = append(np.Ingress, archNetPolPeerSet{Peers: convPeers(r.From), Ports: convPorts(r.Ports)})
		}
		for _, r := range o.Spec.Egress {
			np.Egress = append(np.Egress, archNetPolPeerSet{Peers: convPeers(r.To), Ports: convPorts(r.Ports)})
		}
		d.NetPols = append(d.NetPols, np)
	}

	sort.Slice(d.Workloads, func(i, j int) bool {
		a, b := d.Workloads[i], d.Workloads[j]
		return a.Meta.Namespace+"/"+a.Meta.Name+a.Kind < b.Meta.Namespace+"/"+b.Meta.Name+b.Kind
	})
	sort.Slice(d.Services, func(i, j int) bool {
		return nsKey(d.Services[i].Meta.Namespace, d.Services[i].Meta.Name) < nsKey(d.Services[j].Meta.Namespace, d.Services[j].Meta.Name)
	})
	sort.Slice(d.Ingresses, func(i, j int) bool {
		return nsKey(d.Ingresses[i].Meta.Namespace, d.Ingresses[i].Meta.Name) < nsKey(d.Ingresses[j].Meta.Namespace, d.Ingresses[j].Meta.Name)
	})
	return d
}

type cronSpec struct {
	Schedule                   string `json:"schedule"`
	Suspend                    bool   `json:"suspend"`
	ConcurrencyPolicy          string `json:"concurrencyPolicy"`
	SuccessfulJobsHistoryLimit *int   `json:"successfulJobsHistoryLimit"`
	FailedJobsHistoryLimit     *int   `json:"failedJobsHistoryLimit"`
	StartingDeadlineSeconds    *int   `json:"startingDeadlineSeconds"`
	TimeZone                   string `json:"timeZone"`
	JobTemplate                struct {
		Spec struct {
			BackoffLimit            *int        `json:"backoffLimit"`
			Completions             *int        `json:"completions"`
			Parallelism             *int        `json:"parallelism"`
			ActiveDeadlineSeconds   *int        `json:"activeDeadlineSeconds"`
			TTLSecondsAfterFinished *int        `json:"ttlSecondsAfterFinished"`
			Template                podTemplate `json:"template"`
		} `json:"spec"`
	} `json:"jobTemplate"`
}

type ingBackendSvc struct {
	Name string `json:"name"`
	Port struct {
		Number int    `json:"number"`
		Name   string `json:"name"`
	} `json:"port"`
}

type labelSelector struct {
	MatchLabels      map[string]string `json:"matchLabels"`
	MatchExpressions []json.RawMessage `json:"matchExpressions"`
}

type netpolPeerJSON struct {
	PodSelector       *labelSelector `json:"podSelector"`
	NamespaceSelector *labelSelector `json:"namespaceSelector"`
	IPBlock           *struct {
		CIDR string `json:"cidr"`
	} `json:"ipBlock"`
}

type netpolPortJSON struct {
	Protocol string       `json:"protocol"`
	Port     *intOrString `json:"port"`
}

func convPeers(in []netpolPeerJSON) []archNetPolPeer {
	var out []archNetPolPeer
	for _, p := range in {
		ap := archNetPolPeer{}
		if p.PodSelector != nil {
			ap.PodSelectorSet = true
			ap.PodSelector = p.PodSelector.MatchLabels
		}
		if p.NamespaceSelector != nil {
			ap.NamespaceSet = true
			ap.NamespaceSelector = p.NamespaceSelector.MatchLabels
		}
		if p.IPBlock != nil {
			ap.CIDR = p.IPBlock.CIDR
		}
		out = append(out, ap)
	}
	return out
}

func convPorts(in []netpolPortJSON) []string {
	var out []string
	for _, p := range in {
		if p.Port == nil {
			continue
		}
		if p.Port.IsStr {
			out = append(out, p.Port.StrVal)
		} else {
			out = append(out, itoa(p.Port.IntVal))
		}
	}
	return out
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

func selectorSubset(sel, labels map[string]string) bool {
	for k, v := range sel {
		if labels[k] != v {
			return false
		}
	}
	return true
}

type archNode struct {
	ID    string            `json:"id"`
	Kind  string            `json:"kind"`
	Name  string            `json:"name"`
	NS    string            `json:"ns,omitempty"`
	Sub   string            `json:"sub,omitempty"`
	Sev   int               `json:"sev"`
	Layer int               `json:"layer"`
	Group string            `json:"group,omitempty"`
	Meta  map[string]string `json:"meta,omitempty"`
}

type archEdge struct {
	ID       string   `json:"id"`
	From     string   `json:"from"`
	To       string   `json:"to"`
	Kind     string   `json:"kind"`
	Label    string   `json:"label,omitempty"`
	Evidence []string `json:"evidence,omitempty"`
}

type archGraph struct {
	Context string         `json:"context"`
	Nodes   []archNode     `json:"nodes"`
	Edges   []archEdge     `json:"edges"`
	Stats   map[string]int `json:"stats"`
	Secrets bool           `json:"secretsScanned"`
}

type graphBuilder struct {
	nodes map[string]*archNode
	order []string
	edges map[string]*archEdge
	eord  []string
}

func newGraphBuilder() *graphBuilder {
	return &graphBuilder{nodes: map[string]*archNode{}, edges: map[string]*archEdge{}}
}

func (g *graphBuilder) node(n archNode) *archNode {
	if ex, ok := g.nodes[n.ID]; ok {
		return ex
	}
	nn := n
	g.nodes[n.ID] = &nn
	g.order = append(g.order, n.ID)
	return &nn
}

func (g *graphBuilder) edge(from, to, kind, label, evidence string) {
	if from == to || g.nodes[from] == nil || g.nodes[to] == nil {
		return
	}
	id := from + ">" + to + ">" + kind
	e, ok := g.edges[id]
	if !ok {
		e = &archEdge{ID: id, From: from, To: to, Kind: kind, Label: label}
		g.edges[id] = e
		g.eord = append(g.eord, id)
	}
	if label != "" && e.Label == "" {
		e.Label = label
	}
	if evidence != "" && len(e.Evidence) < 6 {
		for _, x := range e.Evidence {
			if x == evidence {
				return
			}
		}
		e.Evidence = append(e.Evidence, evidence)
	}
}

func workloadID(w *archWorkload) string {
	return "wl:" + strings.ToLower(w.Kind) + ":" + nsKey(w.Meta.Namespace, w.Meta.Name)
}

func inScope(ns, filter string, includeSystem bool) bool {
	if filter != "" {
		return ns == filter
	}
	return includeSystem || !isSystemNS(ns)
}

func collectArchitecture(nsFilter string, includeSystem bool) archGraph {
	d := loadArchData()
	g := newGraphBuilder()

	svcByKey := map[string]*archService{}
	svcNames := map[string][]string{}
	for _, s := range d.Services {
		svcByKey[nsKey(s.Meta.Namespace, s.Meta.Name)] = s
		svcNames[s.Meta.Name] = append(svcNames[s.Meta.Name], s.Meta.Namespace)
	}

	internet := archNode{ID: "ext:internet", Kind: "internet", Name: "Internet", Layer: 0}

	for _, w := range d.Workloads {
		if !inScope(w.Meta.Namespace, nsFilter, includeSystem) {
			continue
		}
		sev := 0
		sub := w.Kind
		switch w.Kind {
		case "CronJob":
			sub = "CronJob · " + w.Schedule
			if w.Suspend {
				sub += " · suspenso"
			}
		default:
			sub = w.Kind + " · " + itoa(w.Ready) + "/" + itoa(w.Desired)
			if w.Desired > 0 && w.Ready == 0 {
				sev = 2
			} else if w.Ready < w.Desired {
				sev = 1
			}
		}
		meta := map[string]string{"kind": w.Kind}
		var images []string
		for _, c := range w.Template.Spec.Containers {
			images = append(images, c.Image)
		}
		meta["images"] = strings.Join(images, "\n")
		if w.Template.Spec.ServiceAccountName != "" {
			meta["serviceAccount"] = w.Template.Spec.ServiceAccountName
		}
		g.node(archNode{ID: workloadID(w), Kind: "workload", Name: w.Meta.Name, NS: w.Meta.Namespace,
			Sub: sub, Sev: sev, Layer: 3, Group: w.Meta.Namespace, Meta: meta})
	}

	for _, s := range d.Services {
		ns := s.Meta.Namespace
		if !inScope(ns, nsFilter, includeSystem) || (ns == "default" && s.Meta.Name == "kubernetes") {
			continue
		}
		var ports []string
		for _, p := range s.Ports {
			ports = append(ports, itoa(p.Port))
		}
		sub := firstNonEmpty(s.Type, "ClusterIP")
		if len(ports) > 0 {
			sub += " · " + strings.Join(ports, ",")
		}
		sid := "svc:" + nsKey(ns, s.Meta.Name)
		g.node(archNode{ID: sid, Kind: "service", Name: s.Meta.Name, NS: ns, Sub: sub, Layer: 2, Group: ns,
			Meta: map[string]string{"type": s.Type}})
		if s.Type == "ExternalName" && s.ExternalName != "" {
			en := externalNode(s.ExternalName)
			g.node(en)
			g.edge(sid, en.ID, "dns", "ExternalName", "")
		}
		if s.Type == "LoadBalancer" || s.Type == "NodePort" {
			g.node(internet)
			label := s.Type
			if len(s.LBHosts) > 0 {
				lb := archNode{ID: "lb:" + s.LBHosts[0], Kind: "lb", Name: shortLB(s.LBHosts[0]), Sub: "Load Balancer", Layer: 1,
					Meta: map[string]string{"host": s.LBHosts[0]}}
				g.node(lb)
				g.edge(internet.ID, lb.ID, "traffic", "", "")
				g.edge(lb.ID, sid, "traffic", label, "")
			} else {
				g.edge(internet.ID, sid, "traffic", label, "")
			}
		}
		if len(s.Selector) == 0 {
			continue
		}
		matched := false
		for _, w := range d.Workloads {
			if w.Meta.Namespace != ns || w.Kind == "CronJob" {
				continue
			}
			if matchSelector(s.Selector, w.Template.Metadata.Labels) {
				g.edge(sid, workloadID(w), "routes", strings.Join(ports, ","), "selector")
				matched = true
			}
		}
		if !matched {
			if n := g.nodes[sid]; n != nil {
				n.Sev = 1
				n.Sub += " · sem workload"
			}
		}
	}

	for _, ing := range d.Ingresses {
		ns := ing.Meta.Namespace
		if !inScope(ns, nsFilter, includeSystem) {
			continue
		}
		var hosts []string
		for _, r := range ing.Rules {
			if r.Host != "" {
				hosts = append(hosts, r.Host)
			}
		}
		iid := "ing:" + nsKey(ns, ing.Meta.Name)
		sub := "Ingress"
		if ing.Class != "" {
			sub += " · " + ing.Class
		}
		meta := map[string]string{"hosts": strings.Join(hosts, "\n")}
		if grp := ing.Meta.Annotations["alb.ingress.kubernetes.io/group.name"]; grp != "" {
			meta["albGroup"] = grp
		}
		g.node(archNode{ID: iid, Kind: "ingress", Name: ing.Meta.Name, NS: ns, Sub: sub, Layer: 1, Group: ns, Meta: meta})
		g.node(internet)
		if len(ing.LBHosts) > 0 {
			lb := archNode{ID: "lb:" + ing.LBHosts[0], Kind: "lb", Name: shortLB(ing.LBHosts[0]), Sub: lbKind(ing), Layer: 0,
				Meta: map[string]string{"host": ing.LBHosts[0]}}
			g.node(lb)
			g.edge(internet.ID, lb.ID, "traffic", "", "")
			g.edge(lb.ID, iid, "traffic", strings.Join(hosts, ", "), "")
		} else {
			g.edge(internet.ID, iid, "traffic", strings.Join(hosts, ", "), "")
		}
		add := func(svc string, port int, path, host string) {
			if svc == "" {
				return
			}
			sid := "svc:" + nsKey(ns, svc)
			if g.nodes[sid] == nil {
				g.node(archNode{ID: sid, Kind: "service", Name: svc, NS: ns, Sub: "não existe", Sev: 2, Layer: 2, Group: ns})
			}
			label := host + path
			g.edge(iid, sid, "routes", label, "")
		}
		add(ing.DefaultService, ing.DefaultPort, "/*", "")
		for _, r := range ing.Rules {
			for _, p := range r.Paths {
				add(p.Service, p.Port, p.Path, r.Host)
			}
		}
	}

	for _, w := range d.Workloads {
		ns := w.Meta.Namespace
		if !inScope(ns, nsFilter, includeSystem) {
			continue
		}
		wid := workloadID(w)
		cms, secs, pvcs := workloadRefs(w)
		for _, cm := range cms {
			id := "cm:" + nsKey(ns, cm)
			sub := "ConfigMap"
			sev := 0
			if _, ok := d.ConfigMaps[nsKey(ns, cm)]; !ok && len(d.ConfigMaps) > 0 {
				sub, sev = "ConfigMap · não existe", 2
			}
			g.node(archNode{ID: id, Kind: "configmap", Name: cm, NS: ns, Sub: sub, Sev: sev, Layer: 4, Group: ns})
			g.edge(wid, id, "config", "", "")
		}
		for _, s := range secs {
			id := "secret:" + nsKey(ns, s)
			g.node(archNode{ID: id, Kind: "secret", Name: s, NS: ns, Sub: "Secret", Layer: 4, Group: ns})
			g.edge(wid, id, "config", "", "")
		}
		for _, p := range pvcs {
			id := "pvc:" + nsKey(ns, p)
			sub := "PVC"
			sev := 0
			if pv := d.PVCs[nsKey(ns, p)]; pv != nil {
				sub = "PVC · " + firstNonEmpty(pv.Size, "?") + " · " + firstNonEmpty(pv.StorageClass, "default")
				if pv.Phase != "" && pv.Phase != "Bound" {
					sev = 2
					sub += " · " + pv.Phase
				}
			}
			g.node(archNode{ID: id, Kind: "pvc", Name: p, NS: ns, Sub: sub, Sev: sev, Layer: 4, Group: ns})
			g.edge(wid, id, "storage", "", "")
		}
		for _, ct := range w.ClaimTpls {
			id := "pvc:" + nsKey(ns, ct.Meta.Name+"-"+w.Meta.Name+"-*")
			g.node(archNode{ID: id, Kind: "pvc", Name: ct.Meta.Name + "-" + w.Meta.Name + "-*", NS: ns,
				Sub: "PVC template · " + firstNonEmpty(ct.Size, "?"), Layer: 4, Group: ns})
			g.edge(wid, id, "storage", "", "")
		}

		for _, ref := range scanWorkloadRefs(w, d, svcByKey, svcNames) {
			if ref.svcKey != "" {
				s := svcByKey[ref.svcKey]
				if s == nil || !inScope(s.Meta.Namespace, nsFilter, includeSystem) {
					if s == nil {
						continue
					}
					g.node(archNode{ID: "svc:" + ref.svcKey, Kind: "service", Name: s.Meta.Name, NS: s.Meta.Namespace,
						Sub: "outro namespace", Layer: 2, Group: s.Meta.Namespace})
				}
				g.edge(wid, "svc:"+ref.svcKey, "calls", ref.port, ref.evidence)
				continue
			}
			en := externalNode(ref.host)
			g.node(en)
			g.edge(wid, en.ID, "calls", ref.port, ref.evidence)
		}
	}

	for _, h := range d.HPAs {
		ns := h.Meta.Namespace
		if !inScope(ns, nsFilter, includeSystem) {
			continue
		}
		var target string
		for _, w := range d.Workloads {
			if w.Meta.Namespace == ns && w.Meta.Name == h.TargetName && strings.EqualFold(w.Kind, h.TargetKind) {
				target = workloadID(w)
			}
		}
		if target == "" {
			continue
		}
		sub := "HPA · " + itoa(h.Min) + "–" + itoa(h.Max)
		if h.CPUTarget > 0 {
			sub += " · cpu " + itoa(h.CPUTarget) + "%"
		}
		sev := 0
		if h.Current >= h.Max && h.Max > 0 {
			sev = 1
			sub += " · no máximo"
		}
		id := "hpa:" + nsKey(ns, h.Meta.Name)
		g.node(archNode{ID: id, Kind: "hpa", Name: h.Meta.Name, NS: ns, Sub: sub, Sev: sev, Layer: 4, Group: ns})
		g.edge(id, target, "scales", "", "")
	}

	for _, np := range d.NetPols {
		ns := np.Meta.Namespace
		if !inScope(ns, nsFilter, includeSystem) {
			continue
		}
		targets := workloadsMatching(d, ns, np.PodSelector, true)
		for _, rule := range np.Ingress {
			port := strings.Join(rule.Ports, ",")
			for _, peer := range rule.Peers {
				if peer.CIDR != "" {
					cid := "cidr:" + peer.CIDR
					g.node(archNode{ID: cid, Kind: "cidr", Name: peer.CIDR, Sub: "ipBlock", Layer: 0})
					for _, t := range targets {
						g.edge(cid, workloadID(t), "netpol", port, "NetworkPolicy "+np.Meta.Name)
					}
					continue
				}
				for _, srcNS := range peerNamespaces(d, ns, peer) {
					var sel map[string]string
					if peer.PodSelectorSet {
						sel = peer.PodSelector
					}
					for _, src := range workloadsMatching(d, srcNS, sel, true) {
						if g.nodes[workloadID(src)] == nil {
							continue
						}
						for _, t := range targets {
							g.edge(workloadID(src), workloadID(t), "netpol", port, "NetworkPolicy "+np.Meta.Name)
						}
					}
				}
			}
		}
	}

	out := archGraph{Context: effectiveContext(), Stats: map[string]int{}, Secrets: archScanSecrets}
	for _, id := range g.order {
		n := g.nodes[id]
		out.Nodes = append(out.Nodes, *n)
		out.Stats[n.Kind]++
	}
	for _, id := range g.eord {
		e := g.edges[id]
		out.Edges = append(out.Edges, *e)
		out.Stats["edge:"+e.Kind]++
	}
	return out
}

func workloadsMatching(d *archData, ns string, sel map[string]string, emptyMatchesAll bool) []*archWorkload {
	var out []*archWorkload
	for _, w := range d.Workloads {
		if w.Meta.Namespace != ns {
			continue
		}
		if len(sel) == 0 {
			if emptyMatchesAll {
				out = append(out, w)
			}
			continue
		}
		if selectorSubset(sel, w.Template.Metadata.Labels) {
			out = append(out, w)
		}
	}
	return out
}

func peerNamespaces(d *archData, ns string, p archNetPolPeer) []string {
	if !p.NamespaceSet {
		return []string{ns}
	}
	var out []string
	for name, labels := range d.Namespaces {
		if len(p.NamespaceSelector) == 0 || selectorSubset(p.NamespaceSelector, labels) {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func workloadRefs(w *archWorkload) (cms, secrets, pvcs []string) {
	seen := map[string]bool{}
	add := func(list *[]string, kind, name string) {
		if name == "" || seen[kind+name] {
			return
		}
		seen[kind+name] = true
		*list = append(*list, name)
	}
	containers := append(append([]containerSpec{}, w.Template.Spec.InitContainers...), w.Template.Spec.Containers...)
	for _, c := range containers {
		for _, ef := range c.EnvFrom {
			if ef.ConfigMapRef != nil {
				add(&cms, "cm", ef.ConfigMapRef.Name)
			}
			if ef.SecretRef != nil {
				add(&secrets, "s", ef.SecretRef.Name)
			}
		}
		for _, e := range c.Env {
			if e.ValueFrom == nil {
				continue
			}
			if e.ValueFrom.ConfigMapKeyRef != nil {
				add(&cms, "cm", e.ValueFrom.ConfigMapKeyRef.Name)
			}
			if e.ValueFrom.SecretKeyRef != nil {
				add(&secrets, "s", e.ValueFrom.SecretKeyRef.Name)
			}
		}
	}
	for _, v := range w.Template.Spec.Volumes {
		switch {
		case v.ConfigMap != nil:
			if v.ConfigMap.Name != "kube-root-ca.crt" {
				add(&cms, "cm", v.ConfigMap.Name)
			}
		case v.Secret != nil:
			add(&secrets, "s", v.Secret.SecretName)
		case v.PersistentVolumeClaim != nil:
			add(&pvcs, "p", v.PersistentVolumeClaim.ClaimName)
		case v.Projected != nil:
			for _, src := range v.Projected.Sources {
				if src.ConfigMap != nil && src.ConfigMap.Name != "kube-root-ca.crt" {
					add(&cms, "cm", src.ConfigMap.Name)
				}
				if src.Secret != nil {
					add(&secrets, "s", src.Secret.Name)
				}
			}
		}
	}
	return
}

type archRef struct {
	svcKey   string
	host     string
	port     string
	evidence string
}

var (
	reSvcFQDN  = regexp.MustCompile(`(?i)\b([a-z0-9](?:[-a-z0-9]*[a-z0-9])?)\.([a-z0-9](?:[-a-z0-9]*[a-z0-9])?)\.svc(?:\.cluster\.local)?(?::(\d{2,5}))?`)
	reURLHost  = regexp.MustCompile(`(?i)\b[a-z][a-z0-9+.-]*://(?:[^@/\s:]+(?::[^@/\s]*)?@)?([a-z0-9](?:[-a-z0-9.]*[a-z0-9])?)(?::(\d{2,5}))?`)
	reHostPort = regexp.MustCompile(`(?i)^([a-z0-9](?:[-a-z0-9.]*[a-z0-9])?)(?::(\d{2,5}))?$`)
	reFQDN     = regexp.MustCompile(`(?i)\b((?:[a-z0-9](?:[-a-z0-9]*[a-z0-9])?\.)+(?:com|net|org|io|xyz|br|cloud|dev|app|internal|amazonaws\.com))(?::(\d{2,5}))?\b`)
	reHostKey  = regexp.MustCompile(`(?i)(HOST|ADDR|ADDRESS|URL|URI|ENDPOINT|SERVER|SERVERS|DSN|BROKER|BROKERS|BOOTSTRAP|CONNECTION|CONN)S?(_[A-Z0-9]+)?$`)
	reNotHost  = regexp.MustCompile(`(?i)(^|\.)(docker\.io|ghcr\.io|gcr\.io|quay\.io|ecr\.[a-z0-9-]+\.amazonaws\.com|dkr\.ecr|w3\.org|schema\.org|example\.com|apache\.org|github\.com|githubusercontent\.com|googleapis\.com|gstatic\.com)$`)
)

func scanWorkloadRefs(w *archWorkload, d *archData, svcByKey map[string]*archService, svcNames map[string][]string) []archRef {
	ns := w.Meta.Namespace
	var refs []archRef
	seen := map[string]bool{}
	push := func(r archRef) {
		k := r.svcKey + "|" + r.host
		if r.host != "" && strings.EqualFold(r.host, w.Meta.Name) {
			return
		}
		if r.svcKey == nsKey(ns, w.Meta.Name) {
			return
		}
		if seen[k] {
			return
		}
		seen[k] = true
		refs = append(refs, r)
	}

	scan := func(key, value, evidence string) {
		if value == "" || len(value) > 20000 {
			return
		}
		for _, m := range reSvcFQDN.FindAllStringSubmatch(value, -1) {
			k := nsKey(strings.ToLower(m[2]), strings.ToLower(m[1]))
			if svcByKey[k] != nil {
				push(archRef{svcKey: k, port: m[3], evidence: evidence})
			}
		}
		hostish := reHostKey.MatchString(key)
		var candidates [][2]string
		for _, m := range reURLHost.FindAllStringSubmatch(value, -1) {
			candidates = append(candidates, [2]string{m[1], m[2]})
		}
		if hostish {
			for _, part := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ' ' || r == ';' }) {
				if m := reHostPort.FindStringSubmatch(strings.TrimSpace(part)); m != nil {
					candidates = append(candidates, [2]string{m[1], m[2]})
				}
			}
		}
		for _, m := range reFQDN.FindAllStringSubmatch(value, -1) {
			candidates = append(candidates, [2]string{m[1], m[2]})
		}
		for _, c := range candidates {
			host := strings.ToLower(strings.TrimSuffix(c[0], "."))
			if strings.Contains(host, ".svc") {
				continue
			}
			if !strings.Contains(host, ".") {
				if svcByKey[nsKey(ns, host)] != nil {
					push(archRef{svcKey: nsKey(ns, host), port: c[1], evidence: evidence})
				}
				continue
			}
			parts := strings.Split(host, ".")
			if len(parts) == 2 && svcByKey[nsKey(parts[1], parts[0])] != nil {
				push(archRef{svcKey: nsKey(parts[1], parts[0]), port: c[1], evidence: evidence})
				continue
			}
			if reNotHost.MatchString(host) || isIPLike(host) && !hostish {
				continue
			}
			if strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".localhost") {
				continue
			}
			push(archRef{host: host, port: c[1], evidence: evidence})
		}
	}

	containers := append(append([]containerSpec{}, w.Template.Spec.InitContainers...), w.Template.Spec.Containers...)
	scannedCM := map[string]bool{}
	scannedSecret := map[string]bool{}
	scanCM := func(name, onlyKey string) {
		data, ok := d.ConfigMaps[nsKey(ns, name)]
		if !ok {
			return
		}
		tag := name + "|" + onlyKey
		if scannedCM[tag] {
			return
		}
		scannedCM[tag] = true
		for k, v := range data {
			if onlyKey != "" && k != onlyKey {
				continue
			}
			scan(k, v, "configmap "+name+" → "+k)
		}
	}
	scanSecret := func(name, onlyKey string) {
		if !archScanSecrets {
			return
		}
		s, ok := d.Secrets[nsKey(ns, name)]
		if !ok {
			return
		}
		tag := name + "|" + onlyKey
		if scannedSecret[tag] {
			return
		}
		scannedSecret[tag] = true
		for k, v := range s.Data {
			if onlyKey != "" && k != onlyKey {
				continue
			}
			scan(k, v, "secret "+name+" → "+k)
		}
	}
	for _, c := range containers {
		for _, e := range c.Env {
			if e.ValueFrom == nil {
				scan(e.Name, e.Value, "env "+e.Name+" ("+c.Name+")")
				continue
			}
			if r := e.ValueFrom.ConfigMapKeyRef; r != nil {
				if v, ok := d.ConfigMaps[nsKey(ns, r.Name)][r.Key]; ok {
					scan(e.Name, v, "env "+e.Name+" ← configmap "+r.Name)
				}
			}
			if r := e.ValueFrom.SecretKeyRef; r != nil && archScanSecrets {
				if v, ok := d.Secrets[nsKey(ns, r.Name)].Data[r.Key]; ok {
					scan(e.Name, v, "env "+e.Name+" ← secret "+r.Name)
				}
			}
		}
		for _, ef := range c.EnvFrom {
			if ef.ConfigMapRef != nil {
				scanCM(ef.ConfigMapRef.Name, "")
			}
			if ef.SecretRef != nil {
				scanSecret(ef.SecretRef.Name, "")
			}
		}
		for _, a := range append(append([]string{}, c.Command...), c.Args...) {
			scan("ARG", a, "args ("+c.Name+")")
		}
	}
	for _, v := range w.Template.Spec.Volumes {
		if v.ConfigMap != nil && v.ConfigMap.Name != "kube-root-ca.crt" {
			scanCM(v.ConfigMap.Name, "")
		}
		if v.Secret != nil {
			scanSecret(v.Secret.SecretName, "")
		}
	}
	return refs
}

func isIPLike(h string) bool {
	for _, r := range h {
		if (r < '0' || r > '9') && r != '.' {
			return false
		}
	}
	return true
}

var externalKinds = []struct {
	re   *regexp.Regexp
	kind string
	sub  string
}{
	{regexp.MustCompile(`\.rds\.amazonaws\.com$`), "db", "Amazon RDS"},
	{regexp.MustCompile(`\.cache\.amazonaws\.com$`), "cache", "ElastiCache"},
	{regexp.MustCompile(`\.mq\.[a-z0-9-]+\.amazonaws\.com$`), "queue", "Amazon MQ"},
	{regexp.MustCompile(`(^|\.)sqs\.[a-z0-9-]+\.amazonaws\.com$`), "queue", "Amazon SQS"},
	{regexp.MustCompile(`(^|\.)sns\.[a-z0-9-]+\.amazonaws\.com$`), "queue", "Amazon SNS"},
	{regexp.MustCompile(`(^|\.)kafka\.[a-z0-9-]+\.amazonaws\.com$`), "queue", "Amazon MSK"},
	{regexp.MustCompile(`(^|\.)s3[.-]([a-z0-9-]+\.)?amazonaws\.com$`), "storage", "Amazon S3"},
	{regexp.MustCompile(`\.es\.amazonaws\.com$|\.aoss\.amazonaws\.com$`), "search", "OpenSearch"},
	{regexp.MustCompile(`\.docdb\.amazonaws\.com$`), "db", "DocumentDB"},
	{regexp.MustCompile(`\.mongodb\.net$`), "db", "MongoDB Atlas"},
	{regexp.MustCompile(`(^|\.)secretsmanager\.[a-z0-9-]+\.amazonaws\.com$`), "aws", "Secrets Manager"},
	{regexp.MustCompile(`\.amazonaws\.com$`), "aws", "AWS"},
	{regexp.MustCompile(`(?i)(postgres|pgsql|mysql|maria|aurora|mongo|db)`), "db", "Banco de dados"},
	{regexp.MustCompile(`(?i)(redis|memcache|valkey)`), "cache", "Cache"},
	{regexp.MustCompile(`(?i)(rabbit|amqp|kafka|nats|sqs)`), "queue", "Mensageria"},
}

func externalNode(host string) archNode {
	host = strings.ToLower(host)
	kind, sub := "external", "Externo"
	for _, ek := range externalKinds {
		if ek.re.MatchString(host) {
			kind, sub = ek.kind, ek.sub
			break
		}
	}
	return archNode{ID: "ext:" + host, Kind: kind, Name: shortHost(host), Sub: sub, Layer: 5,
		Meta: map[string]string{"host": host}}
}

func shortHost(h string) string {
	if len(h) <= 34 {
		return h
	}
	parts := strings.Split(h, ".")
	if len(parts) > 2 {
		return parts[0] + "…" + strings.Join(parts[len(parts)-2:], ".")
	}
	return h[:32] + "…"
}

func shortLB(h string) string {
	if i := strings.Index(h, "."); i > 0 {
		return h[:i]
	}
	return h
}

func lbKind(ing *archIngress) string {
	switch {
	case ing.Class == "alb" || strings.HasPrefix(ing.Meta.Annotations["kubernetes.io/ingress.class"], "alb"):
		return "AWS ALB"
	case strings.Contains(ing.Class, "nginx"):
		return "NGINX LB"
	case strings.Contains(ing.Class, "kong"):
		return "Kong LB"
	}
	return "Load Balancer"
}
