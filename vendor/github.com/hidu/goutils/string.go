package utils

import (
	"crypto/md5"
	"fmt"
	"log"
	"regexp"
	"strings"
)

/**
* parse str
    style='width:1' class="hello" checked=on
  as
  map[style:width:1 class:hello checked:on]
*/
func StringToMap(str string) (data map[string]string) {
	re := regexp.MustCompile(`\s*([\w-]+)\s*=\s*(['"]?)(.*)`)
	data = make(map[string]string)
	matches := re.FindAllStringSubmatch(str, -1)
	if len(matches) > 0 {
		first := matches[0]
		var reg2_txt string
		if first[2] == "'" || first[2] == `"` {
			reg2_txt = fmt.Sprintf(`([^%s]*)%s(\s+.*)?`, first[2], first[2])
		} else if first[2] == "" {
			reg2_txt = `(\S+)\s*(.*)`
		}
		re2 := regexp.MustCompile(reg2_txt)
		subResult := re2.FindAllStringSubmatch(first[3], -1)

		if len(subResult) > 0 && len(subResult[0]) > 1 {
			data[first[1]] = subResult[0][1]
			if len(subResult[0][2]) > 0 {
				_subResult := StringToMap(subResult[0][2])
				for k, v := range _subResult {
					data[k] = v
				}
			}
		}
	}
	return
}

func isChar(ru rune) bool {
	return (ru >= 0 && ru <= 9) || (ru >= 'a' && ru <= 'z') || (ru >= 'A' && ru <= 'Z') || ru == '_' || ru == '-'
}

var regSpace *regexp.Regexp = regexp.MustCompile(`\s+`)

func stringParseTextLineSlice(line string) (result []string) {
	line = strings.TrimSpace(line)
	index := strings.IndexByte(line, '#')
	if index == 0 {
		return
	}
	if index > 0 {
		line = line[:index]
		line = strings.TrimSpace(line)
	}
	if line == "" {
		return
	}
	return regSpace.Split(line, -1)
}

func LoadText2Slice(text string) (result [][]string) {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		lineArr := stringParseTextLineSlice(line)
		if len(lineArr) > 0 {
			result = append(result, lineArr)
		}
	}
	return
}

func LoadText2SliceMap(text string) (result []map[string]string) {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		lineArr := stringParseTextLineSlice(line)
		if len(lineArr) == 0 {
			continue
		}
		lineObj := make(map[string]string)
		isOk := true
		for _, str := range lineArr {
			arr := strings.SplitN(str, ":", 2)
			if len(arr) == 2 {
				lineObj[strings.TrimSpace(arr[0])] = strings.TrimSpace(arr[1])
			} else {
				log.Println("parse line [", line, "] failed,not k:v,[", str, "]")
				isOk = false
				break
			}
		}
		if isOk && len(lineObj) > 0 {
			result = append(result, lineObj)
		}
	}
	return result
}

func StrMd5(mystr string) string {
	h := md5.New()
	h.Write([]byte(mystr))
	return fmt.Sprintf("%x", h.Sum(nil))
}
