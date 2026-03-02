package defaultef

import "sync"

// resetConfigForTesting resets the package configuration singleton for tests.
// It mutates package-level globals and must not be used from t.Parallel tests.
func resetConfigForTesting() {
	config = nil
	configOnce = sync.Once{}
}
