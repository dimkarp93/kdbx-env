package domain

import "sort"

type Mapping struct {
	Name string
	Env  string
}

func MappingsFromMap(m map[string]string) []Mapping {
	out := make([]Mapping, 0, len(m))
	for name, env := range m {
		out = append(out, Mapping{Name: name, Env: env})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func MappingsToMap(pairs []Mapping) map[string]string {
	m := make(map[string]string, len(pairs))
	for _, p := range pairs {
		m[p.Name] = p.Env
	}
	return m
}
