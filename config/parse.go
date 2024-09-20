package config

import (
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"os"
)

func ParseConfig() (Config, error){
	conf := Config{}
	err := godotenv.Load(".env")
	if err != nil{
		return Config{}, errors.New("Please rename/create .env file on root path.")
	}


	conf.Instagram.Login = os.Getenv("INSTAGRAM_LOGIN")
	conf.Instagram.Password = os.Getenv("INSTAGRAM_PASSWORD")

	conf.Api.Host = os.Getenv("API_HOST")
	conf.Api.Port = os.Getenv("API_PORT")
	conf.Api.Verbose = os.Getenv("API_VERBOSE")

	conf.Common.HistoryFile = os.Getenv("OPERATIVE_HISTORY")

	conf.Database.Driver = os.Getenv("DB_DRIVER")
	conf.Database.Name = os.Getenv("DB_NAME")
	conf.Database.Host = os.Getenv("DB_HOST")
	conf.Database.User = os.Getenv("DB_USER")
	conf.Database.Pass = os.Getenv("DB_PASS")

	return conf, nil
}
