package api

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/stashapp/stash/pkg/manager/config"

	"github.com/gorilla/sessions"
)

const cookieName = "session"
const usernameFormKey = "username"
const passwordFormKey = "password"
const userIDKey = "userID"
const maxCookieAge = 60 * 60 * 1 // 1 hours

const returnURLParam = "returnURL"

var sessionStore = sessions.NewCookieStore(config.GetSessionStoreKey())

type LoginTemplateData struct {
	URL   string
	Error string
}

func initSessionStore() {
	sessionStore.MaxAge(maxCookieAge)
}

func redirectToLogin(w http.ResponseWriter, returnURL string, loginError string) {
	data, _ := setupUIBox.Find("login.html")
	templ, err := template.New("Login").Parse(string(data))
	if err != nil {
		http.Error(w, fmt.Sprintf("error: %s", err), http.StatusInternalServerError)
		return
	}

	err = templ.Execute(w, LoginTemplateData{URL: returnURL, Error: loginError})
	if err != nil {
		http.Error(w, fmt.Sprintf("error: %s", err), http.StatusInternalServerError)
	}
}

func getLoginHandler(w http.ResponseWriter, r *http.Request) {
	if !config.HasCredentials() {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	redirectToLogin(w, r.URL.Query().Get(returnURLParam), "")
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue(returnURLParam)
	if url == "" {
		url = "/"
	}

	// ignore error - we want a new session regardless
	newSession, _ := sessionStore.Get(r, cookieName)

	username := r.FormValue("username")
	password := r.FormValue("password")

	// authenticate the user
	if !config.ValidateCredentials(username, password) {
		// redirect back to the login page with an error
		redirectToLogin(w, url, "Username or password is invalid")
		return
	}

	newSession.Values[userIDKey] = username

	err := newSession.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, url, http.StatusFound)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	session, err := sessionStore.Get(r, cookieName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	delete(session.Values, userIDKey)
	session.Options.MaxAge = -1

	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// redirect to the login page if credentials are required
	getLoginHandler(w, r)
}

func getSessionUserID(w http.ResponseWriter, r *http.Request) (string, error) {
	session, err := sessionStore.Get(r, cookieName)
	// ignore errors and treat as an empty user id, so that we handle expired
	// cookie
	if err != nil {
		return "", nil
	}

	if !session.IsNew {
		val := session.Values[userIDKey]

		// refresh the cookie
		err = session.Save(r, w)
		if err != nil {
			return "", err
		}

		ret, _ := val.(string)

		return ret, nil
	}

	return "", nil
}
