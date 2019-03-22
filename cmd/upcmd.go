package main

import (
	"../eioh"
	// "log"
)

var upCmd = &Command{
	Name:    "up",
	Usage:   "",
	Summary: "Migrate the DB to the most recent version available",
	Help:    `up extended help here...`,
	Run:     upRun,
}

func upRun(cmd *Command, args ...string) {

	conf, err := eioh.NewDBConf("../", "development")
	if err != nil {
		// log.Fatal(err)
	}

	target, err := eioh.GetMostRecentDBVersion(conf.MigrationsDir)
	if err != nil {
		// log.Fatal(err)
	}

	if err := eioh.RunMigrations(conf, conf.MigrationsDir, target); err != nil {
		// log.Fatal(err)
	}
}
