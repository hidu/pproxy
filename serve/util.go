package serve

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
	"log"
	"net"
	//	"github.com/vmihailenco/msgpack"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func Int64ToBytes(i int64) []byte {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}

func IsLocalIp(host string) bool {
	ips, _ := net.LookupIP(host)
	for _, ip := range ips {
		if ip.IsLoopback() {
			return true
		}
	}
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, addr := range addrs {
			_, ip_g, err := net.ParseCIDR(addr.String())
			if err == nil {
				for _, ip := range ips {
					if ip_g.Contains(ip) {
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

func gob_encode(data interface{}) string {
	//   var bf bytes.Buffer
	//	enc := gob.NewEncoder(&bf)
	//	err := enc.Encode(data)
	//	 bf, err := msgpack.Marshal(data)
	bf, err := json.Marshal(data)
	if err != nil {
		log.Println("gob encode err", err)
		return ""
	}
	return base64.StdEncoding.EncodeToString(bf)
}

func gob_decode(data_input string, out interface{}) {
	//	dec := gob.NewDecoder(bytes.NewBufferString(data_input))
	//	err:= dec.Decode(&out)
	str_64, err := base64.StdEncoding.DecodeString(data_input)
	if err != nil {
		log.Println("decode64 err:", err)
		return
	}
	dec := json.NewDecoder(bytes.NewBuffer(str_64))
	err = dec.Decode(&out)
	if err != nil {
		log.Println("msgpack decode:", err)
	}
}

func getMapValStr(m map[string]interface{}, k string) string {
	if val, has := m[k]; has {
		return fmt.Sprintf("%s", val)
	}
	return ""
}

func gzipDocode(buf *bytes.Buffer) string {
	gr, err := gzip.NewReader(buf)
	defer gr.Close()
	if err == nil {
		bd_bt, _ := ioutil.ReadAll(gr)
		return string(bd_bt)
	} else {
		log.Println("unzip body failed", err)
		return ""
	}
}
func gzipEncode(data []byte) *bytes.Buffer {
	buf := bytes.NewBuffer([]byte{})
	gw := gzip.NewWriter(buf)
	defer gw.Close()
	gw.Write(data)
	return buf
}

func parseUrlInputAsSlice(input string) []string {
	arr := strings.Split(input, "|")
	result := make([]string, 0)
	for _, val := range arr {
		val = strings.TrimSpace(val)
		if val != "" {
			result = append(result, val)
		}
	}
	return result
}

func GetFormValuesWithPrefix(values url.Values, prefix string) map[string][]string {
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
	var port_str string
	req_host := req.Host
	if !strings.Contains(req_host, ":") {
		if strings.HasPrefix(req.Proto, "HTTP") {
			req_host += ":80"
		}
	}
	host, port_str, err = net.SplitHostPort(req_host)
	if err != nil {
		return
	}
	port, err = strconv.Atoi(port_str)
	if err != nil {
		return
	}
	return
}

func checkUrlValuesChange(first url.Values, second url.Values) (change bool) {
	for k, v := range first {
		sec_v, has := second[k]
		if !has {
			return true
		}
		if len(v) != len(sec_v) || fmt.Sprintf("%v", v) != fmt.Sprintf("%v", sec_v) {
			return true
		}
	}
	for k, v := range second {
		first_v, has := first[k]
		if !has {
			return true
		}
		if len(v) != len(first_v) || fmt.Sprintf("%v", v) != fmt.Sprintf("%v", first_v) {
			return true
		}
	}
	return false
}
