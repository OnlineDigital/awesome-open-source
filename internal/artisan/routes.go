package artisan

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"text/tabwriter"

	"github.com/andreitelteu/inertia-go-solid-template/app/server"
)

func printRoutes(out io.Writer) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	// The registry needs no frontend assets or listening server. Development
	// asset configuration avoids requiring a Vite build on a fresh checkout.
	app, err := server.NewWithOptions(root, server.Options{Demo: true, Environment: "development", DevServerURL: "http://127.0.0.1:5173"})
	if err != nil {
		return err
	}
	response, err := app.Test(httptest.NewRequest("GET", "/routes", nil))
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return fmt.Errorf("cannot read route registry: HTTP %d", response.StatusCode)
	}
	var routes []struct {
		Method string `json:"method"`
		Path   string `json:"path"`
		Name   string `json:"name"`
	}
	if err = json.NewDecoder(response.Body).Decode(&routes); err != nil {
		return err
	}
	writer := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "METHOD\tPATH\tNAME")
	for _, route := range routes {
		fmt.Fprintf(writer, "%s\t%s\t%s\n", route.Method, route.Path, route.Name)
	}
	return writer.Flush()
}
