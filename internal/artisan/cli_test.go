package artisan

import (
	"bytes"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestControllerActionsAreSeparateValidGoFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/starter\n\ngo 1.26\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := scaffold(root, "make:controller", "ProjectsController", "index_page,store_form", &output); err != nil {
		t.Fatal(err)
	}
	for filename, functionName := range map[string]string{"index_page.go": "Index", "store_form.go": "Store"} {
		path := filepath.Join(root, "app/controllers/projects_controller", filename)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = parser.ParseFile(token.NewFileSet(), path, data, parser.AllErrors); err != nil {
			t.Fatal(err)
		}
		for _, required := range []string{"package projects_controller", "func " + functionName + "(", "fiber.Handler", "app.RenderPage", "example.com/starter/app/application"} {
			if !strings.Contains(string(data), required) {
				t.Errorf("missing %s", required)
			}
		}
	}
	if !strings.Contains(output.String(), "routes/web.go") {
		t.Fatal("registration instructions missing")
	}
	if err := scaffold(root, "make:controller", "ProjectsController", "index_page,new_page", &output); err == nil {
		t.Fatal("must refuse existing action")
	}
	if _, err := os.Stat(filepath.Join(root, "app/controllers/projects_controller/new_page.go")); !os.IsNotExist(err) {
		t.Fatal("failed scaffold partially wrote a new action")
	}
}

func TestSolidScaffoldPathsAndOverwriteSafety(t *testing.T) {
	root := t.TempDir()
	var output bytes.Buffer
	for _, test := range []struct{ kind, name, path, component string }{
		{"make:component", "UI/AlertBox", "resources/js/components/ui/alert_box.tsx", "AlertBox"},
		{"make:page", "Projects/Index", "resources/js/pages/projects/index_page.tsx", "IndexPage"},
	} {
		if err := scaffold(root, test.kind, test.name, "", &output); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(test.path)))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "export default function "+test.component) {
			t.Fatalf("wrong component: %s", data)
		}
		if err = scaffold(root, test.kind, test.name, "", &output); err == nil {
			t.Fatal("must refuse overwrite")
		}
	}
	for _, name := range []string{"../Escape", "Good/../../Escape", "/Absolute", "Bad-name", "Bad'Name", "Bad\\Name", "Bad;Name", ""} {
		if err := scaffold(root, "make:page", name, "", &output); err == nil {
			t.Errorf("accepted invalid name %q", name)
		}
	}
}

func TestCLIRejectsWrongOptionsBeforeRunningTools(t *testing.T) {
	for _, args := range [][]string{{"build", "--wat"}, {"build", "unexpected"}, {"build", "--output"}, {"build", "--skip-frontend=yes"}, {"dev", "--port", "0"}, {"serve", "--env", "wrong"}, {"test", "unexpected"}, {"make:page", "../Escape"}, {"make:component", "Name", "--action", "index"}, {"unknown"}} {
		var output bytes.Buffer
		if err := Run(args, &output); err == nil {
			t.Errorf("accepted invalid CLI %v", args)
		}
	}
	for _, command := range []string{"help", "list", "--help", "about"} {
		var output bytes.Buffer
		if err := Run([]string{command}, &output); err != nil {
			t.Fatal(err)
		}
		if output.Len() == 0 {
			t.Fatal("empty CLI output")
		}
	}
}

func TestLiteralDotenvAndExistingVariablePrecedence(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	t.Setenv("ARTISAN_EXISTING", "process-wins")
	for _, key := range []string{"ARTISAN_SINGLE", "ARTISAN_DOUBLE", "ARTISAN_PLAIN", "ARTISAN_LITERAL"} {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}
	content := "export ARTISAN_EXISTING=file\nARTISAN_SINGLE='single # quoted'\nARTISAN_DOUBLE=\"double\\nquoted\" # comment\nARTISAN_PLAIN=plain # comment\nARTISAN_LITERAL=$(touch never-execute)\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	if err := LoadEnv(path); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"ARTISAN_EXISTING": "process-wins", "ARTISAN_SINGLE": "single # quoted", "ARTISAN_DOUBLE": "double\nquoted", "ARTISAN_PLAIN": "plain", "ARTISAN_LITERAL": "$(touch never-execute)"} {
		if got := os.Getenv(key); got != want {
			t.Errorf("%s: got %q want %q", key, got, want)
		}
	}
	if err := os.WriteFile(path, []byte("BAD QUOTED='unterminated\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := LoadEnv(path); err == nil {
		t.Fatal("malformed dotenv accepted")
	}
}

func TestKeyGenerationKeepsOtherSettingsAndNeverPrintsSecret(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".env")
	if err := os.WriteFile(path, []byte("APP_ENV=production\nAPP_ADDR=localhost:9000\nAPP_KEY=\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := generateKey(root, false, &output); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "APP_ADDR=localhost:9000") {
		t.Fatal("other env setting changed")
	}
	var key string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "APP_KEY=") {
			key = strings.TrimPrefix(line, "APP_KEY=")
		}
	}
	if len(key) != 64 {
		t.Fatalf("invalid key size %d", len(key))
	}
	if strings.Contains(output.String(), key) {
		t.Fatal("secret printed")
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(path)
		if info.Mode().Perm() != 0600 {
			t.Fatal("env file is not private")
		}
	}
	if err := generateKey(root, false, &output); err == nil {
		t.Fatal("existing key rotated without --force")
	}
	if err := generateKey(root, true, &output); err != nil {
		t.Fatal(err)
	}
	replaced, _ := os.ReadFile(path)
	if string(replaced) == string(data) {
		t.Fatal("--force did not rotate key")
	}
}

func TestRouteListWorksWithoutFrontendBuild(t *testing.T) {
	// Change only this test's working directory. No server is started, no build
	// directory exists, and the registry still includes native Fiber named routes.
	t.Chdir(t.TempDir())
	var output bytes.Buffer
	if err := Run([]string{"route:list"}, &output); err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"/users/:user", "users.show", "POST", "form.store"} {
		if !strings.Contains(output.String(), marker) {
			t.Errorf("missing route marker %s", marker)
		}
	}
}
