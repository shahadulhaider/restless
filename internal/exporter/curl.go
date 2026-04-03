package exporter

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/shahadulhaider/restless/internal/importer"
	"github.com/shahadulhaider/restless/internal/model"
)

// ToCurl converts a model.Request to a curl command string.
func ToCurl(req model.Request) string {
	return importer.GenerateCurl(req)
}

// CopyToClipboard copies text to the system clipboard by shelling out to a
// platform clipboard tool. Returns an error if the tool is unavailable.
func CopyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try xclip first, then xsel as fallback
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return fmt.Errorf("no clipboard tool found (install xclip or xsel)")
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("clipboard write: %w", err)
	}
	return nil
}
