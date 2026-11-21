package main

import (
	"strings"
	"testing"
)

func TestScanWorkloadRefs(t *testing.T) {
	w := &archWorkload{Kind: "Deployment", Meta: k8sMeta{Name: "api", Namespace: "loja"}}
	w.Template.Spec.Containers = []containerSpec{{
		Name: "api",
		Env: []envVar{
			{Name: "CATALOG_URL", Value: "http://catalog:8080/v1"},
			{Name: "PAY", Value: "payments.pagamentos.svc.cluster.local:9000"},
			{Name: "DB_HOST", Value: "x.cluster-1.sa-east-1.rds.amazonaws.com"},
			{Name: "APP_NAME", Value: "catalog"},
			{Name: "IMAGE_REPO", Value: "123.dkr.ecr.sa-east-1.amazonaws.com/api"},
			{Name: "SELF_URL", Value: "http://api:80"},
		},
	}}
	d := &archData{ConfigMaps: map[string]map[string]string{}, Secrets: map[string]archSecret{}}
	svc := map[string]*archService{
		"loja/catalog":        {Meta: k8sMeta{Name: "catalog", Namespace: "loja"}},
		"loja/api":            {Meta: k8sMeta{Name: "api", Namespace: "loja"}},
		"pagamentos/payments": {Meta: k8sMeta{Name: "payments", Namespace: "pagamentos"}},
	}
	got := map[string]string{}
	for _, r := range scanWorkloadRefs(w, d, svc, nil) {
		got[r.svcKey+r.host] = r.port
	}
	want := map[string]string{
		"loja/catalog":                            "8080",
		"pagamentos/payments":                     "9000",
		"x.cluster-1.sa-east-1.rds.amazonaws.com": "",
	}
	for k, p := range want {
		if gp, ok := got[k]; !ok || gp != p {
			t.Errorf("faltou %s (porta %q), veio %v", k, p, got)
		}
	}
	if len(got) != len(want) {
		t.Errorf("refs a mais: %v", got)
	}
}

func TestExternalNode(t *testing.T) {
	cases := map[string]string{
		"x.cluster-1.sa-east-1.rds.amazonaws.com": "db",
		"loja.abc.0001.sae1.cache.amazonaws.com":  "cache",
		"sqs.sa-east-1.amazonaws.com":             "queue",
		"cluster0.abcd.mongodb.net":               "db",
		"api.gateway.com.br":                      "external",
	}
	for h, k := range cases {
		if n := externalNode(h); n.Kind != k {
			t.Errorf("%s: kind %s, esperado %s", h, n.Kind, k)
		}
	}
}

func TestHCLString(t *testing.T) {
	cases := map[string]string{
		`a"b`:         `"a\"b"`,
		"${HOME}":     `"$${HOME}"`,
		"%{if}":       `"%%{if}"`,
		`c:\x`:        `"c:\\x"`,
		"linha\nnova": `"linha\nnova"`,
	}
	for in, want := range cases {
		if got := hclString(in); got != want {
			t.Errorf("hclString(%q) = %s, esperado %s", in, got, want)
		}
	}
}

func TestTFName(t *testing.T) {
	if got := tfName("loja-dev", "api.v2"); got != "loja_dev_api_v2" {
		t.Errorf("tfName = %s", got)
	}
	if got := tfName("9app"); !strings.HasPrefix(got, "r_") {
		t.Errorf("tfName não pode começar com dígito: %s", got)
	}
}

func TestAlignHCL(t *testing.T) {
	in := "a {\n  name = \"x\"\n  namespace = \"y\"\n  labels = {\n    k = \"v\"\n  }\n}\n"
	out := alignHCL(in)
	if !strings.Contains(out, "  name      = \"x\"\n  namespace = \"y\"") {
		t.Errorf("alinhamento errado:\n%s", out)
	}
}
