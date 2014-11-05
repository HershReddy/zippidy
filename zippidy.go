package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"html/template"
	"io/ioutil"
	"net"
	"net/http"
	"os"
)

type Imagemap map[string][]byte
type Zipmap map[string]Imagemap

var files []os.FileInfo
var zipmap Zipmap

var imageroot = flag.String("rootdir", "images", "root directory containing image archives")

func dirHandler(w http.ResponseWriter, r *http.Request) {
	dirtemplate, err := template.ParseFiles("dir.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type fileDesc struct{ Filename string }
	tempdata := make([]fileDesc, 1)

	for _, file := range files {
		tempdata = append(tempdata, fileDesc{Filename: file.Name()})
	}

	if err = dirtemplate.Execute(w, tempdata); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func imageHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	imgfilename := vars["imgfile"]
	zipfilename := vars["zipfile"]
	fmt.Println(imgfilename)
	fmt.Println(zipfilename)

	w.Header().Set("ContentType", "image/jpeg")
	w.Write(zipmap[zipfilename][imgfilename])
}

func zipHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	zipfilename := vars["zipfile"]
	var imagemap Imagemap
	var ok bool
	if imagemap, ok = zipmap[zipfilename]; !ok {
		// Open a zip archive for reading.
		rz, err := zip.OpenReader(*imageroot + "/" + zipfilename)
		if err != nil {
			fmt.Printf("Oops: %v\n", err)
			return
		}
		defer rz.Close()

		zipmap[zipfilename] = make(Imagemap)
		imagemap = zipmap[zipfilename]

		// Iterate through the files in the archive,
		// printing some of their contents.
		for _, f := range rz.File {
			if _, ok := imagemap[f.Name]; !ok {
				rc, err := f.Open()
				if err != nil {
					fmt.Printf("Oops: %v\n", err)
					return
				}
				imagemap[f.Name], _ = ioutil.ReadAll(rc)
				rc.Close()
			}
		}
	}

	zipdirtemplate, err := template.ParseFiles("zipdir.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	imagenames := make([]string, 1)

	for imagename := range imagemap {
		imagenames = append(imagenames, imagename)
	}
	tempdata := struct {
		Zipfile    string
		Imagenames []string
	}{
		Zipfile:    zipfilename,
		Imagenames: imagenames,
	}

	if err = zipdirtemplate.Execute(w, tempdata); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func printAddress() {
	name, err := os.Hostname()
	if err != nil {
		fmt.Printf("Oops: %v\n", err)
		return
	}

	addrs, err := net.LookupHost(name)
	if err != nil {
		fmt.Printf("Oops: %v\n", err)
		return
	}

	for _, a := range addrs {
		fmt.Println(a)
	}
}

func initFiles() {
	var err error
	files, err = ioutil.ReadDir(*imageroot)
	if err != nil {
		fmt.Printf("Oops: %v\n", err)
		return
	}
	zipmap = make(Zipmap)
}

func main() {
	flag.Parse()
	fmt.Printf("Root directory is: %v \n", *imageroot)
	fmt.Printf("Server started. Hit Control-C to exit.\n Network addresses for this host are: \n")
	printAddress()
	initFiles()

	r := mux.NewRouter()
	r.HandleFunc("/dir", dirHandler)
	r.HandleFunc("/dir/{zipfile}/{imgfile}", imageHandler)
	r.HandleFunc("/dir/{zipfile}", zipHandler)

	http.Handle("/", r)
	http.ListenAndServe(":8080", nil)
}
