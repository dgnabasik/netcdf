package filesystem

// utilities: datetime, file system, error handling, logging, exec.Command, curl utils; TestRemoteAddressPortsOpen()
// ISO 8601 format: use package iso8601 since The built-in RFC3333 time layout in Go is too restrictive to support any ISO8601 date-time.
import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	DateFormat     = "2006-01-02"
	DateTimeFormat = DateFormat + "T15:04:05Z" // RFC3339 format: "2006-01-02T15:04:05Z07:00"
	TimeFormat1    = DateFormat + " 15:04:05"
	TimeFormatNano = DateFormat + "T15:04:05.000Z07:00" // this is the preferred milliseconds version with a time zone offset (but store as UTC).
)

// Regex expressions to handle diacritical marks for non-ASCII SPARQL queries: map[regex.template]exact.match.in.GraphDB : r := regexp.MustCompile(`{[^{}]*}`)	matches := r.FindAllString("{city}, {state} {zip}", -1).  See GraphDB: MINES-IgnoreCase-NonAscii
var NonAsciiMap = map[string]string{ // [çc]
	".*IMT-MINES-Saint-[ÉE]tienne*":             "IMT-MINES-Saint-Étienne",
	".*[ÉE]coleDesMinesSaint[ÉE]tienne*":        "ÉcoleDesMinesSaintÉtienne",
	".*[ÉE]cole des Mines de Saint-[ÉE]tienne*": "École des Mines de Saint-Étienne",
}

var FormatTypes = []string{"csv", "turtle", "rdf", "triples", "json-ld", "mqtt", "quads"}

func GetCurrentDateTime(dateOnly bool) string {
	t := time.Now()
	if dateOnly {
		return t.Format(DateFormat)
	}
	return t.Format(DateTimeFormat)
}

// someTime can be either a long or a readable dateTime string.
func GetStartTimeFromLongint(someTime string) (time.Time, error) {
	isLong, err := strconv.ParseInt(someTime, 10, 64)
	if err == nil {
		return time.Unix(isLong, 0), nil  // 2016-01-03 02:42:50 +0100 CET
	}

	t1, err := time.Parse(DateTimeFormat, someTime)
	if err == nil {
		return t1, nil
	} 

	t2, err := time.Parse(TimeFormat1, someTime)
	if err == nil {
		return t2, nil
	}

	return time.Now(), err
}

// Return yyyy-MM-dd
func GetDateStr(t time.Time) string {
	return t.Format(DateFormat)
}

func StandardDate(dt time.Time) string {
	var a [20]byte
	var b = a[:0]                        // Using the a[:0] notation converts the fixed-size array to a slice type represented by b that is backed by this array.
	b = dt.AppendFormat(b, time.RFC3339) // AppendFormat() accepts type []byte. The allocated memory a is passed to AppendFormat().
	return string(b[0:10])
}

func FormatFloat(str string) float64 {
	f, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0.0
	}
	return f
}

// Parse xsd:date
func ParseDate(dateValue string) time.Time {
	d, err := time.Parse(DateTimeFormat, dateValue)
	if err == nil {
		return d
	} else {
		return time.Now()
	}
}

// non-generic version looks for first embedded string match. Return -1 if not found.
func Find(lines []string, target string) (int, bool) {
	for ndx, v := range lines {
		if strings.Contains(v, target) {
			return ndx, true
		}
	}
	return -1, false
}

var src = rand.New(rand.NewSource(time.Now().UnixNano()))

func GetRandomIdentifier() string {
	const n = 8
	b := make([]byte, (n+1)/2)
	if _, err := src.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)[:n]
	return string(b)
}

/**********************************************************************************/

// All ports must be open.
func TestRemoteAddressPortsOpen(host string, ports []string) (bool, error) {
	for _, port := range ports {
		timeout := time.Second
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
		if err != nil {
			return false, err
		}
		CheckError("TestRemoteAddressPortsOpen(connection error)", err, true)
		if conn != nil {
			defer conn.Close()
			fmt.Println("Connected to server at ", net.JoinHostPort(host, port))
		}
	}
	return true, nil
}

// return (err != nil)
func CheckError(msg string, err error, abort bool) bool {
	if err != nil {
		xmsg := fmt.Sprintf("%s%+v\n", msg, err)
		fmt.Print(xmsg)
		//log.Printf(xmsg)
		if abort {
			os.Exit(1)
		}
		return true
	}
	return false
}

// FileExists Returns false if directory.
func FileExists(filePath string) (bool, error) {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false, nil
	}
	return !info.IsDir(), err
}

// WriteTextLines writes/appends the lines to the given file.
func WriteTextLines(lines []string, filePath string, appendData bool) error {
	if !appendData {
		itExists, _ := FileExists(filePath)
		if itExists {
			_ = os.Remove(filePath)
		}
	}
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if CheckError("WriteTextLines:1 ", err, false) {
		return err
	}
	defer file.Close()
	contents := strings.Join(lines, "\n")
	_, err = file.WriteString(contents)
	if CheckError("WriteTextLines:2 ", err, false) {
		return err
	}
	return nil
}

// Read a large text file. MODIFIED to remove comments, blank lines, and left-justify.
func ReadTextLines(filePath string, normalizeText bool) ([]string, error) {
	file, err := os.Open(filePath)
	if CheckError("ReadTextLines:1 ", err, false) {
		return nil, err
	}

	fi, err := file.Stat()
	if CheckError("ReadTextLines:2 ", err, false) {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	maxCapacity := fi.Size() + 1
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, int(maxCapacity))

	for scanner.Scan() {
		if normalizeText {
			str := strings.TrimSpace(scanner.Text())
			if len(str) > 0 {
				lines = append(lines, str)
			}
		} else {
			lines = append(lines, scanner.Text())
		}
	}
	return lines, scanner.Err()
}

func WriteStringToFile(filePath, str string) error {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("WriteStringToFile: %+v\n", err)
		return err
	}
	defer file.Close()

	if _, err := file.WriteString(str); err != nil {
		log.Printf("WriteStringToFile: %+v\n", err)
	}
	fmt.Println("Wrote to file " + filePath)
	return err
}

/**********************************************************************************/

type SaveOutput struct {
	SavedOutput []byte
}

func (so *SaveOutput) Write(p []byte) (n int, err error) {
	so.SavedOutput = append(so.SavedOutput, p...)
	return os.Stdout.Write(p)
}

// CLEAR GRAPH <https://mines.ontology.datasets.fr/foaf/DavidGnabasik>	// %3Chttps%3A%2F%2Fmines.ontology.datasets.fr%2Ffoaf%2FDavidGnabasik%3E
func ClearNamedGraph(graphDbQueryUrl, urlEncodedNamedGraphIRI string) error {
	var so SaveOutput
	delete := graphDbQueryUrl + "/statements?name=&infer=true&sameAs=false&update=CLEAR%20%20GRAPH%20" + urlEncodedNamedGraphIRI
	cmd := exec.Command("curl", "-X", "POST", "-H", "Accept:application/sparql-update", "--data-binary", "-d", delete)
	cmd.Stdout = &so
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if CheckError("ClearNamedGraph failed with ", err, false) {
		return err
	}
	return nil
}

// Get server files available for import. See https://graphdb.ontotext.com/documentation/10.0/devhub/rest-api/curl-commands.html#data-import
// Upload foaf Turtle file into IMT repository format. cURL is used because GraphDB imports entire ontologies from the ~/graphdb-import folder.
// Execute ClearNamedGraph() first to avoid duplicates.
// http://localhost:7200/rest/repositories/IMT-MINES-Saint-Etienne/import/server
func ImportOntology(graphDbImportUrl, sourceFileName string) error {
	insert := `{ "fileNames": [ "` + sourceFileName + `" ] }`
	var so SaveOutput
	cmd := exec.Command("curl", "-X", "POST", "-H", "Content-Type: application/json", "-d", insert, graphDbImportUrl)
	cmd.Stdout = &so
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if CheckError("ImportOntology.1 failed with ", err, false) {
		return err
	}
	return nil
}

/**********************************************************************************/

type Tree map[string]Tree

func (tree Tree) Add(path string) {
	frags := strings.Split(path, "/")
	tree.add(frags)
}

func (tree Tree) add(frags []string) {
	if len(frags) == 0 {
		return
	}
	nextTree, ok := tree[frags[0]]
	if !ok {
		nextTree = Tree{}
		tree[frags[0]] = nextTree
	}
	nextTree.add(frags[1:])
}

func (tree Tree) Fprint(w io.Writer, root bool, padding string) {
	if tree == nil {
		return
	}
	index := 0
	for k, v := range tree {
		fmt.Fprintf(w, "%s%s\n", padding+getPadding(root, getBoxType(index, len(tree))), k)
		v.Fprint(w, false, padding+getPadding(root, getBoxTypeExternal(index, len(tree))))
		index++
	}
}

type BoxType int

const (
	Regular BoxType = iota
	Last
	AfterLast
	Between
)

func (boxType BoxType) String() string {
	switch boxType {
	case Regular:
		return "\u251c" // ├
	case Last:
		return "\u2514" // └
	case AfterLast:
		return " "
	case Between:
		return "\u2502" // │
	default:
		panic("invalid box type")
	}
}

func getBoxType(index int, len int) BoxType {
	if index+1 == len {
		return Last
	} else if index+1 > len {
		return AfterLast
	}
	return Regular
}

func getBoxTypeExternal(index int, len int) BoxType {
	if index+1 == len {
		return AfterLast
	}
	return Between
}

func getPadding(root bool, boxType BoxType) string {
	if root {
		return ""
	}
	return boxType.String() + " "
}

/**********************************************************************************/
