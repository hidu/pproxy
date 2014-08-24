package serve

import (
	"fmt"
	"github.com/hidu/goutils"
	"github.com/robertkrimen/otto"
	"io/ioutil"
	"log"
	"strings"
)

var rewriteJsTpl = string(utils.DefaultResource.Load("res/sjs/req_rewrite.js"))

type requestModifier struct {
	jsVm   *otto.Otto
	jsPath string
	jsStr  string
	jsFn   otto.Value
	canMod bool
}

func NewRequestModifier(jsPath string) *requestModifier {
	reqMod := &requestModifier{
		jsVm:   otto.New(),
		jsPath: jsPath,
	}
	return reqMod
}

func (reqMod *requestModifier) tryLoadJs() (err error) {
	if utils.File_exists(reqMod.jsPath) {
		script, jsErr := ioutil.ReadFile(reqMod.jsPath)
		if jsErr == nil {
			jsErr = reqMod.parseJs(string(script), false)
			if jsErr != nil {
				fmt.Println("load rewrite js failed:", jsErr)
				return jsErr
			}
		}
	}
	return nil
}
func (reqMod *requestModifier) CanMod() bool {
	return reqMod.canMod
}

func (reqMod *requestModifier) parseJs(jsStr string, save2File bool) error {
	rewriteJs := strings.Replace(rewriteJsTpl, "CUSTOM_JS", jsStr, 1)
	reqMod.jsVm.Run(rewriteJs)
	jsFn, err := reqMod.jsVm.Get("pproxy_rewrite")
	if err != nil {
		log.Println("rewrite js init error:", err)
		return err
	}
	reqMod.jsStr = jsStr
	reqMod.jsFn = jsFn
	reqMod.canMod = true
	if save2File {
		err = utils.File_put_contents(reqMod.jsPath, []byte(jsStr))
		log.Println("save rewritejs ", reqMod.jsPath, err)
	}
	return err
}

func (reqMod *requestModifier) rewrite(data map[string]interface{}) (map[string]interface{}, error) {
	reqJsObj, _ := reqMod.jsVm.Object(`req={}`)
	reqJsObj.Set("origin", data)
	js_ret, err_js := reqMod.jsFn.Call(reqMod.jsFn, reqJsObj)

	if err_js != nil {
		return nil, err_js
	}
	if !js_ret.IsObject() {
		log.Println("wrong req_rewirte return value")
		return nil, fmt.Errorf("wrong req_rewirte return value,not object.%t", js_ret)
	}
	obj, export_err := js_ret.Export()

	if export_err != nil {
		return nil, export_err
	}
	reqObjNew := obj.(map[string]interface{})
	return reqObjNew, nil
}
