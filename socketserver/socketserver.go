// Create a separate Goroutine to serve each TCP client.  Execute netcat to test: nc 127.0.0.1 <port>
// lsof -Pnl +M -i4	or -i6		netstat -tulpn
// sudo nmap localhost => 9898/tcp open  monkeycom  (because it uses gorilla/websocket - haha)
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/apache/iotdb-client-go/client"
	"github.com/gorilla/websocket"
)

const (
	WSHOST        = "localhost" // if not specified, the function listens on all available unicast and anycast IP addresses of the local system.
	SERVER_TYPE   = "tcp4"      // tcp, tcp4, tcp6, unix, or unix packet
	jsonExtension = ".json"
	crlf          = "\n"
)

var WSPORT string = ":9898" // override in main()
var addr = flag.String("addr", WSHOST+WSPORT, "http service address")
var clients = make(map[*websocket.Conn]bool)

// for both IotDB & GraphDB.
var databaseCommands = []string{"login=<my name>", "groups", "group.device=<group>", "timeseries=<group.device>", "data=<group.device> interval=<1s> format=<csv>", "logout"}

///////////////////////////////////////////////////////////////////////////////////////////

// Limit the buffer sizes to the maximum expected message size.
var upgrader = websocket.Upgrader{ //  EnableCompression: true,
	ReadBufferSize:  256,
	WriteBufferSize: 4096,
}

// for javascript client
func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Home Page")
}

// Check if incoming request from a different domain is allowed to connect; get CORS.
// Connections support one concurrent reader and one concurrent writer.
// The server must enforce an origin policy using the Origin request header sent by the browser.
func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	// TODO: check the Origin header before calling upgrader.CheckOrigin.
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	log.Println("Client connected.")
	err = ws.WriteMessage(1, []byte("Available commands: "+strings.Join(databaseCommands, "; ")))
	if err != nil {
		log.Println(err)
	}
	reader(ws)
}

func setupRoutes() {
	http.HandleFunc("/", homePage) // http.FileServer(http.Dir("./jsclient"))
	http.HandleFunc("/ws", wsEndpoint)
}

// Listen indefinitely. The WebSocket protocol distinguishes between TextMessage(UTF-8) and BinaryMessage. The interpretation of binary messages is left to the application.
// The WebSocket protocol defines three types of control messages: close, ping, pong.
func reader(conn *websocket.Conn) {
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Println(string(p)) //<<<
		if err := conn.WriteMessage(messageType, p); err != nil {
			log.Println(err)
			return
		}
	}
}

///////////////////////////////////////////////////////////////////////////////////////////

// abort
func checkErr(title string, err error) {
	if err != nil {
		fmt.Print(title + ": ")
		fmt.Println(err)
		log.Fatal(err)
	}
}

// FileExists Returns false if directory.
func FileExists(filePath string) (bool, error) {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false, nil
	}
	return !info.IsDir(), err
}

///////////////////////////////////////////////////////////////////////////////////////////

type TimeseriesMetadata struct { // <<<
	Description string `json:"description"`
}

func ReadJsonFile(filename string) (TimeseriesMetadata, error) {
	funcName := "TimeseriesMetadata.ReadJsonFile"
	file, err := os.Open(filename)
	checkErr(funcName, err)
	defer file.Close()

	byteValue, err := io.ReadAll(file)
	checkErr(funcName, err)

	var data TimeseriesMetadata
	err = json.Unmarshal(byteValue, &data)
	checkErr(funcName, err)

	return data, nil
}

func GetTimeseriesMetadata(filePath string) (TimeseriesMetadata, bool) {
	found, _ := FileExists(filePath)
	tsmd := TimeseriesMetadata{Description: "not defined"}
	if !found {
		return tsmd, false
	}
	tsmd, err := ReadJsonFile(filePath)
	checkErr("GetTimeseriesMetadata: ", err)
	return tsmd, true
}

func GetMetadataFilename(dataFilePath string) string {
	fileName := filepath.Base(dataFilePath)
	return filepath.Dir(dataFilePath) + "/metadata_" + fileName[:len(fileName)-len(filepath.Ext(fileName))] + jsonExtension
}

///////////////////////////////////////////////////////////////////////////////////////////

type IotdbTimeseriesProfile struct {
	Timeseries         string `json:"timeseries"`
	Alias              string `json:"alias"`
	Database           string `json:"database"`
	DataType           string `json:"datatype"`
	Encoding           string `json:"encoding"`
	Compression        string `json:"compression"`
	Tags               string `json:"tags"`
	Attributes         string `json:"attributes"`
	Deadband           string `json:"deadband"`
	DeadbandParameters string `json:"deadbandparameters"`
}

type IoTDbAccess struct {
	session            client.Session
	Sql                string   `json:"sql"`
	ActiveSession      bool     `json:"activesession"`
	TimeseriesCommands []string `json:"timeseriescommands"` // given as command-line parameters
	QueryResults       []string `json:"queryresults"`
}

func printDataSet(sds *client.SessionDataSet) []string {
	const tab = "\t"
	output := make([]string, 0)
	showTimestamp := !sds.IsIgnoreTimeStamp()
	if showTimestamp {
		output = append(output, "Time"+tab+tab+tab+tab)
	}
	for i := 0; i < sds.GetColumnCount(); i++ {
		output = append(output, sds.GetColumnName(i)+tab)
	}
	output = append(output, crlf)

	for next, err := sds.Next(); err == nil && next; next, err = sds.Next() {
		if showTimestamp {
			output = append(output, sds.GetText(client.TimestampColumnName)+tab)
		}
		for i := 0; i < sds.GetColumnCount(); i++ {
			columnName := sds.GetColumnName(i)
			v := sds.GetValue(columnName)
			if v == nil {
				v = "null"
			}
			str := fmt.Sprintf("%v\t\t", v)
			output = append(output, str)
		}
		output = append(output, crlf)
	}
	return output
}

/* <<<<

// Returns [groupName.deviceName.measurementName]
func (itp IotdbTimeseriesProfile) Format_Timeseries(list []IotdbTimeseriesProfile) []string {
	output := make([]string, len(list))
	for ndx := range list {
		output[ndx] = strings.Replace(list[ndx].Timeseries, EtsidataRoot+".", "", 1)
	}
	return output
}

iotdbTimeseriesList := iot.GetTimeseriesList([]string{iot.GroupName + "."}, iot.DatasetName+".")

// Assign iotAccess.QueryResults; return []IotdbTimeseriesProfile
// |Timeseries|Alias|Database|DataType|Encoding|Compression|Tags|Attributes|Deadband|DeadbandParameters|
func (iotAccess *IoTDbAccess) GetTimeseriesList(groupNames []string, datasetName string) []IotdbTimeseriesProfile {
	var timeout int64 = 1000
	const blockSize = 11
	timeseriesList := make([]IotdbTimeseriesProfile, 0) // want multiples of this number
	timeseriesItem := IotdbTimeseriesProfile{}
	for groupIndex := 0; groupIndex < len(groupNames); groupIndex++ {
		iotAccess.Sql = "show timeseries " + EtsidataRoot + "." + datasetName + groupNames[groupIndex] + "*;"
		sessionDataSet, err := iotAccess.session.ExecuteQueryStatement(iotAccess.Sql, &timeout)
		if err == nil {
			lines := printDataSet(sessionDataSet)
			for ndx, str := range lines { // first 10 lines are column headers, then 3 blank lines between each timeseries.
				if ndx > (blockSize - 1) {
					index := ndx % blockSize
					line := strings.TrimSpace(str)
					switch index {
					case 0:
						timeseriesItem = IotdbTimeseriesProfile{}
						timeseriesItem.Timeseries = line
					case 1:
						timeseriesItem.Alias = line
					case 2:
						timeseriesItem.Database = line
					case 3:
						timeseriesItem.DataType = line
					case 4:
						timeseriesItem.Encoding = line
					case 5:
						timeseriesItem.Compression = line
					case 6:
						timeseriesItem.Tags = line
					case 7:
						timeseriesItem.Attributes = line
					case 8:
						timeseriesItem.Deadband = line
					case 9:
						timeseriesItem.DeadbandParameters = line
					case 10:
						timeseriesList = append(timeseriesList, timeseriesItem)
					}
				}
			}
			sessionDataSet.Close()
		} else {
			checkErr("iotAccess.GetTimeseriesList("+datasetName+")", err)
		}
	} // for groupIndex

	// format/assign iotAccess.QueryResults
	const cwidth0 = 100
	const cwidth1 = 10
	const sep = "|"
	iotAccess.QueryResults = make([]string, 0) // skip Alias,Database,Tags|Attributes|Deadband|DeadbandParameters
	iotAccess.QueryResults = append(iotAccess.QueryResults, "                                     Timeseries |DataType|Encoding|Compress|")
	for ndx := 0; ndx < len(timeseriesList); ndx++ {
		item0 := strings.Repeat(" ", cwidth0-len(timeseriesList[ndx].Timeseries)) + timeseriesList[ndx].Timeseries + sep
		item2 := strings.Repeat(" ", cwidth1-len(timeseriesList[ndx].DataType)) + timeseriesList[ndx].DataType + sep
		item3 := strings.Repeat(" ", cwidth1-len(timeseriesList[ndx].Encoding)) + timeseriesList[ndx].Encoding + sep
		item4 := strings.Repeat(" ", cwidth1-len(timeseriesList[ndx].Compression)) + timeseriesList[ndx].Compression + sep
		iotAccess.QueryResults = append(iotAccess.QueryResults, item0+item2+item3+item4)
	}

	return timeseriesList
}
*/

func main() {
	flag.Parse()
	log.SetFlags(0)
	WSPORT := os.Getenv("WSPORT")
	setupRoutes()
	fmt.Println("The socketserver program only reads from the the IoT and Graph databases.")
	fmt.Println("The client programs display the available commands to the server.")
	fmt.Println("Server is listening on " + WSHOST + WSPORT + " ...")
	log.Fatal(http.ListenAndServe(*addr, nil))
}

/*
metadata, found := GetTimeseriesMetadata(GetMetadataFilename(programArgs[1]))
*/

var homeTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<script>  
window.addEventListener("load", function(evt) {

    var output = document.getElementById("output");
    var input = document.getElementById("input");
    var ws;

    var print = function(message) {
        var d = document.createElement("div");
        d.textContent = message;
        output.appendChild(d);
        output.scroll(0, output.scrollHeight);
    };

    document.getElementById("open").onclick = function(evt) {
        if (ws) {
            return false;
        }
        ws = new WebSocket("{{.}}");
        ws.onopen = function(evt) {
            print("OPEN");
        }
        ws.onclose = function(evt) {
            print("CLOSE");
            ws = null;
        }
        ws.onmessage = function(evt) {
            print("RESPONSE: " + evt.data);
        }
        ws.onerror = function(evt) {
            print("ERROR: " + evt.data);
        }
        return false;
    };

    document.getElementById("send").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        print("SEND: " + input.value);
        ws.send(input.value);
        return false;
    };

    document.getElementById("close").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        ws.close();
        return false;
    };

});
</script>
</head>
<body>
<table>
<tr><td valign="top" width="50%">
<p>Click "Open" to create a connection to the server, 
"Send" to send a message to the server and "Close" to close the connection. 
You can change the message and send multiple times.
<p>
<form>
<button id="open">Open</button>
<button id="close">Close</button>
<p><input id="input" type="text" value="Hello world!">
<button id="send">Send</button>
</form>
</td><td valign="top" width="50%">
<div id="output" style="max-height: 70vh;overflow-y: scroll;"></div>
</td></tr></table>
</body>
</html>
`))
