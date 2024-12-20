package commons

import (
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ConfigManager struct {
	Manager *viper.Viper
}

func NewConfigManager() ConfigManager {
	return ConfigManager{
		Manager: viper.New(),
	}
}

func (cm ConfigManager) LoadFlags(cmd *cobra.Command) {
	err := cm.Manager.BindPFlags(cmd.Flags())
	cobra.CheckErr(err)
}

// LoadEnv from embedded .env file
func (cm ConfigManager) LoadEnv(envString string) {
	slog.Debug("env: loading")
	cm.Manager.AutomaticEnv()

	currentEnv := map[string]bool{}
	rawEnv := os.Environ()
	for _, rawEnvLine := range rawEnv {
		key := strings.Split(rawEnvLine, "=")[0]
		currentEnv[key] = true
	}

	parse, err := godotenv.Unmarshal(envString)
	cobra.CheckErr(err)

	for k, v := range parse {
		if !currentEnv[k] {
			slog.Debug("env: setting env", "key", k, "value", v)
			err := os.Setenv(k, v)
			cobra.CheckErr(err)
		} else {
			slog.Debug("env: skipping env", "key", k)
		}
	}

	slog.Debug("env: loaded")
}

func (cm ConfigManager) DebugToSlog(logger *slog.Logger, level slog.Level) {
	sw := NewSlogWriter(logger, slog.LevelDebug)
	cm.Manager.DebugTo(sw)
}
