package artisan

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// deploy builds the executable and installs or restarts its systemd user
// service. An existing unit file is never read or rewritten, so local edits
// (port, extra arguments, environment) survive every deploy.
func deploy(root string, args []string, out io.Writer) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("deploy needs Linux with systemd")
	}
	op, err := parse(args, "name", "skip-frontend")
	if err != nil {
		return err
	}
	if len(op.args) != 0 {
		return fmt.Errorf("deploy accepts no positional arguments")
	}
	name := op.values["name"]
	if name == "" {
		name = "aos"
	}
	if strings.ContainsAny(name, "/\\ \t\r\n") || strings.HasSuffix(name, ".service") {
		return fmt.Errorf("--name must be a bare service name such as aos")
	}
	buildArgs := []string{}
	if op.booleans["skip-frontend"] {
		buildArgs = append(buildArgs, "--skip-frontend")
	}
	if err = build(root, buildArgs, out); err != nil {
		return err
	}
	binary, err := filepath.Abs(filepath.Join(root, defaultBinary(runtime.GOOS)))
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	unitDir := filepath.Join(home, ".config", "systemd", "user")
	unit := filepath.Join(unitDir, name+".service")
	service := name + ".service"

	_, statErr := os.Stat(unit)
	created := false
	switch {
	case statErr == nil:
		fmt.Fprintf(out, "Service %s exists, leaving %s untouched.\n", service, unit)
	case errors.Is(statErr, os.ErrNotExist):
		if err = os.MkdirAll(unitDir, 0755); err != nil {
			return err
		}
		if err = os.WriteFile(unit, []byte(unitFile(name, root, binary)), 0644); err != nil {
			return err
		}
		created = true
		fmt.Fprintf(out, "Created %s (edit it freely; deploy will not overwrite it).\n", unit)
	default:
		return statErr
	}

	// Always reload so edits made to an existing unit are picked up.
	if err = execute(root, "systemctl", []string{"--user", "daemon-reload"}, nil, out); err != nil {
		return err
	}
	if created {
		if err = execute(root, "systemctl", []string{"--user", "enable", "--now", service}, nil, out); err != nil {
			return err
		}
		fmt.Fprintf(out, "Enabled and started %s.\n", service)
		fmt.Fprintf(out, "To keep it running after logout: loginctl enable-linger $USER\n")
	} else {
		if err = execute(root, "systemctl", []string{"--user", "restart", service}, nil, out); err != nil {
			return err
		}
		fmt.Fprintf(out, "Restarted %s.\n", service)
	}
	return execute(root, "systemctl", []string{"--user", "--no-pager", "--lines=0", "status", service}, nil, out)
}

func unitFile(name, root, binary string) string {
	return fmt.Sprintf(`[Unit]
Description=%s (awesomeopensource.dev)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=%s
ExecStart=%s serve --env production
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`, name, root, binary)
}
