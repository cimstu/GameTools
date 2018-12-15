package main

import (
	"net"
	"log"
	"net/http"
	"io/ioutil"
	"github.com/zserge/webview"
	"GameTools/util"
	"os"
	"fmt"
	"encoding/json"
	"errors"
	"time"
)

func CheckError(err error, pre string, pn bool) bool{
	if err != nil {
		util.MessageBox("Error", pre + ":" + err.Error(), util.MB_OK)
		if pn{
			panic("")
		}

		return true
	}

	return false
}

func startServer() string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal("read start listen failed:", err)
	}

	go func() {
		defer ln.Close()
		http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
			con, err := ioutil.ReadFile("./res/main.html")
			if err != nil {
				log.Fatal("read res file failed:", err)
			}
			writer.Write(con)
		})

		log.Fatal(http.Serve(ln, nil))
	}()

	return "http://"+ln.Addr().String()
}
const (
	winWidth = 400
	winHight = 600

	saveLocal = "./Save-Local"
	saveDownload = "./Save-Down"
	saveUp = "./Save-Up"

	settingFile = "cloudUDrive.setting"
)

type Settings struct {
	ServerUrl string
}
var (
	MainView webview.WebView

	settings Settings
	uploadServer string
	getlistServer string
	downloadServer string
	mainPage string
)


func init() {
	c, err := ioutil.ReadFile("./"+settingFile)
	CheckError(err, "尚未配置云盘地址", true)

	err = json.Unmarshal(c, &settings)
	CheckError(err, "读取配置信息失败", true)

	if settings.ServerUrl == "" {
		CheckError(errors.New(""), "读取配置信息失败", true)
	}

	uploadServer = settings.ServerUrl+"upload"
	getlistServer = settings.ServerUrl+"getlist"
	downloadServer = settings.ServerUrl+"staticfile/"
	mainPage = downloadServer+"contents.html"
}

func handleEvent(v webview.WebView, data string) {
	switch {
	case data == "upload":
		fp := v.Dialog(webview.DialogTypeOpen, 0, "Open file", "")
		handleUpload(fp)
	}
}

func handleUpload(fp string) {
	if fp == "" {
		return
	}

	fi, err := os.Stat(fp)
	if CheckError(err, "上传失败", false) {
		return
	}
	if fi.Size() >= 1024*1024*20 {
		util.MessageBox("Error", "对不起，不能上传大于10M的文件", util.MB_OK)
		return
	}

	_, err = util.PostFile(fp, uploadServer)
	CheckError(err, "上传失败", false)

	go func() {
		time.Sleep(time.Second*1)
		MainView.Dispatch(func() {
			MainView.Eval(`var fr = document.getElementById("contents"); fr.src=fr.src;`)
		})
	}()

	util.MessageBox("成功", "上传成功", util.MB_OK)
}

func main(){
	//res := startServer()
	fmt.Println(mainPage)
	MainView = webview.New(webview.Settings{
		Width: winWidth,
		Height: winHight,
		Title: "My U Drive",
		Resizable: false,
		URL: mainPage,
		ExternalInvokeCallback: handleEvent,
	})

	defer MainView.Exit()
	MainView.Run()
}
