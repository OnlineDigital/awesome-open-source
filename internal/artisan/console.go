package artisan

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"awesomeopensource.dev/app/application"
	"awesomeopensource.dev/app/config"
	"awesomeopensource.dev/app/github"
	"awesomeopensource.dev/app/store"
	"golang.org/x/crypto/bcrypt"
)

// runConsole executes data-maintenance commands against the configured database.
func runConsole(command string, args []string, out io.Writer) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cfg := config.Load()
	app, err := application.NewCore(cfg)
	if err != nil {
		return err
	}
	defer app.Close()

	switch command {
	case "migrate":
		_, err = fmt.Fprintln(out, "database is up to date")
		return err

	case "admin:create":
		o, err := parse(args, "password", "")
		if err != nil {
			return err
		}
		if len(o.args) != 1 {
			return fmt.Errorf("usage: admin:create EMAIL [--password PASSWORD]")
		}
		password := o.values["password"]
		if password == "" {
			password = os.Getenv("ADMIN_PASSWORD")
		}
		if len(password) < 10 {
			return fmt.Errorf("password must have at least 10 characters (use --password or ADMIN_PASSWORD)")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if err := app.Store.UpsertAdmin(ctx, o.args[0], string(hash)); err != nil {
			return err
		}
		_, err = fmt.Fprintf(out, "admin %s saved\n", o.args[0])
		return err

	case "import":
		o, err := parse(args, "category", "")
		if err != nil {
			return err
		}
		if len(o.args) != 1 {
			return fmt.Errorf("usage: import FILE|- [--category SLUG]")
		}
		var catID *int64
		if slug := o.values["category"]; slug != "" {
			c, err := app.Store.CategoryBySlug(ctx, slug, cfg.DefaultLocale)
			if err != nil {
				return fmt.Errorf("category %q not found", slug)
			}
			catID = &c.ID
		}
		var in io.Reader = os.Stdin
		if o.args[0] != "-" {
			f, err := os.Open(o.args[0])
			if err != nil {
				return err
			}
			defer f.Close()
			in = f
		}
		added, existing, invalid := 0, 0, 0
		sc := bufio.NewScanner(in)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			owner, name, ok := github.ParseRepo(strings.Fields(line)[0])
			if !ok {
				invalid++
				fmt.Fprintf(out, "skipped (not a GitHub repo): %s\n", line)
				continue
			}
			if _, err := app.Store.CreateProject(ctx, owner, name, catID); err == store.ErrExists {
				existing++
			} else if err != nil {
				return err
			} else {
				added++
			}
		}
		fmt.Fprintf(out, "added %d, already present %d, invalid %d. Run `sync` to fetch metadata and scores.\n", added, existing, invalid)
		return sc.Err()

	case "sync":
		o, err := parse(args, "id limit", "all")
		if err != nil {
			return err
		}
		if id := o.values["id"]; id != "" {
			n, err := strconv.ParseInt(id, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid --id")
			}
			if err := app.Syncer.SyncOne(ctx, n); err != nil {
				return err
			}
			_, err = fmt.Fprintln(out, "synced")
			return err
		}
		limit := 100000
		if l := o.values["limit"]; l != "" {
			if limit, err = strconv.Atoi(l); err != nil {
				return fmt.Errorf("invalid --limit")
			}
		}
		stale := cfg.SyncStaleAge
		if o.booleans["all"] {
			stale = 0
		}
		if cfg.GitHubToken == "" {
			fmt.Fprintln(out, "warning: GITHUB_TOKEN is not set; unauthenticated GitHub API allows only 60 requests/hour and sees only the default branch")
		}
		ok, failed, err := app.Syncer.SyncStale(ctx, stale, limit)
		fmt.Fprintf(out, "synced %d, failed %d\n", ok, failed)
		return err

	case "search:reindex":
		n, err := app.Indexer.Reindex(ctx)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(out, "indexed %d projects in %d locales\n", n, len(app.I18n.Locales))
		return err

	case "translate":
		o, err := parse(args, "locale id", "all force")
		if err != nil {
			return err
		}
		if !app.Translator.Enabled() {
			return fmt.Errorf("set ANTHROPIC_API_KEY to enable machine translation")
		}
		var locales []string
		if l := o.values["locale"]; l != "" {
			locales = []string{l}
		}
		var ids []int64
		if id := o.values["id"]; id != "" {
			n, err := strconv.ParseInt(id, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid --id")
			}
			ids = []int64{n}
		} else if o.booleans["all"] {
			ps, _, err := app.Store.ListProjects(ctx, store.ProjectFilter{Status: "published", Limit: 200})
			if err != nil {
				return err
			}
			for _, p := range ps {
				ids = append(ids, p.ID)
			}
		} else {
			return fmt.Errorf("usage: translate --all | --id N [--locale xx] [--force]")
		}
		for _, id := range ids {
			done, err := app.TranslateProject(ctx, id, locales, o.booleans["force"])
			fmt.Fprintf(out, "project %d: translated %v\n", id, done)
			if err != nil {
				fmt.Fprintf(out, "  error: %v\n", err)
			} else if len(done) > 0 {
				_ = app.Indexer.IndexProject(ctx, id)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}
		cats, _ := app.Store.ListCategories(ctx, cfg.DefaultLocale)
		for _, c := range cats {
			if done, err := app.TranslateCategory(ctx, c.ID, o.booleans["force"]); err != nil {
				fmt.Fprintf(out, "category %s: %v\n", c.Slug, err)
			} else if len(done) > 0 {
				fmt.Fprintf(out, "category %s: translated %v\n", c.Slug, done)
			}
		}
		return nil

	case "embed":
		if !app.Embedder.Enabled() {
			return fmt.Errorf("set EMBEDDINGS_URL (OpenAI-compatible endpoint) to create embeddings")
		}
		total := 0
		for {
			ids, err := app.Store.EmbeddingTargets(ctx, 50)
			if err != nil || len(ids) == 0 {
				fmt.Fprintf(out, "embedded %d projects\n", total)
				return err
			}
			for _, id := range ids {
				if err := app.Syncer.EmbedProject(ctx, id); err != nil {
					return fmt.Errorf("project %d: %w", id, err)
				}
				total++
				time.Sleep(50 * time.Millisecond)
			}
		}
	}
	return fmt.Errorf("unknown command %q", command)
}
