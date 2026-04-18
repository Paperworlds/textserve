package deps

import (
	"fmt"
	"os/exec"

	"github.com/paperworlds/textserve/internal/registry"
)

// Check runs each dep's cmd via bash and returns an error if any fail.
// On failure the dep's hint is printed to stderr before returning.
func Check(deps []registry.Dep) error {
	for _, d := range deps {
		if err := exec.Command("bash", "-c", d.Cmd).Run(); err != nil {
			fmt.Println("Hint:", d.Hint)
			return fmt.Errorf("dependency check failed: %q", d.Cmd)
		}
	}
	return nil
}
