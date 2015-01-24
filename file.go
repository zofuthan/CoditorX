package main

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"strconv"

	"github.com/b3log/wide/util"
	"encoding/json"
)

type file struct {
	ID      string `json:"id"`
	IsShare bool   `json:"isShare"`
	Name    string `json:"name"`
	Type    string `json:"type"`
}

func fileTreeHandler(w http.ResponseWriter, r *http.Request) {
	// XXX: it's a list now, not a tree

	data := map[string]interface{}{"succ": true}
	defer util.RetJSON(w, r, data)

	httpSession, _ := httpSessionStore.Get(r, "coditor-session")
	userSession := httpSession.Values[user_session]
	if nil == userSession {
		data["succ"] = false
		data["msg"] = "permission denied"
		return
	}

	user := userSession.(*User)
	names := listFiles(user.getWorkspace())
	files := []*file{}
	for i, fname := range names {
		t := filepath.Ext(fname)
		if strings.HasPrefix(t, ".") {
			t = t[1:]
		}

		f := &file{ID: strconv.Itoa(i), IsShare: false, Name: fname, Type: t}

		files = append(files, f)
	}

	data["files"] = files
}

// listFiles lists names of files under the specified dirname.
func listFiles(dirname string) []string {
	f, _ := os.Open(dirname)

	names, _ := f.Readdirnames(-1)
	f.Close()

	sort.Strings(names)

	dirs := []string{}
	files := []string{}

	// sort: directories in front of files
	for _, name := range names {
		path := filepath.Join(dirname, name)
		fio, err := os.Lstat(path)

		if nil != err {
			logger.Warnf("Can't read file info [%s]", path)

			continue
		}

		if strings.HasSuffix(name, ".coditor") { // skip Coditor meta-data files
			continue
		}

		if fio.IsDir() {
			dirs = append(dirs, name)
		} else {
			files = append(files, name)
		}
	}

	return append(dirs, files...)
}

func fileNew(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{"succ": true}
	defer util.RetJSON(w, r, data)

	httpSession, _ := httpSessionStore.Get(r, "coditor-session")
	userSession := httpSession.Values[user_session]
	if nil == userSession {
		data["succ"] = false
		data["msg"] = "permission denied"
		return
	}

	user := userSession.(*User)

	var args map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		logger.Error(err)
		data["succ"] = false
		data["msg"] = "args decode error!"
		return
	}

	fileName := args["name"]
	if fileName == nil || len(fileName.(string)) == 0 {
		data["succ"] = false
		data["msg"] = "fileName can not be null."
		return
	}

	// create file
	path := filepath.Join(conf.Workspace, user.Username, "workspace", fileName.(string))
	file, err := os.Create(path)
	if err != nil {
		logger.Error(err)
		data["succ"] = false
		data["msg"] = err.Error()
		return
	}
	file.Close()
	// create json file
	dmd, err := newDocumentMetaData(path)
	if err != nil{
		logger.Error(err)
		data["succ"] = false
		data["msg"] = err.Error()
		return
	}
	dmd.Owner = user.Username
	err = dmd.save()
	if err != nil{
		logger.Error(err)
		data["succ"] = false
		data["msg"] = err.Error()
		return
	}
	// create log files
	path = path + ".logs.coditor"
	file, err = os.Create(path)
	if err != nil {
		logger.Error(err)
		data["succ"] = false
		data["msg"] = err.Error()
		return
	}
	file.Close()
}

func fileRename(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{"succ": true}
	defer util.RetJSON(w, r, data)

	httpSession, _ := httpSessionStore.Get(r, "coditor-session")
	userSession := httpSession.Values[user_session]
	if nil == userSession {
		data["succ"] = false
		data["msg"] = "permission denied"
		return
	}

	user := userSession.(*User)

	var args map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		logger.Error(err)
		data["succ"] = false
		data["msg"] = "args decode error!"
		return
	}

	oldName := args["oldName"]
	if oldName == nil || len(oldName.(string)) == 0 {
		data["succ"] = false
		data["msg"] = "oldName can not be null."
		return
	}

	newName := args["newName"]
	if newName == nil || len(newName.(string)) == 0 {
		data["succ"] = false
		data["msg"] = "newName can not be null."
		return
	}

	// create file
	oldPath := filepath.Join(conf.Workspace, user.Username, "workspace", oldName.(string))
	newPath := filepath.Join(conf.Workspace, user.Username, "workspace", newName.(string))
	err := os.Rename(oldPath, newPath)
	if err != nil {
		logger.Error(err)
		data["succ"] = false
		data["msg"] = err.Error()
		return
	}

	oldJsonPath := oldPath + ".json.coditor"
	newJsonPath := newPath + ".json.coditor"
	err = os.Rename(oldJsonPath, newJsonPath)
	if err != nil {
		logger.Error(err)
		data["succ"] = false
		data["msg"] = err.Error()
		return
	}

	oldLogPath := oldPath + ".logs.coditor"
	newLogPath := newPath + ".logs.coditor"
	err = os.Rename(oldLogPath, newLogPath)
	if err != nil {
		logger.Error(err)
		data["succ"] = false
		data["msg"] = err.Error()
		return
	}

	// FIXME: change document name if exist.
	dmd, err := newDocumentMetaData(newPath)
	if err != nil {
		logger.Error(err)
		data["succ"] = false
		data["msg"] = err.Error()
		return
	}
	shares := dmd.Editors
	shares = append(shares, dmd.Viewers...)
	newShare := &Share{}
	newShare.Owner = user.Username
	newShare.DocName = newName.(string)
	oldShare := &Share{}
	oldShare.Owner = user.Username
	oldShare.DocName = oldName.(string)
	for _, s := range shares {
		err = checkAndUpdate(s, oldShare, newShare)
		// just log err!
		logger.Error(err)
	}
}

func fileDel(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{"succ": true}
	defer util.RetJSON(w, r, data)

	httpSession, _ := httpSessionStore.Get(r, "coditor-session")
	userSession := httpSession.Values[user_session]
	if nil == userSession {
		data["succ"] = false
		data["msg"] = "permission denied"
		return
	}

	user := userSession.(*User)

	var args map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		logger.Error(err)
		data["succ"] = false
		data["msg"] = "args decode error!"
		return
	}

	fileName := args["name"]
	if fileName == nil || len(fileName.(string)) == 0 {
		data["succ"] = false
		data["msg"] = "fileName can not be null."
		return
	}

	// FIXME: del document if exist.

	path := filepath.Join(conf.Workspace, user.Username, "workspace", fileName.(string))
	// del shares!
	dmd, err := newDocumentMetaData(path)
	if err != nil {
		logger.Error(err)
		data["succ"] = false
		data["msg"] = err.Error()
		return
	}
	shares := dmd.Editors
	shares = append(shares, dmd.Viewers...)
	share := &Share{}
	share.Owner = user.Username
	share.DocName = fileName.(string)
	for _, s := range shares {
		err = checkAndDel(s, share)
		if err != nil {
			// Just log err!
			logger.Error(err)
		}
	}

	// del file
	err = os.Remove(path)
	if err != nil {
		logger.Error(err)
		data["succ"] = false
		data["msg"] = err.Error()
		return
	}
	// del json file
	jsonPath := path + ".json.coditor"
	err = os.Remove(jsonPath)
	if err != nil{
		logger.Error(err)
		data["succ"] = false
		data["msg"] = err.Error()
		return
	}
	// del log files
	logPath := path + ".logs.coditor"
	err = os.Remove(logPath)
	if err != nil {
		logger.Error(err)
		data["succ"] = false
		data["msg"] = err.Error()
		return
	}
}