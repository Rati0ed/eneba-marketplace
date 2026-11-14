package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type authResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
	Error string `json:"error"`
}

// registerUser creates an account through the API and returns its token.
func registerUser(t *testing.T, h *Handler, email string) string {
	t.Helper()
	var res authResponse
	code := call(t, h, "POST", "/api/auth/register", `{"email":"`+email+`","password":"correct horse"}`, "", &res)
	if code != http.StatusCreated || res.Token == "" {
		t.Fatalf("register %s: status %d, %+v", email, code, res)
	}
	return res.Token
}

func TestRegisterAndLogin(t *testing.T) {
	h := newTestHandler(t)

	var res authResponse
	code := call(t, h, "POST", "/api/auth/register",
		`{"email":"  Ana@Example.COM ","password":"correct horse","display_name":"Ana"}`, "", &res)
	if code != http.StatusCreated || res.Token == "" || res.User.Email != "ana@example.com" || res.User.DisplayName != "Ana" {
		t.Fatalf("register: %d %+v", code, res)
	}

	// The password is stored as a bcrypt hash, never as plain text.
	var stored string
	h.DB.QueryRow(`SELECT password_hash FROM users WHERE email = 'ana@example.com'`).Scan(&stored)
	if stored == "correct horse" || !strings.HasPrefix(stored, "$2a$") {
		t.Errorf("password stored as %q, want a bcrypt hash", stored)
	}

	// Same email in different case = same account -> 409.
	code = call(t, h, "POST", "/api/auth/register", `{"email":"ANA@example.com","password":"another one"}`, "", &res)
	if code != http.StatusConflict {
		t.Errorf("duplicate email: status %d, want 409", code)
	}

	// Login works with any case/spaces in the email.
	code = call(t, h, "POST", "/api/auth/login", `{"email":" ana@EXAMPLE.com","password":"correct horse"}`, "", &res)
	if code != http.StatusOK || res.Token == "" {
		t.Errorf("login: %d %+v", code, res)
	}

	// Wrong password and unknown email get the SAME answer (no hint which one was wrong).
	var wrongPass, unknownEmail authResponse
	code1 := call(t, h, "POST", "/api/auth/login", `{"email":"ana@example.com","password":"wrong password"}`, "", &wrongPass)
	code2 := call(t, h, "POST", "/api/auth/login", `{"email":"nobody@example.com","password":"correct horse"}`, "", &unknownEmail)
	if code1 != http.StatusUnauthorized || code2 != http.StatusUnauthorized || wrongPass.Error != unknownEmail.Error {
		t.Errorf("wrong password: %d %q / unknown email: %d %q", code1, wrongPass.Error, code2, unknownEmail.Error)
	}

	// No display name given -> the part of the email before "@".
	call(t, h, "POST", "/api/auth/register", `{"email":"bob.smith@example.com","password":"correct horse"}`, "", &res)
	if res.User.DisplayName != "bob.smith" {
		t.Errorf("default display name = %q", res.User.DisplayName)
	}
}

func TestRegisterValidation(t *testing.T) {
	h := newTestHandler(t)
	long := strings.Repeat("a", 73)
	bad := map[string]string{
		"not an email":      `{"email":"ana","password":"correct horse"}`,
		"no domain dot":     `{"email":"ana@localhost","password":"correct horse"}`,
		"name + address":    `{"email":"Ana <ana@example.com>","password":"correct horse"}`,
		"short password":    `{"email":"ana@example.com","password":"short"}`,
		"73-byte password":  `{"email":"ana@example.com","password":"` + long + `"}`,
		"spaces password":   `{"email":"ana@example.com","password":"          "}`,
		"name with newline": `{"email":"ana@example.com","password":"correct horse","display_name":"A\nB"}`,
		"broken JSON":       `{"email":`,
		"empty body":        ``,
	}
	for name, body := range bad {
		var res authResponse
		code := call(t, h, "POST", "/api/auth/register", body, "", &res)
		if code != http.StatusBadRequest || res.Error == "" {
			t.Errorf("%s: status %d, error %q; want 400 with a message", name, code, res.Error)
		}
	}
}

func TestMeAndTokenChecks(t *testing.T) {
	h := newTestHandler(t)
	token := registerUser(t, h, "ana@example.com")

	var me User
	if code := call(t, h, "GET", "/api/auth/me", "", token, &me); code != http.StatusOK || me.Email != "ana@example.com" {
		t.Fatalf("/me with token: %d %+v", code, me)
	}

	// signToken builds tokens like an attacker would, to check each one is rejected.
	signToken := func(method jwt.SigningMethod, key any, claims jwt.RegisteredClaims) string {
		s, err := jwt.NewWithClaims(method, claims).SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	valid := jwt.RegisteredClaims{
		Subject: "1", Issuer: tokenIssuer, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	expired := valid
	expired.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute))
	noExpiry := valid
	noExpiry.ExpiresAt = nil
	otherUser := valid
	otherUser.Subject = "999"

	// Change one character of the payload (the middle part): the signature no longer matches.
	parts := strings.Split(token, ".")
	tampered := parts[0] + "." + parts[1][:len(parts[1])-2] + "xx" + "." + parts[2]

	bad := map[string]string{
		"no token":           "",
		"garbage":            "not-a-jwt",
		"tampered payload":   tampered,
		"wrong secret":       signToken(jwt.SigningMethodHS256, []byte("some-other-secret-32-bytes-long!!"), valid),
		"alg none":           signToken(jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, valid),
		"expired":            signToken(jwt.SigningMethodHS256, h.JWTSecret, expired),
		"no expiry":          signToken(jwt.SigningMethodHS256, h.JWTSecret, noExpiry),
		"deleted user (999)": signToken(jwt.SigningMethodHS256, h.JWTSecret, otherUser),
	}
	for name, badToken := range bad {
		if code := call(t, h, "GET", "/api/auth/me", "", badToken, nil); code != http.StatusUnauthorized {
			t.Errorf("%s: status %d, want 401", name, code)
		}
	}

	// And a correctly signed token we built ourselves IS accepted (proves the test tokens are realistic).
	good := signToken(jwt.SigningMethodHS256, h.JWTSecret, jwt.RegisteredClaims{
		Subject: strconv.FormatInt(me.ID, 10), Issuer: tokenIssuer, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})
	if code := call(t, h, "GET", "/api/auth/me", "", good, nil); code != http.StatusOK {
		t.Errorf("hand-made valid token: status %d", code)
	}
}

func TestCartQuote(t *testing.T) {
	h := newTestHandler(t)

	// Find one offer with stock, and one without.
	var inStock, stock, soldOut, price int
	h.DB.QueryRow(`SELECT o.id, o.price_cents, COUNT(k.id) FROM offers o JOIN game_keys k ON k.offer_id = o.id
		WHERE k.status = 'available' GROUP BY o.id HAVING COUNT(k.id) >= 3 LIMIT 1`).Scan(&inStock, &price, &stock)
	h.DB.QueryRow(`SELECT o.id FROM offers o WHERE NOT EXISTS (SELECT 1 FROM game_keys k WHERE k.offer_id = o.id) LIMIT 1`).Scan(&soldOut)

	var quote Quote
	// The fake "unit_price_cents": 1 must be ignored: prices come from the database.
	body := `{"items":[{"offer_id":` + strconv.Itoa(inStock) + `,"quantity":2,"unit_price_cents":1}]}`
	if code := call(t, h, "POST", "/api/cart/quote", body, "", &quote); code != http.StatusOK {
		t.Fatalf("quote: %d", code)
	}
	if !quote.CanCheckout || quote.TotalCents != 2*price || quote.Items[0].UnitPriceCents != price {
		t.Errorf("quote = %+v, want total %d", quote, 2*price)
	}

	body = `{"items":[{"offer_id":` + strconv.Itoa(soldOut) + `,"quantity":1},{"offer_id":` + strconv.Itoa(inStock) +
		`,"quantity":` + strconv.Itoa(min(stock+1, 10)) + `},{"offer_id":999999,"quantity":1}]}`
	call(t, h, "POST", "/api/cart/quote", body, "", &quote)
	problems := []string{quote.Items[0].Problem, quote.Items[1].Problem, quote.Items[2].Problem}
	want := []string{"out_of_stock", "not_enough_stock", "not_found"}
	if stock >= 10 {
		want[1] = "" // can't ask for more than 10, so with 10+ keys this line is fine
	}
	for i := range want {
		if problems[i] != want[i] {
			t.Errorf("line %d problem = %q, want %q", i, problems[i], want[i])
		}
	}
	if quote.CanCheckout {
		t.Error("a cart with problems must not be checkout-able")
	}

	// An empty cart is a valid quote (total 0), just not checkout-able.
	call(t, h, "POST", "/api/cart/quote", `{"items":[]}`, "", &quote)
	if quote.CanCheckout || quote.TotalCents != 0 {
		t.Errorf("empty cart: %+v", quote)
	}

	for name, bad := range map[string]string{
		"quantity 0":   `{"items":[{"offer_id":1,"quantity":0}]}`,
		"quantity 11":  `{"items":[{"offer_id":1,"quantity":11}]}`,
		"negative id":  `{"items":[{"offer_id":-1,"quantity":1}]}`,
		"duplicate":    `{"items":[{"offer_id":1,"quantity":1},{"offer_id":1,"quantity":2}]}`,
		"string price": `{"items":"lots"}`,
	} {
		if code := call(t, h, "POST", "/api/cart/quote", bad, "", nil); code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", name, code)
		}
	}
}
