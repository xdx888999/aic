package main

import "testing"

func TestIsVersionFlag(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "double dash version", input: "--version", expected: true},
		{name: "single dash version", input: "-version", expected: true},
		{name: "short version", input: "-v", expected: true},
		{name: "unknown flag", input: "--help", expected: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := isVersionFlag(testCase.input)
			if actual != testCase.expected {
				t.Fatalf("期望参数 %q 的判断结果为 %v，实际为 %v", testCase.input, testCase.expected, actual)
			}
		})
	}
}
