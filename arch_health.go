package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

type workloadHealth struct {
	Pods       int
	Ready      int
	Restarts   int
	Severity   int
	Reason     string
	SamplePod  string
	CPUMilli   int
	MemMi      int
	HasMetrics bool
	PodNames   []string
}

func parseCPUMilli(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	if strings.HasSuffix(s, "m") {
		v, err := strconv.Atoi(strings.TrimSuffix(s, "m"))
		return v, err == nil
	}
	if strings.HasSuffix(s, "n") {
		v, err := strconv.Atoi(strings.TrimSuffix(s, "n"))
		return v / 1000000, err == nil
	}
	f, err := strconv.ParseFloat(s, 64)
	return int(f * 1000), err == nil
}

func parseMemMi(s string) (int, bool) {
	s = strings.TrimSpace(s)
	units := []struct {
		suf string
		mul float64
	}{{"Ki", 1.0 / 1024}, {"Mi", 1}, {"Gi", 1024}, {"Ti", 1024 * 1024}, {"K", 1000.0 / 1048576}, {"M", 1000000.0 / 1048576}, {"G", 1e9 / 1048576}}
	for _, u := range units {
		if strings.HasSuffix(s, u.suf) {
			f, err := strconv.ParseFloat(strings.TrimSuffix(s, u.suf), 64)
			return int(f * u.mul), err == nil
		}
	}
	f, err := strconv.ParseFloat(s, 64)
	return int(f / 1048576), err == nil
}

func fmtCPU(m int) string {
	if m >= 1000 {
		return strconv.FormatFloat(float64(m)/1000, 'f', 1, 64) + " cores"
	}
	return itoa(m) + "m"
}

func fmtMem(mi int) string {
	if mi >= 1024 {
		return strconv.FormatFloat(float64(mi)/1024, 'f', 1, 64) + "Gi"
	}
	return itoa(mi) + "Mi"
}

func collectWorkloadHealth(d *archData, nsFilter string, includeSystem bool) map[string]*workloadHealth {
	var pods []podView
	var met metricsResp
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); pods = allPods() }()
	go func() { defer wg.Done(); met = collectMetrics() }()
	wg.Wait()

	usage := map[string][2]int{}
	hasMet := map[string]bool{}
	for _, m := range met.Pods {
		c, ok1 := parseCPUMilli(m.CPU)
		mm, ok2 := parseMemMi(m.Memory)
		if ok1 || ok2 {
			usage[nsKey(m.NS, m.Name)] = [2]int{c, mm}
			hasMet[nsKey(m.NS, m.Name)] = true
		}
	}

	byNS := map[string][]podView{}
	for _, p := range pods {
		byNS[p.Metadata.Namespace] = append(byNS[p.Metadata.Namespace], p)
	}

	out := map[string]*workloadHealth{}
	for _, w := range d.Workloads {
		if !inScope(w.Meta.Namespace, nsFilter, includeSystem) || w.Kind == "CronJob" || len(w.Selector) == 0 {
			continue
		}
		h := &workloadHealth{}
		worst := -1
		for _, p := range byNS[w.Meta.Namespace] {
			if p.Metadata.DeletionTimestamp != nil || !matchSelector(w.Selector, p.Metadata.Labels) {
				continue
			}
			h.Pods++
			if p.total > 0 && p.ready == p.total {
				h.Ready++
			}
			h.Restarts += p.restarts
			k := nsKey(p.Metadata.Namespace, p.Metadata.Name)
			if u, ok := usage[k]; ok {
				h.CPUMilli += u[0]
				h.MemMi += u[1]
				h.HasMetrics = h.HasMetrics || hasMet[k]
			}
			if len(h.PodNames) < 50 {
				h.PodNames = append(h.PodNames, p.Metadata.Name)
			}
			if p.severity > worst {
				worst = p.severity
				h.Severity = p.severity
				h.Reason = p.reason
				h.SamplePod = p.Metadata.Name
			}
		}
		sort.Strings(h.PodNames)
		out[workloadID(w)] = h
	}
	return out
}

func applyHealth(g *graphBuilder, health map[string]*workloadHealth) {
	for id, h := range health {
		n := g.nodes[id]
		if n == nil {
			continue
		}
		if n.Meta == nil {
			n.Meta = map[string]string{}
		}
		n.Meta["pods"] = itoa(h.Pods)
		n.Meta["podsReady"] = itoa(h.Ready)
		n.Meta["restarts"] = itoa(h.Restarts)
		if h.HasMetrics {
			n.Meta["cpu"] = fmtCPU(h.CPUMilli)
			n.Meta["mem"] = fmtMem(h.MemMi)
		}
		if h.SamplePod != "" && h.Severity > 0 {
			n.Meta["samplePod"] = h.SamplePod
			n.Meta["reason"] = h.Reason
		}
		if len(h.PodNames) > 0 {
			n.Meta["podNames"] = strings.Join(h.PodNames, ",")
		}
		if h.Severity > n.Sev {
			n.Sev = h.Severity
		}
	}
}
