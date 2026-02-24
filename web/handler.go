package web

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"maps"
	"net/http"
	"runtime/debug"

	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
	"github.com/nasermirzaei89/scribble/auth"
	authcontext "github.com/nasermirzaei89/scribble/auth/context"
	"github.com/nasermirzaei89/scribble/contents"
	"github.com/nasermirzaei89/scribble/discuss"
	"github.com/nasermirzaei89/scribble/reactions"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

var (
	//go:embed templates/*
	templatesFS embed.FS

	//go:embed static/*
	staticFS embed.FS
)

const (
	defaultSiteTitle = "Scribble"
	hxRequestTrue    = "true"
)

type Handler struct {
	mux          *http.ServeMux
	handler      http.Handler
	tpl          *template.Template
	static       fs.FS
	authSvc      *auth.Service
	contentsSvc  *contents.Service
	discussSvc   *discuss.Service
	reactionsSvc *reactions.Service
	cookieStore  *sessions.CookieStore
	sessionName  string
	assetHashes  map[string]string
	markdown     goldmark.Markdown
}

var _ http.Handler = (*Handler)(nil)

func NewHandler(
	authSvc *auth.Service,
	contentsSvc *contents.Service,
	discussSvc *discuss.Service,
	reactionsSvc *reactions.Service,
	cookieStore *sessions.CookieStore,
	sessionName string,
	csrfAuthKeys []byte,
	csrfTrustedOrigins []string,
) (*Handler, error) {
	h := &Handler{
		mux:          nil,
		handler:      nil,
		tpl:          nil,
		authSvc:      authSvc,
		contentsSvc:  contentsSvc,
		discussSvc:   discussSvc,
		reactionsSvc: reactionsSvc,
		cookieStore:  cookieStore,
		sessionName:  sessionName,
		assetHashes:  make(map[string]string),
		markdown:     nil,
	}

	{
		h.markdown = goldmark.New(
			goldmark.WithExtensions(
				extension.GFM, // tables, strikethrough, task lists
			),
			goldmark.WithRendererOptions(
				html.WithUnsafe(), // allow raw HTML (REMOVE if you want stricter)
			),
		)
	}

	{
		tpl, err := template.New("").Funcs(h.funcs()).ParseFS(templatesFS, "templates/*.gohtml")
		if err != nil {
			return nil, fmt.Errorf("failed to parse templates: %w", err)
		}

		h.tpl = tpl
	}

	{
		static, err := fs.Sub(staticFS, "static")
		if err != nil {
			return nil, fmt.Errorf("failed to sub static fs: %w", err)
		}

		h.static = static
	}

	{
		h.mux = &http.ServeMux{}
		h.handler = h.mux

		h.registerRoutes()
	}

	{
		h.handler = h.authMiddleware(h.handler)

		{
			csrfMiddleware := csrf.Protect(
				csrfAuthKeys,
				csrf.TrustedOrigins(csrfTrustedOrigins),
			)

			h.handler = csrfMiddleware(h.handler)
		}

		h.handler = recoverMiddleware(h.handler)
	}

	return h, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handler.ServeHTTP(w, r)
}

func (h *Handler) registerRoutes() {
	h.mux.HandleFunc("/", h.HandleIndex)

	h.mux.Handle("GET /register", h.HandleRegisterPage())
	h.mux.Handle("POST /register", h.HandleRegister())
	h.mux.Handle("GET /login", h.HandleLoginPage())
	h.mux.Handle("POST /login", h.HandleLogin())
	h.mux.Handle("GET /logout", h.HandleLogoutPage())
	h.mux.Handle("POST /logout", h.HandleLogout())

	h.mux.Handle("GET /create-post", h.HandleCreatePostPage())
	h.mux.Handle("POST /create-post", h.HandleCreatePost())
	h.mux.Handle("GET /p/{postId}", h.HandleViewPostPage())
	h.mux.Handle("POST /p/{postId}/comment", h.HandlePostComment())
	h.mux.Handle("GET /p/{postId}/comments/{commentId}/reply", h.HandleReplyForm())
	h.mux.Handle("POST /react/{targetType}/{targetId}", h.HandleToggleReaction())
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

func (h *Handler) renderTemplate(w http.ResponseWriter, r *http.Request, name string, extraData map[string]any,
) {
	var currentUser *auth.User

	if isAuthenticated(r) {
		var err error

		currentUser, err = h.authSvc.GetCurrentUser(r.Context())
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to get current user", "error", err)
			http.Error(w, "Failed to get current user", http.StatusInternalServerError)

			return
		}
	}

	data := map[string]any{
		"CurrentPath":     r.URL.Path,
		"Lang":            "en",
		"Dir":             "ltr",
		"IsAuthenticated": isAuthenticated(r),
		"CurrentUser":     currentUser,
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

func (h *Handler) HandleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		h.HandleHomePage(w, r)

		return
	}

	h.HandleStatic(w, r)
}

// HandleStatic serves static files.
func (h *Handler) HandleStatic(w http.ResponseWriter, r *http.Request) {
	// w.Header().Set("Cache-Control", "public, max-age=3600")
	http.FileServer(http.FS(h.static)).ServeHTTP(w, r)
}

func (h *Handler) HandleHomePage(w http.ResponseWriter, r *http.Request) {
	posts, err := h.contentsSvc.ListPosts(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to list posts", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)

		return
	}

	postsWithAuthors, err := h.preloadPostAuthor(
		r.Context(),
		posts,
		h.currentUserIDFromRequest(r),
		"/",
		csrf.TemplateField(r),
	)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to preload post authors", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)

		return
	}

	data := map[string]any{
		"Posts":          postsWithAuthors,
		csrf.TemplateTag: csrf.TemplateField(r),
	}

	h.renderTemplate(w, r, "home-page.gohtml", data)
}

type FullPost struct {
	contents.Post

	Author        *auth.User
	CommentsCount *int
	Comments      []*CommentWithAuthor
	Reactions     *ReactionWidgetData
}

type CommentWithAuthor struct {
	discuss.Comment

	Author *auth.User

	Replies   []*CommentWithAuthor
	Reactions *ReactionWidgetData
}

type ReactionWidgetData struct {
	TargetType      reactions.TargetType
	TargetID        string
	Options         []reactions.ReactionOption
	ReturnTo        string
	IsAuthenticated bool
	CSRFField       template.HTML
}

func (h *Handler) preloadPostAuthor(
	ctx context.Context,
	posts []*contents.Post,
	currentUserID *string,
	returnTo string,
	csrfField template.HTML,
) ([]*FullPost, error) {
	var result []*FullPost

	// TODO: optimize this by batching user retrieval instead of doing it one by one
	for _, post := range posts {
		author, err := h.authSvc.GetUser(ctx, post.AuthorID)
		if err != nil {
			return nil, fmt.Errorf("failed to get author: %w", err)
		}

		commentsCount, err := h.discussSvc.CountComments(ctx, post.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to count comments: %w", err)
		}

		reactionData, err := h.buildReactionWidgetData(
			ctx,
			reactions.TargetTypePost,
			post.ID,
			currentUserID,
			returnTo,
			csrfField,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load post reactions: %w", err)
		}

		result = append(result, &FullPost{
			Post:          *post,
			Author:        author,
			CommentsCount: &commentsCount,
			Reactions:     reactionData,
		})
	}

	return result, nil
}

func (h *Handler) HandleRegisterPage() http.Handler {
	hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			csrf.TemplateTag: csrf.TemplateField(r),
			"SiteTitle":      "Register",
		}

		h.renderTemplate(w, r, "register-page.gohtml", data)
	})

	return h.GuestOnly(hf)
}

func (h *Handler) HandleRegister() http.Handler {
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
			var userAlreadyExistsErr *auth.UserAlreadyExistsError
			switch {
			case errors.As(err, &userAlreadyExistsErr):
				http.Error(w, "Username already exists", http.StatusConflict)
			default:
				slog.ErrorContext(r.Context(), "failed to register user", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}

			return
		}

		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})

	return h.GuestOnly(hf)
}

func (h *Handler) HandleLoginPage() http.Handler {
	hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			csrf.TemplateTag: csrf.TemplateField(r),
			"SiteTitle":      "Login",
		}

		h.renderTemplate(w, r, "login-page.gohtml", data)
	})

	return h.GuestOnly(hf)
}

func (h *Handler) HandleLogin() http.Handler {
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

func (h *Handler) HandleLogoutPage() http.Handler {
	hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			csrf.TemplateTag: csrf.TemplateField(r),
			"SiteTitle":      "Logout",
		}

		h.renderTemplate(w, r, "logout-page.gohtml", data)
	})

	return h.AuthenticatedOnly(hf)
}

func (h *Handler) HandleLogout() http.Handler {
	hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionID, ok := authcontext.SessionIDFromContext(r.Context())
		if ok {
			err := h.authSvc.Logout(r.Context(), sessionID)
			if err != nil {
				slog.ErrorContext(r.Context(), "error on logout", "sessionId", sessionID, "error", err)
				http.Error(w, "error on logout", http.StatusInternalServerError)

				return
			}
		}

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

func (h *Handler) HandleCreatePostPage() http.Handler {
	hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			csrf.TemplateTag: csrf.TemplateField(r),
			"SiteTitle":      "Create Post",
		}

		h.renderTemplate(w, r, "create-post-page.gohtml", data)
	})

	return h.AuthenticatedOnly(hf)
}

func (h *Handler) HandleCreatePost() http.Handler {
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

func (h *Handler) HandleViewPostPage() http.Handler {
	hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		postID := r.PathValue("postId")
		currentUserID := h.currentUserIDFromRequest(r)
		returnTo := "/p/" + postID

		post, err := h.contentsSvc.GetPost(r.Context(), postID)
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to get post", "postId", postID, "error", err)
			http.Error(w, "Post not found", http.StatusNotFound)

			return
		}

		author, err := h.authSvc.GetUser(r.Context(), post.AuthorID)
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to get post author", "authorId", post.AuthorID, "error", err)
			http.Error(w, "Failed to get post author", http.StatusInternalServerError)

			return
		}

		comments, err := h.listCommentsWithAuthors(
			r.Context(),
			post.ID,
			currentUserID,
			returnTo,
			csrf.TemplateField(r),
		)
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to list comments with authors", "postId", post.ID, "error", err)
			http.Error(w, "Failed to list comments", http.StatusInternalServerError)

			return
		}

		reactionData, err := h.buildReactionWidgetData(
			r.Context(),
			reactions.TargetTypePost,
			post.ID,
			currentUserID,
			returnTo,
			csrf.TemplateField(r),
		)
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to load post reactions", "postId", post.ID, "error", err)
			http.Error(w, "Failed to load reactions", http.StatusInternalServerError)

			return
		}

		data := map[string]any{
			"Post": FullPost{Post: *post, Author: author, Comments: comments, Reactions: reactionData},
			// "SiteTitle": "View Post", TODO: set post title as site title
			csrf.TemplateTag: csrf.TemplateField(r),
		}

		h.renderTemplate(w, r, "view-post-page.gohtml", data)
	})

	return hf
}

func (h *Handler) listCommentsWithAuthors(
	ctx context.Context,
	postID string,
	currentUserID *string,
	returnTo string,
	csrfField template.HTML,
) ([]*CommentWithAuthor, error) {
	comments, err := h.discussSvc.ListComments(ctx, postID)
	if err != nil {
		return nil, fmt.Errorf("failed to list comments: %w", err)
	}

	result := make([]*CommentWithAuthor, 0, len(comments))
	commentsByID := make(map[string]*CommentWithAuthor, len(comments))

	for _, comment := range comments {
		author, err := h.authSvc.GetUser(ctx, comment.AuthorID)
		if err != nil {
			return nil, fmt.Errorf("failed to get comment author: %w", err)
		}

		commentWithAuthor := &CommentWithAuthor{
			Comment: *comment,
			Author:  author,
		}

		reactionData, err := h.buildReactionWidgetData(
			ctx,
			reactions.TargetTypeComment,
			comment.ID,
			currentUserID,
			returnTo,
			csrfField,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load comment reactions: %w", err)
		}

		commentWithAuthor.Reactions = reactionData

		result = append(result, commentWithAuthor)
		commentsByID[comment.ID] = commentWithAuthor
	}

	roots := make([]*CommentWithAuthor, 0, len(result))

	for _, comment := range result {
		if comment.ReplyTo == nil {
			roots = append(roots, comment)

			continue
		}

		parent, found := commentsByID[*comment.ReplyTo]
		if !found {
			roots = append(roots, comment)

			continue
		}

		parent.Replies = append(parent.Replies, comment)
	}

	return roots, nil
}

func (h *Handler) currentUserIDFromRequest(r *http.Request) *string {
	if !isAuthenticated(r) {
		return nil
	}

	currentUserID := authcontext.GetSubject(r.Context())
	if currentUserID == authcontext.Anonymous {
		return nil
	}

	return &currentUserID
}

func (h *Handler) buildReactionWidgetData(
	ctx context.Context,
	targetType reactions.TargetType,
	targetID string,
	currentUserID *string,
	returnTo string,
	csrfField template.HTML,
) (*ReactionWidgetData, error) {
	targetReactions, err := h.reactionsSvc.GetTargetReactions(ctx, targetType, targetID, currentUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get target reactions: %w", err)
	}

	return &ReactionWidgetData{
		TargetType:      targetReactions.TargetType,
		TargetID:        targetReactions.TargetID,
		Options:         targetReactions.Options,
		ReturnTo:        returnTo,
		IsAuthenticated: currentUserID != nil,
		CSRFField:       csrfField,
	}, nil
}

func (h *Handler) HandlePostComment() http.Handler {
	hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		postID := r.PathValue("postId")

		err := r.ParseForm()
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to parse form", "error", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)

			return
		}

		content := r.FormValue("comment")
		replyToID := r.FormValue("reply_to_id")

		currentUser, err := h.authSvc.GetCurrentUser(r.Context())
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to get current user", "error", err)
			http.Error(w, "Failed to get current user", http.StatusInternalServerError)

			return
		}

		_, err = h.discussSvc.CreateComment(r.Context(), discuss.CreateCommentRequest{
			PostID:   postID,
			AuthorID: currentUser.ID,
			Content:  content,
			ReplyTo:  replyToID,
		})
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to create comment", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)

			return
		}

		http.Redirect(w, r, "/p/"+postID, http.StatusSeeOther)
	})

	return h.AuthenticatedOnly(hf)
}

func (h *Handler) HandleReplyForm() http.Handler {
	hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("HX-Request") != hxRequestTrue {
			http.Error(w, "Direct access is forbidden", http.StatusForbidden)

			return
		}

		postID := r.PathValue("postId")
		commentID := r.PathValue("commentId")

		data := map[string]any{
			csrf.TemplateTag: csrf.TemplateField(r),
			"PostID":         postID,
			"CommentID":      commentID,
		}

		h.renderTemplate(w, r, "reply-form.gohtml", data)
	})

	return h.AuthenticatedOnly(hf)
}

func (h *Handler) HandleToggleReaction() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetType := reactions.TargetType(r.PathValue("targetType"))
		targetID := r.PathValue("targetId")

		err := r.ParseForm()
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to parse reaction form", "error", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)

			return
		}

		returnTo := r.FormValue("return_to")
		if returnTo == "" {
			returnTo = "/"
		}

		if !isAuthenticated(r) {
			if r.Header.Get("HX-Request") == hxRequestTrue {
				w.Header().Set("HX-Redirect", "/login")
				w.WriteHeader(http.StatusUnauthorized)

				return
			}

			http.Redirect(w, r, "/login", http.StatusSeeOther)

			return
		}

		emoji := r.FormValue("emoji")

		currentUser, err := h.authSvc.GetCurrentUser(r.Context())
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to get current user for reaction", "error", err)
			http.Error(w, "Failed to get current user", http.StatusInternalServerError)

			return
		}

		err = h.reactionsSvc.ToggleReaction(r.Context(), targetType, targetID, currentUser.ID, emoji)
		if err != nil {
			var invalidTargetTypeErr reactions.InvalidTargetTypeError

			var invalidEmojiErr reactions.InvalidEmojiError

			switch {
			case errors.As(err, &invalidTargetTypeErr):
				http.Error(w, "Invalid reaction target", http.StatusBadRequest)
			case errors.As(err, &invalidEmojiErr):
				http.Error(w, "Invalid reaction emoji", http.StatusBadRequest)
			default:
				slog.ErrorContext(r.Context(), "failed to toggle reaction", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}

			return
		}

		if r.Header.Get("HX-Request") != hxRequestTrue {
			http.Redirect(w, r, returnTo, http.StatusSeeOther)

			return
		}

		widgetData, err := h.buildReactionWidgetData(
			r.Context(),
			targetType,
			targetID,
			&currentUser.ID,
			returnTo,
			csrf.TemplateField(r),
		)
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to load updated reaction widget", "error", err)
			http.Error(w, "Failed to load reactions", http.StatusInternalServerError)

			return
		}

		err = h.tpl.ExecuteTemplate(w, "reactions.gohtml", widgetData)
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to render reactions template", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)

			return
		}
	})
}
