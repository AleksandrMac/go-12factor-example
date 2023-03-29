package config

import (
	"encoding/json"
	"fmt"
	"go-example/internal/log"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var viperInstance = viper.New()
var Default Config

func init() {
	log.Debug("INIT CONFIG")
}

// Config struct
type Config struct {
	Server struct {
		Port uint
		Host string
	}
	Database struct {
		URL  string
		Pool struct {
			Max uint
		}
	}
	Log zap.Config
}

func (d Config) String() string {
	b, _ := json.Marshal(d)
	return string(b)
}

// Parse get all config support in app
func Parse() Config {
	if err := viperInstance.Unmarshal(&Default, viper.DecodeHook(mapstructure.TextUnmarshallerHookFunc())); err != nil {
		log.Fatal(
			fmt.Sprintf("Fail to read configuration: %s", err.Error()))
	}
	return Default
}

// Viper instance
func Viper() *viper.Viper {
	return viperInstance
}
