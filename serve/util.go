package serve

import (
	"encoding/binary"
	"net"
	"io"
	"bytes"
	"io/ioutil"
)

 func Int64ToBytes(i int64) []byte {
     var buf = make([]byte, 8)
     binary.BigEndian.PutUint64(buf, uint64(i))
     return buf
 }

func IsLocalIp(host string)bool{
   ips,_:=net.LookupIP(host)
   for _,ip:=range ips{
	   if(ip.IsLoopback()){
	   return true;
	   }
    }
   if addrs, err := net.InterfaceAddrs(); err == nil {  
        for _, addr := range addrs {  
           _,ip_g,err:=net.ParseCIDR(addr.String())  
           if(err==nil){
           for _,ip:=range ips{
	           if(ip_g.Contains(ip)){
	              return true
	               }
               }
           }
        }  
    }
   return  false;
 }
 
 func forgetRead(reader *io.ReadCloser) *bytes.Buffer{
    buf := bytes.NewBuffer([]byte{})
	 io.Copy(buf, *reader)
    *reader = ioutil.NopCloser(buf).(io.ReadCloser)
    return buf
 }