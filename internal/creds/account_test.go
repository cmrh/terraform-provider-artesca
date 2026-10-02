package creds

import "testing"

func TestSplitImportID(t *testing.T) {
	cases := []struct {
		id, account, rest string
		wantErr           bool
	}{
		{"app/alice", "app", "alice", false},
		{"app/alice/inline-policy", "app", "alice/inline-policy", false},
		{"app/arn:aws:iam::123456789012:policy/path/name", "app", "arn:aws:iam::123456789012:policy/path/name", false},
		{"alice", "", "", true},
		{"/alice", "", "", true},
		{"app/", "", "", true},
		{"", "", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			account, rest, err := SplitImportID(tc.id)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if account != tc.account || rest != tc.rest {
				t.Errorf("got (%q, %q), want (%q, %q)", account, rest, tc.account, tc.rest)
			}
		})
	}
}
