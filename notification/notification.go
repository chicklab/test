package notification

import (
	"fmt"
	"bytes"
    "net/http"
	"path/filepath"
    "github.com/kylelemons/go-gypsy/yaml"
	// "log"
)


func Push(text string) {

    conf, err := NewNotifiactionConf("../")

    name := "Go"
    channel := "random"

    jsonStr := `{"channel":"` + channel + `","username":"` + name + `","text":"` + text + `"}`

    req, err := http.NewRequest(
        "POST",
        conf.Url,
        bytes.NewBuffer([]byte(jsonStr)),
    )

    if err != nil {
        fmt.Print(err)
    }

    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        fmt.Print(err)
    }

    // fmt.Print(resp)
    defer resp.Body.Close()
}


type NotificationConf struct {
	Url string
	Env string
}

func NewNotifiactionConf(p string) (*NotificationConf, error) {

    env := "notification"

	cfgFile := filepath.Join(p, "conf.yml")

	f, err := yaml.ReadFile(cfgFile)
	if err != nil {
		return nil, err
	}

	url, err := f.Get(fmt.Sprintf("%s.url", env))
	if err != nil {
		return nil, err
	}

	return &NotificationConf{
        Url: url,
		Env: env,
	}, nil
}