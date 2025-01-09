package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
    scp "github.com/bramvdbogaerde/go-scp"
	"github.com/bramvdbogaerde/go-scp/auth"
	"golang.org/x/crypto/ssh"
    "github.com/joho/godotenv"
    "context"
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

    
    err := godotenv.Load()
    if err != nil {
        fmt.Println("Error loading .env file")
    }

    serverAddress := os.Getenv("SCP_SERVER")
    serverPort := os.Getenv("SCP_PORT")
    serverUser := os.Getenv("SCP_USER")
    serverIdentity := os.Getenv("SCP_IDENTITY")

    clientConfig, _ := auth.PrivateKey(serverUser,serverIdentity, ssh.InsecureIgnoreHostKey())

	for true {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		jsonInput := scanner.Text()

		// 汎用のマップにパース
		var genericEvent map[string]interface{}
		err = json.Unmarshal([]byte(jsonInput), &genericEvent)
		if err != nil {
			fmt.Println("Error parsing JSON:", err)
			return
		}

		// イベントタイプを取得
		eventType, ok := genericEvent["event"].(string)
		if !ok {
			fmt.Println("Invalid event type")
			return
		}

		// イベントタイプに対応する構造体を取得
		eventStruct, ok := EventMap[eventType]
		if !ok {
			fmt.Println("Unknown event type")
			return
		}

		// 対応する構造体にパース
		eventBytes, err := json.Marshal(genericEvent)
		if err != nil {
			fmt.Println("Error marshaling generic event:", err)
			return
		}

		err = json.Unmarshal(eventBytes, &eventStruct)
		if err != nil {
			fmt.Println("Error parsing event to struct:", err)
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

			err = os.WriteFile("log", resOut, 0644)
			if err != nil {
				errRes := UploadErrorResponse{
					Event: "complete",
					Oid:   upload.Oid,
					Error: struct {
						Code    int    `json:"code"`
						Message string `json:"message"`
					}{
						Code:    3,
						Message: "Error Logging: " + err.Error(),
					},
				}
                fmt.Println(errRes)
				return
			}

			fmt.Println(string(resOut))
		} else if eventType == "download" {
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

            err := client.Connect()
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

            f,_:= os.Create(download.Oid)
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
            
            err = os.WriteFile("log", resOut, 0644)
            if err != nil {
                errRes := DownloadErrorResponse{
                    Event: "complete",
                    Oid:   download.Oid,
                    Error: struct {
                        Code    int    `json:"code"`
                        Message string `json:"message"`
                    }{
                        Code:    3,
                        Message: "Error Logging: " + err.Error(),
                    },
                }
                fmt.Println(errRes)
                return
            }

            fmt.Println(string(resOut))

		} else if eventType == "terminate" {
			break
		}
	}
}
