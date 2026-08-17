package model

import "testing"

func TestCanTransition(t *testing.T) {
	cases := []struct {
		from, to Status
		want     bool
	}{
		{StatusPending, StatusAssigned, true},
		{StatusAssigned, StatusPicking, true},
		{StatusPicking, StatusCompleted, true},
		{StatusPicking, StatusFailed, true},
		{StatusFailed, StatusRetrying, true},
		{StatusRetrying, StatusPicking, true},
		{StatusCompleted, StatusRetrying, false},
	}
	for _, c := range cases {
		if got := CanTransition(c.from, c.to); got != c.want {
			t.Fatalf("CanTransition(%s,%s)=%v want %v", c.from, c.to, got, c.want)
		}
	}
}

func TestActiveStatuses(t *testing.T) {
	if !ActiveStatuses[StatusRetrying] {
		t.Fatal("ActiveStatuses must include retrying")
	}
	if ActiveStatuses[StatusCompleted] {
		t.Fatal("ActiveStatuses must not include completed")
	}
}

func TestHasSkill(t *testing.T) {
	p := &Picker{Skills: []string{"coldchain", "forklift"}}
	if !p.HasSkill("coldchain") {
		t.Fatal("expected HasSkill(coldchain) true")
	}
	if p.HasSkill("hazmat-cert") {
		t.Fatal("expected HasSkill(hazmat-cert) false")
	}
}
