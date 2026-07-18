package main

import (
	"encoding/json"
	"testing"
)

func TestClassifyEnv(t *testing.T) {
	cases := []struct{ in, want string }{
		{"production", "prod"},
		{"betgo-prd-eks", "prod"},
		{"staging", "staging"},
		{"betgo-stg-eks", "staging"},
		{"homolog", "staging"},
		{"development", "dev"},
		{"reals-dev-eks", "dev"},
		{"sandbox", "dev"},
		{"kube-system", "other"},
		{"datadog", "other"},
	}
	for _, c := range cases {
		if got := classifyEnv(c.in); got != c.want {
			t.Errorf("classifyEnv(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestClusterShortName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"arn:aws:eks:sa-east-1:229787251669:cluster/reals-dev-eks", "reals-dev-eks"},
		{"arn:aws:eks:sa-east-1:847263217323:cluster/betgo-stg-eks", "betgo-stg-eks"},
		{"minikube", "minikube"},
		{"kind-mycluster", "kind-mycluster"},
	}
	for _, c := range cases {
		if got := clusterShortName(c.in); got != c.want {
			t.Errorf("clusterShortName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestAppName(t *testing.T) {
	cases := []struct {
		name, json, want string
	}{
		{
			"replicaset owner strips hash suffix",
			`{"metadata":{"name":"api-7d8f9c9c8-abcde","ownerReferences":[{"kind":"ReplicaSet","name":"api-7d8f9c9c8"}]}}`,
			"api",
		},
		{
			"non-replicaset owner kept whole",
			`{"metadata":{"name":"job-123","ownerReferences":[{"kind":"Job","name":"cronjob-xyz-29718506"}]}}`,
			"cronjob-xyz-29718506",
		},
		{
			"no owner falls back to label",
			`{"metadata":{"name":"standalone","labels":{"app.kubernetes.io/name":"myapp"}}}`,
			"myapp",
		},
		{
			"no owner, no label falls back to pod name",
			`{"metadata":{"name":"bare-pod"}}`,
			"bare-pod",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var p pod
			if err := json.Unmarshal([]byte(c.json), &p); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got := appName(p); got != c.want {
				t.Errorf("appName() = %q, want %q", got, c.want)
			}
		})
	}
}
