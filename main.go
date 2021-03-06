package main

import "C"

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"

	"bt/common"
	"bt/controller"
	"bt/logger"

	"golang.org/x/crypto/curve25519"
	"unsafe"
	"os"
)

var (
	device *controller.Device
)

//export Callback
type Callback interface {
	CallFd(int)
	CallStatus(int)
	CallUploadFlow(int)
	CallDownloadFlow(int)
	CallDestinationIP(string)
}

/*type Cb struct {

}

func (c Cb)CallFd(fb int)  {
	fmt.Println(fb)
}
func (c Cb)CallStatus(fb int)  {
	fmt.Println(fb)
}
func (c Cb)CallUploadFlow(fb int)  {
	fmt.Println(fb)
}
func (c Cb)CallDownloadFlow(fb int)  {
	fmt.Println(fb)
}

func (c Cb)CallDestinationIP(s string)  {
	fmt.Println(s)
}*/

type configData struct {
	OwnPrivate   string `json:"own_private"`
	OwnPublic    string `json:"own_public"`
	TheirPublic  string `json:"their_public"`
	Endpoint     string `json:"endpoint"`
	AllowIp      string `json:"allow_ip"`
	LogPath      string `json:"log_path"`
	IsIOS        string `json:"is_iOS"`
	Ts           uint32 `json:"ts"`
	Sign         string `json:"sign"`
	Netmask      uint32 `json:"netmask"`
	IntervalTime int64  `json:"interval_time"`
	IsCallbackIp int    `json:"is_callback_ip"`
}

func main(){
	/*private,public := GetPriAndPubKey()

	config := configData{
		OwnPrivate:   private,
		OwnPublic:    public,
		TheirPublic:  "YwCI0t17PegezDkGISuHcOMgYdFCpwvY7C0Q+nTp+Qs=",
		Endpoint:     "152.32.190.101:6666",
		AllowIp:      "0.0.0.0/0, ::0/0",
		LogPath:      "./diary/info_diary.log",
		IsIOS:        "",
		Ts:           0,
		Sign:         "",
		Netmask:      32,
		IntervalTime: 50,
	}

	file ,err:= os.Open("./fb.txt")
	if err != nil{
		fmt.Println(err)
		panic(err)
	}

	fd := file.Fd()

	configStr,err := json.Marshal(config)
	if err != nil{
		fmt.Println(err)
		panic(err)
	}
	str := Init(int(fd),string(configStr))

	fmt.Println(str)
	cb := Cb{}
	if str == "" {
		s := Start(int32(fd),cb)
		fmt.Println(s)
	}*/
	/*[Interface]
	PrivateKey = qFmhY+lwAgoZpAPkw6uSV3RB1a+8tYSgD78e4GgZ3Gg=
	Address = 10.0.0.2/24
	DNS = 8.8.8.8
	MTU = 1420
	[Peer]
	PublicKey = YwCI0t17PegezDkGISuHcOMgYdFCpwvY7C0Q+nTp+Qs=
	Endpoint = 152.32.190.101:6666
	AllowedIPs = 0.0.0.0/0, ::0/0
	PersistentKeepalive = 30*/


	/*{
		"own_private":  string,      //上面接口获取的私钥
		"own_public":   string,     //上面接口获取的公钥
		"their_public": string,     //服务器的公钥
		"endpoint":     string,     //服务器的IP
		"allow_ip":     string,     //分配的客户端虚拟ip
		"log_path":     string,     //存放的日志
		"is_iOS":       string,     //是否是iOS，不是就填空
		"ts":           int,        //到期时间
		"sign":         string,     //签名串
		"netmask":      int,         //固定值32
		"interval_time":  int        //间隔时间，用来统计网络异常时多久上报一次101。默认50s
	}*/

}

//export Init
func Init(fd int, jsonFomt string) string {
	if fd <= 0 {
		return "tun fd有误:" + strconv.Itoa(fd)
	}
	var values configData
	err := json.Unmarshal([]byte(jsonFomt), &values)
	if err != nil {
		return err.Error()
	}

	//init logger
	err = logger.LogDiaryInit(values.LogPath)
	if err != nil {
		return err.Error()
	}

	logger.Wlog.SaveInfoLog("init start，config info:" + jsonFomt)

	if values.IsIOS == "1" {
		controller.IsiOS = true
	}

	controller.Ts = values.Ts
	controller.Sign = values.Sign
	controller.AllowIp = values.AllowIp
	controller.Netmask = values.Netmask
	controller.Endpoint = values.Endpoint
	if values.IntervalTime > 0 {
		controller.IntervalTime = values.IntervalTime
	}

	debug.SetGCPercent(10)

	// open TUN device
	tun, err := controller.CreateTUN(fd)
	if err != nil {
		logger.Wlog.SaveInfoLog(err.Error())
		return err.Error()
	}

	// create controller device
	device = controller.NewDevice(tun)
	config := make([]string, 6)
	config[0] = "own_private=" + values.OwnPrivate
	config[1] = "own_public=" + values.OwnPublic
	config[2] = "their_public=" + values.TheirPublic
	config[3] = "endpoint=" + values.Endpoint
	config[4] = "allowed_ip=0.0.0.0/0"
	config[5] = "persistent_keepalive_interval=15"

	errMsg := controller.SetOperation(device, config)
	if errMsg != "" {
		logger.Wlog.SaveInfoLog(errMsg)
		return errMsg
	}

	logger.Wlog.SaveInfoLog(fmt.Sprintf("Version:%s", time.Now().Format("2006-01-02")))
	logger.Wlog.SaveInfoLog("init finish")

	return ""
}


//export Start
func Start(rfd int32, cb Callback) string {
	logger.Wlog.SaveInfoLog("enter start...")

	defer func() {
		if err := recover(); err != nil {
			logger.Wlog.SaveErrLog(fmt.Sprintln("程序结束:", err))
		}
	}()

	if rfd < 0 {
		msg := "pipe fd is error:" + strconv.Itoa(int(rfd))
		logger.Wlog.SaveInfoLog(msg)
		return msg
	}

	logger.Wlog.SaveInfoLog("starting success...")
	go device.StartConnection()
	go status(cb)

	p := make([]byte, 1)
	//windows.Read(windows.Handle(uintptr(unsafe.Pointer(&rfd))), d)
	//syscall.Read(int(rfd), p)
	syscall.Read(syscall.Handle(uintptr(unsafe.Pointer(&rfd))), p)

	device.Close()

	// clean up UAPI bind
	logger.Wlog.SaveInfoLog("Closing")
	return ""
}

//export status
func status(c Callback) {
	defer func() {
		if err := recover(); err != nil {
			logger.Wlog.SaveErrLog(fmt.Sprintln("recover status err:", err))
		}
	}()
	for {
		select {
		case <-device.WaitChannel():
			return
		//case n := <-controller.UdpfdChan:
		//	c.CallUDPFd(n)
		case n := <-controller.StatusChan:
			c.CallStatus(n)
		case fd := <-controller.FdChan:
			c.CallFd(fd)
			//case n := <-controller.UploadFlowChan:
			//	c.CallUploadFlow(n)
			//case n := <-controller.DownloadFlowChan:
			//	c.CallDownloadFlow(n)
			//case str := <-controller.DestinationIPChan:
			//	c.CallDestinationIP(str)
		}
	}
}


//export GetPriAndPubKey
func GetPriAndPubKey() (string,string) {
	random := rand.Reader

	var pri, pub [32]byte
	_, err := io.ReadFull(random, pri[:])
	if err != nil {
		fmt.Println("Error:",err)
		return "",""
		os.Exit(1)
	}

	pri[0] &= 248
	pri[31] &= 127
	pri[31] |= 64

	curve25519.ScalarBaseMult(&pub, &pri)

	private := base64.StdEncoding.EncodeToString(pri[:])
	public := base64.StdEncoding.EncodeToString(pub[:])

	//return private + "," + public
	fmt.Println(private + "," + public)
	return private,public
}

//export GetDomain
func GetDomain(domain, secret, isAbroad string) string {
	var err error
	var qtMsg []byte
	msgChan := make(chan []byte, 3)
	dnsName := []string{"114.114.114.114:53", "223.5.5.5:53", "119.29.29.29:53", "180.76.76.76:53"}
	if isAbroad == "1" {
		dnsName = []string{"8.8.8.8:53"}
	}

	for _, v := range dnsName {
		go func() {
			qtMsg, err = controller.SendDNSReq(v, domain, msgChan)
			if err != nil {
				fmt.Println("failed to req:", err)
			}
		}()
	}

L:
	for {
		select {
		case b := <-msgChan:
			qtMsg = b
			break L
		case <-time.After(6 * time.Second):
			break L
		}
	}

	if len(qtMsg) == 0 {
		return ""
	}

	if secret == "" {
		secret = "wriQVbeJqBWC2qJ0"
	}

	aesDec := common.AesEncrypt{}
	v, err := aesDec.Decrypt(qtMsg, []byte(secret))
	if err != nil {
		fmt.Println("failed to aes decrypt:", err)
		return ""
	}
	return string(v)
}

func f(){
	udp_addr, err := net.ResolveUDPAddr("udp", ":11110")
	if err != nil{
		panic(err)
	}

	conn, err := net.ListenUDP("udp", udp_addr)
	defer conn.Close()

	controller.GetFD(conn)

}