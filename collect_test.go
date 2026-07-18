package main

import (
	"encoding/json"
	"testing"
)

func TestMatchSelector(t *testing.T) {
	cases := []struct {
		name string
		sel  map[string]string
		lbl  map[string]string
		want bool
	}{
		{"empty selector never matches", map[string]string{}, map[string]string{"app": "x"}, false},
		{"exact match", map[string]string{"app": "x"}, map[string]string{"app": "x"}, true},
		{"extra pod labels ok", map[string]string{"app": "x"}, map[string]string{"app": "x", "tier": "web"}, true},
		{"missing label fails", map[string]string{"app": "x", "tier": "web"}, map[string]string{"app": "x"}, false},
		{"different value fails", map[string]string{"app": "x"}, map[string]string{"app": "y"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := matchSelector(c.sel, c.lbl); got != c.want {
				t.Errorf("matchSelector(%v, %v) = %v, want %v", c.sel, c.lbl, got, c.want)
			}
		})
	}
}

func mustPod(t *testing.T, js string) pod {
	t.Helper()
	var p pod
	if err := json.Unmarshal([]byte(js), &p); err != nil {
		t.Fatalf("unmarshal pod: %v", err)
	}
	return p
}

func TestAnyContainerHasPort(t *testing.T) {
	withPorts := mustPod(t, `{"spec":{"containers":[{"name":"app","ports":[{"containerPort":8080},{"name":"metrics","containerPort":9090}]}]}}`)
	noPorts := mustPod(t, `{"spec":{"containers":[{"name":"app"}]}}`)

	t.Run("numeric port found", func(t *testing.T) {
		found, declared := anyContainerHasPort([]pod{withPorts}, nil, 8080)
		if !found || !declared {
			t.Errorf("found=%v declared=%v, want true,true", found, declared)
		}
	})

	t.Run("numeric port not found but declared", func(t *testing.T) {
		found, declared := anyContainerHasPort([]pod{withPorts}, nil, 9999)
		if found || !declared {
			t.Errorf("found=%v declared=%v, want false,true", found, declared)
		}
	})

	t.Run("named targetPort matches by name", func(t *testing.T) {
		tp := &intOrString{IsStr: true, StrVal: "metrics"}
		found, declared := anyContainerHasPort([]pod{withPorts}, tp, 0)
		if !found || !declared {
			t.Errorf("found=%v declared=%v, want true,true", found, declared)
		}
	})

	t.Run("no ports declared at all: not a mismatch, just unknown", func(t *testing.T) {
		found, declared := anyContainerHasPort([]pod{noPorts}, nil, 8080)
		if found || declared {
			t.Errorf("found=%v declared=%v, want false,false (avoid false-positive mismatch)", found, declared)
		}
	})
}

func TestIntOrStringUnmarshal(t *testing.T) {
	var numeric intOrString
	if err := json.Unmarshal([]byte("8080"), &numeric); err != nil {
		t.Fatalf("unmarshal numeric: %v", err)
	}
	if numeric.IsStr || numeric.IntVal != 8080 {
		t.Errorf("numeric = %+v, want IntVal=8080 IsStr=false", numeric)
	}

	var named intOrString
	if err := json.Unmarshal([]byte(`"http"`), &named); err != nil {
		t.Fatalf("unmarshal string: %v", err)
	}
	if !named.IsStr || named.StrVal != "http" {
		t.Errorf("named = %+v, want StrVal=http IsStr=true", named)
	}
}
