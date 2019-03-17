package main

import (
	"../eioh"
	"log"
	// "fmt"
)

var downCmd = &Command{
	Name:    "down",
	Usage:   "",
	Summary: "Roll back the version by 1",
	Help:    `down extended help here...`,
	Run:     downRun,
}

func downRun(cmd *Command, args ...string) {

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

	// if previous < 0 {
	// 	previous = current
	// }
	// fmt.Println(previous)

	if err = eioh.RunMigrations(conf, conf.MigrationsDir, previous); err != nil {
		log.Fatal(err)
	}
}