package rellog

import "testing"

func TestIsCanonicalScriptPath(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "nested path", value: "scripts/preserve.sh", want: true},
		{name: "root-level path", value: "preserve.sh", want: true},
		{name: "space in segment", value: "scripts/preserve release.sh", want: true},
		{name: "empty", value: "", want: false},
		{name: "whitespace only", value: " \t", want: false},
		{name: "POSIX absolute", value: "/scripts/preserve.sh", want: false},
		{name: "Windows absolute with uppercase drive", value: "C:/scripts/preserve.sh", want: false},
		{name: "Windows absolute with lowercase drive", value: "c:/scripts/preserve.sh", want: false},
		{name: "backslash separator", value: `scripts\preserve.sh`, want: false},
		{name: "leading current-directory segment", value: "./scripts/preserve.sh", want: false},
		{name: "leading parent-directory segment", value: "../scripts/preserve.sh", want: false},
		{name: "nested current-directory segment", value: "scripts/./preserve.sh", want: false},
		{name: "nested parent-directory segment", value: "scripts/../preserve.sh", want: false},
		{name: "empty segment", value: "scripts//preserve.sh", want: false},
		{name: "trailing slash", value: "scripts/", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isCanonicalScriptPath(tt.value); got != tt.want {
				t.Fatalf("isCanonicalScriptPath(%q) = %t, want %t", tt.value, got, tt.want)
			}
		})
	}
}
