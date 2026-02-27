package config

import (
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Port               int           `env:"PORT" envDefault:"8080"`
	Env                string        `env:"ENV" envDefault:"development"`
	DatabaseURL        string        `env:"DATABASE_URL,required"`
	JWTSecret          string        `env:"JWT_SECRET,required"`
	JWTAccessTTL       time.Duration `env:"JWT_ACCESS_TTL" envDefault:"15m"`
	JWTRefreshTTL      time.Duration `env:"JWT_REFRESH_TTL" envDefault:"168h"`
	FBGraphAPIURL      string        `env:"FB_GRAPH_API_URL" envDefault:"https://graph.facebook.com/v21.0"`
	CORSAllowedOrigins []string      `env:"CORS_ALLOWED_ORIGINS" envSeparator:"," envDefault:"http://localhost:5173"`
	RateLimitRPS       int           `env:"RATE_LIMIT_RPS" envDefault:"100"`

	// Cloudflare (Custom Domains)
	CFAPIToken      string `env:"CF_API_TOKEN"`
	CFAccountID     string `env:"CF_ACCOUNT_ID"`
	CFZoneID        string `env:"CF_ZONE_ID"`
	CFKVNamespaceID string `env:"CF_KV_NAMESPACE_ID"`
	CFCNAMETarget   string `env:"CF_CNAME_TARGET" envDefault:"customer.pixlinks.xyz"`

	// S3/R2 Storage
	S3Endpoint  string `env:"S3_ENDPOINT"`
	S3Bucket    string `env:"S3_BUCKET"`
	S3AccessKey string `env:"S3_ACCESS_KEY"`
	S3SecretKey string `env:"S3_SECRET_KEY"`
	S3PublicURL string `env:"S3_PUBLIC_URL"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
