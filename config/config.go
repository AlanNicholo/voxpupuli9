package config

type Config struct{
	Api ApiConfig
	Database DataBase
	Common Common
	Instagram Network
	Twitter Network
}

type Network struct{
	Login string
	Password string
	Api ApiConfig
}

type DataBase struct{
	Name string
	User string
	Pass string
	Host string
	Driver string
	Port string
}

type Common struct{
	HistoryFile string
	ConfigurationFile string
}

type ApiConfig struct{
	Host string
	Port string
	Key string
	SKey string
	Verbose string
}
