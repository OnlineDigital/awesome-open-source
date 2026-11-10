// Package admin_controller handles admin sign-in and the dashboard.
package admin_controller

import (
	"errors"
	"strings"

	"awesomeopensource.dev/app/application"
	"awesomeopensource.dev/app/store"
	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/bcrypt"
)

func LoginPage(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		if app.Admin(c) != nil {
			return c.Redirect().Status(fiber.StatusSeeOther).To("/admin")
		}
		return app.AdminPage(c, "Admin/Login", map[string]any{})
	}
}

type loginRequest struct {
	Email    string `json:"email" form:"email"`
	Password string `json:"password" form:"password"`
}

// dummyHash keeps response time similar for unknown emails.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("not-a-real-password"), bcrypt.DefaultCost)

func Login(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		if !app.AllowLogin(c) {
			return app.Redirect303(c, "/admin/login", map[string]string{"email": "Too many attempts. Try again in a few minutes."}, "")
		}
		var req loginRequest
		if err := c.Bind().Body(&req); err != nil {
			return fiber.ErrBadRequest
		}
		u, err := app.Store.AdminByEmail(c.Context(), strings.TrimSpace(req.Email))
		hash := dummyHash
		if err == nil {
			hash = []byte(u.PasswordHash)
		} else if !errors.Is(err, store.ErrNotFound) {
			return err
		}
		if bcrypt.CompareHashAndPassword(hash, []byte(req.Password)) != nil || u == nil {
			return app.Redirect303(c, "/admin/login", map[string]string{"email": "Invalid email or password."}, "")
		}
		app.ResetLogin(c)
		app.LoginAdmin(c, u.ID)
		return c.Redirect().Status(fiber.StatusSeeOther).To("/admin")
	}
}

func Logout(app *application.Application) fiber.Handler {
	return func(c fiber.Ctx) error {
		app.LogoutAdmin(c)
		return c.Redirect().Status(fiber.StatusSeeOther).To("/admin/login")
	}
}
