package rpccgo

import (
	"os"
	"os/exec"
	"path/filepath"
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
