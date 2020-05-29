package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/jmoiron/sqlx"
	"github.com/xbsoftware/wfs"
	db "github.com/xbsoftware/wfs-db"
)

type RichFile struct {
	wfs.File
	Favorite bool  `json:"star,omitempty"`
	Users    []int `json:"users,omitempty"`
}

func addFilesRoutes(r chi.Router) {

	r.Get("/files", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			id = "/"
		}
		source := r.URL.Query().Get("source")

		var data []wfs.File
		var err error

		if source != "" {
			switch source {
			case "recent":
				data, err = getFromQuery("select entity.* from entity where tree = ? and path != \"/\" and left(path, 1) !=\".\" order by type desc, modified desc, name asc limit 20", User.Root)
			case "favorite":
				data, err = getFromQuery("select entity.* from entity inner join favorite on entity.id = favorite.entity_id where tree = ? and path != \"/\" and left(path, 1) !=\".\" order by type desc, name asc", User.Root)
			case "shared":
				data, err = getFromQuery("select entity.* from entity inner join entity_user on entity.id = entity_user.entity_id where user_id = ? and tree = ? and path != \"/\" and left(path, 1) !=\".\" order by type desc, name asc", User.ID, User.Root)
			case "trash":
				data, err = getFromTrashQuery("select entity.* from entity where left(path,1) = \".\" AND tree = ? AND folder = -1 order by type desc, name asc", User.Root)
			}
		} else {
			search := r.URL.Query().Get("search")

			var config *wfs.ListConfig
			if search == "" {
				config = &wfs.ListConfig{
					Nested:  true,
					Exclude: func(name string) bool { return strings.HasPrefix(name, ".") },
				}
				data, err = drive.List(id, config)
			} else {
				names, tags := processInput(search)
				searchBy := "WHERE tree = ? AND path LIKE ?"
				params := make([]interface{}, len(names)+2)
				params[0] = User.Root
				if id == "/" {
					params[1] = "/%"
				} else {
					params[1] = id + "/%"
				}

				for i := range names {
					params[i+2] = "%" + names[i] + "%"
					searchBy += " AND name LIKE ?"
				}
				if len(tags) > 0 {
					tsql, targs, _ := sqlx.In("WHERE tag.value IN (?)", tags)
					searchBy += ` AND id IN (
						SELECT entity_id FROM entity_tag 
						INNER JOIN tag ON tag.id = entity_tag.tag_id ` + tsql + `) `
					params = append(params, targs...)
				}

				if len(params) > 2 {
					data, err = getFromQuery(`
						SELECT entity.* FROM entity `+searchBy, params...)
				}
			}
		}

		if err != nil {
			format.Text(w, 500, err.Error())
			return
		}

		err = format.JSON(w, 200, enrich(data, conn))
	})

}

type UserShare struct {
	UserID     int    `db:"user_id"`
	EntityPath string `db:"path"`
}

func getFromTrashQuery(sql string, args ...interface{}) ([]wfs.File, error) {
	data := make([]db.DBFile, 0)

	err := conn.Select(&data, sql, args...)
	if err != nil {
		return nil, err
	}

	out := make([]wfs.File, len(data))
	for i, d := range data {
		out[i] = wfs.File{ID: strconv.Itoa(d.ID), Name: d.FileName, Date: d.LastModTime.Unix(), Size: d.FileSize, Type: wfs.GetType(d.FileName, d.IsDir())}
	}

	return out, nil
}

func getFromQuery(sql string, args ...interface{}) ([]wfs.File, error) {
	data := make([]db.DBFile, 0)

	err := conn.Select(&data, sql, args...)
	if err != nil {
		return nil, err
	}

	out := make([]wfs.File, len(data))
	for i, d := range data {
		log.Println(d.Content)
		out[i] = wfs.File{ID: d.Path, Name: d.FileName, Date: d.LastModTime.Unix(), Size: d.FileSize, Type: wfs.GetType(d.FileName, d.IsDir())}
	}

	return out, nil
}

func enrich(data []wfs.File, db *sqlx.DB) []RichFile {
	rfiles := make([]RichFile, len(data))
	ids := make([]string, 0, len(data))
	temp := make(map[string]*RichFile)

	for i := range data {
		rfiles[i] = RichFile{File: data[i]}
		ids = append(ids, data[i].ID)
		temp[data[i].ID] = &rfiles[i]
	}

	favs := make([]string, 0)
	query, args, _ := sqlx.In(`
SELECT entity.path FROM entity 
INNER JOIN favorite ON entity.id = favorite.entity_id 
WHERE entity.path IN (?)  AND favorite.user_id = ? AND tree = ?`, ids, User.ID, User.Root)
	err := db.Select(&favs, query, args...)
	if err != nil {
		log.Print(err.Error())
	}

	users := make([]UserShare, 0)
	query, args, _ = sqlx.In(`
SELECT entity.path, entity_user.user_id 
FROM entity INNER JOIN entity_user ON entity.id = entity_user.entity_id
WHERE entity.path IN (?) AND tree = ?`, ids, User.Root)
	err = db.Select(&users, query, args...)
	if err != nil {
		log.Print(err.Error())
	}
	for _, f := range favs {
		temp[f].Favorite = true
	}

	for _, u := range users {
		t := temp[u.EntityPath]
		if t.Users == nil {
			t.Users = []int{u.UserID}
		} else {
			t.Users = append(t.Users, u.UserID)
		}
	}

	return rfiles
}

func processInput(inp string) (names, tags []string) {
	words := strings.Fields(inp)

	for _, w := range words {
		if strings.HasPrefix(w, "#") {
			if len(w) > 1 {
				tags = append(tags, strings.TrimLeft(w, "#"))
			}
		} else {
			names = append(names, w)
		}
	}

	return
}
