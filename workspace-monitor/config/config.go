package config

import (
	"os"

	"github.com/spf13/viper"
)

type Config struct {
  PersistentDB bool     `mapstructure:"persistent-db"`
  DBFile       string   `mapstructure:"db-file"`
  SaveLayout   bool     `mapstructure:"save-layout"`
  Workspaces   []string `mapstructure:"workspaces"`
}

func New() (Config, error) {
  conf := Config{}

  homeDirPath, _ := os.UserHomeDir()
  var defaultConfigLocations [3]string = [3]string{
    homeDirPath+"/.config/i3wm-hidden-workspaces/",
    ".",
    homeDirPath+"/i3/hidden-workspace/",
  }

  for _,p := range defaultConfigLocations {
    viper.AddConfigPath(p)
  }

  viper.SetConfigName("config")
  viper.SetConfigType("yaml")

  viper.SetDefault("persistent-db", true)
  viper.SetDefault("db-file", "/tmp/i3wm-hidden-workspaces")
  viper.SetDefault("save-layout", false)

  err := viper.ReadInConfig()
  if err != nil {
    return conf, err
  }

  err = viper.Unmarshal(&conf)
  if err != nil {
    return conf, err
  }

  return conf, nil
}
