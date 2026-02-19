package main

import (
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"maps"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/nasermirzaei89/scribble/auth"
	"github.com/nasermirzaei89/scribble/contents"
)

//go:embed templates/*.gohtml
var templatesFS embed.FS

const defaultSiteTitle = "Scribble"

type HTTPHandler struct {
	mux         http.ServeMux
	tpl         *template.Template
	authSvc     *auth.Service
	contentsSvc *contents.Service
	cookieStore *sessions.CookieStore
	sessionName string
}

var _ http.Handler = (*HTTPHandler)(nil)

func NewHTTPHandler(authSvc *auth.Service, contentsSvc *contents.Service, cookieStore *sessions.CookieStore, sessionName string) (*HTTPHandler, error) {
	httpHandler := &HTTPHandler{
		authSvc:     authSvc,
		contentsSvc: contentsSvc,
		cookieStore: cookieStore,
		sessionName: sessionName,
	}

	tpl, err := template.ParseFS(templatesFS, "templates/*.gohtml")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	httpHandler.tpl = tpl

	httpHandler.mux = http.ServeMux{}

	httpHandler.registerRoutes()

	return httpHandler, nil
}

func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *HTTPHandler) registerRoutes() {
	h.mux.HandleFunc("/", h.HandleIndex)

	h.mux.HandleFunc("GET /register", h.HandleRegisterPage)
	h.mux.HandleFunc("POST /register", h.HandleRegister)
	h.mux.HandleFunc("GET /login", h.HandleLoginPage)
	h.mux.HandleFunc("POST /login", h.HandleLogin)
	h.mux.HandleFunc("GET /logout", h.HandleLogoutPage)
	h.mux.HandleFunc("POST /logout", h.HandleLogout)

	h.mux.HandleFunc("GET /create-post", h.HandleCreatePostPage)
	h.mux.HandleFunc("POST /create-post", h.HandleCreatePost)
}

func (h *HTTPHandler) isAuthenticated(r *http.Request) bool {
	sessionID, _ := h.getSessionID(r)
	return sessionID != ""
}

func (h *HTTPHandler) renderTemplate(w http.ResponseWriter, r *http.Request, name string, extraData map[string]any,
) {
	data := map[string]any{
		"CurrentPath":     r.URL.Path,
		"Lang":            "en",
		"Dir":             "ltr",
		"IsAuthenticated": h.isAuthenticated(r),
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

func (h *HTTPHandler) HandleRegisterPage(w http.ResponseWriter, r *http.Request) {
	h.renderTemplate(w, r, "register-page.gohtml", nil)
}

func (h *HTTPHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
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
}

func (h *HTTPHandler) HandleLoginPage(w http.ResponseWriter, r *http.Request) {
	h.renderTemplate(w, r, "login-page.gohtml", nil)
}

func (h *HTTPHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
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

	err = h.setSessionID(w, r, session.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to set session ID", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *HTTPHandler) setSessionID(w http.ResponseWriter, r *http.Request, sessionID string) error {
	session, _ := h.cookieStore.Get(r, h.sessionName)
	session.Values["session_id"] = sessionID

	err := session.Save(r, w)
	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

func (h *HTTPHandler) getSessionID(r *http.Request) (string, error) {
	session, err := h.cookieStore.Get(r, h.sessionName)
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}

	sessionID, ok := session.Values["session_id"].(string)
	if !ok {
		return "", fmt.Errorf("session ID not found in session")
	}

	return sessionID, nil
}

func (h *HTTPHandler) deleteSessionID(w http.ResponseWriter, r *http.Request) error {
	session, err := h.cookieStore.Get(r, h.sessionName)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	delete(session.Values, "session_id")

	err = session.Save(r, w)
	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

func (h *HTTPHandler) HandleLogoutPage(w http.ResponseWriter, r *http.Request) {
	h.renderTemplate(w, r, "logout-page.gohtml", nil)
}

func (h *HTTPHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	sessionID, err := h.getSessionID(r)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to get session ID", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = h.authSvc.Logout(r.Context(), sessionID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to logout user", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = h.deleteSessionID(w, r)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to delete session ID", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *HTTPHandler) HandleCreatePostPage(w http.ResponseWriter, r *http.Request) {
	h.renderTemplate(w, r, "create-post-page.gohtml", nil)
}

func (h *HTTPHandler) HandleCreatePost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to parse form", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	content := r.FormValue("content")

	userID, err := h.getCurrentUserID(r)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to get current user ID", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	_, err = h.contentsSvc.CreatePost(r.Context(), contents.CreatePostRequest{
		AuthorID: userID,
		Content:  content,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to create post", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *HTTPHandler) getCurrentUserID(r *http.Request) (string, error) {
	sessionID, err := h.getSessionID(r)
	if err != nil {
		return "", fmt.Errorf("failed to get session ID: %w", err)
	}

	session, err := h.authSvc.GetSession(r.Context(), sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to find session: %w", err)
	}

	return session.UserID, nil
}
