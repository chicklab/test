package main

import (
	"../eioh"
	"log"
)

var redoCmd = &Command{
	Name:    "redo",
	Usage:   "",
	Summary: "Re-run the latest migration",
	Help:    `redo extended help here...`,
	Run:     redoRun,
}

func redoRun(cmd *Command, args ...string) {
	// conf, err := dbConfFromFlags()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	conf, err := eioh.NewDBConf("../", "development")
	if err != nil {
		// log.Fatal(err)
	}

	current, err := eioh.GetDBVersion(conf)
	if err != nil {
		log.Fatal(err)
	}

	previous, err := eioh.GetPreviousDBVersion(conf.MigrationsDir, current)
	if err != nil {
		log.Fatal(err)
	}

	if err := eioh.RunMigrations(conf, conf.MigrationsDir, previous); err != nil {
		log.Fatal(err)
	}

	if err := eioh.RunMigrations(conf, conf.MigrationsDir, current); err != nil {
		log.Fatal(err)
	}
}