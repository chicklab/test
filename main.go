package main

import (
	"fmt"
	"./migration"
)

func main() {

	conf, err := migration.NewDBConf("./", "development")
	if err != nil {
		fmt.Sprintf("%s.getconf", err)
	}

	db, err := migration.OpenDBFromDBConf(conf)
	if err != nil {
		fmt.Sprintf("%s.dbinit", err)
	}
	defer db.Close()
	// RunMigrations(conf, "./", 0)

	err = migration.RunMigrations(conf, "./history", 20190317)
    // defer rows.Close()
    if err != nil {
        panic(err.Error())
    }

	// fmt.Println(version)
 
    // for rows.Next() {
    //     var id int
    //     var title string
    //     if err := rows.Scan(&id, &title); err != nil {
    //         panic(err.Error())
    //     }
    //     fmt.Println(id, title)
    // }
}