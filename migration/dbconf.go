package migration

import (
	"database/sql"
	"os"
	"path/filepath"
	"fmt"

	"github.com/kylelemons/go-gypsy/yaml"
	_ "github.com/go-sql-driver/mysql"
)

type DBDriver struct {
	Name    string
	OpenStr string
	// Import  string
	Base SqlBase
}

type DBConf struct {
	MigrationsDir string
	Env           string
	Driver        DBDriver
}

func NewDBConf(p, env string) (*DBConf, error) {

	cfgFile := filepath.Join(p, "dbconf.yml")

	f, err := yaml.ReadFile(cfgFile)
	if err != nil {
		return nil, err
	}

	drv, err := f.Get(fmt.Sprintf("%s.driver", env))
	if err != nil {
		return nil, err
	}

	drv = os.ExpandEnv(drv)

	open, err := f.Get(fmt.Sprintf("%s.open", env))
	if err != nil {
		return nil, err
	}
	open = os.ExpandEnv(open)

	d := newDBDriver(drv, open)

	// if imprt, err := f.Get(fmt.Sprintf("%s.import", env)); err == nil {
	// 	d.Import = imprt
	// }

	// if base, err := f.Get(fmt.Sprintf("%s.base", env)); err == nil {
	// 	d.Base = baseByName(base)
	// }

	return &DBConf{
		MigrationsDir: filepath.Join(p, "migrations"),
		Env:           env,
		Driver:        d,
	}, nil
}

func newDBDriver(name, open string) DBDriver {

	d := DBDriver{
		Name:    name,
		OpenStr: open,
	}

	switch name {

	case "mysql":
		// d.Import = "github.com/go-sql-driver/mysql"
		d.Base = &MySqlBase{}

	}
	return d
}

func OpenDBFromDBConf(conf *DBConf) (*sql.DB, error) {
	db, err := sql.Open(conf.Driver.Name, conf.Driver.OpenStr)
	if err != nil {
		return nil, err
	}
	return db, nil
}