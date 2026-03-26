package version

import "testing"

func TestStringUsesInjectedBuildMetadata(t *testing.T) {
	originalVersion := Version
	originalCommit := Commit
	originalBuildTime := BuildTime
	t.Cleanup(func() {
		Version = originalVersion
		Commit = originalCommit
		BuildTime = originalBuildTime
	})

	Version = "v0.1.0"
	Commit = "abc1234"
	BuildTime = "2026-03-27T00:00:00Z"

	actual := String()
	expected := "aic v0.1.0 (commit: abc1234, built: 2026-03-27T00:00:00Z)"
	if actual != expected {
		t.Fatalf("期望版本输出为 %q，实际为 %q", expected, actual)
	}
}
