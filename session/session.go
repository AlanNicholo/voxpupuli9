package session

import (
	"github.com/chzyer/readline"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/pkg/errors"
)

type Session struct{
	Id int `json:"-" gorm:"primary_key:yes;column:id;AUTO_INCREMENT"`
	SessionName string `json:"session_name"`
	Information Information
	Connection Connection `json:"-" sql:"-"`
	Version string            `json:"version" sql:"-"`
	Targets []*Target   `json:"subjects" sql:"-"`
	Modules []Module          `json:"modules" sql:"-"`
	Prompt *readline.Config `json:"-" sql:"-"`
	Stream Stream `json:"-" sql:"-"`
}

type Information struct{
	ApiStatus bool `json:"api_status"`
	ModuleLaunched int `json:"module_launched"`
	Event int `json:"event"`
}

func (i *Information) AddEvent(){
	i.Event = i.Event + 1
	return
}

func (i *Information) AddModule(){
	i.ModuleLaunched = i.ModuleLaunched + 1
	return
}

func (i *Information) SetApi(s bool){
	i.ApiStatus = s
	return
}

func (Session) TableName() string{
	return "sessions"
}


func (s *Session) GetId() int{
	return s.Id
}

func (s *Session) ListType() []string{
	return []string{
		"enterprise",
		"ip_address",
		"website",
		"url",
		"person",
		"social_network",
	}
}

func (s *Session) GetTarget(id string) (*Target, error){
	for _, targ := range s.Targets{
		if targ.GetId() == id{
			return targ, nil
		}
	}
	return nil, errors.New("can't find selected target")
}
