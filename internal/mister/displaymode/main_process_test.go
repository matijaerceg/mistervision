package displaymode

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestMainProcessRecognition(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		want       bool
	}{
		{"MiSTer", "/media/fat/MiSTer", true},
		{"MiSTer_Zaparoo", "/media/fat/zaparoo/MiSTer_Zaparoo", true},
		{"MiSTer_Zaparoo", "/tmp/MiSTer_Zaparoo", false},
		{"MiSTer_Zaparoo", "/media/fat/zaparoo/MiSTer_Zaparoo-other", false},
		{"MiSTer_Zaparoo", "", false},
		{"zaparoo", "/media/fat/zaparoo/zaparoo", false},
		{"MiSTer_Physical", "/media/fat/MiSTer_Physical-CD", true},
		{"MiSTer_Physical", "/tmp/unrelated", false},
		{"MiSTer_Physical", "/media/fat/MiSTer_Physical-CD-other", false},
	} {
		if got := isMainProcess(tc.name, tc.path); got != tc.want {
			t.Fatalf("%q %q: %v", tc.name, tc.path, got)
		}
	}
}

// TestFindMainOwnership covers the process-table decisions before any signal is
// sent. The Zaparoo service is separate from its replacement MiSTer host.
func TestFindMainOwnership(t *testing.T) {
	type process struct{ name, executable, state string }
	standard := process{"MiSTer", "/media/fat/MiSTer", "S (sleeping)"}
	zaparoo := process{"MiSTer_Zaparoo", "/media/fat/zaparoo/MiSTer_Zaparoo", "S (sleeping)"}
	for _, tc := range []struct {
		name      string
		processes []process
		wantPID   int
		wantError string
	}{
		{"standard", []process{standard}, 100, ""},
		{"zaparoo", []process{zaparoo}, 100, ""},
		{"stopped host", []process{{zaparoo.name, zaparoo.executable, "T (stopped)"}}, 100, ""},
		{"physical disc", []process{{"MiSTer_Physical", "/media/fat/MiSTer_Physical-CD", "S (sleeping)"}}, 100, ""},
		{"service and host", []process{{"zaparoo", "/media/fat/zaparoo/zaparoo", "S (sleeping)"}, zaparoo}, 101, ""},
		{"ambiguous", []process{standard, zaparoo}, 0, "multiple MiSTer processes found"},
		{"duplicate host", []process{zaparoo, zaparoo}, 0, "multiple MiSTer processes found"},
		{"zombie", []process{{zaparoo.name, zaparoo.executable, "Z (zombie)"}, standard}, 101, ""},
		{"dead", []process{{zaparoo.name, zaparoo.executable, "X (dead)"}}, 0, "MiSTer process not found"},
		{"vanished status", []process{{zaparoo.name, zaparoo.executable, ""}}, 0, "MiSTer process not found"},
		{"unrelated executable", []process{{zaparoo.name, "/tmp/MiSTer_Zaparoo", zaparoo.state}}, 0, "MiSTer process not found"},
		{"missing", nil, 0, "MiSTer process not found"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for i, p := range tc.processes {
				dir := filepath.Join(root, strconv.Itoa(100+i))
				if err := os.Mkdir(dir, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "comm"), []byte(p.name+"\n"), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(p.executable, filepath.Join(dir, "exe")); err != nil {
					t.Fatal(err)
				}
				if p.state != "" {
					if err := os.WriteFile(filepath.Join(dir, "status"), []byte("State:\t"+p.state+"\n"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			got, err := findMainIn(root)
			if got != tc.wantPID || (err != nil) != (tc.wantError != "") || (err != nil && !strings.Contains(err.Error(), tc.wantError)) {
				t.Fatalf("got (%d, %v), want (%d, %q)", got, err, tc.wantPID, tc.wantError)
			}
		})
	}
}
