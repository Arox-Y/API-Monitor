// Package config loads settings from environment variables.
//
// An "environment variable" is a value that lives outside your code, in the
// operating system. We use them for anything that changes between machines
// (database address) or must stay secret (passwords, keys).
//
// The golden rule: NEVER write a password or secret key directly in a .go
// file. Code goes on GitHub. Secrets must not.
package config

import (
	"fmt"
	"os"      // reading environment variables
	"strconv" // converting text to numbers

	"github.com/joho/godotenv"
)

// Config holds every setting the application needs.
//
// A "struct" in Go is a container that groups related values together, a bit
// like a form with labelled boxes. Config is one form with eight boxes.
type Config struct {
	DatabaseURL       string
	Port              string
	Env               string
	JWTSecret         string
	CORSOrigin        string
	WorkerTickSeconds int
	WorkerConcurrency int
}

// IsDevelopment reports whether we are running on a developer machine rather
// than a real server. We use it later to decide things like whether the login
// cookie requires HTTPS.
//
// This is a "method": a function attached to a type. The (c Config) part says
// "this function belongs to Config", so you call it as cfg.IsDevelopment().
func (c Config) IsDevelopment() bool {
	return c.Env == "development"
}

// Load reads all settings and returns them, or an error explaining what is
// missing.
//
// Go functions can return two values. The convention is (result, error).
// The caller must check the error — Go has no exceptions, so an ignored error
// is simply a bug waiting to happen.
func Load() (Config, error) {
	// Try to read a .env file so developers do not have to set variables by
	// hand. We look in the current folder and one level up, because the app
	// might be started from backend/ or from the project root.
	//
	// Each path is tried SEPARATELY, and we stop at the first one that works.
	//
	// Why not godotenv.Load(".env", "../.env")? Because that helper returns as
	// soon as ANY file is missing, so a missing ".env" would stop it ever
	// looking at "../.env". (Verified in godotenv v1.5.1: its loop does
	// `if err != nil { return }`.) That exact bug stopped this app booting the
	// first time it ran.
	for _, path := range []string{".env", "../.env"} {
		if err := godotenv.Load(path); err == nil {
			break
		}
	}
	// A missing .env is not an error: inside Docker there is no .env file at
	// all, because Compose supplies the variables directly. The validation
	// below is what actually decides whether we have what we need.

	cfg := Config{
		DatabaseURL:       getEnv("DATABASE_URL", ""),
		Port:              getEnv("PORT", "8080"),
		Env:               getEnv("ENV", "development"),
		JWTSecret:         getEnv("JWT_SECRET", ""),
		CORSOrigin:        getEnv("CORS_ORIGIN", "http://localhost:3000"),
		WorkerTickSeconds: getEnvInt("WORKER_TICK_SECONDS", 10),
		WorkerConcurrency: getEnvInt("WORKER_CONCURRENCY", 20),
	}

	// ---- Validation --------------------------------------------------
	//
	// We check the required settings NOW, at startup, rather than letting the
	// app run and crash later when it first touches the database.
	//
	// This idea is called "fail fast": if something is wrong, stop immediately
	// with a clear message. A server that starts successfully and then breaks
	// on a real user's request is much harder to debug.

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is not set (check your .env file)")
	}

	// JWT_SECRET is the key used to sign login tokens. If an attacker learns
	// it, they can forge a token for ANY user.
	//
	// Notice there is no fallback default. A default like "changeme" is
	// dangerous precisely because it works — the app would start fine, ship to
	// production, and anyone who has read the source could log in as anybody.
	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is not set (generate one and put it in .env)")
	}

	if len(cfg.JWTSecret) < 32 {
		return Config{}, fmt.Errorf(
			"JWT_SECRET is too short (%d characters, need at least 32) — a short key is easy to guess",
			len(cfg.JWTSecret),
		)
	}

	return cfg, nil
}

// getEnv returns the environment variable named key, or fallback if it is not
// set. Lowercase name = private to this package; other packages cannot call it.
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getEnvInt is the same, but converts the text into a whole number.
// Environment variables are ALWAYS text — "10" is the two characters 1 and 0,
// not the number ten — so it has to be converted.
func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	number, err := strconv.Atoi(value)
	if err != nil {
		// The value was not a valid number (someone typed "ten"). Rather than
		// crash, fall back to the default.
		return fallback
	}
	return number
}
