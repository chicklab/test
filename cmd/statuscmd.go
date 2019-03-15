package main

import (
	"../eioh"
	// "log"
	// "fmt"
)

var statusCmd = &Command{
	Name:    "status",
	Usage:   "",
	Summary: "",
	Help:    `status extended help here...`,
	Run:     statusRun,
}

func statusRun(cmd *Command, args ...string) {

	// conf, err := dbConfFromFlags()
	conf, err := eioh.NewDBConf("../", "development")
	if err != nil {
		// log.Fatal(err)
	}

	// target, err := eioh.GetMostRecentDBVersion(conf.MigrationsDir)
	// if err != nil {
	// 	// log.Fatal(err)
	// }
	if err := eioh.StatusMigration(conf); err != nil {

	}
	// if err := eioh.RunMigrations(conf, conf.MigrationsDir, target); err != nil {
	// 	// log.Fatal(err)
	// }
}