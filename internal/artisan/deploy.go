package artisan

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
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
	if err = checkUserManager(root); err != nil {
		return err
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
	if err = systemctlUser(root, out, "daemon-reload"); err != nil {
		return err
	}
	if created {
		if err = systemctlUser(root, out, "enable", "--now", service); err != nil {
			return err
		}
		fmt.Fprintf(out, "Enabled and started %s.\n", service)
		fmt.Fprintf(out, "To keep it running after logout: sudo loginctl enable-linger $USER\n")
	} else {
		// The unit may have been created by a deploy that failed before this
		// point, so make sure it also starts at boot (linger) on every run.
		if err = systemctlUser(root, out, "enable", service); err != nil {
			return err
		}
		if err = systemctlUser(root, out, "restart", service); err != nil {
			return err
		}
		fmt.Fprintf(out, "Enabled and restarted %s.\n", service)
	}
	return systemctlUser(root, out, "--no-pager", "--lines=0", "status", service)
}

func systemctlUser(root string, out io.Writer, args ...string) error {
	return execute(root, "systemctl", append([]string{"--user"}, args...), userManagerEnv(), out)
}

// userManagerEnv supplies XDG_RUNTIME_DIR when artisan was started outside a
// login session (su, a service manager, a remote agent). sd-bus then reports
// the unhelpful "Failed to connect to bus: No medium found" instead of looking
// for the user manager socket under /run/user/<uid>.
func userManagerEnv() map[string]string {
	if os.Getenv("XDG_RUNTIME_DIR") != "" {
		return nil
	}
	return map[string]string{"XDG_RUNTIME_DIR": fmt.Sprintf("/run/user/%d", os.Getuid())}
}

// checkUserManager fails before the build when systemctl --user has no user
// manager to talk to, and explains the one-time fix.
func checkUserManager(root string) error {
	child := exec.Command("systemctl", "--user", "--no-pager", "show", "--property=Version", "--value")
	child.Dir = root
	child.Env = mergeEnv(userManagerEnv())
	child.Stdout = io.Discard
	var stderr bytes.Buffer
	child.Stderr = &stderr
	err := child.Run()
	if err == nil {
		return nil
	}
	if errors.Is(err, exec.ErrNotFound) {
		return fmt.Errorf("deploy needs systemctl from systemd")
	}
	uid := os.Getuid()
	account := strconv.Itoa(uid)
	if current, lookupErr := user.Current(); lookupErr == nil && current.Username != "" {
		account = current.Username
	}
	detail := strings.TrimSpace(stderr.String())
	if detail != "" {
		detail = " (" + detail + ")"
	}
	return fmt.Errorf(`systemctl --user cannot reach the systemd user manager for %s%s
Start it once for this account, then deploy again:
    sudo loginctl enable-linger %s
If linger is already enabled, start the manager directly:
    sudo systemctl start user@%d.service`, account, detail, account, uid)
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
