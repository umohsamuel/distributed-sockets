package env

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/umohsamuel/distributed-sockets/config/file"
)

type PostgresDB struct {
	Username string
	Password string
	Host     string
	Port     int
	Name     string
	SSLMode  string
}

type SMTP struct {
	FromAddress string
	Host        string
	Port        int
	Username    string
	Password    string
}

type GoogleOAuth struct {
	GoogleKey      string
	GoogleSecret   string
	GoogleCallback string
}

type GithubOAuth struct {
	GithubKey      string
	GithubSecret   string
	GithubCallback string
}

type OAuthProvider struct {
	Google *GoogleOAuth
	Github *GithubOAuth
}

type GoogleGenerativeAI struct {
	APIKEY    string
	Model     string
	FastModel string
	LiveModel string
}

type EnvironmentVariables struct {
	Port                  string
	JWTSecret             string
	JWTMaxAge             time.Duration
	RefreshJWTSecret      string
	RefreshJWTMaxAge      time.Duration
	CookieSecret          string
	SessionSecret         string
	SessionMaxAge         int
	ProductionEnvironment bool
	AuthRedirectUrl       string
	ClientDomain          string
	EmailQueue            string
	SMSQueue              string
	PushNotificationQueue string
	ProjectName           string
	PostgresDB            *PostgresDB
	SMTP                  *SMTP
	OAuthProvider         *OAuthProvider
	Gemini                *GoogleGenerativeAI
}

func loadEnv() {
	rootPath := file.GetRootPath()
	err := godotenv.Load(rootPath + `/.env`)

	if err != nil {
		log.Fatalf("Error loading .env file")
	}
}

func LoadEnvironment() *EnvironmentVariables {
	loadEnv()
	return &EnvironmentVariables{
		Port:                  getEnv("PORT", ":5000"),
		JWTSecret:             getEnvOrError("JWT_SECRET"),
		JWTMaxAge:             time.Second * time.Duration(getEnvAsInt("JWT_MAX_AGE", 60*15)),
		RefreshJWTSecret:      getEnvOrError("REFRESH_JWT_SECRET"),
		RefreshJWTMaxAge:      time.Second * time.Duration(getEnvAsInt("REFRESH_JWT_MAX_AGE", 60*60*24*31)),
		CookieSecret:          getEnvOrError("COOKIE_SECRET"),
		SessionSecret:         getEnvOrError("SESSIONS_SECRET"),
		SessionMaxAge:         getEnvAsInt("SESSION_MAX_AGE", 86400*300),
		ProductionEnvironment: getEnvAsBool("PRODUCTION_ENVIRONMENT", false),
		ClientDomain:          getEnv("CLIENT_DOMAIN", "localhost"),
		EmailQueue:            getEnv("EMAIL_QUEUE_NAME", "email"),
		SMSQueue:              getEnv("SMS_QUEUE_NAME", "sms"),
		PushNotificationQueue: getEnv("PUSH_NOTIFICATION_QUEUE_NAME", "push_notification"),
		ProjectName:           getEnv("PROJECT_NAME", "rider"),
		PostgresDB: &PostgresDB{
			Username: getEnv("POSTGRES_USER", "postgres"),
			Password: getEnvOrError("POSTGRES_PASSWORD"),
			Host:     getEnv("POSTGRES_HOST", "127.0.0.1"),
			Port:     getEnvAsInt("POSTGRES_PORT", 5432),
			Name:     getEnvOrError("POSTGRES_DB"),
			SSLMode:  getEnv("POSTGRES_SSL", "false"),
		},

		SMTP: &SMTP{
			FromAddress: getEnvOrError("SMTP_FROM_ADDRESS"),
			Host:        getEnvOrError("SMTP_HOST"),
			Port:        getEnvAsInt("SMTP_PORT", 587),
			Username:    getEnvOrError("SMTP_USERNAME"),
			Password:    getEnvOrError("SMTP_PASSWORD"),
		},

		OAuthProvider: &OAuthProvider{
			Google: &GoogleOAuth{
				GoogleKey:      getEnvOrError("GOOGLE_CLIENT_ID"),
				GoogleSecret:   getEnvOrError("GOOGLE_CLIENT_SECRET"),
				GoogleCallback: getEnvOrError("GOOGLE_CALLBACK_URL"),
			},
			Github: &GithubOAuth{
				GithubKey:      getEnvOrError("GITHUB_CLIENT_ID"),
				GithubSecret:   getEnvOrError("GITHUB_CLIENT_SECRET"),
				GithubCallback: getEnvOrError("GITHUB_CALLBACK_URL"),
			},
		},

		Gemini: &GoogleGenerativeAI{
			APIKEY:    getEnvOrError("GOOGLE_API_KEY"),
			Model:     getEnvOrError("GEMINI_MODEL"),
			FastModel: getEnvOrError("GEMINI_FAST_MODEL"),
			LiveModel: getEnvOrError("GEMINI_LIVE_MODEL"),
		},
	}
}

func getEnvOrError(key string) string {
	value, exists := os.LookupEnv(key)
	if exists {
		return value
	}
	panic("Environment variable " + key + " not set")
}

func getEnv(key string, fallback string) string {
	value, exists := os.LookupEnv(key)
	if exists {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	value, exist := os.LookupEnv(key)
	if exist {
		valueInt, err := strconv.Atoi(value)
		if err != nil {
			log.Panicf("Environment variable \"%v\" not set properly", key)
		}
		return valueInt
	}
	return fallback
}

func getEnvAsBool(key string, fallback bool) bool {
	value, exist := os.LookupEnv(key)
	if exist {
		valueBool, err := strconv.ParseBool(value)
		if err != nil {
			log.Panicf("Environment variable \"%v\" not set properly", key)
		}
		return valueBool
	}
	return fallback
}
