package main

import (
	"log"
	"os"
	"path/filepath"
	"fmt"
	"bufio"
	"io"
	"time"
	"bytes"
	"net/http"
	"mime/multipart"
	"archive/zip"
	"os/exec"
	"encoding/json"
	"io/ioutil"
)

var LOG* log.Logger
func init() {
	logPath, _ := filepath.Abs("./saveMonitor.log")
	logFile, _ := os.OpenFile(logPath , os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	LOG = log.New(logFile, "", log.Lshortfile|log.LstdFlags )
}

func compress(file *os.File, prefix string, zw *zip.Writer) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		prefix = prefix + "/" + info.Name()
		fileInfos, err := file.Readdir(-1)
		if err != nil {
			return err
		}
		for _, fi := range fileInfos {
			f, err := os.Open(file.Name() + "/" + fi.Name())
			if err != nil {
				return err
			}
			err = compress(f, prefix, zw)
			if err != nil {
				return err
			}
		}
	} else {
		header, err := zip.FileInfoHeader(info)
		header.Name = prefix + "/" + header.Name
		if err != nil {
			return err
		}
		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, file)
		file.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func DeCompress(zipFile, dest string) error {
	reader, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}

	err = os.MkdirAll(dest, 0755)

	defer reader.Close()
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		_, n := filepath.Split(file.Name)
		filename := filepath.Join( dest, n)

		if err != nil {
			return err
		}
		w, err := os.Create(filename)
		if err != nil {
			return err
		}
		defer w.Close()
		_, err = io.Copy(w, rc)
		if err != nil {
			return err
		}
		w.Close()
		rc.Close()
	}
	return nil
}

func Compress(files []*os.File, dest string) error {
	d, _ := os.Create(dest)
	defer d.Close()
	w := zip.NewWriter(d)
	defer w.Close()
	for _, file := range files {
		err := compress(file, "", w)
		if err != nil {
			return err
		}
	}
	return nil
}

func DownloadFile(filepath string, url string) error {
	// Create the file
	out, err := os.OpenFile(filepath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func postFile(filename string, target_url string) (*http.Response, error) {
	body_buf := bytes.NewBufferString("")
	body_writer := multipart.NewWriter(body_buf)

	_, file := filepath.Split(filename)
	// use the body_writer to write the Part headers to the buffer
	_, err := body_writer.CreateFormFile("uploadfile", file)
	if err != nil {
		fmt.Println("error writing to buffer")
		return nil, err
	}

	// the file data will be the second part of the body
	fh, err := os.Open(filename)
	if err != nil {
		fmt.Println("error opening file")
		return nil, err
	}
	// need to know the boundary to properly close the part myself.
	boundary := body_writer.Boundary()
	//close_string := fmt.Sprintf("\r\n--%s--\r\n", boundary)
	close_buf := bytes.NewBufferString(fmt.Sprintf("\r\n--%s--\r\n", boundary))

	// use multi-reader to defer the reading of the file data until
	// writing to the socket buffer.
	request_reader := io.MultiReader(body_buf, fh, close_buf)
	fi, err := fh.Stat()
	if err != nil {
		fmt.Printf("Error Stating file: %s", filename)
		return nil, err
	}
	req, err := http.NewRequest("POST", target_url, request_reader)
	if err != nil {
		return nil, err
	}

	// Set headers for multipart, and Content Length
	req.Header.Add("Content-Type", "multipart/form-data; boundary="+boundary)
	req.ContentLength = fi.Size() + int64(body_buf.Len()) + int64(close_buf.Len())

	return http.DefaultClient.Do(req)
}

var
(
	diabloPath = "F:\\迅雷下载\\Diablo.II.v1.13c.CHS.CHT.Green.Edition-ALI213\\Diablo II\\"
	//diabloPath = "E:\\Soft\\ForMine\\diablo1.13C\\"
	diabloPathSave = filepath.Join(diabloPath, "Save")
	diabloPathExe = filepath.Join(diabloPath, "d2loader.exe")
	saveLocal = filepath.Join(diabloPath, "Save-Local")
	saveDown = filepath.Join(diabloPath, "Save-Down")
	saveUp = filepath.Join(diabloPath, "Save-Upload")

	serverUrl = "http://129.204.48.44:80/"
	uploadServer = serverUrl+"upload"
	getlistServer = serverUrl+"getlist"
	downloadServer = serverUrl+"staticfile/"

	stdinReader = bufio.NewReader(os.Stdin)
)

func saveInLocal() {
	err := os.Mkdir(saveLocal, 0777)
	dest := fmt.Sprintf("%s/%s.local", saveLocal, time.Now().Format("2006-01-02-15.04.05"))

	saveFolder, err := os.Open(diabloPathSave); if err != nil {
		errorQuit("save folder not found")
	}
	defer saveFolder.Close()

	if err := Compress(append([]*os.File{}, saveFolder), dest); err != nil {
		errorQuit("Save Local Compress Failed:"+err.Error())
	}
}

func downloadProfile()  {
	//get list
	resp, err := http.Get(getlistServer)
	if err != nil {
		errorQuit("Get list failed:"+err.Error())
	}
	defer resp.Body.Close()

	fileContent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		errorQuit("Download file list failed:" + err.Error())
	}

	if len(fileContent) <= 0 {
		return
	}

	var fileList []string
	if err = json.Unmarshal(fileContent, &fileList); err != nil {
		errorQuit("Download file list unmarsal json failed:" + err.Error())
	}

	//check newest
	newestFile := ""
	for _, file := range fileList {
		if newestFile == "" {
			newestFile = file
		} else if file > newestFile {
			newestFile = file
		}
	}

	if newestFile == "" {
		return
	}

	//download
	os.MkdirAll(saveDown, 0777)

	downDest := filepath.Join(saveDown, newestFile)
	if err = DownloadFile(downDest, downloadServer+newestFile); err != nil {
		errorQuit("Download newest profile failed:"+err.Error())
	}

	//DeCompress
	if err = DeCompress(downDest, diabloPathSave); err != nil {
		errorQuit("DeCompress newest profile failed:"+err.Error())
	}
}

func uploadProfile() {
	os.Mkdir(saveUp, 0777)

	dest := fmt.Sprintf("%s/%s.up", saveUp, time.Now().Format("2006-01-02-15.04.05"))
	dest, _ = filepath.Abs(dest)

	var fileList []*os.File
	saveFolder, err := os.Open(diabloPathSave)
	if err != nil {
		errorQuit("Cann't find Dialbo save folder")
	}
	defer saveFolder.Close()

	if err := Compress(append(fileList, saveFolder), dest); err != nil {
		errorQuit("Upload Compress Failed:" + err.Error())
	}

	if _, err := postFile(dest, uploadServer); err != nil {
		errorQuit("Upload failed:"+err.Error())
	}
}

func errorQuit(err string) {
	fmt.Println(err)
	fmt.Println("Press Enter To Quit!")
	txt, _ := stdinReader.ReadString('\n')
	txt = txt
	panic("")
}

func main(){
	//Save profile first
	saveInLocal()

	//Download remote profile
	downloadProfile()

	//Monitor
	reader := bufio.NewReader(os.Stdin)
	cmd := exec.Command(diabloPathExe, "-w")
	if err := cmd.Start() ; err != nil {
		fmt.Println("Diablo is loading failed:", err)
		txt, _ := reader.ReadString('\n')
		txt = txt
		return
	}

	fmt.Println("Diablo is running...")
	cmd.Wait()
	fmt.Println("Diablo quit, Saving game...")

	//upload
	uploadProfile()

	fmt.Println("Saving complete!")
	txt, _ := reader.ReadString('\n')
	txt = txt

	return
}