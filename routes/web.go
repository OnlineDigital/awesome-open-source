// Package routes is the central route table. Controllers are packages/folders;
// each exported action lives in its own source file.
package routes

import (
	"awesomeopensource.dev/app/application"
	"awesomeopensource.dev/app/controllers/admin_category_controller"
	"awesomeopensource.dev/app/controllers/admin_controller"
	"awesomeopensource.dev/app/controllers/admin_project_controller"
	"awesomeopensource.dev/app/controllers/admin_settings_controller"
	"awesomeopensource.dev/app/controllers/admin_topic_controller"
	"awesomeopensource.dev/app/controllers/api_controller"
	"awesomeopensource.dev/app/controllers/category_controller"
	"awesomeopensource.dev/app/controllers/health_controller"
	"awesomeopensource.dev/app/controllers/home_controller"
	"awesomeopensource.dev/app/controllers/llms_controller"
	"awesomeopensource.dev/app/controllers/project_controller"
	"awesomeopensource.dev/app/controllers/search_controller"
	"awesomeopensource.dev/app/controllers/seo_controller"
	"awesomeopensource.dev/app/controllers/topic_controller"
	"awesomeopensource.dev/app/mcp"
	"awesomeopensource.dev/internal/router"
	"github.com/gofiber/fiber/v3"
)

func publicCache(c fiber.Ctx) error {
	c.Set("Cache-Control", "public, max-age=60, s-maxage=300, stale-while-revalidate=600")
	c.Vary("X-Inertia")
	return c.Next()
}

func Web(app *application.Application, fiberApp *fiber.App) *router.Router {
	r := router.New(fiberApp)
	app.RouteURL = r.URL
	app.RouteList = func() any { return r.Routes() }

	r.Get("/health", health_controller.Index(app)).Name("health")
	// MCP for AI agents: the secret in the URL is managed in Admin > Settings.
	mcpServer := mcp.New(app)
	r.Get("/mcp/:token/sse", mcpServer.SSE()).Name("mcp.sse")
	r.Post("/mcp/:token/messages", mcpServer.Messages()).Name("mcp.messages")
	r.Post("/mcp/:token", mcpServer.HTTP()).Name("mcp.http")
	r.Delete("/mcp/:token", mcpServer.HTTP()).Name("mcp.http.delete")
	r.Get("/mcp/:token", mcpServer.HTTP()).Name("mcp.http.get")

	r.Get("/robots.txt", seo_controller.Robots(app)).Name("robots")
	r.Get("/sitemap.xml", seo_controller.Index(app)).Name("sitemap")
	r.Get("/sitemap-general.xml", seo_controller.General(app)).Name("sitemap.general")
	r.Get("/sitemap-project-:file", seo_controller.Projects(app)).Name("sitemap.projects")

	// Read-only JSON for WebMCP tools and agents.
	r.Group("/api", "api.", nil, func(api *router.Router) {
		api.Get("/search", api_controller.Search(app)).Name("search")
		api.Get("/categories", api_controller.Categories(app)).Name("categories")
		api.Get("/projects/:slug", api_controller.Project(app)).Name("project")
		api.Get("/projects/:slug/deepwiki", api_controller.DeepWiki(app)).Name("project.deepwiki")
		api.Get("/topics", api_controller.Topics(app)).Name("topics")
		api.Get("/topics/:slug", api_controller.Topic(app)).Name("topic")
	})

	// Public site: default locale at the root, every other locale under /{locale}.
	for _, loc := range app.I18n.Locales {
		prefix := ""
		if loc != app.Cfg.DefaultLocale {
			prefix = "/" + loc
		}
		r.Group(prefix, loc+".", []router.Middleware{app.LocaleMiddleware(loc), publicCache}, func(g *router.Router) {
			g.Get("/", home_controller.Index(app)).Name("home")
			g.Get("/category/:slug", category_controller.Show(app)).Name("category")
			g.Get("/topic/:slug", topic_controller.Show(app)).Name("topic")
			g.Get("/project/:slug", project_controller.Show(app)).Name("project")
			g.Get("/search", search_controller.Index(app)).Name("search")
			g.Get("/llms.txt", llms_controller.Index(app)).Name("llms")
			g.Get("/llms/category/:file", llms_controller.Category(app)).Name("llms.category")
			g.Get("/llms/topic/:file", llms_controller.Topic(app)).Name("llms.topic")
			g.Get("/llms/:file", llms_controller.Project(app)).Name("llms.project")
		})
	}

	r.Group("/admin", "admin.", []router.Middleware{app.SameOrigin}, func(admin *router.Router) {
		admin.Group("", "", []router.Middleware{app.LoadAdmin}, func(guest *router.Router) {
			guest.Get("/login", admin_controller.LoginPage(app)).Name("login")
			guest.Post("/login", admin_controller.Login(app)).Name("login.store")
		})
		admin.Group("", "", []router.Middleware{app.RequireAdmin}, func(g *router.Router) {
			g.Post("/logout", admin_controller.Logout(app)).Name("logout")
			g.Get("/", admin_controller.Dashboard(app)).Name("dashboard")
			g.Post("/reindex", admin_controller.Reindex(app)).Name("reindex")
			g.Post("/sync", admin_controller.SyncStale(app)).Name("sync")

			g.Get("/projects", admin_project_controller.Index(app)).Name("projects.index")
			g.Post("/projects", admin_project_controller.Store(app)).Name("projects.store")
			g.Get("/projects/:id", admin_project_controller.Edit(app)).Name("projects.edit")
			g.Put("/projects/:id", admin_project_controller.Update(app)).Name("projects.update")
			g.Delete("/projects/:id", admin_project_controller.Destroy(app)).Name("projects.destroy")
			g.Post("/projects/:id/sync", admin_project_controller.Sync(app)).Name("projects.sync")
			g.Put("/projects/:id/translations", admin_project_controller.SaveTranslations(app)).Name("projects.translations")
			g.Post("/projects/:id/translate", admin_project_controller.Translate(app)).Name("projects.translate")
			g.Post("/projects/:id/screenshots", admin_project_controller.UploadScreenshot(app)).Name("screenshots.store")
			g.Put("/projects/:id/screenshots/:shot", admin_project_controller.UpdateScreenshot(app)).Name("screenshots.update")
			g.Delete("/projects/:id/screenshots/:shot", admin_project_controller.DeleteScreenshot(app)).Name("screenshots.destroy")

			g.Get("/categories", admin_category_controller.Index(app)).Name("categories.index")
			g.Post("/categories", admin_category_controller.Store(app)).Name("categories.store")
			g.Put("/categories/:id", admin_category_controller.Update(app)).Name("categories.update")
			g.Delete("/categories/:id", admin_category_controller.Destroy(app)).Name("categories.destroy")
			g.Put("/projects/:id/topics", admin_topic_controller.ProjectTopics(app)).Name("projects.topics")

			g.Get("/topics", admin_topic_controller.Index(app)).Name("topics.index")
			g.Post("/topics", admin_topic_controller.Store(app)).Name("topics.store")
			g.Get("/topics/:id", admin_topic_controller.Edit(app)).Name("topics.edit")
			g.Put("/topics/:id", admin_topic_controller.Update(app)).Name("topics.update")
			g.Delete("/topics/:id", admin_topic_controller.Destroy(app)).Name("topics.destroy")
			g.Put("/topics/:id/features", admin_topic_controller.SaveFeatures(app)).Name("topics.features")
			g.Put("/topics/:id/values", admin_topic_controller.SetValue(app)).Name("topics.values")
			g.Post("/topics/:id/projects", admin_topic_controller.AddProject(app)).Name("topics.projects.add")
			g.Delete("/topics/:id/projects/:project", admin_topic_controller.RemoveProject(app)).Name("topics.projects.remove")
			g.Post("/topics/:id/translate", admin_topic_controller.Translate(app)).Name("topics.translate")
			g.Get("/settings", admin_settings_controller.Index(app)).Name("settings")
			g.Post("/settings/mcp-tokens", admin_settings_controller.Create(app)).Name("settings.mcp.store")
			g.Put("/settings/mcp-tokens/:id", admin_settings_controller.Rename(app)).Name("settings.mcp.rename")
			g.Post("/settings/mcp-tokens/:id/rotate", admin_settings_controller.Rotate(app)).Name("settings.mcp.rotate")
			g.Post("/settings/mcp-tokens/:id/revoke", admin_settings_controller.Revoke(app)).Name("settings.mcp.revoke")
			g.Delete("/settings/mcp-tokens/:id", admin_settings_controller.Destroy(app)).Name("settings.mcp.destroy")
			g.Post("/categories/:id/translate", admin_category_controller.Translate(app)).Name("categories.translate")
		})
	})
	return r
}
