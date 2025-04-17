package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	LiveKit struct {
		URL       string `mapstructure:"url"`
		APIKey    string `mapstructure:"api_key"`
		APISecret string `mapstructure:"api_secret"`
		RoomName  string `mapstructure:"room_name"`
		Identity  string `mapstructure:"identity"`
	} `mapstructure:"livekit"`

	Health      HealthConfig  `mapstructure:"health"`
	Monitor     MonitorConfig `mapstructure:"monitor"`
	RTSPStreams []RTSPStream  `mapstructure:"rtsp_streams"`
	TsDump      TsConfig      `mapstructure:"ts"`
}

type RTSPStream struct {
	Name string `mapstructure:"name"`
	URL  string `mapstructure:"url"`
}

type HealthConfig struct {
	Parallel int `mapstructure:"parallel"`
	ChkSec   int `mapstructure:"chk_sec"`
}

type MonitorConfig struct {
	Parallel    int `mapstructure:"parallel"`
	DurationSec int `mapstructure:"duration_sec"`
}

type TsConfig struct {
	Duration int `mapstructure:"duration"`
}

var Cfg Config

func LoadConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("설정 파일 로드 실패: %v", err)
	}

	if err := viper.Unmarshal(&Cfg); err != nil {
		log.Fatalf("설정 파일 파싱 실패: %v", err)
	}
}
