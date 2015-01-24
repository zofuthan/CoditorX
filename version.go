package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/b3log/wide/util"
)

// 暂时放在这里
func InitDocumentHolder() {
	documentHolder = newDocumentHolder()
}

func openDocHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{"succ": true}
	defer util.RetJSON(w, r, data)

	var args map[string]interface{}

	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		logger.Error(err)
		data["succ"] = false
		data["msg"] = "args decode error!"
		return
	}

	var fileName string
	fileNameArg := args["fileName"]
	if fileNameArg == nil {
		data["succ"] = false
		data["msg"] = "fileName can not bu nil."
		return
	}
	fileName = fileNameArg.(string)

	httpSession, _ := httpSessionStore.Get(r, "coditor-session")
	userSession := httpSession.Values[user_session]
	if nil == userSession {
		data["succ"] = false
		data["msg"] = "permission denied"
		return
	}

	user := userSession.(*User)
	// FIXME: doc permission check

	docName := filepath.Clean(fileName)

	doc := documentHolder.getDoc(docName)

	if doc == nil {
		metaData, err := newDocumentMetaData(docName)
		if err != nil {
			data["succ"] = false
			data["msg"] = err.Error()
			return
		}

		// FIXME: 设置为 owner 不对
		metaData.Owner = user.Username

		logger.Debugf("load doc [%s] into memory", docName)
		doc, err = newDocument(metaData, 10)
		if err != nil {
			data["succ"] = false
			data["msg"] = err.Error()
			return
		}

		documentHolder.setDoc(docName, doc)
	}

	docMap := make(map[string]interface{}, 0)
	version, err := doc.getVersion(user.Username)
	if err != nil {
		data["succ"] = false
		data["msg"] = err.Error()
		return
	}
	docMap["version"] = version
	content, err := doc.getContents(user.Username)
	if err != nil {
		data["succ"] = false
		data["msg"] = err.Error()
		return
	}
	docMap["content"] = content
	data["doc"] = docMap
}

func commitDocHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{"succ": true}
	defer util.RetJSON(w, r, data)

	var args map[string]interface{}

	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		logger.Error(err)
		data["succ"] = false
		data["msg"] = "args decode error!"
		return
	}

	fileArgs := args["file"].(map[string]interface{})
	if fileArgs == nil {
		data["succ"] = false
		data["msg"] = "args error!"
		return
	}

	fileName := fileArgs["name"].(string)
	fileContent := fileArgs["content"].(string)
	fileVersion := DocVersion(uint32(fileArgs["version"].(float64)))

	httpSession, _ := httpSessionStore.Get(r, "coditor-session")
	userSession := httpSession.Values[user_session]
	if nil == userSession {
		data["succ"] = false
		data["msg"] = "permission denied"
		return
	}

	user := userSession.(*User)
	// FIXME: doc permission check

	docName := filepath.Clean(fileName)

	doc := documentHolder.getDoc(docName)
	if doc == nil {
		data["succ"] = false
		data["msg"] = "document not exist!"
		return
	}

	patchsStr, version, err := doc.merge(fileContent, fileVersion, user.Username)
	if err != nil {
		data["succ"] = false
		data["msg"] = "document not exist!"
		return
	}

	output := make(map[string]interface{})
	output["patchs"] = patchsStr
	output["version"] = version

	data["output"] = output
}

func fetchDocHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{"succ": true}
	defer util.RetJSON(w, r, data)

	var args map[string]interface{}

	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		logger.Error(err)
		data["succ"] = false
		data["msg"] = "args decode error!"
		return
	}

	fileArgs := args["file"].(map[string]interface{})
	if fileArgs == nil {
		data["succ"] = false
		data["msg"] = "args error!"
		return
	}

	fileName := fileArgs["name"].(string)
	fileVersion := DocVersion(uint32(fileArgs["version"].(float64)))

	httpSession, _ := httpSessionStore.Get(r, "coditor-session")
	userSession := httpSession.Values[user_session]
	if nil == userSession {
		data["succ"] = false
		data["msg"] = "permission denied"
		return
	}

	user := userSession.(*User)
	// FIXME: doc permission check

	docName := filepath.Clean(fileName)

	doc := documentHolder.getDoc(docName)
	if doc == nil {
		data["succ"] = false
		data["msg"] = "document not exist!"
		return
	}

	patchss, err := doc.tail(fileVersion, user.Username)
	if err != nil {
		data["succ"] = false
		data["msg"] = err.Error()
		return
	}

	data["patchss"] = patchss
	version, err := doc.getVersion(user.Username)
	if err != nil {
		data["succ"] = false
		data["msg"] = err.Error()
		return
	}
	data["version"] = version
}

func getHeadDocHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{"succ": true}
	defer util.RetJSON(w, r, data)

	var args map[string]interface{}

	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		logger.Error(err)
		data["succ"] = false
		data["msg"] = "args decode error!"
		return
	}

	var fileName string
	fileNameArg := args["fileName"]
	if fileNameArg == nil {
		data["succ"] = false
		data["msg"] = "fileName can not bu nil."
		return
	}

	fileName = fileNameArg.(string)

	httpSession, _ := httpSessionStore.Get(r, "coditor-session")
	userSession := httpSession.Values[user_session]
	if nil == userSession {
		data["succ"] = false
		data["msg"] = "permission denied"
		return
	}

	user := userSession.(*User)
	// FIXME: doc permission check

	docName := filepath.Clean(fileName)

	doc := documentHolder.getDoc(docName)
	if doc == nil {
		data["succ"] = false
		data["msg"] = "document not exist!"
		return
	}

	output := make(map[string]interface{})
	content, err := doc.getContents(user.Username)
	if err != nil {
		data["succ"] = false
		data["msg"] = err.Error()
		return
	}

	output["content"] = content
	version, _ := doc.getVersion(user.Username)
	output["version"] = version

	data["output"] = output
}