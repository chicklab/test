package eioh

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
	"sort"

	_ "github.com/go-sql-driver/mysql"
	"github.com/olekukonko/tablewriter"
)

//マイグレーション+スラック通知

const sqlCmdPrefix = "-- +eioh "

type migrationSorter []*Migration


func (ms migrationSorter) Len() int           { return len(ms) }
func (ms migrationSorter) Swap(i, j int)      { ms[i], ms[j] = ms[j], ms[i] }
func (ms migrationSorter) Less(i, j int) bool { return ms[i].Version < ms[j].Version }

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

func splitSQLStatements(r io.Reader, direction bool) (stmts []string) {

	var buf bytes.Buffer
	scanner := bufio.NewScanner(r)

	upSections := 0
	downSections := 0

	statementEnded := false
	ignoreSemicolons := false
	directionIsActive := false

	for scanner.Scan() {

		line := scanner.Text()
		if strings.HasPrefix(line, sqlCmdPrefix) {
			cmd := strings.TrimSpace(line[len(sqlCmdPrefix):])
			switch cmd {
			case "up":
				directionIsActive = (direction == true)
				upSections++
				break

			case "down":
				directionIsActive = (direction == false)
				downSections++
				break

			case "statementbegin":
				if directionIsActive {
					ignoreSemicolons = true
				}
				break

			case "statementend":
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

		if (!ignoreSemicolons && endsWithSemicolon(line)) || statementEnded {
			statementEnded = false
			stmts = append(stmts, buf.String())
			buf.Reset()
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("scanning migration: %v", err)
	}

	if ignoreSemicolons {
		log.Println("WARNING: saw '-- +eioh statementbegin' with no matching '-- +eioh statementend'")
	}

	if bufferRemaining := strings.TrimSpace(buf.String()); len(bufferRemaining) > 0 {
		log.Printf("WARNING: Unexpected unfinished SQL query: %s. Missing a semicolon?\n", bufferRemaining)
	}

	if upSections == 0 && downSections == 0 {
		log.Fatalf(`ERROR: no up/down annotations found, so no statements were executed.
			See https://bitbucket.org/liamstask/eioh/overview for details.`)
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

func RunMigrationsOnDb(conf *DBConf, migrationsDir string, target int64, db *sql.DB) (err error) {

	current, err := EnsureDBVersion(conf, db)

	if err != nil {
		return err
	}


	migrations, err := CollectMigrations(migrationsDir, current, target)
	

	if err != nil {
		return err
	}
	fmt.Println(migrations);

	if len(migrations) == 0 {
		fmt.Printf("eioh: no migrations to run. current version: %d\n", current)
		return nil
	}

	ms := migrationSorter(migrations)
	direction := current < target
	ms.Sort(direction)

	fmt.Printf("eioh: migrating db environment '%v', current version: %d, target: %d\n",
		conf.Env, current, target)

	for _, m := range ms {

		switch filepath.Ext(m.Source) {
		case ".sql":
			err = runSQLMigration(conf, db, m.Source, m.Version, direction)
		}

		if err != nil {
			return errors.New(fmt.Sprintf("FAIL %v, quitting migration", err))
		}

		fmt.Println("OK   ", filepath.Base(m.Source))
	}

	return nil
}

func runSQLMigration(conf *DBConf, db *sql.DB, scriptFile string, v int64, direction bool) error {

	txn, err := db.Begin()
	if err != nil {
		log.Fatal("db.Begin:", err)
	}

	f, err := os.Open(scriptFile)
	if err != nil {
		log.Fatal(err)
	}

	for _, query := range splitSQLStatements(f, direction) {
		fmt.Println(query)
		if _, err = txn.Exec(query); err != nil {
			txn.Rollback()
			log.Fatalf("FAIL %s (%v), quitting migration.", filepath.Base(scriptFile), err)
			return err
		}
	}

	if err = FinalizeMigration(conf, txn, direction, v); err != nil {
		log.Fatalf("error finalizing migration %s, quitting. (%v)", filepath.Base(scriptFile), err)
	}

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
				m = append(m, newMigration(v, name))
			}
		}
		return nil
	})

	return m, nil
}

func newMigration(v int64, src string) *Migration {
	return &Migration{v, -1, -1, src}
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

func (ms migrationSorter) Sort(direction bool) {

	if direction {
		sort.Sort(ms)
	} else {
		sort.Sort(sort.Reverse(ms))
	}

	for i, m := range ms {
		prev := int64(-1)
		if i > 0 {
			prev = ms[i-1].Version
			ms[i-1].Next = m.Version
		}
		ms[i].Previous = prev
	}
}

type MigrationRecord struct {
	VersionId int64
	CreateDate time.Time
	Status bool
}

type Migration struct {
	Version  int64
	Next     int64
	Previous int64
	Source   string
}

func EnsureDBVersion(conf *DBConf, db *sql.DB) (int64, error) {

	rows, err := conf.Driver.Base.dbVersionQuery(db)
	if err != nil {
		if err == ErrTableDoesNotExist {
			return 0, createVersionTable(conf, db)
		}
		return 0, err
	}
	defer rows.Close()
	

	toSkip := make([]int64, 0)

	for rows.Next() {
		
		var row MigrationRecord
		if err = rows.Scan(&row.VersionId, &row.Status, &row.CreateDate); err != nil {
			log.Fatal("error scanning rows:", err)
		}

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

		if row.Status {
			return row.VersionId, nil
		}
		toSkip = append(toSkip, row.VersionId)
	}

	return 0, err
}

func showDBStatus(conf *DBConf, db *sql.DB) error {

	rows, err := conf.Driver.Base.dbVersionQuery(db)
	if err != nil {
		if err != ErrTableDoesNotExist {
			return err
		}
		if err = createVersionTable(conf, db); err != nil {
			return err
		}
		if rows, err = conf.Driver.Base.dbVersionQuery(db); err != nil {
			return err
		}
	}
	defer rows.Close()

	data := make([][]string, 0)
	count := 0

	for rows.Next() {
		var row MigrationRecord
		if err = rows.Scan(&row.VersionId, &row.Status, &row.CreateDate); err != nil {
			log.Fatal("error scanning rows:", err)
		}
		// a := strconv.FormatBool(row.Status)
		a := "up"
		if !row.Status {
			a = "down"
		}
		b := strconv.FormatInt(row.VersionId, 10)
		c := row.CreateDate

		data = append(data, []string{a, b, c.String()})
		count++
	}

	if count == 0 {
		log.Fatal("no valid version found")
		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Status", "MigrationId", "CreateDate"})
	for _, v := range data {
		table.Append(v)
	}
	table.Render()

	return nil
}

func StatusMigration(conf *DBConf) (err error) {

	db, err := OpenDBFromDBConf(conf)

	if err != nil {
		return err
	}
	defer db.Close()

	err = showDBStatus(conf, db)
	if err != nil {
		return err
	}

	return nil
}

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

	return txn.Commit()
}


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

func GetPreviousDBVersion(dirpath string, version int64) (previous int64, err error) {

	previous = -1
	sawGivenVersion := false

	filepath.Walk(dirpath, func(name string, info os.FileInfo, walkerr error) error {

		if !info.IsDir() {
			if v, e := NumericComponent(name); e == nil {
				if v > previous && v < version {
					previous = v
				}
				if v == version {
					sawGivenVersion = true
				}
			}
		}

		return nil
	})

	if previous == -1 {
		if sawGivenVersion {
			previous = 0
		} else {
			err = ErrNoPreviousVersion
		}
	}

	return
}

func GetMostRecentDBVersion(dirpath string) (version int64, err error) {

	version = -1

	filepath.Walk(dirpath, func(name string, info os.FileInfo, walkerr error) error {
		if walkerr != nil {
			return walkerr
		}

		if !info.IsDir() {
			if v, e := NumericComponent(name); e == nil {
				if v > version {
					version = v
				}
			}
		}
		return nil
	})

	if version == -1 {
		err = errors.New("no valid version found")
	}

	return
}

func CreateMigration(name, dir string, t time.Time) (path string, err error) {

	timestamp := t.Format("20060102150405")
	filename := fmt.Sprintf("%v_%v.sql", timestamp, name)

	fpath := filepath.Join(dir, filename)

	var tmpl *template.Template
	tmpl = sqlMigrationTemplate

	path, err = writeTemplateToFile(fpath, tmpl, timestamp)

	return
}

func FinalizeMigration(conf *DBConf, txn *sql.Tx, direction bool, v int64) error {

	stmt := conf.Driver.Base.insertVersionSql()
	if _, err := txn.Exec(stmt, v, direction); err != nil {
		txn.Rollback()
		return err
	}

	return txn.Commit()
}

var sqlMigrationTemplate = template.Must(template.New(".sql-migration").Parse(`
-- +eioh up
-- SQL in section 'up' is executed when this migration is applied


-- +eioh down
-- SQL section 'down' is executed when this migration is rolled back

`))
