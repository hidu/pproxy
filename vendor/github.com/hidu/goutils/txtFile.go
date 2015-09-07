package utils

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"
)

type TxtLine struct {
	No      int
	Str     string
	Comment string
	Origin  string
}

func (line *TxtLine) String() string {
	if line.Comment == "" {
		return line.Str
	} else {
		return fmt.Sprintf("%s #%s", line.Str, line.Comment)
	}
}

func (line *TxtLine) Empty() bool {
	return line.Str == ""
}

func (line *TxtLine) Slice() []string {
	return regSpace.Split(line.Str, -1)
}

func (line *TxtLine) KvMap(splitStr string) (result map[string]string, err error) {
	if line.Empty() {
		return
	}
	result = make(map[string]string)
	slice := line.Slice()
	for _, str := range slice {
		arr := strings.SplitN(str, splitStr, 2)
		if len(arr) == 2 {
			result[strings.TrimSpace(arr[0])] = strings.TrimSpace(arr[1])
		} else {
			err = fmt.Errorf("lineNo:%d,[%s],not k:v", line.No, line.Str)
			return nil, err
		}
	}
	return result, nil
}

type TxtFile struct {
	Lines []*TxtLine
}

func NewTxtFile(path string) (*TxtFile, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return NewTxtFileFromString(string(data)), nil
}

func NewTxtFileFromString(text string) *TxtFile {
	linesObj := []*TxtLine{}
	lines := strings.Split(text, "\n")
	for lineNo, lineStr := range lines {
		line := &TxtLine{No: lineNo + 1, Origin: lineStr}

		lineStr = strings.TrimSpace(lineStr)
		index := strings.IndexByte(lineStr, '#')
		if index > -1 {
			line.Str = strings.TrimSpace(string(lineStr[:index]))
			line.Comment = strings.TrimSpace(string(lineStr[index+1:]))
		} else {
			line.Str = lineStr
		}
		linesObj = append(linesObj, line)
	}
	txtFile := &TxtFile{Lines: linesObj}
	return txtFile
}

func (txt *TxtFile) KvMapSlice(splitStr string, ignoreError bool, fields map[string]string) (result []map[string]string, err error) {
	for _, line := range txt.Lines {
		if line.Empty() {
			continue
		}
		kv, err := line.KvMap(splitStr)
		if err != nil {
			if ignoreError {
				log.Println("parse failed,ignore,", err)
				continue
			} else {
				return nil, err
			}
		}

		if kv == nil || len(kv) == 0 {
			continue
		}

		for k, v := range fields {
			cur_v, has := kv[k]
			if !has || cur_v == "" {
				if ignoreError {
					if v == "required" {
						log.Println("ignore line:", line.Origin, "miss required field:", k)
						kv = nil
						break
					} else {
						kv[k] = v
					}
				} else {
					return nil, fmt.Errorf("fields miss at line:", line.Origin)
				}
			}
		}
		if kv != nil {
			result = append(result, kv)
		}
	}
	return result, nil
}
