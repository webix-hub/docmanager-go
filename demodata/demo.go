package demodata

import (
	"database/sql"
	"github.com/jmoiron/sqlx"
	"github.com/xbsoftware/wfs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func must(r sql.Result, e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func ResetDemoData(drive wfs.Drive, db *sqlx.DB) {
	must(db.Exec("truncate table entity"))
	must(db.Exec("truncate table entity_edit"))
	must(db.Exec("truncate table entity_tag"))
	must(db.Exec("truncate table entity_user"))

	must(db.Exec("truncate table comment"))
	must(db.Exec("truncate table favorite"))
	must(db.Exec("truncate table tag"))
	must(db.Exec("truncate table user"))

	ImportDemoData(drive, db)
}

func ImportDemoData(drive wfs.Drive, db *sqlx.DB) {
	tcount := struct{ Count int }{}
	must(nil, db.Get(&tcount, "select count(entity.id) as count from entity"))
	if tcount.Count > 0 {
		return
	}

	importDemoTags(db)
	importDemoUsers(db)
	importDemoEntities(drive, db)
}

func importDemoTags(db *sqlx.DB) {
	must(db.Exec("INSERT INTO tag (id, name, value, color) VALUES (1, 'Review', 'Review', '#ddaaff')"))
	must(db.Exec("INSERT INTO tag (id, name, value, color) VALUES (2, 'Accepted', 'Accepted', '#00ffbb')"))
	must(db.Exec("INSERT INTO tag (id, name, value, color) VALUES (3, 'Denied', 'Denied', '#bb00ff')"))
	must(db.Exec("INSERT INTO tag (id, name, value, color) VALUES (4, 'Personal', 'Personal', '#aa00aa')"))
}

func importDemoUsers(db *sqlx.DB) {
	must(db.Exec("INSERT INTO user (id, email, name, avatar) VALUES (1, 'alastor@ya.ru', 'Alastor Moody', '/users/1/avatar/1.jpg')"))
	must(db.Exec("INSERT INTO user (id, email, name, avatar) VALUES (2, 'johndawlish@gmail.com', 'John Dawlish', '/users/2/avatar/2.jpg')"))
	must(db.Exec("INSERT INTO user (id, email, name, avatar) VALUES (3, 'sirius@gmail.com', 'Sirius Black', '/users/3/avatar/3.jpg')"))
	must(db.Exec("INSERT INTO user (id, email, name, avatar) VALUES (4, 'nymphadora@gmail.com', 'Nymphadora Tonks', '/users/4/avatar/4.jpg')"))
}

func importDemoEntities(drive wfs.Drive, db *sqlx.DB) {
	demoRoot, err := filepath.Abs("./demodata/files")
	if err != nil {
		return
	}

	filepath.Walk(demoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || path == demoRoot {
			return err
		}

		name := filepath.Base(path)
		dir := strings.Replace(strings.Replace(strings.Replace(path, demoRoot, "", 1), name, "", 1), "\\", "/", -1)
		if len(dir) > 1 {
			dir = dir[0 : len(dir)-1]
		}

		if info.IsDir() {
			drive.Make(dir, name, true)
		} else {
			id, _ := drive.Make(dir, name, false)

			reader, _ := os.Open(path)
			drive.Write(id, reader)
			reader.Close()
		}

		return nil
	})
}
