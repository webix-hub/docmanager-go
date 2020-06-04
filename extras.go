package main

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
)

type TagInfo struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
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
	Content  string    `json:"name"`
	Modified time.Time `json:"date"`
	User     int       `db:"user_id" json:"user"`
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
		conn.Select(&info, "select id, name, value from tag")

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
		conn.Select(&versions, "SELECT id,content,modified,user_id,origin FROM entity_edit WHERE entity_id = ? ORDER BY modified desc", did)

		format.JSON(w, 200, versions)
	})
}
