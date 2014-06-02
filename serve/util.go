package serve

import (
    "bytes"
    "encoding/binary"
    "io"
    "io/ioutil"
    "log"
    "net"
    //	"github.com/vmihailenco/msgpack"
    "encoding/base64"
    "encoding/json"
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
