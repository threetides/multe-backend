package auth

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var AppConfig oauth2.Config
var AllowedReturnOrigins []string

func GoogleConfig() oauth2.Config {
	AllowedReturnOrigins = strings.Split(os.Getenv("CORS_ORIGIN"), ",")
	AppConfig = oauth2.Config{
		RedirectURL:  fmt.Sprintf("%v/api/auth/google/callback", os.Getenv("REDIRECT_URL")),
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		Scopes: []string{"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint: google.Endpoint,
	}
	return AppConfig
}
