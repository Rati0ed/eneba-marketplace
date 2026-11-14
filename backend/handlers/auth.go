package handlers

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

// How authentication works:
//  1. Register/login: the server checks the password and replies with a signed token (a JWT).
//  2. The browser stores the token and sends it on every protected request:
//     Authorization: Bearer <token>
//  3. RequireAuth checks the token's signature and expiry, loads the user, and lets the
//     request through. A token can't be forged without JWT_SECRET, which only the server knows.

const (
	tokenLifetime  = 24 * time.Hour
	tokenIssuer    = "gamekey"
	minPasswordLen = 8
	// bcrypt only uses the first 72 bytes of a password. Longer passwords would be silently
	// cut, so "72 x a" + "anything" would equal "72 x a". We reject them instead.
	maxPasswordBytes = 72
	maxNameLen       = 40
)

// bcryptCost = how slow hashing is. Each +1 doubles the work: 10 ≈ 50-100 ms per hash.
// Slow is the point: an attacker with a stolen database can only test a few guesses per second.
// A variable (not a constant) so tests can make it fast.
var bcryptCost = bcrypt.DefaultCost

// User is the public view of an account. The password hash is never sent anywhere.
type User struct {
	ID          int64  `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

// ---------- Register ----------

// Register answers POST /api/auth/register {"email", "password", "display_name"}.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}
	if !readJSON(w, r, &body) {
		return
	}

	email, err := normalizeEmail(body.Email)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validatePassword(body.Password); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	name, err := normalizeDisplayName(body.DisplayName, email)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// bcrypt adds a random "salt" to every hash, so two users with the same password still
	// get different hashes, and precomputed tables of common passwords are useless.
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcryptCost)
	if err != nil {
		serverError(w, err)
		return
	}

	res, err := h.DB.ExecContext(r.Context(),
		`INSERT INTO users (email, password_hash, display_name) VALUES (?, ?, ?)`, email, string(hash), name)
	// The UNIQUE constraint on users.email is the real duplicate check: it also works if
	// two people register the same email at the same moment.
	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
		writeError(w, http.StatusConflict, "An account with this email already exists. Try signing in.")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	id, _ := res.LastInsertId()

	h.writeAuthResponse(w, http.StatusCreated, User{ID: id, Email: email, DisplayName: name})
}

// normalizeEmail trims and lowercases the email (so "Ana@X.com" and "ana@x.com" are the
// same account) and checks it looks like a real address.
func normalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	invalid := errors.New("Please enter a valid email address, like name@example.com.")
	if email == "" || len(email) > 254 {
		return "", invalid
	}
	// net/mail parses addresses the way email software does. It also accepts forms like
	// "Ana <ana@x.com>", so we require that the parsed address is exactly what was typed.
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return "", invalid
	}
	// Require a dot in the domain: "ana@localhost" is technically valid but not for a shop.
	_, domain, _ := strings.Cut(email, "@")
	if !strings.Contains(domain, ".") || strings.HasSuffix(domain, ".") {
		return "", invalid
	}
	return email, nil
}

func validatePassword(password string) error {
	if utf8.RuneCountInString(password) < minPasswordLen {
		return errors.New("Password must be at least 8 characters long.")
	}
	if len(password) > maxPasswordBytes {
		return errors.New("Password is too long (max 72 characters).")
	}
	if strings.TrimSpace(password) == "" {
		return errors.New("Password can't be only spaces.")
	}
	return nil
}

// normalizeDisplayName trims the name; if empty, it uses the part of the email before "@".
func normalizeDisplayName(raw, email string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		name, _, _ = strings.Cut(email, "@")
	}
	if utf8.RuneCountInString(name) > maxNameLen {
		return "", errors.New("Name is too long (max 40 characters).")
	}
	for _, r := range name {
		if r < 32 || r == 127 { // control characters (newlines, tabs...) don't belong in a name
			return "", errors.New("Name contains invalid characters.")
		}
	}
	return name, nil
}

// ---------- Login ----------

// dummyHash is compared against when the email doesn't exist, so a wrong email takes as long
// as a wrong password. Otherwise an attacker could time the responses to find out which
// emails have accounts ("user enumeration").
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy password for timing"), bcrypt.DefaultCost)

// Login answers POST /api/auth/login {"email", "password"}.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &body) {
		return
	}

	// One message for every failure: never say WHICH part was wrong.
	const wrongCredentials = "Wrong email or password."
	email := strings.ToLower(strings.TrimSpace(body.Email))

	var user User
	var hash string
	err := h.DB.QueryRowContext(r.Context(),
		`SELECT id, email, display_name, password_hash FROM users WHERE email = ?`, email,
	).Scan(&user.ID, &user.Email, &user.DisplayName, &hash)

	if errors.Is(err, sql.ErrNoRows) {
		bcrypt.CompareHashAndPassword(dummyHash, []byte(body.Password)) // same work as a real check
		writeError(w, http.StatusUnauthorized, wrongCredentials)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}

	// CompareHashAndPassword hashes the typed password with the salt stored inside `hash`
	// and compares the results in constant time.
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.Password)) != nil {
		writeError(w, http.StatusUnauthorized, wrongCredentials)
		return
	}

	h.writeAuthResponse(w, http.StatusOK, user)
}

// Me answers GET /api/auth/me: who does this token belong to? (RequireAuth already checked it.)
// The frontend calls it on page load to restore the login after a refresh.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, currentUser(r))
}

// writeAuthResponse sends {"token": "...", "user": {...}} after register/login.
func (h *Handler) writeAuthResponse(w http.ResponseWriter, status int, user User) {
	token, err := h.issueToken(user.ID)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, status, map[string]any{"token": token, "user": user})
}

// ---------- Tokens (JWT) ----------

// issueToken creates a signed JWT saying "this is user <id>, valid for 24 hours".
// A JWT has 3 parts: header.payload.signature. Anyone can READ the payload (it's only
// base64), so it holds no secrets; but nobody can CHANGE it without breaking the signature.
func (h *Handler) issueToken(userID int64) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10), // "sub": who the token is about
		Issuer:    tokenIssuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(tokenLifetime)),
	}
	// HS256 = HMAC-SHA256: a signature made with our secret key.
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(h.JWTSecret)
}

// parseToken checks a token and returns the user id inside it.
func (h *Handler) parseToken(tokenString string) (int64, error) {
	var claims jwt.RegisteredClaims
	_, err := jwt.ParseWithClaims(tokenString, &claims,
		func(*jwt.Token) (any, error) { return h.JWTSecret, nil },
		// Only accept HS256. Without this, a forged token saying "alg": "none" (no signature)
		// could be accepted by careless code: a famous JWT attack.
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(tokenIssuer),
		jwt.WithExpirationRequired(), // reject tokens without an expiry date
	)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(claims.Subject, 10, 64)
}

// ---------- Middleware: protect routes ----------

// contextKey is our own type for context keys, so no other package can accidentally use
// the same key (a plain string "user" could clash).
type contextKey string

const userContextKey contextKey = "user"

// RequireAuth wraps a handler so it only runs for logged-in users:
//
//	mux.HandleFunc("GET /api/orders", h.RequireAuth(h.ListOrders))
//
// It reads "Authorization: Bearer <token>", checks the token, loads the user from the
// database and stores it in the request's context, where the handler finds it.
func (h *Handler) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !found || tokenString == "" {
			writeError(w, http.StatusUnauthorized, "Please sign in to continue.")
			return
		}

		const expired = "Your session has expired. Please sign in again."
		userID, err := h.parseToken(tokenString)
		if err != nil {
			writeError(w, http.StatusUnauthorized, expired)
			return
		}

		// The token is genuine, but does the user still exist? Our database is re-created on
		// every server start, so an old token may point to an account that's gone.
		var user User
		err = h.DB.QueryRowContext(r.Context(),
			`SELECT id, email, display_name FROM users WHERE id = ?`, userID,
		).Scan(&user.ID, &user.Email, &user.DisplayName)
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, expired)
			return
		}
		if err != nil {
			serverError(w, err)
			return
		}

		// context.WithValue makes a copy of the request's context with the user attached.
		// Python comparison: like Flask's `g.user`, but passed along explicitly with the request.
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next(w, r.WithContext(ctx))
	}
}

// currentUser returns the logged-in user. Only call it in handlers wrapped by RequireAuth.
func currentUser(r *http.Request) User {
	user, _ := r.Context().Value(userContextKey).(User)
	return user
}
