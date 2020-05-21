package main

import (
	"errors"
	"html"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/sergi/go-diff/diffmatchpatch"
)

type TagInfo struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
	Color string `json:"color"`
}

type UserInfo struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Avatar string `json:"avatar"`
}

type CommentInfo struct {
	ID       int       `json:"id"`
	Content  string    `json:"text"`
	Modified time.Time `json:"date"`
	UserId   int       `db:"user_id" json:"user_id"`
}

type CurrentUser struct {
	ID   int
	Root int
}

type EditInfo struct {
	ID       int       `json:"id"`
	Modified time.Time `json:"date"`
	User     int       `db:"user_id" json:"user"`
	Content  string    `json:"content"`
	Origin   time.Time `json:"origin"`
}

var User = CurrentUser{1, 1}

func dbID(id string) (res int) {
	conn.Get(&res, "select id from entity where path =? and tree = ?", id, User.Root)
	return res
}

func addExtrasRoutes(r chi.Router) {
	r.Get("/tags/all", func(w http.ResponseWriter, r *http.Request) {
		info := make([]TagInfo, 0)
		conn.Select(&info, "select id, name, value, color from tag")

		format.JSON(w, 200, info)
	})

	r.Get("/tags", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		did := dbID(id)

		ids := make([]int, 0)
		conn.Select(&ids, "select tag_id from entity_tag where entity_id = ? ", did)

		format.JSON(w, 200, ids)
	})

	r.Put("/tags", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		id := r.Form.Get("id")
		did := dbID(id)
		tags := strings.Split(r.Form.Get("value"), ",")

		conn.Exec("delete from entity_tag WHERE entity_id = ?", did)
		for _, tag := range tags {
			conn.Exec("insert into entity_tag(entity_id, tag_id) VALUES(?, ?)", did, tag)
		}

		format.JSON(w, 200, Response{ID: id})
	})

	r.Post("/tags", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		name := r.Form.Get("name")
		color := r.Form.Get("color")
		value := strings.ReplaceAll(name, " ", "")

		res, _ := conn.Exec("INSERT INTO tag (name, value, color) VALUES (?, ?, ?)", name, value, color)

		cid, _ := res.LastInsertId()
		format.JSON(w, 200, Response{ID: strconv.FormatInt(cid, 10)})
	})

	r.Put("/tags/{id}", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		id := chi.URLParam(r, "id")
		name := r.Form.Get("name")
		color := r.Form.Get("color")
		value := strings.ReplaceAll(name, " ", "")

		conn.Exec("UPDATE tag SET name = ?, value = ?, color = ? WHERE id = ?", name, value, color, id)
		format.JSON(w, 200, Response{ID: id})
	})

	r.Delete("/tags/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		conn.Exec("DELETE FROM tag WHERE id = ?", id)
		format.JSON(w, 200, Response{ID: id})
	})

	r.Get("/users/all", func(w http.ResponseWriter, r *http.Request) {
		info := make([]UserInfo, 0)
		conn.Select(&info, "select id, name, email, avatar from user")

		format.JSON(w, 200, info)
	})
	r.Get("/users/{id}/avatar/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		http.ServeFile(w, r, filepath.Join(Config.DataFolder, "avatars", name))
	})

	r.Post("/favorite", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		id := r.Form.Get("id")
		did := dbID(id)

		conn.Exec("INSERT INTO favorite(entity_id, user_id) values(?, ?)", did, User.ID)
		format.JSON(w, 200, Response{ID: id})
	})

	r.Delete("/favorite", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		did := dbID(id)

		conn.Exec("DELETE FROM favorite WHERE entity_id = ? and user_id = ?", did, User.ID)
		format.JSON(w, 200, Response{ID: id})
	})

	r.Post("/share", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		id := r.Form.Get("id")
		uid := r.Form.Get("user")
		did := dbID(id)

		conn.Exec("INSERT INTO entity_user(entity_id, user_id) values(?, ?)", did, uid)
		format.JSON(w, 200, Response{ID: id})
	})

	r.Delete("/share", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		uid := r.URL.Query().Get("user")
		did := dbID(id)

		conn.Exec("DELETE FROM entity_user WHERE entity_id = ? and user_id = ?", did, uid)
		format.JSON(w, 200, Response{ID: id})
	})

	r.Get("/comments", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		did := dbID(id)

		comments := make([]CommentInfo, 0)
		conn.Select(&comments, "select id,content,user_id,modified from comment where entity_id = ?", did)

		format.JSON(w, 200, comments)
	})

	r.Post("/comments", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		id := r.URL.Query().Get("id")
		did := dbID(id)
		content := r.Form.Get("value")

		res, _ := conn.Exec("insert into comment(entity_id, user_id, content)  values(?, ?, ?)", did, User.ID, content)
		cid, _ := res.LastInsertId()
		format.JSON(w, 200, Response{ID: strconv.FormatInt(cid, 10)})
	})

	r.Put("/comments/{id}", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		id := chi.URLParam(r, "id")
		content := r.Form.Get("value")

		var uid int
		conn.Get(&uid, "select user_id from comment where id = ?", id)
		if uid != User.ID {
			format.Text(w, 500, "Access Denied")
			return
		}

		conn.Exec("update comment SET content = ? WHERE id = ?", content, id)
		format.JSON(w, 200, Response{ID: id})
	})

	r.Delete("/comments/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		var uid int
		conn.Get(&uid, "select user_id from comment where id = ?", id)
		if uid != User.ID {
			format.Text(w, 500, "Access Denied")
			return
		}

		conn.Exec("delete from comment WHERE id = ?", id)
		format.JSON(w, 200, Response{ID: id})
	})

	r.Get("/versions", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		did := dbID(id)

		versions := make([]EditInfo, 0)
		err := conn.Select(&versions, "SELECT id,modified,user_id,origin FROM entity_edit WHERE entity_id = ? ORDER BY modified desc", did)
		if err != nil {
			panic(err)
		}

		format.JSON(w, 200, versions)
	})

	r.Get("/versions/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		_, diff := r.URL.Query()["diff"]

		var content, previous string
		conn.Get(&content, "SELECT content FROM entity_edit WHERE id = ?", id)
		if diff {
			conn.Get(&previous, "SELECT previous FROM entity_edit WHERE id = ?", id)
		}

		mode := r.URL.Query().Get("mode")
		if mode == "text" {
			w.Header().Add("Content-type", "text/plain")
			text2 := getTextFromFile(filepath.Join(Config.DataFolder, content))

			if previous != "" {
				text1 := getTextFromFile(filepath.Join(Config.DataFolder, previous))
				dmp := diffmatchpatch.New()
				diffs := dmp.DiffMain(text1, text2, false)
				response := dmp.DiffPrettyHtml(diffs)

				io.WriteString(w, response)
				return
			}

			io.WriteString(w, html.EscapeString(text2))

		} else if mode == "binary" {
			data, err := os.Open(filepath.Join(Config.DataFolder, content))
			if err != nil {
				panic(errors.New("Can't open file for reading"))
			}
			disposition := "inline"
			w.Header().Set("Content-Disposition", disposition+"; filename=\""+content+"\"")
			http.ServeContent(w, r, "", time.Now(), data)
		}
	})

	r.Post("/versions", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		id := r.Form.Get("id")
		version := r.Form.Get("version")

		var edit EditInfo
		conn.Get(&edit, "SELECT content, modified FROM entity_edit WHERE id = ?", version)

		file, err := os.Open(filepath.Join(Config.DataFolder, edit.Content))
		if err != nil {
			panic(errors.New("Can't open file for reading"))
		}

		err = drive.Write(id, file)
		if err != nil {
			panic(err)
		}

		info, _ := saveVersion(id, edit.Modified)

		format.JSON(w, 200, info)
	})
}

func getTextFromFile(path string) string {
	d, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	return string(d)
}
