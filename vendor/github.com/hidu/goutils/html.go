package utils

import (
	"fmt"
	"html/template"
	"regexp"
)

func Html_input_tag(tagType string, name string, value string, other_params ...interface{}) (html string) {
	params := make(map[string]string)
	if len(name) > 0 {
		params["name"] = name
	}
	params["type"] = tagType
	params["value"] = value
	params = params_merge(params, other_params)
	html = "<input " + paramsAsString(params) + "/>"
	return
}

func paramsAsString(more_params ...interface{}) string {
	params := make(map[string]string)
	params = params_merge(params, more_params)
	html := ""
	for k, v := range params {
		html += " " + k + `="` + template.HTMLEscapeString(v) + `"`
	}
	return html
}

func params_merge(params map[string]string, more_params []interface{}) map[string]string {
	for _, param := range more_params {
		switch param.(type) {
		case map[string]string:
			for k, v := range param.(map[string]string) {
				params[k] = v
			}
		case string:
			_params := StringToMap(fmt.Sprint(param))
			for k, v := range _params {
				params[k] = v
			}
		}
	}
	return params
}

func Html_input_text(name string, value string, other_params ...interface{}) string {
	return Html_input_tag("text", name, value, other_params...)
}

func Html_input_hidden(name string, value string, other_params ...interface{}) string {
	return Html_input_tag("hidden", name, value, other_params...)
}
func Html_input_password(name string, value string, other_params ...interface{}) string {
	return Html_input_tag("password", name, value, other_params...)
}
func Html_input_email(name string, value string, other_params ...interface{}) string {
	return Html_input_tag("email", name, value, other_params...)
}
func Html_input_url(name string, value string, other_params ...interface{}) string {
	return Html_input_tag("url", name, value, other_params...)
}
func Html_input_search(name string, value string, other_params ...interface{}) string {
	return Html_input_tag("search", name, value, other_params...)
}
func Html_input_file(name string, value string, other_params ...interface{}) string {
	return Html_input_tag("file", name, value, other_params...)
}
func Html_input_submit(value string, other_params ...interface{}) string {
	return Html_input_tag("submit", "", value, other_params...)
}
func Html_input_reset(value string, other_params ...interface{}) string {
	return Html_input_tag("reset", "", value, other_params...)
}

func Html_link(url string, text string, more_params ...interface{}) string {
	params := make(map[string]string)
	params["href"] = url
	params["title"] = text
	html := "<a " + paramsAsString(params_merge(params, more_params)) + ">" + template.HTMLEscapeString(text) + "</a>"
	return html
}

func Html_checkBox(name string, value string, label string, isChecked bool, other_params ...interface{}) string {
	params := make(map[string]string)
	if isChecked {
		params["checked"] = "checked"
	}
	params = params_merge(params, other_params)
	html := "<label>" + Html_input_tag("checkbox", name, value, params) + template.HTMLEscapeString(label) + "</label>"
	return html
}

func Html_datalist(id string, values []string) string {
	html := "<datalist id='" + id + "'>"
	for _, v := range values {
		html += "<option value='" + template.HTMLEscapeString(v) + "'>"
	}
	html += "</datalist>"
	return html
}

func Html_textArea(name string, value string, more_params ...interface{}) string {
	params := make(map[string]string)
	params["name"] = name
	params["value"] = value
	html := "<textarea" + paramsAsString(params_merge(params, more_params)) + ">" + template.HTMLEscapeString(value) + "</textarea>"
	return html
}

type Html_Options struct {
	Items []*html_option
}

type html_option struct {
	Name    interface{}
	Value   interface{}
	Checked bool
	Params  map[string]string
}

func NewHtml_Options() *Html_Options {
	return new(Html_Options)
}

func (options *Html_Options) AddOption(name interface{}, value interface{}, checked bool) {
	option := new(html_option)
	option.Name = name
	option.Value = value
	option.Checked = checked
	options.Items = append(options.Items, option)
}

func Html_select(name string, options *Html_Options, other_params ...interface{}) string {
	params := make(map[string]string)
	params["name"] = name
	html := "<select" + paramsAsString(params_merge(params, other_params)) + ">\n"
	for _, option := range options.Items {
		option_fmt := "<option value='%s'%s>%s</option>\n"
		select_str := ""
		if option.Checked {
			select_str = " selected='selected'"
		}
		name := fmt.Sprintf("%s", option.Name)
		value := fmt.Sprintf("%s", option.Value)
		option_str := fmt.Sprintf(option_fmt, template.HTMLEscapeString(name), select_str, template.HTMLEscapeString(value))
		html += option_str
	}
	html += "</select>"
	return html
}

var html_tag_reg *regexp.Regexp = regexp.MustCompile(`>\s+<`)

func Html_reduceSpace(html string) string {
	return html_tag_reg.ReplaceAllString(html, "><")
}
