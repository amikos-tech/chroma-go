package defaultef

import "sync"

// resetConfigForTesting resets the package configuration singleton for tests.
func resetConfigForTesting() {
	config = nil
	configOnce = sync.Once{}
}
