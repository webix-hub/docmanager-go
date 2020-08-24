package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/xbsoftware/wfs"
)

var nonLatin = regexp.MustCompile(("[[:^ascii:]]"))

func getIconURL(size, ftype, name string) string {
	var re = regexp.MustCompile(`[^A-Za-z0-9.]`)

	size = re.ReplaceAllString(size, "")
	name = "icons/" + size + "/" + re.ReplaceAllString(name, "")
	ftype = "icons/" + size + "/types/" + re.ReplaceAllString(ftype, "")

	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		name = ftype + filepath.Ext(name)
	}

	return name
}

func serveIconPreview(w http.ResponseWriter, r *http.Request, info wfs.File) {
	http.ServeFile(w, r, getIconURL("big", info.Type, filepath.Ext(info.Name)[1:]+".svg"))
}

func getFilePreview(w http.ResponseWriter, r *http.Request) {
	if Config.Preview == "none" {
		format.Text(w, 500, "Previews not configured")
		return
	}

	id := r.URL.Query().Get("id")
	info, err := drive.Info(id)
	if err != nil {
		format.Text(w, 500, "Access denied")
		return
	}

	widthStr := r.URL.Query().Get("width")
	heightStr := r.URL.Query().Get("height")
	width, err := strconv.Atoi(widthStr)
	if err != nil {
		format.Text(w, 500, "incorrect width value")
		return
	}
	height, err := strconv.Atoi(heightStr)
	if err != nil {
		format.Text(w, 500, "incorrect height value")
		return
	}

	if info.Size > 50*1000*1000 || width > 2000 || height > 2000 {
		// file is too large, still it is a valid use-case so return some image
		serveIconPreview(w, r, info)
		return
	}

	target := getImagePreviewName(Config.DataFolder, id, widthStr, heightStr)

	// check previously generated preview
	ext := ".jpg"
	ps, err := os.Stat(target + ext)
	if err != nil {
		ext = ".png"
		ps, err = os.Stat(target + ext)
	}
	if err == nil {
		if ps.Size() == 0 {
			// there is a preview placeholder, which means preview can't be generated for this file
			serveIconPreview(w, r, info)
			return
		} else {
			http.ServeFile(w, r, target+ext)
		}
		return
	}

	source, _ := drive.Read(id)
	if x, ok := source.(io.Closer); ok {
		defer x.Close()
	}

	if Config.Preview != "" {
		ext, err = getExternalPreview(source, target, info.Name, width, height)
	} else {
		if info.Type == "image" {
			ext, err = getImagePreview(source, target, info.Name, width, height)
		}
	}

	if err != nil {
		log.Print(err.Error())
		ioutil.WriteFile(target+".jpg", []byte{}, 0664)
		serveIconPreview(w, r, info)
		return
	}
	http.ServeFile(w, r, target+ext)
}

func getImagePreviewName(base, id, width, height string) string {
	//folder := filepath.Join(base, filepath.Dir(id), ".preview")
	err := os.MkdirAll("/tmp/preview", 0777)
	if err != nil {
		log.Println("Can't create folder for previews")
	}
	return filepath.Join("/tmp/preview", width+"x"+height+"___"+strings.Replace(id, "/", "___", -1))
	// return filepath.Join(folder, filepath.Base(id)+"___"+width+"x"+height)
}
func getImagePreview(source io.Reader, target, name string, width, height int) (string, error) {
	src, err := imaging.Decode(source)
	if err != nil {
		return "", err
	}

	ext := strings.ToLower(filepath.Ext(name))
	targetExt := ".jpg"
	if ext == ".png" || ext == ".gif" {
		targetExt = ".png"
	}
	dst := imaging.Thumbnail(src, width, height, imaging.Lanczos)
	err = imaging.Save(dst, target+targetExt)

	return targetExt, err
}

func getExternalPreview(source io.ReadSeeker, target, name string, width, height int) (string, error) {
	body, writer := io.Pipe()
	defer body.Close()

	form := multipart.NewWriter(writer)
	safeName := nonLatin.ReplaceAllLiteralString(name,"x")

	go func() {
		defer writer.Close()
		defer form.Close()

		fw, err := form.CreateFormField("width")
		if err != nil {
			log.Println(err.Error())
			return
		}
		io.Copy(fw, bytes.NewBufferString(strconv.Itoa(width)))

		fw, err = form.CreateFormField("height")
		if err != nil {
			log.Println(err.Error())
			return
		}
		io.Copy(fw, bytes.NewBufferString(strconv.Itoa(height)))

		fw, err = form.CreateFormField("name")
		if err != nil {
			log.Println(err.Error())
			return
		}
		io.Copy(fw, bytes.NewBufferString(safeName))

		fw, err = form.CreateFormFile("file", safeName)
		if err != nil {
			log.Println(err.Error())
			return
		}
		io.Copy(fw, source)
	}()

	req, err := http.NewRequest(http.MethodPost, Config.Preview, body)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", form.FormDataContentType())

	client := &http.Client{}
	res, err := client.Do(req)

	if err != nil {
		return "", fmt.Errorf("preview service %w", err)
	}
	if res.StatusCode != 200 {
		return "", fmt.Errorf("preview service %d", res.StatusCode)
	}
	ext := ".jpg"
	if res.Header.Get("Content-type") == "image/png" {
		ext = ".png"
	}

	defer res.Body.Close()
	fw, err := os.Create(target + ext)
	if err != nil {
		return "", err
	}
	defer fw.Close()
	_, err = io.Copy(fw, res.Body)

	return ext, err
}
