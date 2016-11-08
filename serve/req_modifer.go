package serve

import (
	"fmt"
	"github.com/hidu/goutils"
	"github.com/robertkrimen/otto"
	"io/ioutil"
	"log"
	"strings"
	"sync"
)

var rewriteJsTpl = Assest.GetContent("res/sjs/req_rewrite.js")

/**
*request动态修改引擎
*使用javascript 来对请求进行修改
 */
type requestModifier struct {
	mu     sync.RWMutex
	jsVm   *otto.Otto
	jsFns  map[string]*otto.Value
	canMod bool
	ser    *ProxyServe
}

func NewRequestModifier(ser *ProxyServe) *requestModifier {
	reqMod := &requestModifier{
		jsVm:  otto.New(),
		jsFns: make(map[string]*otto.Value),
		ser:   ser,
	}
	return reqMod
}
func (reqMod *requestModifier) getJsPath(name string) string {
	baseName := fmt.Sprintf("%s/req_rewrite_%d", reqMod.ser.configDir, reqMod.ser.conf.Port)
	if name == "" {
		return fmt.Sprintf("%s.js", baseName)
	} else {
		return fmt.Sprintf("%s_%s.js", baseName, name)
	}
}

func (reqMod *requestModifier) tryLoadJs(name string) (err error) {
	jsContent, err := reqMod.getJsContent(name)
	if jsContent != "" && err == nil {
		err = reqMod.parseJs(jsContent, name, false)
		if err != nil {
			log.Println("load rewrite js failed:", err)
			return err
		}
		log.Println("load rewrite js[", name, "] suc")
	}
	return nil
}

func (reqMod *requestModifier) loadAllJs() error {
	if !reqMod.ser.conf.ModifyRequest {
		log.Println("ignore requestModifier loadAllJs")
		return nil
	}
	names := []string{""}
	for _, user := range reqMod.ser.Users {
		names = append(names, user.Name)
	}
	for _, name := range names {
		err := reqMod.tryLoadJs(name)
		if err != nil {
			return err
		}
	}
	return nil
}

func (reqMod *requestModifier) getJsContent(name string) (content string, err error) {
	jsPath := reqMod.getJsPath(name)
	if utils.File_exists(jsPath) {
		script, err := ioutil.ReadFile(jsPath)
		if err == nil {
			return string(script), nil
		} else {
			return "", err
		}
	}
	return "", nil
}

func (reqMod *requestModifier) CanMod() bool {
	return reqMod.canMod
}

func (reqMod *requestModifier) parseJs(jsStr string, name string, save2File bool) error {
	jsStr = strings.TrimSpace(jsStr)
	rewriteJs := strings.Replace(rewriteJsTpl, "CUSTOM_JS", jsStr, 1)
	rewriteJs = strings.Replace(rewriteJs, "PPROXY_HOST", fmt.Sprintf("127.0.0.1:%d", reqMod.ser.conf.Port), 1)

	reqMod.mu.Lock()
	defer reqMod.mu.Unlock()
	if reqMod.ser.Debug {
		log.Println("jsvm_execute:", rewriteJs)
	}
	reqMod.jsVm.Run(rewriteJs)
	jsFn, err := reqMod.jsVm.Get("pproxy_rewrite")
	if err != nil {
		log.Println("rewrite js init error:", err)
		return err
	}

	if strings.HasPrefix(jsStr, "//ignore") {
		if _, has := reqMod.jsFns[name]; has {
			delete(reqMod.jsFns, name)
		}
		log.Println("req_mod [", name, "] ignore")
	} else {
		reqMod.jsFns[name] = &jsFn
		log.Println("req_mod [", name, "] register suc")
	}
	reqMod.canMod = true
	if save2File {
		jsPath := reqMod.getJsPath(name)
		err = utils.File_put_contents(jsPath, []byte(jsStr))
		log.Println("save rewritejs ", jsPath, err)
	}
	return err
}
func (reqMod *requestModifier) getJsFnByName(name string) (*otto.Value, error) {
	names := []string{name, ""}
	for _, name := range names {
		if jsFn, has := reqMod.jsFns[name]; has {
			return jsFn, nil
		}
	}
	return nil, fmt.Errorf("no rewrite rules")
}

func (reqMod *requestModifier) rewrite(data map[string]interface{}, name string) (map[string]interface{}, error) {

	reqMod.mu.Lock()
	defer reqMod.mu.Unlock()

	reqJsObj, _ := reqMod.jsVm.Object(`req={}`)
	reqJsObj.Set("origin", data)

	jsFn, err := reqMod.getJsFnByName(name)

	if err != nil {
		return nil, err
	}

	defer func() {
		if caught := recover(); caught != nil {
			log.Println("fatal:requestModifer  recover:", caught)
		}
	}()

	js_ret, err_js := (*jsFn).Call(*jsFn, reqJsObj)

	if err_js != nil {
		log.Println("parse js error:", err_js)
		return nil, err_js
	}
	if !js_ret.IsObject() {
		log.Println("wrong req_rewirte return value,not object:", js_ret)
		return nil, fmt.Errorf("wrong req_rewirte return value,not object.%t", js_ret)
	}
	obj, export_err := js_ret.Export()

	if export_err != nil {
		return nil, export_err
	}
	reqObjNew := obj.(map[string]interface{})
	return reqObjNew, nil
}
