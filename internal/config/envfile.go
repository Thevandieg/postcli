package config

import (
	"bufio"
	"os"
	"sort"
	"strings"
)

// LoadEnvMap reads key=value pairs from EnvPath.
func LoadEnvMap() (map[string]string, error) {
	out := map[string]string{}
	b, err := os.ReadFile(EnvPath())
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// SaveEnvMap writes key=value pairs to EnvPath atomically.
func SaveEnvMap(values map[string]string) error {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("# Managed by postx channels configure\n")
	for _, k := range keys {
		v := strings.TrimSpace(values[k])
		if v == "" {
			continue
		}
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(v)
		b.WriteString("\n")
	}
	tmp := EnvPath() + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, EnvPath())
}
