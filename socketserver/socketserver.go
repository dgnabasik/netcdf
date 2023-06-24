// Create a separate Goroutine to serve each TCP client.  Execute netcat to test: nc 127.0.0.1 <port>
// lsof -Pnl +M -i4	or -i6		netstat -tulpn
// sudo nmap localhost => 9898/tcp open  monkeycom  (because it uses gorilla/websocket - haha)
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/apache/iotdb-client-go/client"
	"github.com/gorilla/websocket"
)

const (
	timeFormat    = "2006-01-02 15:04:05" // RFC3339 format.
	EtsidataRoot  = "root.etsidata"
	WSHOST        = "localhost" // if not specified, the function listens on all available unicast and anycast IP addresses of the local system.
	SERVER_TYPE   = "tcp4"      // tcp, tcp4, tcp6, unix, or unix packet
	jsonExtension = ".json"
	crlf          = "\n"
)

var WSPORT string = ":9898" // override in main()
var addr = flag.String("addr", WSHOST+WSPORT, "http service address")
var iotdbParameters IoTDbProgramParameters
var clientConfig *client.Config

// for both IotDB & GraphDB.
var databaseCommands = []string{"login <myName>", "groups", "group.device <group>", "timeseries <group.device>", "data <group.device> interval <1s> format <csv>", "logout"}

///////////////////////////////////////////////////////////////////////////////////////////

// for javascript client
func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Home Page")
}

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

func testRemoteAddressPortsOpen(host string, ports []string) (bool, error) {
	for _, port := range ports {
		timeout := time.Second
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
		if err != nil {
			return false, err
		}
		checkErr("testRemoteAddressPortsOpen(connection error)", err)
		if conn != nil {
			defer conn.Close()
			fmt.Println("Connected to IoTDB at ", net.JoinHostPort(host, port))
		}
	}
	return true, nil
}

///////////////////////////////////////////////////////////////////////////////////////////

type IoTDbProgramParameters struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}

// First read environment variables: IOTDB_PASSWORD, IOTDB_USER, IOTDB_HOST, IOTDB_PORT; then read override parameters from command-line.
// Assigns global iotdbParameters and returns client.Config. iotdbParameters is a superset of client.Config.
func configureIotdbAccess() *client.Config {
	iotdbParameters = IoTDbProgramParameters{
		Host:     os.Getenv("IOTDB_HOST"),
		Port:     os.Getenv("IOTDB_PORT"),
		User:     os.Getenv("IOTDB_USER"),
		Password: os.Getenv("IOTDB_PASSWORD"),
	}
	envFound := len(iotdbParameters.Host) > 0 && len(iotdbParameters.Port) > 0
	if !envFound {
		flag.StringVar(&iotdbParameters.Host, "host", "127.0.0.1", "--host=10.103.4.83")
		flag.StringVar(&iotdbParameters.Port, "port", "6667", "--port=6667") // sudo netstat -peanut | grep 6667 ==> 3 lines
		flag.StringVar(&iotdbParameters.User, "user", "root", "--user=root")
		flag.StringVar(&iotdbParameters.Password, "password", "root", "--password=root")
		flag.Parse()
	}
	config := &client.Config{
		Host:     iotdbParameters.Host,
		Port:     iotdbParameters.Port,
		UserName: iotdbParameters.User,
		Password: iotdbParameters.Password,
	}
	return config
}

// Assigns global clientConfig.
func Init_IoTDB(testIotdbAccess bool) (string, bool) {
	fmt.Println("Initializing IoTDB client...")
	clientConfig = configureIotdbAccess()
	isOpen, err := testRemoteAddressPortsOpen(clientConfig.Host, []string{clientConfig.Port})
	connectStr := clientConfig.Host + ":" + clientConfig.Port
	if testIotdbAccess && !isOpen {
		fmt.Printf("%s%v%s", "Expected IoTDB to be available at "+connectStr+" but got ERROR: ", err, "\n")
		fmt.Printf("Please execute:  cd ~/iotdb && sbin/start-standalone.sh && sbin/start-cli.sh -h 127.0.0.1 -p 6667 -u root -pw root")
		return connectStr, false
	}
	return connectStr, true
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

// Returns [groupName.deviceName.measurementName]  	Skip Alias,Database,Tags|Attributes|Deadband|DeadbandParameters
func (itp IotdbTimeseriesProfile) Format_Timeseries(list []IotdbTimeseriesProfile) []string {
	const cwidth0 = 100
	const cwidth1 = 10
	const sep = "|"
	output := make([]string, len(list)+1)
	output[0] = "                                     Timeseries |DataType|Encoding|Compress|"
	for ndx := 0; ndx < len(list); ndx++ {
		item0 := strings.Repeat(" ", cwidth0-len(list[ndx].Timeseries)) + list[ndx].Timeseries + sep
		item2 := strings.Repeat(" ", cwidth1-len(list[ndx].DataType)) + list[ndx].DataType + sep
		item3 := strings.Repeat(" ", cwidth1-len(list[ndx].Encoding)) + list[ndx].Encoding + sep
		item4 := strings.Repeat(" ", cwidth1-len(list[ndx].Compression)) + list[ndx].Compression + sep
		output[ndx+1] = item0 + item2 + item3 + item4
	}

	return output
}

///////////////////////////////////////////////////////////////////////////////////////////

type IoTDbAccess struct {
	session            client.Session
	Sql                string                   `json:"sql"`
	ActiveSession      bool                     `json:"activesession"`
	TimeseriesCommands []string                 `json:"timeseriescommands"` // given as command-line parameters
	QueryResults       []string                 `json:"queryresults"`       // every message the client will ever get
	socketserver       *websocket.Conn          // new fields
	UserName           string                   `json:"username"`
	TimeseriesList     []IotdbTimeseriesProfile `json:"timeserieslist"`
	QueryIndex         int                      `json:"queryindex"` // index of current message sent to client
}

// No header returned
func (iotAccess *IoTDbAccess) Format_Timeseries() {
	iotAccess.QueryResults = make([]string, len(iotAccess.TimeseriesList))
	for ndx := 0; ndx < len(iotAccess.TimeseriesList); ndx++ {
		iotAccess.QueryResults[ndx] = strings.Replace(iotAccess.TimeseriesList[ndx].Timeseries, EtsidataRoot+".", "", 1)
	}
	fmt.Println(len(iotAccess.QueryResults)) //<<<
}

func (iotAccess *IoTDbAccess) PrintDataSet(sds *client.SessionDataSet) []string {
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

// Assign iotAccess.[]IotdbTimeseriesProfile
// |Timeseries|Alias|Database|DataType|Encoding|Compression|Tags|Attributes|Deadband|DeadbandParameters|
func (iotAccess *IoTDbAccess) GetTimeseriesList(groupNames []string, datasetName string) {
	if err := clients.session.Open(false, 0); err != nil {
		checkErr("GetTimeseriesList: ", err)
	}
	defer clients.session.Close()

	var timeout int64 = 1000
	const blockSize = 11
	iotAccess.TimeseriesList = make([]IotdbTimeseriesProfile, 0) // want multiples of this number
	timeseriesItem := IotdbTimeseriesProfile{}
	for groupIndex := 0; groupIndex < len(groupNames); groupIndex++ {
		iotAccess.Sql = "show timeseries " + EtsidataRoot + "." + datasetName + groupNames[groupIndex] + "*;"
		fmt.Println(iotAccess.Sql)
		sessionDataSet, err := iotAccess.session.ExecuteQueryStatement(iotAccess.Sql, &timeout)
		if err == nil {
			lines := iotAccess.PrintDataSet(sessionDataSet)
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
						iotAccess.TimeseriesList = append(iotAccess.TimeseriesList, timeseriesItem)
					}
				}
			}
			sessionDataSet.Close()
		} else {
			checkErr("iotAccess.GetTimeseriesList("+datasetName+")", err)
		}
	} // for groupIndex
}

// map to databaseCommands = []string{"login <myName>", "groups", "group.device <group>", "timeseries <group.device>", "data <group.device> interval <1s> format <csv>", "logout"}
func (iotAccess *IoTDbAccess) RoutingParser(clientCommand string) {
	tokens := strings.Split(clientCommand, " ")
	baseCommand := strings.ToLower(tokens[0])
	switch baseCommand {
	case "login":
		iotAccess.ActiveSession = true
		iotAccess.UserName = tokens[1]
		iotAccess.QueryResults = []string{iotAccess.UserName + "successfully logged in at " + time.Now().Format(timeFormat)}

	case "groups":
		iotAccess.GetTimeseriesList([]string{""}, "*")
		iotAccess.Format_Timeseries()

	case "group.device":
		tokens2 := strings.Split(tokens[1], ".")
		groupName := tokens2[0]
		datasetName := tokens2[1]
		iotAccess.GetTimeseriesList([]string{groupName}, datasetName)
		iotAccess.Format_Timeseries()

	case "timeseries": // <group.device>
		tokens2 := strings.Split(tokens[1], ".")
		groupName := tokens2[0]
		datasetName := tokens2[1]
		iotAccess.GetTimeseriesList([]string{groupName}, datasetName)
		iotAccess.Format_Timeseries()

	case "data": //<<<< <group.device> interval <1s> format <csv>"

	case "logout":
		iotAccess.ActiveSession = false
		iotAccess.QueryResults = []string{"thank you ... logging out at " + time.Now().Format(timeFormat)}

	default:
		iotAccess.QueryResults = []string{"invalid command"}
	}
}

// Check if incoming request from a different domain is allowed to connect; get CORS.
// Connections support one concurrent reader and one concurrent writer.
// The server must enforce an origin policy using the Origin request header sent by the browser.
func (iotAccess *IoTDbAccess) wsEndpoint(w http.ResponseWriter, r *http.Request) {
	// Limit the buffer sizes to the maximum expected message size.
	var upgrader = websocket.Upgrader{ //  EnableCompression: true,
		ReadBufferSize:  256,
		WriteBufferSize: 4096,
	}

	// TODO: check the Origin header before calling upgrader.CheckOrigin.
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	iotAccess.socketserver, _ = upgrader.Upgrade(w, r, nil)
	log.Println("Client connected.") // 1=TextMessage
	err := iotAccess.socketserver.WriteMessage(1, []byte("Available commands: "+strings.Join(databaseCommands, "; ")))
	if err != nil {
		log.Println(err)
	}
	iotAccess.reader()
}

// Listen indefinitely. The WebSocket protocol distinguishes between TextMessage(UTF-8) and BinaryMessage. The interpretation of binary messages is left to the application.
// The WebSocket protocol defines three types of control messages: close, ping, pong.
func (iotAccess *IoTDbAccess) reader() { // conn *websocket.Conn) {
	for {
		messageType, clientRequest, err := iotAccess.socketserver.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Println("reader: " + string(clientRequest))
		iotAccess.RoutingParser(string(clientRequest))
		for iotAccess.QueryIndex = 0; iotAccess.QueryIndex < len(iotAccess.QueryResults); iotAccess.QueryIndex++ {
			message := []byte(iotAccess.QueryResults[iotAccess.QueryIndex])
			if err := iotAccess.socketserver.WriteMessage(messageType, message); err != nil {
				log.Println(err)
				return
			}
		}
	}
}

///////////////////////////////////////////////////////////////////////////////////////////

var clients IoTDbAccess

//var clients = make(map[*websocket.Conn]bool)

func setupRoutes() {
	iotdbConnection, ok := Init_IoTDB(true)
	if !ok {
		checkErr("Init_IoTDB: ", errors.New(iotdbConnection))
	}
	clients = IoTDbAccess{ActiveSession: true, session: client.NewSession(clientConfig)}

	http.HandleFunc("/", homePage) // http.FileServer(http.Dir("./jsclient"))
	http.HandleFunc("/ws", clients.wsEndpoint)
	fmt.Println("Routes established.")
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	WSPORT := ":" + os.Getenv("WSPORT")
	fmt.Println("The socketserver program only reads from the the IoT and Graph databases.")
	fmt.Println("Server is listening on " + WSHOST + WSPORT + " ...")
	setupRoutes()
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
