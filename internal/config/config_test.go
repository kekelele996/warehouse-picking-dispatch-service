package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("WAREHOUSE_WORKERS", "")
	t.Setenv("WAREHOUSE_RETRY_LIMIT", "")
	t.Setenv("WAREHOUSE_RESERVE_TIMEOUT_MS", "")
	t.Setenv("WAREHOUSE_POLL_INTERVAL_MS", "")
	c := Load()
	if c.Workers != 4 || c.RetryLimit != 3 {
		t.Fatalf("Workers=%d RetryLimit=%d", c.Workers, c.RetryLimit)
	}
	if len(c.ZoneRoutes) == 0 {
		t.Fatal("ZoneRoutes should be populated by defaults")
	}
}

func TestLoadCustomZoneRoutes(t *testing.T) {
	t.Setenv("WAREHOUSE_ZONE_ROUTES", "fresh:coldchain,fragile:fragile-cert")
	c := Load()
	if c.ZoneRoutes["fresh"] != "coldchain" {
		t.Fatalf("fresh route=%q", c.ZoneRoutes["fresh"])
	}
}
