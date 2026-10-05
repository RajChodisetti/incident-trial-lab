package candidates

// IDs returns a fresh finite allowlist. Selection does not establish correctness.
func IDs() []string { return []string{"fixed", "ineffective", "misleading"} }

func ValidID(id string) bool {
	for _, allowed := range IDs() {
		if id == allowed {
			return true
		}
	}
	return false
}
