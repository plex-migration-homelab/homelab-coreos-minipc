package steps

import "github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"

// ensureCanonicalMarker checks for the canonical completion marker and migrates any legacy markers
// to the canonical name to maintain backward compatibility.
// This function is designed to be race-safe when called concurrently by multiple processes.
func ensureCanonicalMarker(cfg *config.Config, canonical string, legacy ...string) (bool, error) {
	// First check if canonical marker exists (fast path for completed steps)
	if cfg.IsComplete(canonical) {
		return true, nil
	}

	// Check for legacy markers and migrate them
	for _, legacyName := range legacy {
		if legacyName == "" || legacyName == canonical {
			continue
		}

		if !cfg.IsComplete(legacyName) {
			continue
		}

		// Atomically create canonical marker (race-safe)
		// If another process already created it between our check and now, that's fine
		wasCreated, err := cfg.MarkCompleteIfNotExists(canonical)
		if err != nil {
			return false, err
		}

		// Best-effort cleanup of the legacy marker. Ignore errors since it's non-critical.
		// Only remove if we were the ones who created the canonical marker
		if wasCreated {
			_ = cfg.ClearMarker(legacyName)
		}
		return true, nil
	}

	return false, nil
}
