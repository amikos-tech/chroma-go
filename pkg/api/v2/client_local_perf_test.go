//go:build soak && !cloud

package v2

import (
	"testing"
)

func TestLocalPersistentPerformance(t *testing.T) {
	cfg, err := perfBuildRuntimeConfig()
	if err != nil {
		t.Fatalf("failed to initialize runtime config: %v", err)
	}

	thresholds := defaultPerfThresholds()
	scenarios := perfBuildScenarios(cfg)
	if len(scenarios) == 0 {
		t.Fatal("no performance scenarios were configured")
	}

	summaries := make([]perfSummary, 0, len(scenarios))
	t.Cleanup(func() {
		if len(summaries) == 0 {
			return
		}
		mdPath, mdErr := perfWriteMarkdownSummary(cfg.ReportDir, cfg.Profile, summaries)
		if mdErr != nil {
			t.Errorf("failed to write markdown summary: %v", mdErr)
			return
		}
		t.Logf("performance markdown summary: %s", mdPath)
		t.Logf("performance report directory: %s", cfg.ReportDir)
	})

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			var runErr error
			var summary perfSummary
			switch scenario.Kind {
			case perfScenarioKindSynthetic:
				summary, runErr = perfRunSyntheticScenario(cfg, thresholds, scenario)
			case perfScenarioKindChurn:
				summary, runErr = perfRunChurnScenario(cfg, thresholds, scenario)
			default:
				t.Fatalf("unsupported scenario kind: %s", scenario.Kind)
			}
			if runErr != nil {
				t.Fatalf("scenario %s failed: %v", scenario.Name, runErr)
			}

			reportPath, writeErr := perfWriteScenarioJSON(cfg.ReportDir, summary)
			if writeErr != nil {
				t.Fatalf("failed to write scenario json summary: %v", writeErr)
			}
			summary.ReportJSONPath = reportPath
			t.Logf("scenario report: %s", reportPath)

			summaries = append(summaries, summary)

			if cfg.Enforce && len(summary.Failures) > 0 {
				for _, failure := range summary.Failures {
					t.Errorf("threshold failure: %s", failure)
				}
			}
		})
	}
}
