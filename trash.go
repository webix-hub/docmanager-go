package main

import (
	"net/http"
	"path"

	"github.com/go-chi/chi"
	"github.com/jmoiron/sqlx"
	db "github.com/xbsoftware/wfs-db"
)

func addTrashRoutes(r chi.Router) {
	r.Post("/delete", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		id := r.Form.Get("id")
		if id == "" || id[0:2] == "./" {
			panic("path not provided")
		}

		obj, err := drive.Info(id)
		if err != nil {
			format.JSON(w, 500, Response{Invalid: true, Error: err.Error()})
			return
		}

		_, err = conn.Exec("update entity set path = ?, folder = -1 where path = ? AND tree = ?", "."+id, id, User.Root)
		if err != nil {
			format.JSON(w, 500, Response{Invalid: true, Error: err.Error()})
			return
		}

		// mark all files in deleted folder
		if obj.Type == "folder" {
			_, err = conn.Exec("update entity set path = concat(\".\", path) where path LIKE ? AND tree = ?", id+"/%", User.Root)
			if err != nil {
				format.JSON(w, 500, Response{Invalid: true, Error: err.Error()})
				return
			}
		}

		format.JSON(w, 200, Response{})
	})

	r.Put("/delete", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		id := r.Form.Get("id")

		obj := db.DBFile{}
		conn.Get(&obj, "select * from entity where id=? and tree=?", id, User.Root)
		if obj.ID == 0 || obj.Path[0:2] != "./" {
			panic("wrong id provided")
		}

		// all involved files
		ids := make([]int, 0)
		if obj.Type == 2 {
			ids = append(ids, selectIdRec(obj.ID)...)
		}

		deletedPath := obj.Path[1:]
		restorePath := path.Dir(obj.Path[1:])
		// check restore folder
		newRoot := 0
		conn.Get(&newRoot, "SELECT id FROM entity WHERE path = ? AND tree = ?", restorePath, User.Root)
		if newRoot == 0 {
			conn.Get(&newRoot, "SELECT id FROM entity WHERE path = \"/\" AND tree = ?", User.Root)
		}

		targetName := obj.FileName
		targetPath := path.Join(restorePath, path.Base(obj.Path[1:]))
		// ensure that file name is not occupied
		for {
			fileUsed := 0
			conn.Get(&fileUsed, "SELECT id FROM entity WHERE path = ? AND tree = ?", targetPath, User.Root)
			if fileUsed == 0 {
				break
			}
			targetName += ".restored"
			targetPath += ".restored"
		}

		// restore the object
		_, err := conn.Exec("UPDATE entity set name = ?, path = ?, folder = ? where id = ? AND tree = ?", targetName, targetPath, newRoot, id, User.Root)
		if err != nil {
			format.JSON(w, 500, Response{Invalid: true, Error: err.Error()})
			return
		}

		// and all inner objects
		if len(ids) > 0 {
			sql, args, _ := sqlx.In("UPDATE entity set path = REPLACE(path, ?, ?) where id IN (?) AND tree = ?", "."+deletedPath+"/", targetPath+"/", ids, User.Root)
			_, err = conn.Exec(sql, args...)
			if err != nil {
				format.JSON(w, 500, Response{Invalid: true, Error: err.Error()})
				return
			}
		}

		info, err := drive.Info(targetPath)
		format.JSON(w, 200, info)
	})

	r.Delete("/delete", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		obj := db.DBFile{}

		conn.Get(&obj, "select * from entity where id=? and tree=?", id, User.Root)
		if obj.ID == 0 || obj.Path[0:2] != "./" {
			panic("wrong id provided")
		}

		// all involved files
		ids := []int{obj.ID}
		if obj.Type == 2 {
			ids = append(ids, selectIdRec(obj.ID)...)
		}

		// delete related markers
		idStr, args, _ := sqlx.In("entity_id IN(?) AND user_id = ? ", ids, User.Root)
		conn.Exec("DELETE FROM favorite WHERE "+idStr, args...)
		idStr, args, _ = sqlx.In("entity_id IN(?)", ids)
		conn.Exec("DELETE FROM entity_tag WHERE "+idStr, args...)
		conn.Exec("DELETE FROM entity_user WHERE "+idStr, args...)

		// delete file itself
		idStr, args, _ = sqlx.In("DELETE FROM entity where id in (?)", ids)
		_, err := conn.Exec(idStr, args...)
		if err != nil {
			format.JSON(w, 500, Response{Invalid: true, Error: err.Error()})
			return
		}

		format.JSON(w, 200, Response{})
	})
}

func selectIdRec(folder int) []int {
	var ids = make([]db.DBFile, 0)
	conn.Select(&ids, "select id,type FROM entity where folder = ? AND tree = ?", folder, User.Root)

	out := make([]int, 0, len(ids))
	for i := range ids {
		out = append(out, ids[i].ID)
		if ids[i].Type == 2 {
			out = append(out, selectIdRec(ids[i].ID)...)
		}
	}

	return out
}
