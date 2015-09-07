package utils

import (
	"fmt"
	"os"
	"testing"
)

func TestFile_get_contents(t *testing.T) {
	res, _ := File_get_contents("fs.go")
	if len(res) == 0 {
		t.FailNow()
	}
	md5_str, err := File_Md5("./fs.go")
	fmt.Println("fs.go md5:", md5_str, err)
}
func TestFile_put_contents(t *testing.T) {
	test_data := "hello"
	if File_exists("aaa") {
		os.Remove("aaa")
	}
	File_put_contents("aaa", []byte(test_data))
	res, _ := File_get_contents("aaa")
	if string(res) != test_data {
		t.FailNow()
	}
	File_put_contents("aaa", []byte("nihao"), FILE_APPEND)
	res, _ = File_get_contents("aaa")
	if string(res) != test_data+"nihao" {
		t.FailNow()
	}
	os.Remove("aaa")
}
