package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"wfs-ls/demodata"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/jinzhu/configor"
	"github.com/unrolled/render"

	"github.com/xbsoftware/wfs"
	db "github.com/xbsoftware/wfs-db"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var format = render.New()

type Response struct {
	Invalid bool   `json:"invalid"`
	Error   string `json:"error"`
	ID      string `json:"id"`
}

type FSFeatures struct {
	Preview map[string]bool `json:"preview"`
	Meta    map[string]bool `json:"meta"`
}

var drive wfs.Drive
var conn *sqlx.DB
var features = FSFeatures{
	Preview: map[string]bool{},
	Meta:    map[string]bool{},
}

type AppConfig struct {
	DataFolder   string `default:"/tmp/docs"`
	Port         string
	Preview      string
	UploadLimit  int64
	Readonly     bool
	ResetOnStart bool

	DB DBConfig
}

type DBConfig struct {
	Host     string `default:"localhost"`
	Port     string `default:"3306"`
	User     string `default:"root"`
	Password string `default:"1"`
	Database string `default:"files"`
}

var Config AppConfig

func main() {
	flag.StringVar(&Config.DataFolder, "data", "", "location of data folder")
	flag.StringVar(&Config.Preview, "preview", "", "url of preview generation service")
	flag.BoolVar(&Config.ResetOnStart, "reset", false, "reset data in DB")
	flag.BoolVar(&Config.Readonly, "readonly", false, "readonly mode")
	flag.Int64Var(&Config.UploadLimit, "limit", 10_000_000, "max file size to upload")
	flag.StringVar(&Config.Port, "port", ":3200", "port for web server")
	flag.Parse()

	configor.New(&configor.Config{ENVPrefix: "APP", Silent: true}).Load(&Config, "config.yml")

	// configure features
	features.Meta["audio"] = true
	features.Meta["image"] = true
	if Config.Preview != "none" {
		features.Preview["image"] = true
		if Config.Preview != "" {
			features.Preview["document"] = true
			features.Preview["code"] = true
		}
	}

	// common drive access
	var err error
	driveConfig := wfs.DriveConfig{Verbose: true}
	driveConfig.Operation = &wfs.OperationConfig{PreventNameCollision: true}
	if Config.Readonly {
		temp := wfs.Policy(&wfs.ReadOnlyPolicy{})
		driveConfig.Policy = &temp
	}

	connStr := fmt.Sprintf("%s:%s@(%s:%s)/%s?multiStatements=true&parseTime=true",
		Config.DB.User, Config.DB.Password, Config.DB.Host, Config.DB.Port, Config.DB.Database)
	conn, err = sqlx.Connect("mysql", connStr)
	if err != nil {
		log.Fatal(err)
	}

	migration(conn)

	os.Mkdir(Config.DataFolder, 0777)
	drive, err = db.NewDBDrive(conn, Config.DataFolder, "entity", User.Root, &driveConfig)
	if err != nil {
		log.Fatal(err)
	}

	if Config.ResetOnStart {
		demodata.ResetDemoData(drive, conn)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	cors := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	})
	r.Use(cors.Handler)

	addExtrasRoutes(r)
	addFilesRoutes(r)
	addTrashRoutes(r)

	r.Get("/icons/{size}/{type}/{name}", func(w http.ResponseWriter, r *http.Request) {
		size := chi.URLParam(r, "size")
		name := chi.URLParam(r, "name")
		ftype := chi.URLParam(r, "type")

		http.ServeFile(w, r, getIconURL(size, ftype, name))
	})

	r.Get("/preview", getFilePreview)

	r.Get("/search", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		search := r.URL.Query().Get("search")

		data, err := drive.Search(id, search)

		if err != nil {
			format.Text(w, 500, err.Error())
			return
		}
		format.JSON(w, 200, data)
	})

	r.Get("/folders", func(w http.ResponseWriter, r *http.Request) {
		data, err := drive.List("/", &wfs.ListConfig{
			Nested:     true,
			SubFolders: true,
			SkipFiles:  true,
			Exclude:    func(name string) bool { return strings.HasPrefix(name, ".") },
		})

		if err != nil {
			format.Text(w, 500, err.Error())
			return
		}

		err = format.JSON(w, 200, data)
	})

	r.Post("/copy", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		id := r.Form.Get("id")
		to := r.Form.Get("to")
		if id == "" || to == "" {
			panic("both, 'id' and 'to' parameters must be provided")
		}

		id, err := drive.Copy(id, to, "")
		if err != nil {
			format.JSON(w, 500, Response{Invalid: true, Error: err.Error()})
			return
		}

		info, err := drive.Info(id)
		if err != nil {
			format.JSON(w, 500, Response{Invalid: true, Error: err.Error()})
			return
		}

		format.JSON(w, 200, info)
	})

	r.Post("/move", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		id := r.Form.Get("id")
		to := r.Form.Get("to")
		if id == "" || to == "" {
			panic("both, 'id' and 'to' parameters must be provided")
		}

		id, err := drive.Move(id, to, "")
		if err != nil {
			format.JSON(w, 500, Response{Invalid: true, Error: err.Error()})
			return
		}

		info, err := drive.Info(id)
		if err != nil {
			format.JSON(w, 500, Response{Invalid: true, Error: err.Error()})
			return
		}

		format.JSON(w, 200, info)
	})

	r.Post("/rename", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		id := r.Form.Get("id")
		name := r.Form.Get("name")
		if id == "" || name == "" {
			panic("both, 'id' and 'name' parameters must be provided")
		}

		id, err := drive.Move(id, "", name)
		if err != nil {
			format.JSON(w, 500, Response{Invalid: true, Error: err.Error()})
			return
		}

		format.JSON(w, 200, Response{ID: id})
	})

	r.Post("/upload", func(w http.ResponseWriter, r *http.Request) {
		handleUpload(w, r, true)
	})

	r.Post("/makefile", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		id := r.Form.Get("id")
		name := r.Form.Get("name")
		if id == "" || name == "" {
			panic("both, 'id' and 'name' parameters must be provided")
		}

		id, err := drive.Make(id, name, false)
		if err != nil {
			format.JSON(w, 500, Response{Invalid: true, Error: err.Error()})
			return
		}

		info, err := drive.Info(id)
		if err != nil {
			format.JSON(w, 500, Response{Invalid: true, Error: err.Error()})
			return
		}

		format.JSON(w, 200, info)
	})

	r.Post("/makedir", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		id := r.Form.Get("id")
		name := r.Form.Get("name")
		if id == "" || name == "" {
			panic("both, 'id' and 'name' parameters must be provided")
		}

		id, err := drive.Make(id, name, true)
		if err != nil {
			format.JSON(w, 500, Response{Invalid: true, Error: err.Error()})
			return
		}

		info, err := drive.Info(id)
		if err != nil {
			format.JSON(w, 500, Response{Invalid: true, Error: err.Error()})
			return
		}

		format.JSON(w, 200, info)
	})

	r.Get("/text", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			panic("id not provided")
		}

		data, err := drive.Read(id)
		if err != nil {
			panic(err)
		}

		w.Header().Add("Content-type", "text/plain")
		io.Copy(w, data)
	})

	r.Post("/text", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()

		id := r.Form.Get("id")
		content := r.Form.Get("content")
		if id == "" {
			panic("id not provided")
		}

		err = drive.Write(id, strings.NewReader(content))
		if err != nil {
			panic(err)
		}

		info, _ := saveVersion(id, nil)

		format.JSON(w, 200, info)
	})

	r.Get("/direct", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			panic("id not provided")
		}

		info, err := drive.Info(id)
		if err != nil {
			format.Text(w, 500, "Access denied")
			return
		}

		data, err := drive.Read(id)
		if err != nil {
			format.Text(w, 500, "Access denied")
			return
		}

		disposition := "inline"
		_, ok := r.URL.Query()["download"]
		if ok {
			disposition = "attachment"
		}

		w.Header().Set("Content-Disposition", disposition+"; filename=\""+info.Name+"\"")
		http.ServeContent(w, r, "", time.Now(), data)
	})

	r.Post("/direct", func(w http.ResponseWriter, r *http.Request) {
		handleUpload(w, r, false)
	})

	r.Get("/info", getInfo)
	r.Get("/meta", getMetaInfo)

	log.Printf("Starting webserver at port " + Config.Port)
	http.ListenAndServe(Config.Port, r)
}

func handleUpload(w http.ResponseWriter, r *http.Request, makeNew bool) {
	// buffer for file parsing, this is NOT the max upload size
	var limit = int64(32 << 20) // default is 32MB
	if Config.UploadLimit < limit {
		limit = Config.UploadLimit
	}

	// this one limit max upload size
	r.Body = http.MaxBytesReader(w, r.Body, Config.UploadLimit)
	r.ParseMultipartForm(limit)

	file, handler, err := r.FormFile("upload")
	if err != nil {
		panic("Error Retrieving the File")
	}
	defer file.Close()

	fileID := r.URL.Query().Get("id")
	if makeNew {
		fileID, err = drive.Make(fileID, handler.Filename, false)
		if err != nil {
			format.Text(w, 500, "Access Denied")
			return
		}
	}

	err = drive.Write(fileID, file)
	if err != nil {
		format.Text(w, 500, "Access Denied")
		return
	}

	info, err := saveVersion(fileID, nil)
	format.JSON(w, 200, info)
}

func saveVersion(id string, restore *time.Time) (*wfs.File, error) {
	var data db.DBFile

	err := conn.Get(&data, "select entity.* from entity where path = ?", id)
	if err != nil {
		return nil, err
	}

	out := &wfs.File{ID: data.Path, Name: data.FileName, Date: data.LastModTime.Unix(), Size: data.FileSize, Type: wfs.GetType(data.FileName, data.IsDir())}

	// get previous version
	var older db.DBFile
	err = conn.Get(&older, "select content from entity_edit where entity_id = ? order by modified desc limit 1;", data.ID)

	// write new edit version
	_, err = conn.Exec("INSERT INTO entity_edit(entity_id, content, modified, user_id, previous, origin) VALUES(?, ?, ?, ?, ?, ?)", data.ID, data.Content, data.LastModTime, User.Root, older.Content, restore)

	if err != nil {
		log.Println(err)
		return out, err
	}

	return out, nil
}
