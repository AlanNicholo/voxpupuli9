package session

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/graniet/go-pretty/table"
	"github.com/segmentio/ksuid"
)

type Target struct {
	Id           int                         `json:"-" gorm:"primary_key:yes;column:id;AUTO_INCREMENT"`
	SessionId    int                         `json:"-" gorm:"column:session_id"`
	TargetId     string                      `json:"id" gorm:"column:target_id"`
	Sess         *Session                    `json:"-" gorm:"-"`
	Name         string                      `json:"name" gorm:"column:target_name"`
	Type         string                      `json:"type" gorm:"column:target_type"`
	Results      map[string][]*TargetResults `sql:"-" json:"results"`
	TargetLinked []Linking                   `json:"target_linked" sql:"-"`
	Notes        []Note                      `json:"notes" sql:"-"`
	Tags         []Tags                      `json:"tags"  sql:"-"`
}

type Linking struct {
	LinkingId       int    `json:"-" gorm:"primary_key:yes;column:id;AUTO_INCREMENT"`
	SessionId       int    `json:"session_id" gorm:"column:session_id"`
	TargetBase      string `json:"target_base" gorm:"column:target_base"`
	TargetId        string `json:"target_id" gorm:"column:target_id"`
	TargetName      string `json:"target_name" gorm:"column:target_name"`
	TargetType      string `json:"target_type" gorm:"column:target_type"`
	TargetResultId  string `json:"target_result_id" gorm:"column:target_result_id"`
	TargetResultsTo string `json:"target_results_to"`
}

func (Linking) TableName() string {
	return "target_links"
}

func (target *Target) GetId() string {
	return target.TargetId
}

func (target *Target) GetName() string {
	return target.Name
}

func (target *Target) GetType() string {
	return target.Type
}

func (target *Target) GetResults() map[string][]*TargetResults {
	return target.Results
}

func (target *Target) GetLinked() []Linking {
	return target.TargetLinked
}

func (target *Target) PushLinked(t Linking) {
	target.TargetLinked = append(target.TargetLinked, t)
}

func (target *Target) CheckType() bool {
	for _, sType := range target.Sess.ListType() {
		if sType == target.GetType() {
			return true
		}
	}
	return false
}

func (target *Target) Link(target2 Linking) {
	if target.GetId() == target2.TargetId {
		return
	}
	t2, err := target.Sess.GetTarget(target2.TargetId)
	if err != nil {
		target.Sess.Stream.Error(err.Error())
		return
	}

	/*for _, trg := range target.TargetLinked {
		if trg.TargetId == t2.GetId() {
			return
		}
	}*/

	target2.TargetType = t2.GetType()
	target2.TargetName = t2.GetName()
	target2.TargetBase = target.GetId()
	target2.SessionId = target.Sess.GetId()
	target.PushLinked(target2)

	t2.PushLinked(Linking{
		SessionId:      target.Sess.GetId(),
		TargetBase:     t2.GetId(),
		TargetName:     target.GetName(),
		TargetId:       target.GetId(),
		TargetType:     target.GetType(),
		TargetResultId: target2.TargetResultId,
	})
	target.Sess.Connection.ORM.Create(&target2)
	target.Sess.Connection.ORM.Create(&Linking{
		SessionId:      target.Sess.GetId(),
		TargetBase:     t2.GetId(),
		TargetName:     target.GetName(),
		TargetId:       target.GetId(),
		TargetType:     target.GetType(),
		TargetResultId: target2.TargetResultId,
	})
}

func (target *Target) GetResult(id string) (*TargetResults, error) {
	for _, module := range target.Results {
		for _, result := range module {
			if result.ResultId == id {
				return result, nil
			}
		}
	}
	return &TargetResults{}, errors.New("Result as been not found.")
}

func (target *Target) Linked() {
	t := target.Sess.Stream.GenerateTable()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{
		"TARGET",
		"NAME",
		"TYPE",
		"RESULT ID",
	})
	for _, element := range target.TargetLinked {
		t.AppendRow(table.Row{
			element.TargetId,
			element.TargetName,
			element.TargetType,
			element.TargetResultId,
		})
	}
	target.Sess.Stream.Render(t)
}

func (target *Target) GetSeparator() string {
	return base64.StdEncoding.EncodeToString([]byte(";operativeframework;"))[0:5]
}

func (target *Target) ResultExist(result TargetResults) bool {
	for module, results := range target.Results {
		if module == result.ModuleName {
			for _, r := range results {
				if r.Header == result.Header {
					if r.Value == result.Value {
						return true
					}
				}
			}
		}
	}
	return false
}

func (target *Target) Save(module Module, result TargetResults) bool {
	result.ResultId = "R_" + ksuid.New().String()
	result.TargetId = target.GetId()
	result.SessionId = target.Sess.GetId()
	result.ModuleName = module.Name()
	result.CreatedAt = time.Now()

	if !target.ResultExist(result) {
		target.Results[module.Name()] = append(target.Results[module.Name()], &result)
		target.Sess.Connection.ORM.Create(&result).Table("target_results")
		targets, err := target.Sess.FindLinked(module.Name(), result)
		if err == nil {
			for _, id := range targets {
				target.Link(Linking{
					TargetId:       id,
					TargetResultId: result.ResultId,
				})
			}
		}
	}
	module.SetExport(result)
	return true
}

func (target *Target) GetModuleResults(name string) ([]*TargetResults, error) {

	for moduleName, results := range target.Results {
		if moduleName == name {
			return results, nil
		}
	}
	return []*TargetResults{}, errors.New("result not found for this module")
}

func (target *Target) GetFormatedResults(module string) ([]map[string]string, error) {
	var formated []map[string]string
	results, err := target.GetModuleResults(module)
	if err != nil {
		return formated, err
	}

	for _, result := range results {
		resultMap := make(map[string]string)
		separator := target.GetSeparator()
		header := strings.Split(result.Header, separator)
		res := strings.Split(result.Value, separator)
		for k, r := range res {
			resultKey := strings.Replace(strings.ToLower(header[k]), " ", "_", -1)
			if len(header) < len(res) && k > len(header) {
				resultMap[ksuid.New().String()] = r
			} else {
				resultMap[resultKey] = r
			}
		}
		formated = append(formated, resultMap)
	}
	return formated, nil
}

func (target *Target) GetLastResults(module string) {
	if results, ok := target.Results[module]; ok {
		fmt.Println(results)
	}
}

func (s *Session) GetResultsAfter(results []*TargetResults, afterTime time.Time) []*TargetResults {
	var scopes []*TargetResults
	for _, result := range results {
		if result.CreatedAt.After(afterTime) {
			scopes = append(scopes, result)
		}
	}

	return scopes
}

func (s *Session) GetResultsBefore(results []*TargetResults, beforeTime time.Time) []*TargetResults {
	var scopes []*TargetResults
	for _, result := range results {
		if result.CreatedAt.Before(beforeTime) {
			scopes = append(scopes, result)
		}
	}

	return scopes
}
