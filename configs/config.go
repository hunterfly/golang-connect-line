package configs

import (
	"log"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// DefaultSystemPrompt is the default system prompt for LM Studio
const DefaultSystemPrompt = "You are a helpful assistant responding via LINE messaging"

// Config struct
type Config struct {
	App      `mapstructure:"app"`
	Postgres `mapstructure:"postgres"`
	Line     `mapstructure:"line"`
	LMStudio `mapstructure:"lmstudio"`
	Session  `mapstructure:"session"`
}

// App struct
type App struct {
	Debug bool   `mapstructure:"debug"`
	Env   string `mapstructure:"env"`
	Port  string `mapstructure:"port"`
}

// Postgres struct
type Postgres struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DbName   string `mapstructure:"database"`
	SSLMode  bool   `mapstructure:"sslmode"`
}

// Line struct
type Line struct {
	ChannelSecret string `mapstructure:"channel_secret"`
	ChannelToken  string `mapstructure:"channel_token"`
}

// LMStudio struct - Configuration for LM Studio client
type LMStudio struct {
	BaseURL      string `mapstructure:"base_url"`
	Model        string `mapstructure:"model"`
	Timeout      int    `mapstructure:"timeout"`
	SystemPrompt string `mapstructure:"system_prompt"`
}

// Session struct - Configuration for user session management
type Session struct {
	Timeout  int `mapstructure:"timeout"`
	MaxTurns int `mapstructure:"max_turns"`
}

var config Config

// InitViper func
func InitViper(path, env string) {
	getConfig(path, env)
}

// GetViper func
func GetViper() *Config {
	return &config
}

func getConfig(path, env string) {
	viper.SetConfigName("config")
	viper.AddConfigPath(path)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Println("Config file has changed: ", e.Name)
	})
	err = viper.Unmarshal(&config)
	if err != nil {
		log.Fatalln(err)
	}
}
