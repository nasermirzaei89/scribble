package web

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"maps"
	"net/http"
	"runtime/debug"

	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
	"github.com/nasermirzaei89/scribble/auth"
	authcontext "github.com/nasermirzaei89/scribble/auth/context"
	"github.com/nasermirzaei89/scribble/contents"
)

//go:embed templates/*.gohtml
var templatesFS embed.FS

const defaultSiteTitle = "Scribble"

type HTTPHandler struct {
	mux         *http.ServeMux
	handler     http.Handler
	tpl         *template.Template
	authSvc     *auth.Service
	contentsSvc *contents.Service
	cookieStore *sessions.CookieStore
	sessionName string
}

var _ http.Handler = (*HTTPHandler)(nil)

func NewHTTPHandler(
	authSvc *auth.Service,
	contentsSvc *contents.Service,
	cookieStore *sessions.CookieStore,
	sessionName string,
	csrfAuthKeys []byte,
	csrfTrustedOrigins []string,
) (*HTTPHandler, error) {
	httpHandler := &HTTPHandler{
		authSvc:     authSvc,
		contentsSvc: contentsSvc,
		cookieStore: cookieStore,
		sessionName: sessionName,
	}

	{
		tpl, err := template.ParseFS(templatesFS, "templates/*.gohtml")
		if err != nil {
			return nil, fmt.Errorf("failed to parse templates: %w", err)
		}

		httpHandler.tpl = tpl
	}

	{
		httpHandler.mux = &http.ServeMux{}
		httpHandler.handler = httpHandler.mux

		httpHandler.registerRoutes()
	}

	httpHandler.handler = httpHandler.authMiddleware(httpHandler.handler)

	{
		csrfMiddleware := csrf.Protect(
			csrfAuthKeys,
			csrf.TrustedOrigins(csrfTrustedOrigins),
		)

		httpHandler.handler = csrfMiddleware(httpHandler.handler)
	}

	httpHandler.handler = recoverMiddleware(httpHandler.handler)

	return httpHandler, nil
}

func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handler.ServeHTTP(w, r)
}

func (h *HTTPHandler) registerRoutes() {
	h.mux.HandleFunc("/", h.HandleIndex)

	h.mux.Handle("GET /register", h.HandleRegisterPage())
	h.mux.Handle("POST /register", h.HandleRegister())
	h.mux.Handle("GET /login", h.HandleLoginPage())
	h.mux.Handle("POST /login", h.HandleLogin())
	h.mux.Handle("GET /logout", h.HandleLogoutPage())
	h.mux.Handle("POST /logout", h.HandleLogout())

	h.mux.Handle("GET /create-post", h.HandleCreatePostPage())
	h.mux.Handle("POST /create-post", h.HandleCreatePost())
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func(ctx context.Context) {
			if err := recover(); err != nil {
				slog.ErrorContext(
					ctx,
					"recovered from panic",
					"error",
					err,
					"stack",
					string(debug.Stack()),
				)

				http.Error(w, "internal error occurred", http.StatusInternalServerError)
			}
		}(r.Context())

		next.ServeHTTP(w, r)
	})
}
func (h *HTTPHandler) renderTemplate(w http.ResponseWriter, r *http.Request, name string, extraData map[string]any,
) {
	data := map[string]any{
		"CurrentPath":     r.URL.Path,
		"Lang":            "en",
		"Dir":             "ltr",
		"IsAuthenticated": isAuthenticated(r),
	}

	maps.Copy(data, extraData)

	data["SiteTitle"] = defaultSiteTitle

	if extraData["SiteTitle"] != nil {
		data["SiteTitle"] = fmt.Sprintf("%s | %s", extraData["SiteTitle"], data["SiteTitle"])
	}

	err := h.tpl.ExecuteTemplate(w, name, data)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to render template", "name", name, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *HTTPHandler) HandleIndex(w http.ResponseWriter, r *http.Request) {
	h.HandleHomePage(w, r)
}

func (h *HTTPHandler) HandleHomePage(w http.ResponseWriter, r *http.Request) {
	posts, err := h.contentsSvc.ListPosts(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to list posts", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Posts": posts,
	}

	h.renderTemplate(w, r, "home-page.gohtml", data)
}

func (h *HTTPHandler) HandleRegisterPage() http.Handler {
	hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			csrf.TemplateTag: csrf.TemplateField(r),
		}

		h.renderTemplate(w, r, "register-page.gohtml", data)
	})

	return h.GuestOnly(hf)
}

func (h *HTTPHandler) HandleRegister() http.Handler {
	hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to parse form", "error", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		err = h.authSvc.Register(r.Context(), username, password)
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to register user", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})

	return h.GuestOnly(hf)
}

func (h *HTTPHandler) HandleLoginPage() http.Handler {
	hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			csrf.TemplateTag: csrf.TemplateField(r),
		}

		h.renderTemplate(w, r, "login-page.gohtml", data)
	})

	return h.GuestOnly(hf)
}

func (h *HTTPHandler) HandleLogin() http.Handler {
	hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to parse form", "error", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		session, err := h.authSvc.Login(r.Context(), username, password)
		if err != nil {
			switch {
			case errors.Is(err, auth.ErrInvalidCredentials):
				http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			default:
				slog.ErrorContext(r.Context(), "failed to login user", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}

			return
		}

		err = h.setSessionValue(w, r, sessionIDKey, session.ID)
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to set session ID", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	return h.GuestOnly(hf)
}

func (h *HTTPHandler) HandleLogoutPage() http.Handler {
	hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			csrf.TemplateTag: csrf.TemplateField(r),
		}

		h.renderTemplate(w, r, "logout-page.gohtml", data)
	})

	return h.AuthenticatedOnly(hf)
}

func (h *HTTPHandler) HandleLogout() http.Handler {
	hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionID, ok := authcontext.SessionIDFromContext(r.Context())
		if ok {
			slog.DebugContext(r.Context(), "logging out session", "sessionId", sessionID)
			err := h.authSvc.Logout(r.Context(), sessionID)
			if err != nil {
				slog.ErrorContext(r.Context(), "error on logout", "sessionId", sessionID, "error", err)
				http.Error(w, "error on logout", http.StatusInternalServerError)

				return
			}
		}

		slog.DebugContext(r.Context(), "deleting session value", "key", sessionIDKey)

		err := h.deleteSessionValue(w, r, sessionIDKey)
		if err != nil {
			slog.ErrorContext(
				r.Context(),
				"error on deleting session value",
				"key",
				sessionIDKey,
				"error",
				err,
			)
			http.Error(w, "error on deleting session value", http.StatusInternalServerError)

			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	return h.AuthenticatedOnly(hf)
}

func (h *HTTPHandler) HandleCreatePostPage() http.Handler {
	hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			csrf.TemplateTag: csrf.TemplateField(r),
		}

		h.renderTemplate(w, r, "create-post-page.gohtml", data)
	})

	return h.AuthenticatedOnly(hf)
}

func (h *HTTPHandler) HandleCreatePost() http.Handler {
	hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to parse form", "error", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		content := r.FormValue("content")

		currentUser, err := h.authSvc.GetCurrentUser(r.Context())
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to get current user", "error", err)
			http.Error(w, "Failed to get current user", http.StatusInternalServerError)

			return
		}

		_, err = h.contentsSvc.CreatePost(r.Context(), contents.CreatePostRequest{
			AuthorID: currentUser.ID,
			Content:  content,
		})
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to create post", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	return h.AuthenticatedOnly(hf)
}
