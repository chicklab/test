package migration

import (
	"bufio"
	"bytes"
	"database/sql"
	"io"
	"log"
	"strings"
	"text/template"
	"path/filepath"
	"os"
	"time"
	"fmt"
	"errors"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
)

//マイグレーション+スラック通知

const sqlCmdPrefix = "-- +goose "

// Checks the line to see if the line has a statement-ending semicolon
// or if the line contains a double-dash comment.
func endsWithSemicolon(line string) bool {

	prev := ""
	scanner := bufio.NewScanner(strings.NewReader(line))
	scanner.Split(bufio.ScanWords)

	for scanner.Scan() {
		word := scanner.Text()
		if strings.HasPrefix(word, "--") {
			break
		}
		prev = word
	}

	return strings.HasSuffix(prev, ";")
}

// Split the given sql script into individual statements.
//
// The base case is to simply split on semicolons, as these
// naturally terminate a statement.
//
// However, more complex cases like pl/pgsql can have semicolons
// within a statement. For these cases, we provide the explicit annotations
// 'StatementBegin' and 'StatementEnd' to allow the script to
// tell us to ignore semicolons.
func splitSQLStatements(r io.Reader, direction bool) (stmts []string) {

	var buf bytes.Buffer
	scanner := bufio.NewScanner(r)

	// track the count of each section
	// so we can diagnose scripts with no annotations
	upSections := 0
	downSections := 0

	statementEnded := false
	ignoreSemicolons := false
	directionIsActive := false

	for scanner.Scan() {

		line := scanner.Text()

		// handle any goose-specific commands
		if strings.HasPrefix(line, sqlCmdPrefix) {
			cmd := strings.TrimSpace(line[len(sqlCmdPrefix):])
			switch cmd {
			case "Up":
				directionIsActive = (direction == true)
				upSections++
				break

			case "Down":
				directionIsActive = (direction == false)
				downSections++
				break

			case "StatementBegin":
				if directionIsActive {
					ignoreSemicolons = true
				}
				break

			case "StatementEnd":
				if directionIsActive {
					statementEnded = (ignoreSemicolons == true)
					ignoreSemicolons = false
				}
				break
			}
		}

		if !directionIsActive {
			continue
		}

		if _, err := buf.WriteString(line + "\n"); err != nil {
			log.Fatalf("io err: %v", err)
		}

		// Wrap up the two supported cases: 1) basic with semicolon; 2) psql statement
		// Lines that end with semicolon that are in a statement block
		// do not conclude statement.
		if (!ignoreSemicolons && endsWithSemicolon(line)) || statementEnded {
			statementEnded = false
			stmts = append(stmts, buf.String())
			buf.Reset()
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("scanning migration: %v", err)
	}

	// diagnose likely migration script errors
	if ignoreSemicolons {
		log.Println("WARNING: saw '-- +goose StatementBegin' with no matching '-- +goose StatementEnd'")
	}

	if bufferRemaining := strings.TrimSpace(buf.String()); len(bufferRemaining) > 0 {
		log.Printf("WARNING: Unexpected unfinished SQL query: %s. Missing a semicolon?\n", bufferRemaining)
	}

	if upSections == 0 && downSections == 0 {
		log.Fatalf(`ERROR: no Up/Down annotations found, so no statements were executed.
			See https://bitbucket.org/liamstask/goose/overview for details.`)
	}

	return
}


func RunMigrations(conf *DBConf, migrationsDir string, target int64) error {

	db, err := OpenDBFromDBConf(conf)
	if err != nil {
		return err
	}
	defer db.Close()

	return RunMigrationsOnDb(conf, migrationsDir, target, db)
}

// Runs migration on a specific database instance.
func RunMigrationsOnDb(conf *DBConf, migrationsDir string, target int64, db *sql.DB) (err error) {
	current, err := EnsureDBVersion(conf, db)
	if err != nil {
		return err
	}
	fmt.Println(current)

	migrations, err := CollectMigrations(migrationsDir, current, target)
	fmt.Println(migrations);
	fmt.Println(err);
	if err != nil {
		return err
	}


	// if len(migrations) == 0 {
	// 	fmt.Printf("goose: no migrations to run. current version: %d\n", current)
	// 	return nil
	// }

	// ms := migrationSorter(migrations)
	// direction := current < target
	// ms.Sort(direction)

	// fmt.Printf("goose: migrating db environment '%v', current version: %d, target: %d\n",
	// 	conf.Env, current, target)

	// for _, m := range ms {

	// 	switch filepath.Ext(m.Source) {
	// 	case ".sql":
	// 		// err = runSQLMigration(conf, db, m.Source, m.Version, direction)
	// 	}

	// 	if err != nil {
	// 		return errors.New(fmt.Sprintf("FAIL %v, quitting migration", err))
	// 	}

	// 	fmt.Println("OK   ", filepath.Base(m.Source))
	// }

	return nil
}

func NumericComponent(name string) (int64, error) {

	base := filepath.Base(name)

	if ext := filepath.Ext(base); ext != ".sql" {
		return 0, errors.New("ファイルの拡張子を確認してください。")
	}

	idx := strings.Index(base, "_")
	if idx < 0 {
		return 0, errors.New("versionId_ファイル名.sqlという形式にしてください。")
	}
	
	n, e := strconv.ParseInt(base[:idx], 10, 64)
	if e == nil && n <= 0 {
		return 0, errors.New("IDは0以上を使用してください。")
	}

	n, err := time.Parse(RFC3339, n)
    if err == nil {
        return 0, errors.New("日付形式で入力してください。")
    }

	return n, e
}


func CollectMigrations(dirpath string, current, target int64) (m []*Migration, err error) {

	filepath.Walk(dirpath, func(name string, info os.FileInfo, err error) error {

		if v, e := NumericComponent(name); e == nil {
			
			for _, g := range m {
				if v == g.Version {
					log.Fatalf("more than one file specifies the migration for version %d (%s and %s)",
						v, g.Source, filepath.Join(dirpath, name))
				}
			}
			
			if versionFilter(v, current, target) {
				// m = append(m, newMigration(v, name))
			}
		}
		return nil
	})

	return m, nil
}

func versionFilter(v, current, target int64) bool {

	if target > current {
		return v > current && v <= target
	}

	if target < current {
		return v <= current && v > target
	}

	return false
}

// func (ms migrationSorter) Sort(direction bool) {

// 	// sort ascending or descending by version
// 	if direction {
// 		sort.Sort(ms)
// 	} else {
// 		sort.Sort(sort.Reverse(ms))
// 	}

// 	// now that we're sorted in the appropriate direction,
// 	// populate next and previous for each migration
// 	for i, m := range ms {
// 		prev := int64(-1)
// 		if i > 0 {
// 			prev = ms[i-1].Version
// 			ms[i-1].Next = m.Version
// 		}
// 		ms[i].Previous = prev
// 	}
// }

// look for migration scripts with names in the form:
//  XXX_descriptivename.ext
// where XXX specifies the version number
// and ext specifies the type of migration
// func NumericComponent(name string) (int64, error) {

// 	base := filepath.Base(name)

// 	if ext := filepath.Ext(base); ext != ".go" && ext != ".sql" {
// 		return 0, errors.New("not a recognized migration file type")
// 	}

// 	idx := strings.Index(base, "_")
// 	if idx < 0 {
// 		return 0, errors.New("no separator found")
// 	}

// 	n, e := strconv.ParseInt(base[:idx], 10, 64)
// 	if e == nil && n <= 0 {
// 		return 0, errors.New("migration IDs must be greater than zero")
// 	}

// 	return n, e
// }
type MigrationRecord struct {
	VersionId int64
	TStamp    time.Time
	IsApplied bool // was this a result of up() or down()
}

type Migration struct {
	Version  int64
	Next     int64  // next version, or -1 if none
	Previous int64  // previous version, -1 if none
	Source   string // path to .go or .sql script
}
// retrieve the current version for this DB.
// Create and initialize the DB version table if it doesn't exist.
func EnsureDBVersion(conf *DBConf, db *sql.DB) (int64, error) {

	rows, err := conf.Driver.Base.dbVersionQuery(db)
	if err != nil {
		if err == ErrTableDoesNotExist {
			return 0, createVersionTable(conf, db)
		}
		return 0, err
	}
	defer rows.Close()

	// The most recent record for each migration specifies
	// whether it has been applied or rolled back.
	// The first version we find that has been applied is the current version.

	toSkip := make([]int64, 0)

	for rows.Next() {
		var row MigrationRecord
		if err = rows.Scan(&row.VersionId, &row.IsApplied); err != nil {
			log.Fatal("error scanning rows:", err)
		}

		// have we already marked this version to be skipped?
		skip := false
		for _, v := range toSkip {
			if v == row.VersionId {
				skip = true
				break
			}
		}

		if skip {
			continue
		}

		// if version has been applied we're done
		if row.IsApplied {
			return row.VersionId, nil
		}

		// latest version of migration has not been applied.
		toSkip = append(toSkip, row.VersionId)
	}

	panic("failure in EnsureDBVersion()")
}

// Create the goose_db_version table
// and insert the initial 0 value into it
func createVersionTable(conf *DBConf, db *sql.DB) error {
	txn, err := db.Begin()
	if err != nil {
		return err
	}

	d := conf.Driver.Base

	if _, err := txn.Exec(d.createVersionTableSql()); err != nil {
		txn.Rollback()
		return err
	}

	version := 0
	applied := true
	if _, err := txn.Exec(d.insertVersionSql(), version, applied); err != nil {
		txn.Rollback()
		return err
	}

	return txn.Commit()
}

// DBバージョン取得
func GetDBVersion(conf *DBConf) (version int64, err error) {

	db, err := OpenDBFromDBConf(conf)
	if err != nil {
		return -1, err
	}
	defer db.Close()

	version, err = EnsureDBVersion(conf, db)
	if err != nil {
		return -1, err
	}

	return version, nil
}

// func GetPreviousDBVersion(dirpath string, version int64) (previous int64, err error) {

// 	previous = -1
// 	sawGivenVersion := false

// 	filepath.Walk(dirpath, func(name string, info os.FileInfo, walkerr error) error {

// 		if !info.IsDir() {
// 			if v, e := NumericComponent(name); e == nil {
// 				if v > previous && v < version {
// 					previous = v
// 				}
// 				if v == version {
// 					sawGivenVersion = true
// 				}
// 			}
// 		}

// 		return nil
// 	})

// 	if previous == -1 {
// 		if sawGivenVersion {
// 			// the given version is (likely) valid but we didn't find
// 			// anything before it.
// 			// 'previous' must reflect that no migrations have been applied.
// 			previous = 0
// 		} else {
// 			err = ErrNoPreviousVersion
// 		}
// 	}

// 	return
// }

// // helper to identify the most recent possible version
// // within a folder of migration scripts
// func GetMostRecentDBVersion(dirpath string) (version int64, err error) {

// 	version = -1

// 	filepath.Walk(dirpath, func(name string, info os.FileInfo, walkerr error) error {
// 		if walkerr != nil {
// 			return walkerr
// 		}

// 		if !info.IsDir() {
// 			if v, e := NumericComponent(name); e == nil {
// 				if v > version {
// 					version = v
// 				}
// 			}
// 		}

// 		return nil
// 	})

// 	if version == -1 {
// 		err = errors.New("no valid version found")
// 	}

// 	return
// }

// func CreateMigration(name, migrationType, dir string, t time.Time) (path string, err error) {

// 	if migrationType != "go" && migrationType != "sql" {
// 		return "", errors.New("migration type must be 'go' or 'sql'")
// 	}

// 	timestamp := t.Format("20060102150405")
// 	filename := fmt.Sprintf("%v_%v.%v", timestamp, name, migrationType)

// 	fpath := filepath.Join(dir, filename)

// 	var tmpl *template.Template
// 	if migrationType == "sql" {
// 		tmpl = sqlMigrationTemplate
// 	} else {
// 		tmpl = goMigrationTemplate
// 	}

// 	path, err = writeTemplateToFile(fpath, tmpl, timestamp)

// 	return
// }

// Update the version table for the given migration,
// and finalize the transaction.
func FinalizeMigration(conf *DBConf, txn *sql.Tx, direction bool, v int64) error {

	// XXX: drop goose_db_version table on some minimum version number?
	stmt := conf.Driver.Base.insertVersionSql()
	if _, err := txn.Exec(stmt, v, direction); err != nil {
		txn.Rollback()
		return err
	}

	return txn.Commit()
}


var sqlMigrationTemplate = template.Must(template.New(".sql-migration").Parse(`
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied


-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

`))
