package artisan

import (
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var identifier = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
var acronymBoundary = regexp.MustCompile(`([A-Z]+)([A-Z][a-z])`)
var wordBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)

func snake(value string) string {
	return strings.ToLower(wordBoundary.ReplaceAllString(acronymBoundary.ReplaceAllString(value, "${1}_${2}"), "${1}_${2}"))
}
func pascal(value string) string {
	parts := strings.Split(snake(value), "_")
	for n, part := range parts {
		if part != "" {
			parts[n] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

func modulePath(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			return strings.Trim(fields[1], "\""), nil
		}
	}
	return "", fmt.Errorf("go.mod has no module declaration")
}

func scaffold(root, kind, name, actions string, out io.Writer) error {
	parts := strings.Split(name, "/")
	for _, part := range parts {
		if !identifier.MatchString(part) {
			return fmt.Errorf("names must contain identifier segments only, without dots or traversal")
		}
	}
	if kind == "make:controller" && len(parts) != 1 {
		return fmt.Errorf("controller names must be a single identifier")
	}
	files := map[string][]byte{}
	if kind == "make:controller" {
		module, err := modulePath(root)
		if err != nil {
			return err
		}
		folder := snake(name)
		if !strings.HasSuffix(folder, "_controller") {
			folder += "_controller"
		}
		if actions == "" {
			actions = "index_page"
		}
		functions := map[string]bool{}
		for _, action := range strings.Split(actions, ",") {
			if !identifier.MatchString(action) {
				return fmt.Errorf("invalid action %q", action)
			}
			filename := snake(action)
			functionName := filename
			for _, suffix := range []string{"_page", "_form", "_data"} {
				functionName = strings.TrimSuffix(functionName, suffix)
			}
			functionName = pascal(functionName)
			if functionName == "" || functions[functionName] {
				return fmt.Errorf("duplicate or invalid action function %q", functionName)
			}
			functions[functionName] = true
			pageName := pascal(strings.TrimSuffix(folder, "_controller")) + "/" + functionName
			source := fmt.Sprintf("package %s\n\nimport (\n%q\n%q\ninertia %q\n)\n\nfunc %s(app *application.Application) fiber.Handler {\nreturn func(c fiber.Ctx) error {\nreturn app.RenderPage(c, %q, inertia.Props{\"title\": %q})\n}\n}\n", folder, module+"/app/application", "github.com/gofiber/fiber/v3", "github.com/inertia-go/inertia-go", functionName, pageName, pageName)
			formatted, err := format.Source([]byte(source))
			if err != nil {
				return err
			}
			files[filepath.Join("app", "controllers", folder, filename+".go")] = formatted
		}
	} else {
		base := "components"
		suffix := ""
		if kind == "make:page" {
			base = "pages"
			suffix = "_page"
		}
		for n, part := range parts {
			parts[n] = snake(part)
		}
		last := parts[len(parts)-1]
		if suffix != "" && !strings.HasSuffix(last, suffix) {
			last += suffix
		}
		parts[len(parts)-1] = last + ".tsx"
		component := pascal(last)
		var source string
		if kind == "make:component" {
			source = fmt.Sprintf("import type { ParentProps } from 'solid-js'\n\nexport default function %s(props: ParentProps) {\n  return <section>{props.children}</section>\n}\n", component)
		} else {
			source = fmt.Sprintf("import { Title } from '@solidjs/meta'\n\nexport default function %s() {\n  return <section><Title>%s</Title><h1>%s</h1></section>\n}\n", component, name, name)
		}
		files[filepath.Join(append([]string{"resources", "js", base}, parts...)...)] = []byte(source)
	}
	for path := range files {
		target := filepath.Join(root, path)
		if _, err := os.Lstat(target); err == nil {
			return fmt.Errorf("refusing to overwrite %s", path)
		} else if !os.IsNotExist(err) {
			return err
		}
		for parent := filepath.Dir(target); parent != root && parent != filepath.Dir(parent); parent = filepath.Dir(parent) {
			if info, err := os.Lstat(parent); err == nil && info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("refusing symlink scaffold directory %s", parent)
			}
		}
	}
	for path, data := range files {
		target := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return err
		}
		_, writeErr := file.Write(data)
		closeErr := file.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
		fmt.Fprintln(out, "Created", path)
	}
	if kind == "make:controller" {
		fmt.Fprintln(out, "Next: import the controller package and register its actions in routes/web.go; add matching Solid page resolver entries.")
	}
	if kind == "make:page" {
		fmt.Fprintf(out, "Next: import this page in resources/js/app.tsx and register resolver key %q with the persistent layout.\n", name)
	}
	return nil
}
