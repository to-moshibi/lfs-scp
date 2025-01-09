package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	scp "github.com/bramvdbogaerde/go-scp"
	"github.com/bramvdbogaerde/go-scp/auth"
	"golang.org/x/crypto/ssh"
)

type Init struct {
	Event               string `json:"event"`
	Operation           string `json:"operation"`
	Remote              string `json:"remote"`
	Concurrent          bool   `json:"concurrent"`
	Concurrenttransfers int    `json:"concurrenttransfers"`
}

type Upload struct {
	Event  string `json:"event"`
	Oid    string `json:"oid"`
	Size   int    `json:"size"`
	Path   string `json:"path"`
	Action struct {
		Href   string `json:"href"`
		Header struct {
			Key string `json:"key"`
		} `json:"header"`
	} `json:"action"`
}

type Download struct {
	Event  string `json:"event"`
	Oid    string `json:"oid"`
	Size   int    `json:"size"`
	Action struct {
		Href   string `json:"href"`
		Header struct {
			Key string `json:"key"`
		} `json:"header"`
	} `json:"action"`
}

type Terminate struct {
	Event string `json:"event"`
}

type InitResponse struct{}

type InitErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type UploadResponse struct {
	Event string `json:"event"`
	Oid   string `json:"oid"`
}

type UploadErrorResponse struct {
	Event string `json:"event"`
	Oid   string `json:"oid"`
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type DownloadResponse struct {
	Event string `json:"event"`
	Oid   string `json:"oid"`
	Path  string `json:"path"`
}

type DownloadErrorResponse struct {
	Event string `json:"event"`
	Oid   string `json:"oid"`
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type ProgressResponse struct {
	Event          string `json:"event"`
	Oid            string `json:"oid"`
	BytesSoFar     int    `json:"bytesSoFar"`
	BytesSinceLast int    `json:"bytesSinceLast"`
}

var EventMap = map[string]interface{}{
	"init":      Init{},
	"upload":    Upload{},
	"download":  Download{},
	"terminate": Terminate{},
}

func main() {

    serverAddress := os.Args[1]
    serverPort := os.Args[2]
    serverUser := os.Args[3]
    serverIdentity := os.Args[4]

    clientConfig, _ := auth.PrivateKey(serverUser,serverIdentity, ssh.InsecureIgnoreHostKey())

	for {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		jsonInput := scanner.Text()

		// 汎用のマップにパース
		var genericEvent map[string]interface{}
		err := json.Unmarshal([]byte(jsonInput), &genericEvent)
		if err != nil {
			return
		}

		// イベントタイプを取得
		eventType, ok := genericEvent["event"].(string)
		if !ok {
			return
		}

		// イベントタイプに対応する構造体を取得
		eventStruct, ok := EventMap[eventType]
		if !ok {
			return
		}

		// 対応する構造体にパース
		eventBytes, err := json.Marshal(genericEvent)
		if err != nil {
			return
		}

		err = json.Unmarshal(eventBytes, &eventStruct)
		if err != nil {
			return
		}

		if eventType == "init" {
			fmt.Println(InitResponse{})
		} else if eventType == "upload" {

			var upload Upload
			err = json.Unmarshal(eventBytes, &upload)
			if err != nil {
				errRes := UploadErrorResponse{
					Event: "complete",
					Oid:   upload.Oid,
					Error: struct {
						Code    int    `json:"code"`
						Message string `json:"message"`
					}{
						Code:    1,
						Message: "Error Unmarshal: " + err.Error(),
					},
				}
                fmt.Println(errRes)
				return
			}

            client := scp.NewClient(serverAddress+":"+serverPort, &clientConfig)

            err := client.Connect()
            if err != nil {
                errRes := UploadErrorResponse{
                    Event: "complete",
                    Oid:   upload.Oid,
                    Error: struct {
                        Code    int    `json:"code"`
                        Message string `json:"message"`
                    }{
                        Code:    4,
                        Message: "Error Connecting: " + err.Error(),
                    },
                }
                fmt.Println(errRes)
                return
            }

            f,_:= os.Open(upload.Path)
            defer client.Close()
            defer f.Close()

			err = client.CopyFile(context.Background(), f, "/home/"+serverUser+"/storage/"+upload.Oid, "0655")
            if err != nil {
                errRes := UploadErrorResponse{
                    Event: "complete",
                    Oid:   upload.Oid,
                    Error: struct {
                        Code    int    `json:"code"`
                        Message string `json:"message"`
                    }{
                        Code:    5,
                        Message: "Error Copying: " + err.Error(),
                    },
                }
                fmt.Println(errRes)
                return
            }

			res := UploadResponse{
				Event: "complete",
				Oid:   upload.Oid,
			}

            resOut,err := json.Marshal(res)
			if err != nil {
				errRes := UploadErrorResponse{
					Event: "complete",
					Oid:   upload.Oid,
					Error: struct {
						Code    int    `json:"code"`
						Message string `json:"message"`
					}{
						Code:    2,
						Message: "Error Marshal Response: " + err.Error(),
					},
				}
                fmt.Println(errRes)
				return
			}

			fmt.Println(string(resOut))
		} else if eventType == "download" {
            cmd := exec.Command("git", "rev-parse", "--git-dir")
            cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
            out,err := cmd.Output()
            if(err != nil){
                fmt.Println(err)
                return
            }
            path := strings.TrimSpace(string(out))
            gitDir,err := absPath(path)
            if(err != nil){
                fmt.Println(err)
                return
            }

            var download Download
            err = json.Unmarshal(eventBytes, &download)
            if err != nil {
                errRes := DownloadErrorResponse{
                    Event: "complete",
                    Oid:   download.Oid,
                    Error: struct {
                        Code    int    `json:"code"`
                        Message string `json:"message"`
                    }{
                        Code:    1,
                        Message: "Error Unmarshal: " + err.Error(),
                    },
                }
                fmt.Println(errRes)
                return
            }

            client := scp.NewClient(serverAddress+":"+serverPort, &clientConfig)

            err = client.Connect()
            if err != nil {
                errRes := DownloadErrorResponse{
                    Event: "complete",
                    Oid:   download.Oid,
                    Error: struct {
                        Code    int    `json:"code"`
                        Message string `json:"message"`
                    }{
                        Code:    4,
                        Message: "Error Connecting: " + err.Error(),
                    },
                }
                fmt.Println(errRes)
                return
            }

            dlFileName := downloadTempPath(gitDir, download.Oid)
            
            f,_:= os.OpenFile(dlFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
            defer client.Close()
            defer f.Close()
            
            err = client.CopyFromRemote(context.Background(), f, "/home/"+serverUser+"/storage/"+download.Oid)
            if err != nil {
                errRes := DownloadErrorResponse{
                    Event: "complete",
                    Oid:   download.Oid,
                    Error: struct {
                        Code    int    `json:"code"`
                        Message string `json:"message"`
                    }{
                        Code:    5,
                        Message: "Error Copying: " + err.Error(),
                    },
                }
                fmt.Println(errRes)
                return
            }
            
            res := DownloadResponse{
                Event: "complete",
                Oid:   download.Oid,
                Path: dlFileName,

            }

            resOut,err := json.Marshal(res)
            if err != nil {
                errRes := DownloadErrorResponse{
                    Event: "complete",
                    Oid:   download.Oid,
                    Error: struct {
                        Code    int    `json:"code"`
                        Message string `json:"message"`
                    }{
                        Code:    2,
                        Message: "Error Marshal Response: " + err.Error(),
                    },
                }
                fmt.Println(errRes)
                return
            }
            
            f.Close()

            fmt.Println(string(resOut))

		} else if eventType == "terminate" {
			break
		}
	}
}

func absPath(path string) (string, error) {
	if len(path) > 0 {
		path, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		return filepath.EvalSymlinks(path)
	}
	return "", nil
}

func downloadTempPath(gitDir string, oid string) string {
	tmpfld := filepath.Join(gitDir, "lfs", "tmp")
	os.MkdirAll(tmpfld, os.ModePerm)
	return filepath.Join(tmpfld, oid + ".tmp")
}