package utils

import (
	"fmt"
	//    "github.com/bmizerany/assert"
	"testing"
	//    "reflect"
)

var html_print bool = false

func TestHtml_input_text(t *testing.T) {
	test := Html_input_text("a", "b", "style='color:red'", "class='c'")
	if html_print {
		fmt.Println(test)
	}

	html_option := new(Html_Options)
	html_option.AddOption("a", "b", false)
	sele := Html_select("a", html_option, "id='a'")
	if html_print {
		fmt.Println(sele)
	}
	//    fmt.Println("type:",c.Kind())
	link := Html_link("http://www.baidu.com", "baidu-百度")
	if html_print {
		fmt.Println(link)
	}

	checkBox := Html_checkBox("name", "good", "haohaohao", false, "style='width:100px'")
	if html_print {
		fmt.Println(checkBox)
	}
}
