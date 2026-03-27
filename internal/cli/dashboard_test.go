package cli

import (
	"testing"
)

func TestDashboardOutputJSON_EmptySubscriptions(t *testing.T) {
	out := DashboardOutputJSON{
		Subscriptions: []DashboardSubJSON{},
		TotalUnread:   0,
		Version:       "1.0.0",
	}
	if len(out.Subscriptions) != 0 {
		t.Error("empty subscriptions should have length 0")
	}
}

func TestDashboardSubJSON_Fields(t *testing.T) {
	sub := DashboardSubJSON{
		Topic:  "silicon-chips",
		Unread: 5,
		Total:  12,
	}
	if sub.Topic != "silicon-chips" {
		t.Errorf("topic = %q, want %q", sub.Topic, "silicon-chips")
	}
	if sub.Unread != 5 {
		t.Errorf("unread = %d, want %d", sub.Unread, 5)
	}
}

func TestShowGettingStarted(t *testing.T) {
	// Should not panic
	showGettingStarted()
}

func TestRootCmdHasRunE(t *testing.T) {
	if rootCmd.RunE == nil {
		t.Error("rootCmd.RunE should be set for dashboard")
	}
}
