package serve

import (
	"fmt"
	"github.com/hidu/goutils"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type webFileInfo struct {
	Name         string
	RootDir      string
	IsDir        bool
	Size         int64
	Link         string
	fullPath     string
	file         *os.File
	subFileInfos []*webFileInfo
}

func newWebFileInfo(rootDir, name string) (*webFileInfo, error) {
	rootDir = filepath.Clean(rootDir + "/")
	fullPath := filepath.Clean(fmt.Sprintf("%s/%s", rootDir, name))
	if !strings.HasPrefix(fullPath, rootDir) {
		return nil, fmt.Errorf("unsafe path:%s", name)
	}
	f, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	info := &webFileInfo{
		Name:     strings.TrimLeft(fullPath[len(rootDir):], "/"),
		RootDir:  rootDir,
		IsDir:    stat.IsDir(),
		Size:     stat.Size(),
		fullPath: fullPath,
		file:     f,
	}
	info.Link = info.link()
	return info, nil
}

func (f *webFileInfo) String() string {
	return fmt.Sprintf("Name:%s\nRootDir:%s\nisDir:%v\nSize:%d\nfullPath:%s\n", f.Name, f.RootDir, f.IsDir, f.Size, f.fullPath)
}
func (f *webFileInfo) link() string {
	values := make(url.Values)
	values.Set("name", f.Name)
	if !f.IsDir {
		values.Set("op", "edit")
	}
	return "/file?" + values.Encode()
}
func (f *webFileInfo) getContent() string {
	if f.IsDir {
		return ""
	}
	data, err := ioutil.ReadAll(f.file)
	if err != nil {
		log.Println("read file failed:", err)
		return ""
	}
	return string(data)
}

func (f *webFileInfo) Close() {
	f.file.Close()
	if f.subFileInfos != nil {
		for _, info := range f.subFileInfos {
			info.Close()
		}
	}
}

func (f *webFileInfo) subFiles() ([]*webFileInfo, error) {
	names, err := f.file.Readdirnames(0)
	if err != nil {
		return nil, err
	}
	fileInfos := make([]*webFileInfo, 0)

	for _, name := range names {
		info, err := newWebFileInfo(f.RootDir, fmt.Sprintf("%s/%s", f.Name, name))
		if err != nil {
			log.Println("read file err,skip.name=", name, err)
		} else {
			fileInfos = append(fileInfos, info)
		}
	}
	f.subFileInfos = fileInfos
	return fileInfos, nil

}

func (ser *ProxyServe) getWebFilePath(name string) (fullPath string, nameNew string, err error) {
	rootDir := filepath.Clean(ser.conf.FileDir + "/")
	fullPath = filepath.Clean(fmt.Sprintf("%s/%s", rootDir, name))
	if !strings.HasPrefix(fullPath, rootDir) {
		return "", "", fmt.Errorf("unsafe path:%s", name)
	}
	nameNew = fullPath[len(rootDir):]
	re := regexp.MustCompile(`^[\w/\-\.]*$`)
	if !re.MatchString(nameNew) {
		err = fmt.Errorf("illegal path:%s", nameNew)
	}
	return fullPath, nameNew, err
}

func (ctx *webRequestCtx) handle_file() {
	if !ctx.isLogin {
		ctx.showError("need login")
		return
	}

	opMap := make(map[string]func())
	opMap["edit"] = ctx.handle_file_edit
	opMap["new"] = ctx.handle_file_new
	opMap["del"] = ctx.handle_file_del
	opMap["save"] = ctx.handle_file_save

	op := ctx.req.FormValue("op")
	if fn, has := opMap[op]; has {
		fn()
		return
	}
	name := ctx.req.FormValue("name")
	if !ctx.isAdmin && name == "" {
		name = ctx.user.Name
		dirFullPath, _, err := ctx.ser.getWebFilePath(name)
		if err != nil {
			ctx.showError("file dir wrong")
			return
		}
		if !utils.File_exists(dirFullPath) {
			os.MkdirAll(dirFullPath, os.ModePerm)
		}
	}

	dirInfo, err := newWebFileInfo(ctx.ser.conf.FileDir, name)
	if err != nil {
		ctx.showError("open file dir failed:" + name)
		return
	}
	defer dirInfo.Close()

	ctx.values["currentDir"] = dirInfo.Name
	ctx.values["isSubDir"] = dirInfo.Name != ""
	if !ctx.isAdmin && !strings.Contains(dirInfo.Name, "/") {
		ctx.values["isSubDir"] = false
	}
	files, err := dirInfo.subFiles()
	if err != nil {
		ctx.showError(err.Error())
		return
	}
	ctx.values["files"] = files

	ctx.render("file.html", true)
}

func (ctx *webRequestCtx) handle_file_edit() {
	name := ctx.req.FormValue("name")
	if name == "" {
		ctx.showError("params wrong")
		return
	}
	info, err := newWebFileInfo(ctx.ser.conf.FileDir, name)
	if err != nil {
		ctx.showError("read file info failed:" + err.Error())
		return
	}
	defer info.Close()
	if info.IsDir {
		ctx.showError("params wrong,only file can view")
		return
	}
	ctx.values["file"] = info
	fileContent := info.getContent()
	ctx.values["fileContent"] = fileContent
	ctx.values["fileContentRows"] = len(strings.Split(fileContent, "\n")) + 8
	ctx.render("file_edit.html", true)
}

func (ctx *webRequestCtx) handle_file_del() {

}

func (ctx *webRequestCtx) handle_file_new() {
	dirFullPath, dirNew, err := ctx.ser.getWebFilePath(ctx.req.FormValue("dir"))
	if err != nil {
		ctx.showErrorOrAlert("params err:" + err.Error())
		return
	}
	ctx.values["dir"] = dirNew

	if ctx.req.Method == "GET" {
		finfo, fErr := os.Stat(dirFullPath)
		if fErr != nil || !finfo.IsDir() {
			ctx.showErrorOrAlert("open dir failed")
			return
		}
		ctx.render("file_new.html", true)
	} else if ctx.req.Method == "POST" {
		name := strings.TrimSpace(ctx.req.FormValue("name"))
		if name == "" {
			ctx.jsAlert("empty filename")
			return
		}
		fpath := ctx.req.FormValue("dir") + "/" + ctx.req.FormValue("name")

		fileFullPath, fileName, err := ctx.ser.getWebFilePath(fpath)

		if err != nil {
			ctx.jsAlert("wrong fileName")
			return
		}

		if fileName == "" || strings.HasSuffix(fileName, "/") {
			ctx.jsAlert("wrong file name")
			return
		}

		if utils.File_exists(fileFullPath) {
			ctx.jsAlert("file already exists")
			return
		}

		if !strings.HasPrefix(fileFullPath, dirFullPath) {
			ctx.jsAlert("file name wrong")
			return
		}

		if !ctx.user.IsAdmin && !strings.HasPrefix(fileName+"/", "/"+ctx.user.Name+"/") {
			ctx.jsAlert("file path wrong:" + fileName)
			return
		}

		dirName := filepath.Dir(fileFullPath)
		if !utils.File_exists(dirName) {
			os.MkdirAll(dirName, os.ModePerm)
		}
		content := ctx.req.FormValue("content")

		wErr := utils.File_put_contents(fileFullPath, []byte(content))
		if wErr != nil {
			ctx.jsAlert("write file failed")
			return
		}

		finfo, _ := newWebFileInfo(ctx.ser.conf.FileDir, fpath)
		defer finfo.Close()

		ctx.jsAlertJump("save suc", finfo.link())
	}
}
func (ctx *webRequestCtx) handle_file_save() {
	nameOrigin := ctx.req.PostFormValue("nameOrigin")
	name := ctx.req.PostFormValue("name")
	content := ctx.req.PostFormValue("content")

	fullPath, nameFix, err := ctx.ser.getWebFilePath(name)
	if err != nil {
		ctx.jsAlert("file path wrong:" + err.Error())
		return
	}
	if name == "" || nameFix == "" || strings.HasSuffix(nameFix, "/") {
		ctx.jsAlert("wrong file name")
		return
	}
	fullPathOrigin, _, err := ctx.ser.getWebFilePath(nameOrigin)

	if fullPathOrigin == "" && err != nil {
		ctx.jsAlert("origin file path wrong:" + err.Error())
		return
	}

	dirName := filepath.Dir(fullPath)
	if !utils.File_exists(dirName) {
		os.MkdirAll(dirName, os.ModePerm)
	}

	errWrite := utils.File_put_contents(fullPath, []byte(content))
	if errWrite != nil {
		ctx.jsAlert("save failed:" + errWrite.Error())
		return
	}
	if fullPath != fullPathOrigin {
		os.Remove(fullPathOrigin)
	}
	info, _ := newWebFileInfo(ctx.ser.conf.FileDir, name)
	ctx.jsAlertJump("save suc", info.link())
}
