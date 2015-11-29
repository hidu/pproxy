package serve

import (
	"bytes"
	"compress/gzip"
	//	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	//	"gopkg.in/vmihailenco/msgpack.v2"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Int64ToBytes int64转换为byte
func Int64ToBytes(i int64) []byte {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}

// IntToBytes int转换为byte
func IntToBytes(i int) []byte {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}

// IsLocalIP 判断一个host是否本地ip
func IsLocalIP(host string) bool {
	ips, _ := net.LookupIP(host)
	for _, ip := range ips {
		if ip.IsLoopback() {
			return true
		}
	}
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, addr := range addrs {
			_, ipG, err := net.ParseCIDR(addr.String())
			if err == nil {
				for _, ip := range ips {
					if ipG.Contains(ip) {
						return true
					}
				}
			}
		}
	}
	return false
}

func forgetRead(reader *io.ReadCloser) *bytes.Buffer {
	buf := bytes.NewBuffer([]byte{})
	io.Copy(buf, *reader)
	*reader = ioutil.NopCloser(buf).(io.ReadCloser)
	return bytes.NewBuffer(buf.Bytes())
}

func dataEncode(data interface{}) []byte {
	bf, err := json.Marshal(data)
	if err != nil {
		log.Println("data_encode_err", err)
		return bf
	}
	return bf
}
func dataDecode(dataInput []byte, out interface{}) error {
	if len(dataInput) == 0 {
		return fmt.Errorf("empty dataInput")
	}
	err := json.Unmarshal(dataInput, &out)
	if err != nil {
		log.Println("json_decode_err:", err, "dataInput:", string(dataInput))
		return err
	}
	return err
}

func getMapValStr(m map[string]interface{}, k string) string {
	if val, has := m[k]; has {
		return fmt.Sprintf("%s", val)
	}
	return ""
}

func gzipDocode(buf *bytes.Buffer) string {
	if buf.Len() < 1 {
		return ""
	}
	gr, err := gzip.NewReader(buf)
	defer gr.Close()
	if err == nil {
		bdBt, _ := ioutil.ReadAll(gr)
		return string(bdBt)
	}
	log.Println("unzip body failed", err)
	return ""
}
func gzipEncode(data []byte) *bytes.Buffer {
	buf := bytes.NewBuffer([]byte{})
	gw := gzip.NewWriter(buf)
	defer gw.Close()
	gw.Write(data)
	return buf
}

func parseURLInputAsSlice(input string) []string {
	arr := strings.Split(input, "|")
	var result []string
	for _, val := range arr {
		val = strings.TrimSpace(val)
		if val != "" {
			result = append(result, val)
		}
	}
	return result
}

func getFormValuesWithPrefix(values url.Values, prefix string) map[string][]string {
	result := make(map[string][]string)
	for k, v := range values {
		if strings.HasPrefix(k, prefix) {
			k1 := strings.TrimPrefix(k, prefix)
			result[k1] = v
		}
	}
	return result
}

func getTextAreaHeightByString(mystr string, minHeight int) int {
	height := (len(strings.Split(mystr, "\n")) + 1) * 25
	if height < minHeight {
		height = minHeight
	}
	return height
}

func getHostPortFromReq(req *http.Request) (host string, port int, err error) {
	host, port, err = parseHostPort(req.Host)
	if err == nil && port == 0 {
		switch req.URL.Scheme {
		case "http":
			port = 80
			break
		case "https":
			port = 443
			break
		default:
			break
		}
	}
	return
}

func parseHostPort(hostPortstr string) (host string, port int, err error) {
	var portStr string
	if !strings.Contains(hostPortstr, ":") {
		hostPortstr += ":0"
	}
	host, portStr, err = net.SplitHostPort(hostPortstr)
	if err != nil {
		return
	}
	port, err = strconv.Atoi(portStr)
	if err != nil {
		return
	}
	return
}

func checkURLValuesChange(first url.Values, second url.Values) (change bool) {
	for k, v := range first {
		secV, has := second[k]
		if !has {
			return true
		}
		if len(v) != len(secV) || fmt.Sprintf("%v", v) != fmt.Sprintf("%v", secV) {
			return true
		}
	}
	for k, v := range second {
		firstV, has := first[k]
		if !has {
			return true
		}
		if len(v) != len(firstV) || fmt.Sprintf("%v", v) != fmt.Sprintf("%v", firstV) {
			return true
		}
	}
	return false
}

func parseDocID(strid string) (docid int, err error) {
	docid64, parseErr := strconv.ParseUint(strid, 10, 64)
	if parseErr == nil {
		return int(docid64), nil
	}
	return 0, parseErr
}

func removeHeader(req *http.Request) {
	for k := range req.Header {
		if len(k) > 5 && k[:6] == "Proxy-" {
			req.Header.Del(k)
		}
	}
}

func getPostData(req *http.Request) (post *url.Values) {
	post = new(url.Values)
	if strings.Contains(req.Header.Get("Content-Type"), "x-www-form-urlencoded") {
		buf := forgetRead(&req.Body)
		var bodyStr string
		if req.Header.Get(contentEncoding) == "gzip" {
			bodyStr = gzipDocode(buf)
		} else {
			bodyStr = buf.String()
		}
		var err error
		*post, err = url.ParseQuery(bodyStr)
		if err != nil {
			log.Println("parse post err", err, "url=", req.URL.String())
		}

	}
	return post
}

func headerEncode(data []byte) []byte {
	t := bytes.Replace(data, []byte("\r"), []byte("\\r"), -1)
	t = bytes.Replace(t, []byte("\n"), []byte("\\n"), -1)
	return t
}
