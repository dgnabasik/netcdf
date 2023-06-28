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
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/apache/iotdb-client-go/client"
	"github.com/gorilla/websocket"
)

const (
	TimeFormat    = "2006-01-02T15:04:05Z" // RFC3339 format.
	EtsidataRoot  = "root.etsidata"
	WSHOST        = "localhost" // if not specified, the function listens on all available unicast and anycast IP addresses of the local system.
	JsonExtension = "json"
	CsvExtension  = "csv"
	crlf          = "\n"
)

var WSPORT string = ":9898" // override in main()
var addr = flag.String("addr", WSHOST+WSPORT, "http service address")
var iotdbParameters IoTDbProgramParameters
var clientConfig *client.Config
var timeout int64 = 1000

// for both IotDB & GraphDB.
var DatabaseCommands = []string{"login <myName>", "groups", "group.device <group>", "timeseries <group.device>", "count <group.device>", "data <group.device> interval <1s> format <csv>", "logout", "stop"}
var OutputFormats = []string{CsvExtension, JsonExtension}
var ParameterList = []string{"interval", "format", "limit", "startdate", "enddate", "loop"}

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

// Non-generic version looks for first embedded string match. Return empty string if not found.
func find(lines []string, target string) (string, int) {
	for ndx, v := range lines {
		if strings.Contains(v, target) {
			return lines[ndx], ndx
		}
	}
	return "", -1
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

type TimeseriesMetadata struct {
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
	return filepath.Dir(dataFilePath) + "/metadata_" + fileName[:len(fileName)-len(filepath.Ext(fileName))] + "." + JsonExtension
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
	Interval           float64                  `json:"interval"`   // seconds
	OutputFormat       string                   `json:"outputformat"`
	RowLimit           int                      `json:"rowlimit"`
	StartDate          time.Time                `json:"startdate"`
	EndDate            time.Time                `json:"enddate"`
	LoopOutput         int                      `json:"loopoutput"`
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

// data synthetic.IoT_Motion_Light interval 0.5 format csv
func (iotAccess *IoTDbAccess) GetTimeseriesData(datasetName, groupName string, parameters []string) {
	if err := clients.session.Open(false, 0); err != nil {
		checkErr("GetTimeseriesData: ", err)
	}
	defer clients.session.Close()

	// process parameters
	for ndx := 0; ndx < len(parameters); ndx += 2 {
		pm, index := find(ParameterList, strings.ToLower(parameters[ndx]))

		if pm == "interval" && index >= 0 {
			interval, err := strconv.ParseFloat(parameters[ndx+1], 64)
			if err != nil || interval < 0 || interval > 3600 {
				fmt.Println("Could not parse interval " + parameters[ndx+1])
				iotAccess.Interval = 0
			} else {
				iotAccess.Interval = interval
			}
		}

		if pm == "format" && index >= 0 && len(parameters) >= ndx {
			str, ndx2 := find(OutputFormats, strings.ToLower(parameters[ndx+1]))
			if ndx2 >= 0 {
				iotAccess.OutputFormat = str
			} else {
				fmt.Println("Could not parse format " + parameters[ndx+1])
				iotAccess.OutputFormat = OutputFormats[0]
			}
		}

		if pm == "limit" && index >= 0 && len(parameters) >= ndx {
			limit, err := strconv.Atoi(parameters[ndx+1])
			if err != nil || limit < 0 || limit > 1000000 {
				fmt.Println("Could not parse limit " + parameters[ndx+1])
				iotAccess.RowLimit = limit
			} else {
				iotAccess.RowLimit = 0
			}
		}

		if pm == "loop" && index >= 0 && len(parameters) >= ndx {
			loop, err := strconv.Atoi(parameters[ndx+1])
			if err != nil || loop < 0 || loop > 1000000 {
				fmt.Println("Could not parse loop " + parameters[ndx+1])
				iotAccess.LoopOutput = loop
			} else {
				iotAccess.LoopOutput = 0
			}
		}

		if pm == "startdate" && index >= 0 && len(parameters) >= ndx {
			startdate, err := time.Parse(TimeFormat, parameters[ndx+1])
			if err != nil {
				fmt.Println("Could not parse startdate " + parameters[ndx+1])
				iotAccess.StartDate = startdate
			} else {
				iotAccess.StartDate = time.Time{}
			}
		}

		if pm == "enddate" && index >= 0 && len(parameters) >= ndx {
			enddate, err := time.Parse(TimeFormat, parameters[ndx+1])
			if err != nil {
				fmt.Println("Could not parse enddate " + parameters[ndx+1])
				iotAccess.EndDate = enddate
			} else {
				iotAccess.EndDate = time.Time{}
			}
		}
	}

	iotAccess.Sql = "SELECT * FROM " + EtsidataRoot + "." + datasetName + "." + groupName + " ORDER BY time ASC LIMIT 101;" //<<<
	sessionDataSet, err := iotAccess.session.ExecuteQueryStatement(iotAccess.Sql, &timeout)
	// 'Time' + each measurement name (unordered) + DatasetName + 2 blank lines + each measurement value + 2 blank lines
	if err == nil {
		fmt.Println(iotAccess.Sql)
		lines := iotAccess.PrintDataSet(sessionDataSet)
		sessionDataSet.Close()
		// collect headers. Time column always has value 0.
		headers := make(map[string]int)
		ndx := 0
		processing := true
		var sb strings.Builder
		for processing {
			measurementName := strings.Replace(strings.TrimSpace(lines[ndx]), EtsidataRoot+"."+datasetName+"."+groupName+".", "", 1)
			headers[measurementName] = ndx
			sb.WriteString(measurementName + ",") // csv
			processing = len(strings.TrimSpace(lines[ndx+1])) > 0
			ndx++
		}
		iotAccess.QueryResults = make([]string, 0)
		iotAccess.QueryResults = append(iotAccess.QueryResults, sb.String()[0:len(sb.String())-1]) // remove trailing ','
		// process unknown number of data elements; 1 per line
		ndx++
		processing = true
		nResults := 1
		for processing {
			sb.Reset()
			for processing {
				sb.WriteString(strings.TrimSpace(lines[ndx]) + ",") // csv
				processing = len(lines) > ndx+1
				if processing {
					processing = len(strings.TrimSpace(lines[ndx])) > 0
				}
				ndx++
			}
			iotAccess.QueryResults = append(iotAccess.QueryResults, sb.String()[0:len(sb.String())-2]) // remove trailing ',,'
			nResults++
			processing = len(lines) >= ndx+1
		}
		fmt.Print(len(iotAccess.QueryResults))
		fmt.Println(" rows will be sent to the client.")
	} else {
		checkErr("GetTimeseriesData("+datasetName+")", err)
	}
}

// SELECT COUNT(*) FROM root.etsidata.synthetic.IoT_Motion_Light;
func (iotAccess *IoTDbAccess) GetTimeseriesCount(datasetName, groupName string) {
	const cwidth1 = 9

	if err := clients.session.Open(false, 0); err != nil {
		checkErr("GetTimeseriesCount: ", err)
	}
	defer clients.session.Close()

	iotAccess.Sql = "SELECT COUNT(*) FROM " + EtsidataRoot + "." + datasetName + "." + groupName + ";"
	sessionDataSet, err := iotAccess.session.ExecuteQueryStatement(iotAccess.Sql, &timeout)
	nameList := make([]string, 0)
	countList := make([]string, 0)
	maxLen := 0
	if err == nil {
		fmt.Println(iotAccess.Sql)
		lines := iotAccess.PrintDataSet(sessionDataSet)
		sessionDataSet.Close()
		for ndx := range lines {
			if strings.Contains(lines[ndx], "COUNT(") {
				name := strings.TrimSpace(strings.Replace(lines[ndx], "COUNT("+EtsidataRoot+".", "", 1))
				name = strings.Replace(name, ")", "", 1)
				nameList = append(nameList, name)
				if len(name) > maxLen {
					maxLen = len(name)
				}
			} else {
				count := strings.TrimSpace(lines[ndx])
				if len(count) > 0 {
					countList = append(countList, count)
				}
			}
		}
		iotAccess.QueryResults = make([]string, len(nameList))
		for ndx := 0; ndx < len(nameList); ndx++ {
			str := strings.Repeat(" ", maxLen+1-len(nameList[ndx])) + nameList[ndx] + strings.Repeat(" ", cwidth1-len(countList[ndx])) + countList[ndx]
			iotAccess.QueryResults[ndx] = str
		}
		fmt.Print(len(iotAccess.QueryResults))
		fmt.Println(" rows sent to the client.")
	} else {
		checkErr("GetTimeseriesCount("+datasetName+")", err)
	}
}

// Assign iotAccess.[]IotdbTimeseriesProfile
// |Timeseries|Alias|Database|DataType|Encoding|Compression|Tags|Attributes|Deadband|DeadbandParameters|
func (iotAccess *IoTDbAccess) GetTimeseriesList(datasetName string, groupNames []string) {
	if err := clients.session.Open(false, 0); err != nil {
		checkErr("GetTimeseriesList: ", err)
	}
	defer clients.session.Close()

	const blockSize = 11
	iotAccess.TimeseriesList = make([]IotdbTimeseriesProfile, 0) // want multiples of this number
	timeseriesItem := IotdbTimeseriesProfile{}
	for groupIndex := 0; groupIndex < len(groupNames); groupIndex++ {
		iotAccess.Sql = "show timeseries " + EtsidataRoot + "." + datasetName + groupNames[groupIndex] + "*;"
		sessionDataSet, err := iotAccess.session.ExecuteQueryStatement(iotAccess.Sql, &timeout)
		if err == nil {
			fmt.Println(iotAccess.Sql)
			lines := iotAccess.PrintDataSet(sessionDataSet)
			sessionDataSet.Close()
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
		} else {
			checkErr("GetTimeseriesList("+datasetName+")", err)
		}
	} // for groupIndex
	iotAccess.QueryResults = make([]string, len(iotAccess.TimeseriesList))
	for ndx := 0; ndx < len(iotAccess.TimeseriesList); ndx++ {
		iotAccess.QueryResults[ndx] = strings.Replace(iotAccess.TimeseriesList[ndx].Timeseries, EtsidataRoot+".", "", 1)
	}
	fmt.Print(len(iotAccess.QueryResults))
	fmt.Println(" rows sent to the client.")
}

func prettifyInput(clientCommand string) string {
	space := regexp.MustCompile(`\s+`)
	cc := space.ReplaceAllString(strings.TrimSpace(clientCommand), " ")
	return cc
}

// map to DatabaseCommands = []string{"login <myName>", "groups", "group.device <group>", "timeseries <group.device>", "data <group.device> interval <1s> format <csv>", "logout", "stop"}
// iotAccess.Sql = "show timeseries " + EtsidataRoot + "." + datasetName + groupNames[groupIndex] + "*;"
func (iotAccess *IoTDbAccess) RoutingParser(clientCommand string) {
	cc := prettifyInput(clientCommand)
	tokens := strings.Split(cc, " ")
	baseCommand := strings.ToLower(tokens[0])
	iotAccess.ActiveSession = true

	switch baseCommand {
	case "login":
		iotAccess.UserName = tokens[1]
		iotAccess.QueryResults = []string{iotAccess.UserName + " successfully logged in at " + time.Now().Format(TimeFormat)}

	case "groups":
		datasetName := "*"
		groupNames := []string{""}
		iotAccess.GetTimeseriesList(datasetName, groupNames)

	case "group.device": // show timeseries root.etsidata.synthetic.**;
		datasetName := tokens[1] + "."
		groupNames := []string{"*"}
		iotAccess.GetTimeseriesList(datasetName, groupNames)

	case "timeseries": // <group.device>    show timeseries root.etsidata.synthetic.IoT_Motion_Light.*;
		tokens2 := strings.Split(tokens[1], ".")
		datasetName := tokens2[0] + "."
		groupNames := []string{tokens2[1] + "."}
		iotAccess.GetTimeseriesList(datasetName, groupNames)

	case "count": // <group.device>		SELECT COUNT(*) FROM root.etsidata.synthetic.IoT_Motion_Light;
		tokens2 := strings.Split(tokens[1], ".")
		datasetName := tokens2[0]
		groupName := tokens2[1]
		iotAccess.GetTimeseriesCount(datasetName, groupName)

	case "data": // data <group.device> interval <1s> format <csv>    SELECT status, temperature FROM root.ln.wf01.wt01 WHERE temperature < 24 and time > 2017-11-1 0:13:00
		tokens2 := strings.Split(tokens[1], ".")
		datasetName := tokens2[0]
		groupName := tokens2[1]
		parameters := tokens[2:]
		iotAccess.GetTimeseriesData(datasetName, groupName, parameters)

	case "logout":
		iotAccess.ActiveSession = false
		iotAccess.QueryResults = []string{"thank you ... logging out at " + time.Now().Format(TimeFormat)}

	case "stop":
		iotAccess.ActiveSession = false
		iotAccess.QueryResults = []string{}

	default:
		iotAccess.ActiveSession = false
		iotAccess.QueryResults = []string{"invalid command: " + baseCommand}
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
	err := iotAccess.socketserver.WriteMessage(1, []byte("Available commands: "+strings.Join(DatabaseCommands, "; ")))
	if err != nil {
		log.Println(err)
	}
	iotAccess.reader()
}

// Listen indefinitely. The WebSocket protocol distinguishes between TextMessage(UTF-8) and BinaryMessage.
// The interpretation of binary messages is left to the application.
func (iotAccess *IoTDbAccess) reader() { // conn *websocket.Conn) {
	for {
		messageType, clientRequest, err := iotAccess.socketserver.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		request := string(clientRequest)
		fmt.Println("reader: " + request)
		iotAccess.RoutingParser(request)
		// send data back to the client, one row per time interval.
		fmt.Println("Started sending data to client at " + time.Now().Format(TimeFormat) + " with parameters " + request)
		var duration time.Duration = time.Duration(iotAccess.Interval * 1000)
		continuous := true
		for continuous {
			for iotAccess.QueryIndex = 0; iotAccess.QueryIndex < len(iotAccess.QueryResults); iotAccess.QueryIndex++ {
				message := []byte(iotAccess.QueryResults[iotAccess.QueryIndex])
				if err := iotAccess.socketserver.WriteMessage(messageType, message); err != nil {
					log.Println(err)
					return
				}
				time.Sleep(duration * time.Millisecond)
			}
			continuous = iotAccess.LoopOutput > 0
		}
		fmt.Println("Finished sending data to client at " + time.Now().Format(TimeFormat))
	}
}

///////////////////////////////////////////////////////////////////////////////////////////

var clients IoTDbAccess

/* <<<var clients = make(map[*websocket.Conn]bool)
metadata, found := GetTimeseriesMetadata(GetMetadataFilename(programArgs[1]))
*/

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
