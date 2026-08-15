package poc

import "sort"

// Registry holds every registered PoC module. Modules self-register via
// init(); the map is written once at startup and read-only afterwards, so no
// locking is required.
var Registry = map[string]Module{}

// Register adds a module. Duplicate identifiers panic so developer mistakes
// surface immediately instead of silently shadowing a module.
func Register(m Module) {
	if m == nil {
		panic("poc: cannot register a nil module")
	}
	meta := m.Meta()
	if meta.ID == "" {
		panic("poc: module registered with empty ID")
	}
	if _, dup := Registry[meta.ID]; dup {
		panic("poc: duplicate module ID " + meta.ID)
	}
	Registry[meta.ID] = m
}

// Get returns a registered module by id.
func Get(id string) (Module, bool) {
	m, ok := Registry[id]
	return m, ok
}

// List returns all registered modules sorted by ID so reports and the CLI are
// deterministic.
func List() []Module {
	ids := make([]string, 0, len(Registry))
	for id := range Registry {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]Module, 0, len(ids))
	for _, id := range ids {
		out = append(out, Registry[id])
	}
	return out
}

// ListIDs returns the sorted module IDs.
func ListIDs() []string {
	ids := make([]string, 0, len(Registry))
	for id := range Registry {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
