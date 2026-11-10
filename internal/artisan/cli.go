// Package artisan implements the project's native, cross-platform command line.
package artisan

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/andreitelteu/inertia-go-solid-template/app/server"
)

const help = `inertia-go-solid-template artisan

  help, list                 Show commands
  install                    npm ci and go mod download
  dev [--host HOST --port N --vite-port N]
                             Vite HMR and Go rebuild watcher
  build [--os OS --arch ARCH --output FILE --skip-frontend]
                             Build one executable with embedded frontend
  start [--host HOST --port N --env production|demo]
                             Run the built executable (default: production)
  serve [--host HOST --port N --env ENV]
                             Serve this executable directly
  test                       Run npm test
  route:list, routes         Print the named route registry (no build required)
  key:generate [--force]      Write APP_KEY to .env without printing it
  make:controller Name [--action index_page,store_form]
                             One snake_case file per action
  make:component Name        Create a reusable Solid component
  make:page Name             Create a Solid page (nested names supported)
  about                      Show runtime and project information

Scaffolds refuse overwrites. Register generated controllers/pages explicitly.
Build needs Go, Node and npm; the production executable needs none of them.
`

type options struct {
	values   map[string]string
	booleans map[string]bool
	args     []string
}

func parse(args []string, valueFlags, boolFlags string) (options, error) {
	o := options{values: map[string]string{}, booleans: map[string]bool{}}
	valueSet := map[string]bool{}
	boolSet := map[string]bool{}
	for _, key := range strings.Fields(valueFlags) {
		valueSet[key] = true
	}
	for _, key := range strings.Fields(boolFlags) {
		boolSet[key] = true
	}
	for n := 0; n < len(args); n++ {
		arg := args[n]
		if !strings.HasPrefix(arg, "--") {
			o.args = append(o.args, arg)
			continue
		}
		key, val, assigned := strings.Cut(strings.TrimPrefix(arg, "--"), "=")
		if boolSet[key] {
			if assigned {
				return o, fmt.Errorf("--%s does not take a value", key)
			}
			o.booleans[key] = true
			continue
		}
		if !valueSet[key] {
			return o, fmt.Errorf("unknown option --%s", key)
		}
		if !assigned {
			n++
			if n >= len(args) || strings.HasPrefix(args[n], "--") {
				return o, fmt.Errorf("--%s needs a value", key)
			}
			val = args[n]
		}
		if val == "" {
			return o, fmt.Errorf("--%s needs a value", key)
		}
		if _, exists := o.values[key]; exists {
			return o, fmt.Errorf("--%s specified twice", key)
		}
		o.values[key] = val
	}
	return o, nil
}

func Run(args []string, out io.Writer) error {
	if len(args) == 0 {
		_, err := io.WriteString(out, help)
		return err
	}
	if args[0] == "help" || args[0] == "list" || args[0] == "--help" || args[0] == "-h" {
		_, err := io.WriteString(out, help)
		return err
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	if err = LoadEnv(filepath.Join(root, ".env")); err != nil {
		return err
	}
	command, rest := args[0], args[1:]
	switch command {
	case "make:controller", "make:component", "make:page":
		op, err := parse(rest, "action", "")
		if err != nil {
			return err
		}
		if len(op.args) != 1 {
			return fmt.Errorf("usage: artisan %s Name", command)
		}
		if command != "make:controller" && op.values["action"] != "" {
			return fmt.Errorf("--action applies only to make:controller")
		}
		return scaffold(root, command, op.args[0], op.values["action"], out)
	case "key:generate":
		op, err := parse(rest, "", "force")
		if err != nil {
			return err
		}
		if len(op.args) != 0 {
			return fmt.Errorf("key:generate accepts no positional arguments")
		}
		return generateKey(root, op.booleans["force"], out)
	case "build":
		return build(root, rest, out)
	case "dev":
		op, err := parse(rest, "host port vite-port", "")
		if err != nil {
			return err
		}
		if len(op.args) != 0 {
			return fmt.Errorf("dev accepts no positional arguments")
		}
		if err = configureAddress(op); err != nil {
			return err
		}
		if port := op.values["vite-port"]; port != "" {
			if err = validatePort(port); err != nil {
				return err
			}
			os.Setenv("VITE_PORT", port)
		}
		return execute(root, "node", []string{"scripts/dev.mjs"}, nil, out)
	case "start", "serve":
		op, err := parse(rest, "host port env output", "")
		if err != nil {
			return err
		}
		if len(op.args) != 0 {
			return fmt.Errorf("%s accepts no positional arguments", command)
		}
		if err = configureAddress(op); err != nil {
			return err
		}
		env := op.values["env"]
		if env == "" && command == "start" {
			env = "production"
		}
		if env != "" {
			switch env {
			case "production", "demo", "development", "test":
				os.Setenv("APP_ENV", env)
			default:
				return fmt.Errorf("invalid --env %q", env)
			}
		}
		if command == "serve" {
			return server.RunFromEnvironment()
		}
		binary := op.values["output"]
		if binary == "" {
			binary = defaultBinary(runtime.GOOS)
		}
		binary, err = filepath.Abs(binary)
		if err != nil {
			return err
		}
		if _, err = os.Stat(binary); err != nil {
			return fmt.Errorf("build first with ./artisan build: %w", err)
		}
		return execute(root, binary, nil, nil, out)
	case "install", "test", "routes", "route:list", "about":
		if len(rest) != 0 {
			return fmt.Errorf("%s accepts no arguments", command)
		}
		switch command {
		case "install":
			if err = execute(root, "npm", []string{"ci"}, nil, out); err != nil {
				return err
			}
			return execute(root, "go", []string{"mod", "download"}, nil, out)
		case "test":
			return execute(root, "npm", []string{"test"}, nil, out)
		case "routes", "route:list":
			return printRoutes(out)
		default:
			_, err = fmt.Fprintf(out, "inertia-go-solid-template\nGo: %s\nPlatform: %s/%s\nRoot: %s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH, root)
			return err
		}
	default:
		return fmt.Errorf("unknown command %q; run ./artisan help", command)
	}
}

func defaultBinary(goos string) string {
	name := "inertia-go-solid-template"
	if goos == "windows" {
		name += ".exe"
	}
	return filepath.Join("build", name)
}

func build(root string, args []string, out io.Writer) error {
	op, err := parse(args, "os arch output", "skip-frontend")
	if err != nil {
		return err
	}
	if len(op.args) != 0 {
		return fmt.Errorf("build accepts no positional arguments")
	}
	goos, arch := op.values["os"], op.values["arch"]
	if goos == "" {
		goos = runtime.GOOS
	}
	if arch == "" {
		arch = runtime.GOARCH
	}
	if !op.booleans["skip-frontend"] {
		if err = execute(root, "npm", []string{"run", "build"}, nil, out); err != nil {
			return err
		}
	}
	if _, err = os.Stat(filepath.Join(root, "public", "build", ".vite", "manifest.json")); err != nil {
		return fmt.Errorf("frontend build is missing; run npm run build before --skip-frontend: %w", err)
	}
	output := op.values["output"]
	if output == "" {
		output = defaultBinary(goos)
	}
	if err = os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	err = execute(root, "go", []string{"build", "-buildvcs=false", "-trimpath", "-tags", "production", "-o", output, "."}, map[string]string{"GOOS": goos, "GOARCH": arch, "CGO_ENABLED": "0"}, out)
	if err == nil {
		fmt.Fprintf(out, "Built %s (%s/%s), frontend embedded.\n", output, goos, arch)
	}
	return err
}

func configureAddress(o options) error {
	if o.values["host"] == "" && o.values["port"] == "" {
		return nil
	}
	host, port := "127.0.0.1", "8080"
	if configured := os.Getenv("APP_ADDR"); configured != "" {
		h, p, err := net.SplitHostPort(configured)
		if err != nil {
			return fmt.Errorf("invalid APP_ADDR: %w", err)
		}
		host, port = h, p
	}
	if value := o.values["host"]; value != "" {
		host = value
	}
	if value := o.values["port"]; value != "" {
		port = value
	}
	if strings.ContainsAny(host, "/\\\r\n") {
		return fmt.Errorf("invalid --host")
	}
	if err := validatePort(port); err != nil {
		return err
	}
	return os.Setenv("APP_ADDR", net.JoinHostPort(host, port))
}
func validatePort(value string) error {
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}

func execute(root, command string, args []string, env map[string]string, out io.Writer) error {
	if runtime.GOOS == "windows" && command == "npm" {
		args = append([]string{"/d", "/c", "npm"}, args...)
		command = os.Getenv("ComSpec")
		if command == "" {
			command = "cmd.exe"
		}
	}
	child := exec.Command(command, args...)
	child.Dir = root
	child.Stdin = os.Stdin
	child.Stdout = out
	child.Stderr = os.Stderr
	child.Env = os.Environ()
	for key, value := range env {
		prefix := key + "="
		filtered := child.Env[:0]
		for _, entry := range child.Env {
			if !strings.HasPrefix(entry, prefix) {
				filtered = append(filtered, entry)
			}
		}
		child.Env = append(filtered, prefix+value)
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	if err := child.Start(); err != nil {
		return fmt.Errorf("%s failed: %w", command, err)
	}
	done := make(chan error, 1)
	go func() { done <- child.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("%s failed: %w", command, err)
		}
		return nil
	case sig := <-signals:
		// Node's development supervisor closes Vite and its Go child on SIGTERM.
		// Windows does not implement Process.Signal; Kill is the supported fallback.
		if err := child.Process.Signal(sig); err != nil {
			_ = child.Process.Kill()
		}
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = child.Process.Kill()
			<-done
		}
		return nil
	}
}
