package rpccgo

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestReleaseVerificationCoversNestedExampleModules(t *testing.T) {
	for _, dir := range []string{
		filepath.Join("examples", "minimal-greeter"),
		filepath.Join("examples", "full-greeter"),
	} {
		dir := dir
		t.Run(dir, func(t *testing.T) {
			cmd := exec.Command("go", "test", "./...")
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("go test ./... in %s failed: %v\n%s", dir, err, out)
			}
		})
	}
}

func TestReadmeInternalLinksExist(t *testing.T) {
	source, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("ReadFile(README.md) error = %v", err)
	}
	linkPattern := regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)
	for _, match := range linkPattern.FindAllStringSubmatch(string(source), -1) {
		target := match[1]
		if strings.Contains(target, "://") || strings.HasPrefix(target, "#") {
			continue
		}
		if _, err := os.Stat(target); err != nil {
			t.Fatalf("README.md link target %q is not readable: %v", target, err)
		}
	}
}
