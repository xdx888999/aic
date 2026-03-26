package detector

import (
	"testing"

	"github.com/xdx888999/aic/internal/registry"
)

func TestCompareVersions(t *testing.T) {
	testCases := []struct {
		name     string
		current  string
		latest   string
		expected int
	}{
		{
			name:     "current older",
			current:  "2.4.27",
			latest:   "2.6.18",
			expected: -1,
		},
		{
			name:     "current newer",
			current:  "3.5.37",
			latest:   "2.3.12786",
			expected: 1,
		},
		{
			name:     "same version",
			current:  "1.9552.21",
			latest:   "1.9552.21",
			expected: 0,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := CompareVersions(testCase.current, testCase.latest)
			if actual != testCase.expected {
				t.Fatalf("期望比较结果为 %d，实际为 %d", testCase.expected, actual)
			}
		})
	}
}

func TestReadNPMGlobalVersionParsesInstalledPackageVersion(t *testing.T) {
	output := "├── @google/gemini-cli@0.33.1"
	actual := parseNPMGlobalVersion(output, "@google/gemini-cli", "semver")
	if actual != "0.33.1" {
		t.Fatalf("期望 npm 全局版本解析为 0.33.1，实际为 %q", actual)
	}
}

func TestParseNPMDistTagJSONValue(t *testing.T) {
	raw := "\"0.33.1\""
	actual := registry.ParseVersion(normalizeJSONCommandValue(raw), "semver")
	if actual != "0.33.1" {
		t.Fatalf("期望 dist-tag latest 解析为 0.33.1，实际为 %q", actual)
	}
}
