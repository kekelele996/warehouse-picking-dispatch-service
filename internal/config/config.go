package config

import (
	"os"
	"strconv"
	"strings"
)

// Config 汇聚服务运行期配置，全部来自环境变量。
type Config struct {
	Workers          int
	RetryLimit       int
	ReserveTimeoutMs int
	PollIntervalMs   int
	// ZoneRoutes 记录 SKU 品类到所需技能标签的映射，用于派单时匹配拣货员。
	ZoneRoutes map[string]string
}

// defaultZoneRoutes 返回内置的 SKU 品类 -> 技能标签默认路由表。
func defaultZoneRoutes() map[string]string {
	return map[string]string{
		"fresh":   "coldchain",
		"fragile": "fragile-cert",
		"bulk":    "forklift",
		"hazmat":  "hazmat-cert",
		"default": "general",
	}
}

// loadZoneRoutes 从环境变量加载自定义路由，未设置时回退到内置默认表。
func loadZoneRoutes() map[string]string {
	v := os.Getenv("WAREHOUSE_ZONE_ROUTES")
	if v == "" {
		return defaultZoneRoutes()
	}
	routes := map[string]string{}
	for _, pair := range strings.Split(v, ",") {
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) == 2 {
			routes[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	if len(routes) == 0 {
		return defaultZoneRoutes()
	}
	return routes
}

// Load 从环境变量加载配置；未设置时使用内置默认值。
func Load() *Config {
	return &Config{
		Workers:          getInt("WAREHOUSE_WORKERS", 4),
		RetryLimit:       getInt("WAREHOUSE_RETRY_LIMIT", 3),
		ReserveTimeoutMs: getInt("WAREHOUSE_RESERVE_TIMEOUT_MS", 2000),
		PollIntervalMs:   getInt("WAREHOUSE_POLL_INTERVAL_MS", 500),
		ZoneRoutes:       loadZoneRoutes(),
	}
}

func getInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}
