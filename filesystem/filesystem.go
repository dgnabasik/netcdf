package filesystem
// file system, error handling, logging utils.
import (
	"bufio"
	"fmt"
	"net"
	//"log"
	"os"
	"strings"
	"time"
)

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

