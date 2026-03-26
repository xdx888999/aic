package version

import "testing"

func TestStringUsesInjectedBuildMetadata(t *testing.T) {
	originalVersion := Version
	t.Cleanup(func() {
		Version = originalVersion
	})

	Version = "v0.1.0"

	actual := String()
	expected := "aic v0.1.0"
	if actual != expected {
		t.Fatalf("期望版本输出为 %q，实际为 %q", expected, actual)
	}
}
