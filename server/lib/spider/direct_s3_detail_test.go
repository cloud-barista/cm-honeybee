package spider

import "testing"

func TestPolicyAllowsAnyone(t *testing.T) {
	cases := []struct {
		name   string
		policy string
		want   bool
	}{
		{"no policy", "", false},
		{"principal star", `{"Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:GetObject"}]}`, true},
		{"aws star", `{"Statement":[{"Effect":"Allow","Principal":{"AWS":"*"}}]}`, true},
		{"aws star list", `{"Statement":[{"Effect":"Allow","Principal":{"AWS":["arn:aws:iam::1:root","*"]}}]}`, true},
		{"single statement object", `{"Statement":{"Effect":"Allow","Principal":"*"}}`, true},
		{"deny star", `{"Statement":[{"Effect":"Deny","Principal":"*"}]}`, false},
		{"named account", `{"Statement":[{"Effect":"Allow","Principal":{"AWS":"arn:aws:iam::1:root"}}]}`, false},
		{"broken json", `{"Statement":`, false},
	}
	for _, c := range cases {
		if got := policyAllowsAnyone(c.policy); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestSplitCSV(t *testing.T) {
	s := "GET, PUT,,POST "
	got := splitCSV(&s)
	if len(got) != 3 || got[0] != "GET" || got[1] != "PUT" || got[2] != "POST" {
		t.Errorf("got %q", got)
	}
	if splitCSV(nil) != nil {
		t.Error("nil must give nil")
	}
}
