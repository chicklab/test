package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"../eioh"
	"text/template"

	// "log"
)

// global options. available to any subcommands.
var flagPath = flag.String("path", "db", "folder containing db info")
var flagEnv = flag.String("env", "development", "which DB environment to use")
// var flagPgSchema = flag.String("pgschema", "", "which postgres-schema to migrate (default = none)")

// helper to create a DBConf from the given flags
func dbConfFromFlags() (dbconf *eioh.DBConf, err error) {
	return eioh.NewDBConf(*flagPath, *flagEnv)
}



//Sync 環境をシンクさせる
//TableSettings生成
//オプションスラック通知をしない
//seed機能


var commands = []*Command{
	upCmd,
	// downCmd,
	// redoCmd,
	statusCmd,
	createCmd,
	// dbVersionCmd,
}

func main() {

	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 || args[0] == "-h" {
		flag.Usage()
		return
	}

	var cmd *Command
	name := args[0]
	for _, c := range commands {
		if strings.HasPrefix(c.Name, name) {
			cmd = c
			break
		}
	}

	if cmd == nil {
		fmt.Printf("error: unknown command %q\n", name)
		flag.Usage()
		os.Exit(1)
	}

	cmd.Exec(args[1:])

	// notification.slack()


	// conf, err := eioh.NewDBConf("./", "development")
	// if err != nil {
	// 	fmt.Sprintf("%s.getconf", err)
	// }
	// // fmt.Println(conf.MigrationsDir)

	// target, err := eioh.GetMostRecentDBVersion(conf.MigrationsDir)
	// // fmt.Println(target)
	// if err != nil {
	// 	fmt.Sprintf("%s.dbinit", err)
	// }


	// db, err := eioh.OpenDBFromDBConf(conf)
	// if err != nil {
	// 	fmt.Sprintf("%s.dbinit", err)
	// }
	// defer db.Close()
	// // RunMigrations(conf, "./", 0)

	// // err = migration.RunMigrations(conf, "./history", 20190317)
	// if err := eioh.RunMigrations(conf, conf.MigrationsDir, target); err != nil {
	// 	fmt.Sprintf("%s.dbinit", err)
	// }
}


func usage() {
	// fmt.Print(usagePrefix)
	// flag.PrintDefaults()
	// usageTmpl.Execute(os.Stdout, commands)
}

var usagePrefix = `
eioh is a database migration management system.

Usage:
    eioh [options] <subcommand> [subcommand options]

Options:
`
var usageTmpl = template.Must(template.New("usage").Parse(
	`
Commands:{{range .}}
    {{.Name | printf "%-10s"}} {{.Summary}}{{end}}
`))

