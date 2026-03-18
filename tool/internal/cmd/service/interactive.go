package service

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ssgohq/ssgo/tool/internal/manifest"
	"github.com/ssgohq/ssgo/tool/internal/scanner"
)

// Lipgloss styles for output.
var (
	styleBold    = lipgloss.NewStyle().Bold(true)
	styleWarn    = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	styleOK      = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	styleConflict = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	styleHeader  = lipgloss.NewStyle().Bold(true).Underline(true)
)

// InteractiveSetup prompts the user for key manifest decisions when running
// in an interactive terminal. It is a no-op when NonInteractive is true.
//
// Prompts covered:
//   - go mod init if go.mod is absent
//   - binary mode (split vs single)
//   - transport selection (api / rpc / both)
func InteractiveSetup(dir string, result *scanner.ScanResult, nonInteractive bool) (*manifest.ServiceManifest, error) {
	m := &manifest.ServiceManifest{}

	if nonInteractive {
		return m, nil
	}

	r := bufio.NewReader(os.Stdin)

	// Check for go.mod.
	if err := checkGoMod(dir, r); err != nil {
		return nil, err
	}

	// Binary mode selection.
	mode, err := promptBinaryMode(r)
	if err != nil {
		return nil, err
	}
	m.Binary.Mode = mode

	// Transport selection.
	api, rpc, err := promptTransports(r, result)
	if err != nil {
		return nil, err
	}
	m.Transports.API.Enabled = api
	m.Transports.RPC.Enabled = rpc

	PrintPlanSummary(result, m)
	return m, nil
}

// checkGoMod asks whether to run `go mod init` when go.mod is absent.
func checkGoMod(dir string, r *bufio.Reader) error {
	goModPath := dir + "/go.mod"
	if _, err := os.Stat(goModPath); err == nil {
		return nil // already exists
	}

	fmt.Fprintf(os.Stdout, "%s go.mod not found in %s\n", styleWarn.Render("!"), dir)
	answer, err := prompt(r, "Run `go mod init`? [Y/n]", "y")
	if err != nil {
		return err
	}

	if strings.ToLower(answer) == "y" || answer == "" {
		fmt.Fprintf(os.Stdout, "%s Skipping go mod init — please run it manually before gen.\n", styleOK.Render("i"))
		fmt.Fprintf(os.Stdout, "  Example: go mod init <module-path>\n")
	}
	return nil
}

// promptBinaryMode asks for split vs single binary layout.
func promptBinaryMode(r *bufio.Reader) (manifest.BinaryMode, error) {
	fmt.Fprintln(os.Stdout, styleBold.Render("\nBinary mode:"))
	fmt.Fprintln(os.Stdout, "  1) split  — separate cmd/api and cmd/rpc entry points (default)")
	fmt.Fprintln(os.Stdout, "  2) single — one binary that hosts both transports")

	answer, err := prompt(r, "Choose [1/2]", "1")
	if err != nil {
		return "", err
	}

	if answer == "2" {
		return manifest.BinaryModeSingle, nil
	}
	return manifest.BinaryModeSplit, nil
}

// promptTransports asks which transports to enable.
func promptTransports(r *bufio.Reader, result *scanner.ScanResult) (api, rpc bool, err error) {
	hasAPI := len(result.APIContracts()) > 0
	hasRPC := len(result.RPCContracts()) > 0

	fmt.Fprintln(os.Stdout, styleBold.Render("\nTransport selection:"))
	fmt.Fprintln(os.Stdout, "  1) api  — HTTP/REST (Hertz)")
	fmt.Fprintln(os.Stdout, "  2) rpc  — gRPC/Kitex")
	fmt.Fprintln(os.Stdout, "  3) both — enable both transports")

	// Default based on discovered contracts.
	defaultChoice := "3"
	switch {
	case hasAPI && !hasRPC:
		defaultChoice = "1"
	case hasRPC && !hasAPI:
		defaultChoice = "2"
	}

	answer, err := prompt(r, fmt.Sprintf("Choose [1/2/3] (default: %s)", defaultChoice), defaultChoice)
	if err != nil {
		return false, false, err
	}

	switch answer {
	case "1":
		return true, false, nil
	case "2":
		return false, true, nil
	default:
		return true, true, nil
	}
}

// prompt prints a question and returns the user's trimmed response.
// If the response is empty, defaultVal is returned.
func prompt(r *bufio.Reader, question, defaultVal string) (string, error) {
	fmt.Fprintf(os.Stdout, "  %s: ", question)
	line, err := r.ReadString('\n')
	if err != nil {
		return defaultVal, nil //nolint:nilerr — EOF treated as default
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal, nil
	}
	return line, nil
}

// PrintPlanSummary prints a styled summary of the resolved manifest + plan.
func PrintPlanSummary(result *scanner.ScanResult, m *manifest.ServiceManifest) {
	fmt.Fprintln(os.Stdout, "\n"+styleHeader.Render("Generation Plan"))
	fmt.Fprintf(os.Stdout, "  Directory : %s\n", result.Root)
	fmt.Fprintf(os.Stdout, "  State     : %s\n", string(result.State))
	fmt.Fprintf(os.Stdout, "  Binary    : %s\n", styleBold.Render(string(m.EffectiveBinaryMode())))
	fmt.Fprintf(os.Stdout, "  Transports: %s\n", styleBold.Render(strings.Join(m.ActiveTransports(), ", ")))
}

// PrintConflicts prints a styled list of detected conflicts.
func PrintConflicts(conflicts []string) {
	if len(conflicts) == 0 {
		return
	}
	fmt.Fprintln(os.Stdout, "\n"+styleConflict.Render("Conflicts detected:"))
	for _, cf := range conflicts {
		fmt.Fprintf(os.Stdout, "  %s %s\n", styleConflict.Render("x"), cf)
	}
}
