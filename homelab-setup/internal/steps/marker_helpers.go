package steps

import "github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"

// ensureCanonicalMarker checks for the canonical completion marker and migrates any legacy markers
// to the canonical name to maintain backward compatibility.
// This function is designed to be race-safe when called concurrently by multiple processes.
func ensureCanonicalMarker(markers *config.Markers, canonical string, legacy ...string) (bool, error) {
	// First check if canonical marker exists (fast path for completed steps)
	exists, err := markers.Exists(canonical)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}

	// Check for legacy markers and migrate them
	for _, legacyName := range legacy {
		if legacyName == "" || legacyName == canonical {
			continue
		}

		legacyExists, err := markers.Exists(legacyName)
		if err != nil {
			return false, err
		}
		if !legacyExists {
			continue
		}

		// Atomically create canonical marker (race-safe)
		// If another process already created it between our check and now, that's fine
		wasCreated, err := markers.CreateIfNotExists(canonical)
		if err != nil {
			return false, err
		}

		// Best-effort cleanup of the legacy marker. Ignore errors since it's non-critical.
		// Only remove if we were the ones who created the canonical marker
		if wasCreated {
			_ = markers.Remove(legacyName)
		}
		return true, nil
	}

	return false, nil
}
