package main

import (
	"fmt"
	"os"
	"../eioh"
	"time"
	"path/filepath"
	// "log"
)


var createCmd = &Command{
	Name:    "create",
	Usage:   "",
	Summary: "Create the scaffolding for a new migration",
	Help:    `create extended help here...`,
	Run:     createRun,
}

func createRun(cmd *Command, args ...string) {

	if len(args) < 1 {
		// log.Fatal("eioh create: migration name required")
		return
	}

	// conf, err := dbConfFromFlags()
	conf, err := eioh.NewDBConf("../", "development")
	if err != nil {
		// log.Fatal(err)
	}

	if err = os.MkdirAll(conf.MigrationsDir, 0777); err != nil {
		// log.Fatal(err)
	}



	n, err := eioh.CreateMigration(args[0], conf.MigrationsDir, time.Now())
	if err != nil {
		// log.Fatal(err)
	}

	fmt.Println("kita")

	a, e := filepath.Abs(n)
	if e != nil {
		// log.Fatal(e)
	}

	fmt.Println("eioh: created", a)
}