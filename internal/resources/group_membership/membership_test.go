package groupmembership

import "testing"

func TestInGroup(t *testing.T) {
	groups := []string{"admins", "devs"}
	for name, want := range map[string]bool{"admins": true, "Admins": true, "DEVS": true, "ops": false, "": false} {
		if got := inGroup(groups, name); got != want {
			t.Errorf("inGroup(%q) = %v, want %v", name, got, want)
		}
	}
}
