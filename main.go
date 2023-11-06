package main // netcdf inserts data into IotDB.
// Getting started with Golang multi-module workspaces: https://go.dev/doc/tutorial/workspaces
// Define TimeseriesDataset as a queryable set of zero or more sequences of Timeseries measurements.
// Define TimeseriesDataStream as a sequence of measurements all with the same type of unit collected at periodic intervals but may include missing values.
// IotDB Data Model: https://iotdb.apache.org/UserGuide/Master/Data-Concept/Data-Model-and-Terminology.html
/* Datatype properties (attributes) relate individuals to literal data whereas object properties relate individuals to other individuals.
   Time series as a sequence of measurements, where each measurement is defined as an object with a value, a named event, and a metric.
   A time series object binds a metric to a resource.
   Equivalent time series => owl:sameAs: hasUnit, hasEquation, hasDistribution.
   Produce json file, DatatypeProperty ontology file, and SPARQL query files file from ncdump outputs.
   Use Named Graphs (Identifier+Title+DatastreamName) as Publish/Subscribe topics?
   /usr/bin/ncdump -k cdf.nc			==> get file type {classic, netCDF-4, others...}
   /usr/bin/ncdump -c Jan_clean.nc		==> gives header + indexed {id, time} data
   https://docs.unidata.ucar.edu/nug/current/index.html		Golang supports compressed file formats {rardecode(rar), gz/gzip, zlib, lzw, bzip2}
   Curated data: Each data row is indexed by {a house ID, a time value}. Programmatically import the data into GraphDB.
   nc files: ./github.com/go-native-netcdf/netcdf/*.nc	./github.com/netcdf-c/*.nc	./Documents/digital-twins/Entity/*.nc
   h5 files: ./Documents/digital-twins/AMP/AMPds2.h5	./github.com/netcdf-c/nc_test4/*.h5	./github.com/nci-doe-data-sharing/flaskProject/mt-cnn/mt_cnn_model.h5 ./github.com/go-native-netcdf/netcdf/hdf5/testdata
   csv files: many.
*/

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"filesystem" // work module
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/apache/iotdb-client-go/client"
	"github.com/fhs/go-netcdf/netcdf"
)

const ( // these do not include trailing >
	HomeDirectory   = "/home/david/" // davidgnabasik
	sparqlExtension = ".sparql"
	csvExtension    = ".csv"
	varExtension    = ".var"
	ncExtension     = ".nc"
	crlf            = "\n"
	endOfFields     = "\\"
	timeAlias       = "Time1"
	maxColumns      = 512
	interpolated    = "interpolated"
	unitsName       = "units"
	LastColumnName  = "DatasetName"
	unknown         = "???"
	zero            = 0.0
	graphDbPrefix   = "PREFIX%20%3A%3Chttp%3A%2F%2Fwww.ontotext.com%2Fgraphdb%2Fsimilarity%2F%3E%0APREFIX%20inst%3A%3Chttp%3A%2F%2Fwww.ontotext.com%2Fgraphdb%2Fsimilarity%2Finstance%2F%3E%0APREFIX%20psi%3A%3Chttp%3A%2F%2Fwww.ontotext.com%2Fgraphdb%2Fsimilarity%2Fpsi%2F%3E%0A"
	graphDbPostfix  = "%3E%3B%0Apsi%3AsearchPredicate%20%3Chttp%3A%2F%2Fwww.ontotext.com%2Fgraphdb%2Fsimilarity%2Fpsi%2Fany%3E%3B%0A%3AsearchParameters%20%22-numsearchresults%208%22%3B%0Apsi%3AentityResult%20%3Fresult%20.%0A%3Fresult%20%3Avalue%20%3Fentity%20%3B%0A%3Ascore%20%3Fscore%20.%20%7D%0A"
)

// Non-generic version looks for first embedded string match. Return empty string if not found.
func find(lines []string, target string) (string, int) {
	for ndx, v := range lines {
		if strings.Contains(v, target) {
			return lines[ndx], ndx
		}
	}
	return "", -1
}

// abort
func checkErr(title string, err error) {
	if err != nil {
		fmt.Print(title + ": ")
		fmt.Println(err)
		log.Fatal(err)
	}
}

// var xsdDatatypeMap = map[string]string{"string": "string", "int": "integer", "integer": "integer", "longint": "long", "int64": "long", "float": "decimal", "double": "decimal", "boolean": "byte", "datetime": "dateTime"} // map cdf to xsd datatypes.
// var DiscreteDistributions = []string{"discreteUniform", "discreteBernoulli", "discreteBinomial", "discretePoisson"}
// var ContinuousDistributions = []string{"continuousNormal", "continuousStudent_t_test", "continuousExponential", "continuousGamma", "continuousWeibull"}
var NetcdfFileFormats = []string{"classic", "netCDF", "netCDF-4", "HDF5"}
var rowsXsdMap = map[string]string{"dateTime": "datetime", "Unicode": "string", "unicode": "string", "Float": "float", "float": "float", "Integer": "integer", "integer": "integer", "Longint": "int64", "longint": "int64", "Double": "double", "double": "double"}

func getBlockSize(nMeasurements int) int {
	if nMeasurements < 256 {
		return 131072 // 131072=16*8192
	}
	if nMeasurements < 512 {
		return 4 * 8192
	}
	return 8192
}

func GetSummaryFilename(dataFilePath string) string {
	fileName := filepath.Base(dataFilePath)
	return filepath.Dir(dataFilePath) + "/summary_" + fileName[:len(fileName)-len(filepath.Ext(fileName))] + csvExtension
}

// Contains an entire month's worth of Entity data for every Variable where each row is indexed by {HouseIndex+LongtimeIndex}.
type EntityCleanData struct {
	HouseIndex    []string   `json:"houseindex"`
	LongtimeIndex []string   `json:"longtimeindex"`
	Data          [][]string `json:"data"`
}

// EntityCleanData initializer. NOT USED
func MakeEntityCleanData(rows int, vars []MeasurementVariable) EntityCleanData {
	ecd := EntityCleanData{}
	cols := len(vars)
	ecd.Data = make([][]string, rows) // make a slice of rows slices
	for i := 0; i < rows; i++ {
		ecd.Data[i] = make([]string, cols) // make a slice of cols in each of rows slices
	}
	return ecd
}

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

// Returns [groupName.deviceName.measurementName]
func (itp IotdbTimeseriesProfile) Format_Timeseries(list []IotdbTimeseriesProfile) []string {
	output := make([]string, len(list))
	for ndx := range list {
		output[ndx] = strings.Replace(list[ndx].Timeseries, ".", "", 1)
	}
	return output
}

func GetTimeseriesCommands(programArgs []string) []string {
	timeseriesCommands := make([]string, 0)
	for ndx := range programArgs {
		cmd, index := find(timeSeriesCommands, strings.ToLower(programArgs[ndx]))
		if index >= 0 {
			timeseriesCommands = append(timeseriesCommands, strings.ToLower(cmd))
		}
	}
	return timeseriesCommands
}

// Format: root.etsidata.<group>.<device>	The Ecobee {id} value acts as a device.  Specific Measurement names are appended to this prefix.
func IotDatasetPrefix(identifier, device string) string {
	return identifier + "." + device
}

// mapped to original column name
type MeasurementItem struct {
	MeasurementName  string `json:"measurementname"`  // Original name
	MeasurementAlias string `json:"measurementalias"` // Names that fit IotDB format; see MeasurementName() => [0-9 a-z A-Z _ ]
	MeasurementType  string `json:"measurementtype"`  // XSD data type
	MeasurementUnits string `json:"measurementunits"`
	ColumnOrder      int    `json:"columnorder"` // Column order from data file
	Ignore           bool   `json:"ignore"`      // in case there is no data in the file
}

func (mi MeasurementItem) ToString() string {
	str := mi.MeasurementName + " : " + mi.MeasurementAlias + " : " + mi.MeasurementType + " : " + mi.MeasurementUnits
	if mi.Ignore {
		str += " : IGNORED!"
	}
	return str
}

// Expects the parameters to every Variable to be all the (2) dimensions (except for the dimension variables).
type MeasurementVariable struct {
	MeasurementItem `json:"measurementitem"`
	DimensionIndex  int    `json:"dimensionindex"` // {0,1,2} Default 0 signifies the Variable is not a Dimension.
	FillValue       string `json:"fillvalue"`
	Comment         string `json:"comment"`
	Calendar        string `json:"calendar"`
}

// Separate struct if we want slice of these in container class.
// Keep these field names different from container structs to avoid confusion.
type IoTDbAccess struct {
	session            client.Session
	Sql                string   `json:"sql"`
	ActiveSession      bool     `json:"activesession"`
	TimeseriesCommands []string `json:"timeseriescommands"` // given as command-line parameters
	QueryResults       []string `json:"queryresults"`
}

// Generic NetCDF container. Not stored in IotDB.
type NetCDF struct {
	IoTDbAccess
	Identifier          string                          `json:"identifier"` // Unique ID to distinguish among different datasets.
	NetcdfType          string                          `json:"netcdftype"` // /usr/bin/ncdump -k cdf.nc ==> NetcdfFileFormats
	Dimensions          map[string]int                  `json:"dimensions"`
	Title               string                          `json:"title"`
	Description         string                          `json:"description"` // dummy
	Conventions         string                          `json:"conventions"`
	Institution         string                          `json:"institution"`
	Code_url            string                          `json:"code_url"`
	Location_meaning    string                          `json:"location_meaning"`
	Datastream_name     string                          `json:"datastream_name"`
	Input_files         string                          `json:"input_files"`
	History             string                          `json:"history"`
	TimeMeasurementName string                          `json:"timemeasurementname"` // index into Measurements map;
	Measurements        map[string]*MeasurementVariable `json:"measurements"`
	HouseIndices        []string                        `json:"houseindices"`    // these unique 2 indices are specific to Ecobee datasets.
	LongtimeIndices     []string                        `json:"longtimeindices"` // Different months will have slightly different HouseIndices!
	// these are not in the source file.
	DataFilePath string     `json:"datafilepath"`
	DatasetName  string     `json:"datasetname"`
	Summary      [][]string `json:"summary"` // from summary file
	Dataset      [][]string `json:"dataset"` // actual data
}

func (cdf NetCDF) ToString(outputVariables bool) string {
	const sep = "\n"
	var sb strings.Builder
	sb.WriteString(sep)
	sb.WriteString("Identifier     : " + cdf.Identifier + sep)
	sb.WriteString("NetcdfType     : " + cdf.NetcdfType + sep)
	sb.WriteString("Dimensions     : ")
	for k, v := range cdf.Dimensions {
		sb.WriteString(k + "=")
		sb.WriteString(strconv.Itoa(v) + "; ")
	}
	sb.WriteString(sep)
	sb.WriteString("Title          : " + cdf.Title + sep)
	sb.WriteString("Description    : " + cdf.Description + sep)
	sb.WriteString("Conventions    : " + cdf.Conventions + sep)
	sb.WriteString("Institution    : " + cdf.Institution + sep)
	sb.WriteString("CodeURL        : " + cdf.Code_url + sep)
	sb.WriteString("LocationMeaning: " + cdf.Location_meaning + sep)
	sb.WriteString("DatastreamName : " + cdf.Datastream_name + sep)
	sb.WriteString("InputFiles     : " + cdf.Input_files + sep)
	sb.WriteString("History        : " + cdf.History + sep)
	sb.WriteString("Measurements      : ")
	sb.WriteString(strconv.Itoa(len(cdf.Measurements)) + sep)

	if outputVariables {
		for _, tv := range cdf.Measurements {
			sb.WriteString(" MeasurementName: " + tv.MeasurementItem.MeasurementName + "; ReturnType: " + tv.MeasurementItem.MeasurementType + "; ")
			if len(tv.MeasurementItem.MeasurementAlias) > 0 {
				sb.WriteString(" LongName: " + tv.MeasurementItem.MeasurementAlias + ";")
			}
			if len(tv.MeasurementItem.MeasurementUnits) > 0 {
				sb.WriteString(" Units: " + tv.MeasurementItem.MeasurementUnits + ";")
			}
			if len(tv.Comment) > 0 {
				sb.WriteString(" Comment: " + tv.Comment + ";")
			}
			if len(tv.FillValue) > 0 {
				sb.WriteString(" FillValue: " + tv.FillValue + ";")
			}
			if len(tv.Calendar) > 0 {
				sb.WriteString(" Calendar: " + tv.Calendar + ";")
			}
			sb.WriteString(sep)
		}
	}

	sb.WriteString(sep)
	return sb.String()
}

// Output struct as JSON. NOT USED.
func (cdf NetCDF) Format_Json() ([]string, error) {
	lines := make([]string, 0)
	json, err := json.MarshalIndent(cdf, "", "  ")
	if err != nil {
		return lines, err
	}
	lines = strings.Split(string(json), "\n")
	return lines, nil
}

// Return list of dataset column names.
func (cdf NetCDF) FormattedColumnNames() string {
	var sb strings.Builder
	for ndx := 0; ndx < len(cdf.Measurements); ndx++ {
		for _, v := range cdf.Measurements {
			if v.ColumnOrder == ndx && !v.Ignore {
				sb.WriteString(v.MeasurementAlias + ",")
			}
		}
	}
	str := sb.String()[0:len(sb.String())-1] + " "
	return str
}

// accept either original or alias names.
func (cdf NetCDF) GetMeasurementVariableFromName(name string) (MeasurementVariable, bool) {
	item, ok := cdf.Measurements[name]
	if ok {
		return *item, true
	}
	// check alias names
	for _, item := range cdf.Measurements {
		if strings.EqualFold(name, item.MeasurementAlias) || strings.EqualFold(name, item.MeasurementItem.MeasurementName) {
			return *item, true
		}
	}
	return MeasurementVariable{}, false
}

// Return all column values except header row. Return false if column name not found. Fill in default values.
func (cdf NetCDF) GetSummaryStatValues(columnName string) ([]string, bool) {
	_, columnIndex := find(summaryColumnNames, columnName)
	if columnIndex < 0 {
		return []string{}, false
	}
	stats := make([]string, len(cdf.Measurements))
	for ndx := 0; ndx < len(cdf.Measurements)-1; ndx++ {
		_, aliasName := StandardName(cdf.Summary[ndx+1][0])
		item, _ := cdf.GetMeasurementVariableFromName(aliasName) // MeasurementVariable
		stats[ndx] = strings.TrimSpace(cdf.Summary[ndx+1][columnIndex])
		if strings.HasPrefix(strings.ToLower(item.MeasurementType), "int") || strings.HasPrefix(strings.ToLower(item.MeasurementType), "long") {
			index := strings.Index(stats[ndx], ".")
			if index >= 0 {
				stats[ndx] = stats[ndx][0:index]
			}
		}
		if len(stats[ndx]) == 0 {
			stats[ndx] = strings.TrimSpace(cdf.Summary[ndx+1][1]) // this type is nearly always Unicode
		}
	}
	return stats, true
}

// Expects comma-separated files. Assigns Dataset or Summary.
func (cdf *NetCDF) ReadCsvFile(filePath string, isDataset bool) error {
	f, err := os.Open(filePath)
	checkErr("Unable to read csv file: ", err)
	defer f.Close()
	fmt.Println("Reading " + filePath)
	csvReader := csv.NewReader(f)
	if !isDataset {
		cdf.Summary, err = csvReader.ReadAll()
	} else {
		cdf.Dataset, err = csvReader.ReadAll()
	}
	checkErr("Unable to parse file as CSV for "+filePath, err)
	return err
}

// Index each dimension
func (cdf *NetCDF) getDimensionMap() map[string]int {
	dimensionMap := make(map[string]int, len(cdf.Dimensions))
	index := 1
	for k := range cdf.Dimensions {
		dimensionMap[k] = index
		index++
	}
	return dimensionMap
}

// Return string-formatted value from: columnName={summaryColumnNames}, fieldName={measurement names}
func (cdf *NetCDF) GetSummaryValue(columnName, fieldName string) string {
	columnNames, found1 := cdf.GetSummaryStatValues(summaryColumnNames[0])
	columnValues, found2 := cdf.GetSummaryStatValues(columnName)
	_, index := find(columnNames, fieldName) // exact match
	if !found1 || !found2 || index < 0 {
		return ""
	}
	return columnValues[index]
}

// search for either original or alias name
func (cdf *NetCDF) GetMeasurementItemFromName(name string) (*MeasurementVariable, bool) {
	item, ok := cdf.Measurements[name]
	if ok {
		return item, true
	}
	for _, item := range cdf.Measurements {
		if name == item.MeasurementName || name == item.MeasurementAlias {
			return item, true
		}
	}
	return item, false
}

func (cdf *NetCDF) GetColumnNumberFromName(columnName string) int {
	for ndx := 0; ndx < len(cdf.Summary[0]); ndx++ { // iterate over summary header row
		if strings.EqualFold(cdf.Summary[0][ndx], columnName) {
			return ndx
		}
	}
	return -1
}

/*
func (cdf *NetCDF) GetRowNumberFromName(rowName string) int {
	for ndx := 0; ndx < maxRows; ndx++ {
		if strings.ToLower(cdf.Summary[ndx][0]) == strings.ToLower(rowName) {
			return ndx
		}
	}
	return -1
}

// Reconcile various timestamp formats. Return xsd:date format (yyyy-MM-dd)
// TimeMeasurementName: {ecobee:3:units=yyyy-MM-dd hh:mm:ss}
func (cdf *NetCDF) GetStartEndDates() (string, string) {
	timeRow := cdf.GetRowNumberFromName(cdf.TimeMeasurementName)  // 2
	sDate := cdf.Summary[timeRow][3]	// const
	eDate := cdf.Summary[timeRow][4]
	unitsColumn := cdf.GetColumnNumberFromName(unitsName)

	if cdf.Summary[0][unitsColumn] == "unixutc" {
		startTime, err := filesystem.GetStartTimeFromLongint(sDate)
		if err != nil {
			sDate = filesystem.GetDateStr(startTime)
		}
		endTime, err := filesystem.GetStartTimeFromLongint(eDate)
		if err != nil {
			eDate = filesystem.GetDateStr(endTime)
		}
	} else {
		sDate = sDate[0:10]
		eDate = eDate[0:10]
	}
	return sDate, eDate
} */

// Expects {Units, DatasetName} fields to have been appended to the summary file. Assign []Measurements. Expects Summary to be assigned. Use XSD data types.
// len() only returns the length of the "external" array.
func (cdf *NetCDF) XsvSummaryTypeMap() {
	cdf.Measurements = make(map[string]*MeasurementVariable, 0)
	// get units column
	unitsColumn := cdf.GetColumnNumberFromName(unitsName)
	ndx1 := 0
	dimMap := cdf.getDimensionMap()
	for ndx := 0; ndx < maxColumns; ndx++ { // iterate over summary file rows.
		dataColumnName, aliasName := StandardName(cdf.Summary[ndx+1][0]) // NOTE!! does not include cdf.Summary[ndx+1][0] == "time" ||
		ignore := (cdf.Summary[ndx+1][2] == "0" && cdf.Summary[ndx+1][4] == "0") || dataColumnName == interpolated
		if ignore {
			fmt.Println("Ignoring empty data column " + dataColumnName)
		}
		endOfMeasurements := cdf.Summary[ndx+1][0] == interpolated || strings.TrimSpace(cdf.Summary[ndx+1][0]) == endOfFields
		if endOfMeasurements {
			cdf.Identifier = cdf.Summary[ndx+1][1]
			ndx1 = ndx
			break
		}
		if !endOfMeasurements {
			mi := MeasurementItem{
				MeasurementName:  dataColumnName,
				MeasurementAlias: aliasName,
				MeasurementType:  rowsXsdMap[cdf.Summary[ndx+1][1]],
				MeasurementUnits: cdf.Summary[ndx+1][unitsColumn],
				ColumnOrder:      ndx,
				Ignore:           ignore,
			}
			dimIndex, _ := dimMap[dataColumnName]
			mv := MeasurementVariable{
				MeasurementItem: mi,
				DimensionIndex:  dimIndex,
				FillValue:       "0. ", // or ""
				Comment:         "",
				Calendar:        "",
			}
			cdf.Measurements[dataColumnName] = &mv // add to map using original name
		} else {
			break
		}
	}
	// add DatasetName timerseries in case data column names are the same for different sampling intervals.
	mi := MeasurementItem{
		MeasurementName:  LastColumnName,
		MeasurementAlias: LastColumnName,
		MeasurementType:  "string",
		MeasurementUnits: "unitless",
		ColumnOrder:      ndx1,
		Ignore:           false,
	}
	dimIndex, _ := dimMap[LastColumnName]
	mv := MeasurementVariable{
		MeasurementItem: mi,
		DimensionIndex:  dimIndex,
		FillValue:       "0. ", // or ""
		Comment:         "",
		Calendar:        "",
	}
	cdf.Measurements[LastColumnName] = &mv
}

// Change tiny values to 0, not null.
func (cdf *NetCDF) NormalizeValues() {
	const minimum = 10e-10
	changed := 0
	for ndx1 := 0; ndx1 < len(cdf.Dataset[0]); ndx1++ { // number of columns
		for ndx2 := 1; ndx2 < len(cdf.Dataset); ndx2++ { // number of rows
			fval, err := strconv.ParseFloat(cdf.Dataset[ndx2][ndx1], 64)
			if err == nil && fval < minimum {
				cdf.Dataset[ndx2][ndx1] = "0"
				changed++
			}
		}
	}
	if changed > 0 {
		fmt.Print(changed)
		fmt.Print(" tiny values changed to 0 [< ")
		fmt.Print(minimum)
		fmt.Println("]")
	}
}

// If there are no values at all in the block, IoTDB does not write the measurement, so get mismatch between number of INSERT field names and VALUES (e.g., HeatingEquipmentStage3_RunTime)
// Ids are consecutive: data rows = 8838720; 990 distinct id values; ==> 8928 rows per id-device
func (cdf *NetCDF) CopyCsvTimeseriesDataIntoIotDB() error {
	fileToRead := cdf.DataFilePath + "/csv/" + cdf.DatasetName + csvExtension
	err := cdf.ReadCsvFile(fileToRead, true) // isDataset: yes
	cdf.NormalizeValues()
	checkErr("cdf.ReadCsvFile ", err)
	nBlocks := cdf.Dimensions["id"]     //=990	 len(cdf.Dataset)/(blockSize) + 1
	blockSize := cdf.Dimensions["time"] // scope override
	timeIndex := 1                      // to calculate
	fmt.Printf("%s%d%s", "Writing ", nBlocks, " blocks: ")
	for block := 0; block < nBlocks; block++ {
		fmt.Print(block + 1)
		fmt.Print(" ")
		var sb strings.Builder
		var insert strings.Builder
		insert.WriteString("INSERT INTO " + IotDatasetPrefix(cdf.Identifier, cdf.HouseIndices[block]) + " (time," + cdf.FormattedColumnNames() + ") ALIGNED VALUES ")
		startRow := blockSize*block + 1
		endRow := startRow + blockSize
		if block == nBlocks-1 {
			endRow = len(cdf.Dataset)
		}
		for r := startRow; r < endRow; r++ {
			sb.Reset()
			startTime, err := filesystem.GetStartTimeFromString(cdf.Dataset[r][timeIndex])
			if err != nil {
				fmt.Println("Appears to be a bad time: " + cdf.Dataset[r][timeIndex])
				break
			}
			sb.WriteString("(" + strconv.FormatInt(startTime.UTC().Unix(), 10) + ",")
			for c := 0; c < len(cdf.Dataset[r]); c++ {
				for _, item := range cdf.Measurements {
					if item.ColumnOrder == c && !item.Ignore {
						sb.WriteString(formatDataItem(cdf.Dataset[r][c], item.MeasurementItem.MeasurementType) + ",")
					}
				}
			}
			sb.WriteString(formatDataItem(cdf.DatasetName, "string") + ")")
			if r < endRow-1 {
				sb.WriteString(",")
			}
			insert.WriteString(sb.String())
		}
		_, err := cdf.IoTDbAccess.session.ExecuteNonQueryStatement(insert.String() + ";") // (r *common.TSStatus, err error)
		checkErr("ExecuteNonQueryStatement(insertStatement)", err)
	}
	fmt.Println()
	return err
}

// Assume time series have been created; erase existing data; insert data. Assigns cdf.Dataset. INCOMPLETE! Converted *.nc files to CSV files. See CopyCSVTimeseriesDataIntoIotDB().
// mapNetcdfGolangTypes: "byte": "int8", "ubyte": "uint8", "char": "string", "short": "int16", "ushort": "uint16", "int": "int32", "uint": "uint32", "int64": "int64", "uint64": "uint64", "float": "float32", "double": "float64"
func (cdf *NetCDF) CopyNcTimeseriesDataIntoIotDB() error {
	iotPrefix := IotDatasetPrefix(cdf.Identifier, "{device-id}")
	const createMsg string = " time series not found -- run this program with the `create` parameter first " // also get this if no data in column
	fileToRead := cdf.DataFilePath + "/" + cdf.DatasetName + ncExtension
	nc, err := netcdf.OpenFile(fileToRead, netcdf.NOWRITE)
	if err != nil {
		checkErr("Could not access "+fileToRead, err)
	}
	defer nc.Close()

	// Read every NetCDF variable to construct an aligned time series.  For Jan_clean: id = 990; time = 8928 => datasetSize := 8838720
	for ndx := 0; ndx < len(cdf.Measurements); ndx++ {
		for _, item := range cdf.Measurements { // LastColumnName values do not exist in any *.nc data file.
			if item.ColumnOrder == ndx && !item.Ignore {
				vr, err := nc.Var(item.MeasurementAlias) // nc names
				if err != nil {
					fmt.Println(err)
				}
				//checkErr("["+iotPrefix+"]"+item.MeasurementName+createMsg, err)
				// Get the length of the dimensions of this data column.
				dims, err := vr.LenDims()
				if err != nil {
					fmt.Println(err)
				}
				// Read the entire variable v into data, which must have enough space for all the values (i.e. len(data) must be at least v.Len()).
				// CHAR is a scalar in NetCDF and Go has no scalar character type. Scalar characters in NetCDF will be returned as strings of length one.
				datasetSize := dims[0] * dims[1] // outer; inner
				switch strings.ToLower(item.MeasurementItem.MeasurementType) {
				case "decimal", "double":
					data := make([]float64, datasetSize)
					err := vr.ReadFloat64s(data)
					checkErr(iotPrefix+item.MeasurementAlias+createMsg, err)
				case "unicode", "string", "datetime":
					fmt.Println(vr.Name()) //<<<
					//data := make([]string, datasetSize)
					//err := vr.ReadBytes(data)
					//checkErr(iotPrefix+item.MeasurementAlias+createMsg, err)
					//fmt.Println(vr.Dims())  // [{65536 0} {65536 1}]
					//fmt.Println(vr.Len())	// 8838720
					//fmt.Println()
				case "integer", "int", "int32":
					data := make([]int32, datasetSize)
					err := vr.ReadInt32s(data)
					checkErr(iotPrefix+item.MeasurementAlias+createMsg, err)
				case "longint", "int64":
					data := make([]int64, datasetSize)
					err := vr.ReadInt64s(data)
					checkErr(iotPrefix+item.MeasurementAlias+createMsg, err)
				case "float":
					data := make([]float32, datasetSize)
					err := vr.ReadFloat32s(data)
					checkErr(iotPrefix+item.MeasurementAlias+createMsg, err)
				case "boolean":
					data := make([]byte, datasetSize) // int8{}
					err := vr.ReadBytes(data)         // ReadInt8s(data)
					checkErr(iotPrefix+item.MeasurementAlias+createMsg, err)
				}
			}
		}
	}
	return nil
}

// Similar to the iot version but not the same.
func (cdf *NetCDF) ProcessTimeseries() error {
	if cdf.IoTDbAccess.ActiveSession {
		cdf.IoTDbAccess.session = client.NewSession(clientConfig)
		if err := cdf.IoTDbAccess.session.Open(false, 0); err != nil {
			checkErr("ProcessTimeseries(cdf.IoTDbAccess.session.Open): ", err)
		}
		defer cdf.IoTDbAccess.session.Close()
	}
	fmt.Println("Processing time series for NC dataset " + cdf.DatasetName + " ...")

	for _, command := range cdf.TimeseriesCommands {
		switch command {
		case "createdb":
			sql := "CREATE DATABASE " + cdf.Identifier
			_, err := cdf.IoTDbAccess.session.ExecuteNonQueryStatement(sql)
			checkErr("ExecuteNonQueryStatement(createDBstatement)", err)
			fmt.Println(sql)

		case "dropts": // time series schema; uses single statement;
			for id := 0; id < len(cdf.HouseIndices); id++ {
				sql := "DROP TIMESERIES " + IotDatasetPrefix(cdf.Identifier, cdf.HouseIndices[id]) + "*"
				_, err := cdf.IoTDbAccess.session.ExecuteNonQueryStatement(sql)
				checkErr("ExecuteNonQueryStatement(dropStatement)", err)
			}
			for k := range cdf.Measurements { // RETEST!
				delete(cdf.Measurements, k)
			}

		case "createts":
			// Setting an alias, tag, and attribute for an aligned timeseries is supported as of Nov. 3, 2023 (v1.2.2).
			// CREATE timeseries root.turbine.d1.s1(temprature) WITH datatype = FLOAT, encoding = RLE, compression = SNAPPY, 'max_point_number' = '5' TAGS('tag1' = 'v1', 'tag2'= 'v2') ATTRIBUTES('attr1' = 'v1', 'attr2' = 'v2')
			// Note: For a group of aligned timeseries, Iotdb does not support different compressions.
			// https://iotdb.apache.org/UserGuide/V1.0.x/Reference/SQL-Reference.html#schema-statement
			var sb strings.Builder
			var sql string
			// Use each id as a 'device'; read from var file; are unique.
			for id := 0; id < len(cdf.HouseIndices); id++ {
				sb.Reset()
				sb.WriteString("CREATE ALIGNED TIMESERIES " + IotDatasetPrefix(cdf.Identifier, cdf.HouseIndices[id]) + "(")
				for ndx := 0; ndx < len(cdf.Measurements); ndx++ {
					for _, v := range cdf.Measurements {
						if v.ColumnOrder == ndx && !v.Ignore {
							dataType, encoding, compressor := getClientStorage(v.MeasurementItem.MeasurementType)
							// Note: It is not currently supported to set an alias, tag, and attribute for aligned timeseries.
							//<<<attributes := " ATTRIBUTES('datatype'='" + v.MeasurementType  + "') TAGS('units'='" + v.MeasurementUnits + "')"
							sb.WriteString(v.MeasurementAlias + " " + dataType + " encoding=" + encoding + " compressor=" + compressor + ",") // attributes +
						}
					}
				}
				sql = sb.String()[0:len(sb.String())-1] + ");" // replace trailing comma
				_, err := cdf.IoTDbAccess.session.ExecuteNonQueryStatement(sql)
				checkErr("ExecuteNonQueryStatement(createStatement)", err)
			}
			fmt.Println("IOTDB TEST QUERY: show timeseries " + cdf.Identifier + ".**;")

		case "delete": // remove all data; retain schema; multiple commands.
			for id := 0; id < len(cdf.HouseIndices); id++ {
				deleteStatements := make([]string, 0)
				for _, item := range cdf.Measurements {
					deleteStatements = append(deleteStatements, "DELETE FROM "+IotDatasetPrefix(cdf.DatasetName, cdf.HouseIndices[id])+item.MeasurementName+";")
				}
				_, err := cdf.IoTDbAccess.session.ExecuteBatchStatement(deleteStatements) // (r *common.TSStatus, err error)
				checkErr("ExecuteBatchStatement(deleteStatements)", err)
			}

		case "insert": // insert(append) data; retain schema; either single or multiple statements;
			// Automatically inserts long time column as first column (which should be UTC). Save in blocks.
			err := cdf.CopyCsvTimeseriesDataIntoIotDB()
			checkErr("ExecuteNonQueryStatement(insertStatements)", err)
			fmt.Println("IOTDB TEST QUERY: SELECT COUNT(*) FROM " + cdf.Identifier + ".*;")
		}
		fmt.Println("Timeseries <" + command + "> completed.")
	} // for

	return nil
}

// Return NetCDF struct by parsing Jan_clean.var file that is output from /usr/bin/ncdump -c Jan_clean.nc. Var files are specific to *.nc datasets.
func ParseVariableFile(varFile, filetype, dataSetIdentifier, description string, programArgs []string, isActive bool) (NetCDF, error) {
	lines, err := filesystem.ReadTextLines(varFile, false)
	checkErr("Could not access "+varFile, err)
	datasetPathName := filepath.Dir(programArgs[1]) // does not include trailing slash
	ioTDbAccess := IoTDbAccess{ActiveSession: isActive}
	xcdf := NetCDF{IoTDbAccess: ioTDbAccess, Description: description, DataFilePath: datasetPathName, DatasetName: dataSetIdentifier, NetcdfType: filetype}
	xcdf.TimeseriesCommands = GetTimeseriesCommands(programArgs)
	lineIndex := 0
	tokens := strings.Split(lines[lineIndex], " ")
	xcdf.Identifier = tokens[1] // override
	if len(dataSetIdentifier) > 0 {
		xcdf.Identifier = dataSetIdentifier
	}
	xcdf.Dimensions = make(map[string]int, 0)
	xcdf.Measurements = make(map[string]*MeasurementVariable, 0)
	lineIndex++

	if strings.Contains(lines[lineIndex], "dimensions:") {
		variables := strings.Contains(lines[lineIndex], "variables:")
		for !variables {
			lineIndex++
			tokens := strings.Split(strings.TrimSpace(lines[lineIndex]), " ")
			size, e := strconv.Atoi(tokens[2])
			if e == nil {
				xcdf.Dimensions[tokens[0]] = size
			} else {
				fmt.Println("Error converting Dimension " + tokens[0])
			}
			variables = strings.Contains(lines[lineIndex+1], "variables:")
		}
		lineIndex++
	}

	dimMap := xcdf.getDimensionMap()
	if strings.Contains(lines[lineIndex], "variables:") {
		columnIndex := 0
		offset := 1
		data := strings.Contains(lines[lineIndex], "data:")
		for !data {
			lineIndex++
			if len(strings.TrimSpace(lines[lineIndex])) == 0 {
				break
			}
			tokens := strings.Split(strings.TrimSpace(lines[lineIndex]), " ")
			standardname := strings.TrimSpace(strings.Split(tokens[1], "(")[0])
			standardname, aliasname := StandardName(standardname)
			if aliasname == "=" {
				continue
			}
			tmpVar := MeasurementVariable{}
			tmpVar.MeasurementItem.MeasurementName = standardname
			tmpVar.MeasurementItem.MeasurementAlias = aliasname
			tmpVar.MeasurementItem.MeasurementType = tokens[0]
			tmpVar.MeasurementItem.ColumnOrder = columnIndex
			columnIndex++
			val, ok := dimMap[tmpVar.MeasurementItem.MeasurementName]
			if ok {
				tmpVar.DimensionIndex = val
			}
			thisVariable := strings.Contains(lines[lineIndex], tmpVar.MeasurementItem.MeasurementName)
			for thisVariable {
				lineIndex++
				tokens = strings.Split(strings.TrimSpace(lines[lineIndex]), "=")
				// The :units input is ignored because it is always "unitless". Taken from Ecobee_dataset_cleaning_report.docx.
				if strings.Contains(lines[lineIndex], ":units") {
					tmpVar.MeasurementItem.MeasurementUnits = prettifyString(tokens[offset]) // usually unitless
					if strings.Index(tmpVar.MeasurementItem.MeasurementUnits, " ") > 0 {
						tmpVar.MeasurementItem.MeasurementUnits = "unixutc"
					}
					if strings.Contains(standardname, "Temperature") || strings.Contains(standardname, "Setpoint") {
						tmpVar.MeasurementItem.MeasurementUnits = "°F"
					} else {
						if strings.Contains(standardname, "RunTime") {
							tmpVar.MeasurementItem.MeasurementUnits = "seconds"
						} else {
							if strings.Contains(standardname, "Humidity") {
								tmpVar.MeasurementItem.MeasurementUnits = "%rh"
							} else {
								if strings.Contains(standardname, "DetectedMotion") {
									tmpVar.MeasurementItem.MeasurementUnits = "boolean" // 0/1
								} else {
									if strings.Contains(standardname, "Mode") {
										tmpVar.MeasurementItem.MeasurementUnits = "unitless" // 0/1
									}
								}
							}
						}
					}
				}
				// skip ":standard_name"
				if strings.Contains(lines[lineIndex], ":long_name") {
					tmpVar.MeasurementItem.MeasurementAlias = prettifyString(tokens[offset])
				}
				if strings.Contains(lines[lineIndex], ":_FillValue") {
					tmpVar.FillValue = prettifyString(tokens[offset])
				}
				if strings.Contains(lines[lineIndex], ":comment") {
					tmpVar.Comment = prettifyString(tokens[offset])
				}
				if strings.Contains(lines[lineIndex], ":calendar") {
					tmpVar.Calendar = prettifyString(tokens[offset])
				}
				thisVariable = strings.Contains(lines[lineIndex+1], tmpVar.MeasurementItem.MeasurementName+":")
			}
			if len(tmpVar.MeasurementItem.MeasurementUnits) == 0 {
				tmpVar.MeasurementItem.MeasurementUnits = "unitless"
			}
			xcdf.Measurements[tmpVar.MeasurementItem.MeasurementName] = &tmpVar
		}
	}

	lineIndex = lineIndex + 2
	offset := 1
	for lineIndex < len(lines) {
		if strings.Contains(lines[lineIndex], "data:") {
			break
		}
		tokens := strings.Split(lines[lineIndex], "\"")
		if strings.Contains(lines[lineIndex], ":title") {
			xcdf.Title = tokens[offset]
		}
		if strings.Contains(lines[lineIndex], ":description") {
			xcdf.Description = tokens[offset]
		}
		if strings.Contains(lines[lineIndex], ":conventions") {
			xcdf.Conventions = tokens[offset]
		}
		if strings.Contains(lines[lineIndex], ":institution") {
			xcdf.Institution = tokens[offset]
		}
		if strings.Contains(lines[lineIndex], ":code_url") {
			xcdf.Code_url = tokens[offset]
		}
		if strings.Contains(lines[lineIndex], ":location_meaning") {
			xcdf.Location_meaning = tokens[offset]
		}
		if strings.Contains(lines[lineIndex], ":datastream_name") {
			xcdf.Datastream_name = tokens[offset]
		}
		if strings.Contains(lines[lineIndex], ":input_files") {
			xcdf.Input_files = tokens[offset]
		}
		if strings.Contains(lines[lineIndex], ":history") {
			xcdf.History = tokens[offset]
		}
		lineIndex++
	}

	lineIndex = lineIndex + 2
	xcdf.HouseIndices, lineIndex = parseDimensionIndices(xcdf.Dimensions["id"], lineIndex, lines)
	xcdf.LongtimeIndices, _ = parseDimensionIndices(xcdf.Dimensions["time"], lineIndex, lines)
	return xcdf, nil
}

var EntityCommentMap = map[string]string{
	"ic-data:DataPoint": "A data point is a quantity that is extended with various pieces of process information, namely \n" +
		" - A creation time (instant). This is the point in time when the data point was created, which is not necessarily the time for which it is valid. In the case of soft-sensors or forecasters, a data point might have been created ahead of time, in the case of a direcet measurement a data point might created at its time of validity (or at the end of its validity time interval) and in the case of an archived value the data point might have been created after the fact.\n" +
		" - A validity time (temporal entity) which will be named 'time stamp'. The validity time is the instant or interval in time in which a specific quantity is in effect. For example a room temperature might be measured at 12:00, which means it is in effect at this very instant. A specific amount of energy might me expended within the time-slot between 12:30 and 12:45, which means that the energy measurement is in effect during this time interval.\n" +
		" - A location or topological association. For example, a measurement might be taken in a specific room, a power avarage might have been measured by a specific meter, a forecast might be valid for a specific region or grid segment. This association is therefore not always a location.",
	"ic-data:TimeSeries":            "An ordered sequence of data points of a quantity observed at spaced time intervals is referred to as a time series. Time series can be a result of prediction algorithm.",
	"ic-data:Usage":                 "The usage of a datapoint, time series or message.",
	"ic-data:hasEffectivePeriod":    "This connects to the temporal entity which describes when (time interval) the quantity of this data point was, is, or will be in effect. This is the time interval which is covered by the forecast. This should be equivalent to the time interval covered by the time-series that express the forecast.",
	"ic-data:hasUsage":              "This property provides the possibility to add some additional information about the usage of a data-point or time-series. For example, a data point or time series can be used as an upper limit, lower limit or a baseline, a maximum versus minimum value, or a consumption versus a production value.",
	"ic-data:hasDataPoint":          "This relationship connects a time series to data point.",
	"ic-data:hasCreationTime":       "The time instant that defines the creation time of a data point or quantity or forecast or similar entities. This is not the same as the time at which the quantity is in effect. For example, if a temperature is forecasted today at 12:30 (creation time of the forecast) for the following day at 14:45 (time when the temperature is expected to be in effect), the this instant should be 12:30 of today. A creation time (instant). This is the point in time when the data point was created, which is not necessarily the time for which it is valid. In the case of soft-sensors or forecasters, a data point might have been created ahead of time, in the case of a direcet measurement a data point might created at its time of validity (or at the end of its validity time interval) and in the case of an archived value the data point might have been created after the fact.",
	"ic-data:hasTemporalResolution": "The resolution is the distance between two measurement time-stapms. This only makes sense if the measurements are equidistant.",
	"ic-data:hasUpdateRate":         "The rate at which a data point or time-series or forecast or other data entity is being updated.",
	"saref:Measurement":             "???",
	"saref:UnitOfMeasure":           "???",
	"s4envi:FrequencyUnit":          "???",
	"s4envi:FrequencyMeasurement":   "???",
	"s4auto/Confidence":             "???",
}

///////////////////////////////////////////////////////////////////////////////////////////
/* ###  http://ontology.tno.nl/interconnect/datapoint#TimeSeries
ic-data:TimeSeries rdf:type owl:Class ;
                   rdfs:subClassOf [ rdf:type owl:Restriction ;
                                     owl:onProperty ic-data:hasEffectivePeriod ;
                                     owl:allValuesFrom time:Interval
                                   ] ,
                                   [ rdf:type owl:Restriction ;
                                     owl:onProperty ic-data:hasUsage ;
                                     owl:allValuesFrom ic-data:Usage
                                   ] ,
                                   [ rdf:type owl:Restriction ;
                                     owl:onProperty ic-data:hasDataPoint ;
                                     owl:minQualifiedCardinality "0"^^xsd:nonNegativeInteger ;
                                     owl:onClass ic-data:DataPoint
                                   ] ,
                                   [ rdf:type owl:Restriction ;
                                     owl:onProperty ic-data:hasCreationTime ;
                                     owl:maxQualifiedCardinality "1"^^xsd:nonNegativeInteger ;
                                     owl:onClass time:Instant
                                   ] ,
                                   [ rdf:type owl:Restriction ;
                                     owl:onProperty ic-data:hasTemporalResolution ;
                                     owl:maxQualifiedCardinality "1"^^xsd:nonNegativeInteger ;
                                     owl:onClass time:TemporalDuration
                                   ] ,
                                   [ rdf:type owl:Restriction ;
                                     owl:onProperty ic-data:hasUpdateRate ;
                                     owl:maxQualifiedCardinality "1"^^xsd:nonNegativeInteger ;
                                     owl:onClass time:TemporalDuration
                                   ] ;
-------------------------------------------------------------------
ObjectProperty::
ic-data:hasEffectivePeriod rdf:type owl:ObjectProperty ;
    rdfs:range time:TemporalEntity ;
    rdfs:comment """This connects to the temporal entity which describes when (time interval) the quantity of this data point was, is, or will be in effect. This is the time interval which is covered by the forecast.
This should be equivalent to the time interval covered by the time-series that express the forecast. *A potential application of SHACL?*""" ;

ic-data:hasUsage rdf:type owl:ObjectProperty ;
    rdfs:range ic-data:Usage ;
    rdfs:comment "This property provides the possibility to add some additional information about the usage of a data-point or time-series. For example, a data point or time series can be used as an upper limit, lower limit or a baseline, a maximum versus minimum value, or a consumption versus a production value."@en ;

ic-data:hasDataPoint rdf:type owl:ObjectProperty ;
    rdfs:domain ic-data:TimeSeries ;
    rdfs:range ic-data:DataPoint ;
    rdfs::comment "This relationship connects a time series to data point."@en ;

ic-data:hasCreationTime rdf:type owl:ObjectProperty ;
    rdfs:range time:Instant ;
    rdfs:comment """The time instant that defines the creation time of a data point or quantity or forecast or similar entities. This is not the same as the time at which the quantity is in effect. For example, if a temperature is forecasted today at 12:30 (creation time of the forecast) for the following day at 14:45 (time when the temperature is expected to be in effect), the this instant should be 12:30 of today.
A creation time (instant). This is the point in time when the data point was created, which is not necessarily the time for which it is valid. In the case of soft-sensors or forecasters, a data point might have been created ahead of time, in the case of a direcet measurement a data point might created at its time of validity (or at the end of its validity time interval) and in the case of an archived value the data point might have been created after the fact."""@en ;

ic-data:hasTemporalResolution rdf:type owl:ObjectProperty ;
	rdfs:range time:TemporalDuration ;
    rdfs:comment "The resolution is the distance between two measurement time-stapms. This only makes sense if the measurements are equidistant." ;

###  http://ontology.tno.nl/interconnect/datapoint#hasUpdateRate
ic-data:hasUpdateRate rdf:type owl:ObjectProperty ;
	rdfs:range time:TemporalDuration ;
    rdfs:comment """The rate at which a data point or time-series or forecast or other data entity is being updated.

Classes::
ic-data:Usage rdf:type owl:Class ;
	rdfs:comment "The usage of a datapoint, time series or message."@en ;

ic-data:DataPoint rdf:type owl:Class ;
	rdfs:subClassOf saref:Measurement ,
    rdfs:comment """A data point is a quantity that is extended with various pieces of process information, namely
 	  - A creation time (instant). This is the point in time when the data point was created, which is not necessarily the time for which it is valid. In the case of soft-sensors or forecasters, a data point might have been created ahead of time, in the case of a direcet measurement a data point might created at its time of validity (or at the end of its validity time interval) and in the case of an archived value the data point might have been created after the fact.
	  - A validity time (temporal entity) which will be named \"time stamp\". The validity time is the instant or interval in time in which a specific quantity is in effect. For example a room temperature might be measured at 12:00, which means it is in effect at this very instant. A specific amount of energy might me expended within the time-slot between 12:30 and 12:45, which means that the energy measurement is in effect during this time interval.
	  - A location or topological association. For example, a measurement might be taken in a specific room, a power avarage might have been measured by a specific meter, a forecast might be valid for a specific region or grid segment. This association is therefore not always a location."""@en ;
*/

/* https://www.w3.org/TR/vocab-dcat-3/#basic-example
ex:dataset-001
  a dcat:Dataset ;
	...
  dcat:temporalResolution "P1D"^^xsd:duration ;
  dcat:distribution ex:dataset-001-csv ;
  .
*/

func prettifyString(str string) string {
	s := strings.TrimSpace(strings.ReplaceAll(str, "\"", ""))
	if strings.HasSuffix(s, ";") {
		s = s[0 : len(s)-1]
	}
	return strings.TrimSpace(s)
}

// Return extracted data, new lineIndex.
func parseDimensionIndices(nDimensions, lineIndex int, lines []string) ([]string, int) {
	dimIndex := 0
	output := make([]string, nDimensions)
	for lineIndex < len(lines) {
		if len(strings.TrimSpace(lines[lineIndex])) == 0 {
			break
		}
		tokens := strings.Split(strings.TrimSpace(lines[lineIndex]), ",")
		if len(tokens) == 1 { // last element terminated by semi-colon; no comma.
			tokens = strings.Split(strings.TrimSpace(lines[lineIndex]), ";")
		}
		for ndx := 0; ndx < len(tokens)-1; ndx++ {
			output[dimIndex] = prettifyString(tokens[ndx])
			if strings.Contains(output[dimIndex], "=") { // remove variable names
				tok2 := strings.Split(output[dimIndex], "=")
				output[dimIndex] = strings.TrimSpace(tok2[1])
			}
			dimIndex++
		}
		lineIndex++
	}
	return output, lineIndex

}

func ShowLastExternalBackup() string {
	fp := HomeDirectory + "lastExternalBackup.txt"
	exists, _ := filesystem.FileExists(fp)
	if !exists {
		return "The database has never been backed up to an external drive."
	}
	lines, _ := filesystem.ReadTextLines(fp, false)
	msg := strings.Replace(lines[0], "               ", "Last backup at ", 1)
	return msg
}

func GetOutputPath(filePath, ext string) string {
	return filePath[:len(filePath)-len(filepath.Ext(filePath))] + ext
}

func CreateIotSession(programArgs []string) bool {
	createIotSession := !(len(programArgs) == 3 && programArgs[2] == timeSeriesCommands[0])
	fmt.Println("createIotSession")
	if createIotSession {
		iotdbConnection, ok := Init_IoTDB(createIotSession)
		if !ok {
			checkErr("Init_IoTDB: ", errors.New(iotdbConnection))
		}
	}
	return createIotSession
}

// Produce *.var file using: /usr/bin/ncdump -k cdf.nc  &&  /usr/bin/ncdump -c Jan_clean.nc
func ProcessCsvSensorData(programArgs []string) {
	createIotSession := CreateIotSession(programArgs)
	iotdbDataFile, err := Initialize_IoTDbCsvDataFile(createIotSession, programArgs)
	checkErr("Initialize_IoTDbCsvDataFile: ", err)
	err = iotdbDataFile.ProcessTimeseries()
	checkErr("ProcessTimeseries(csv)", err)
}

func isAccessibleSensorDataFile(dataFilePath string) error {
	exists, err := filesystem.FileExists(dataFilePath)
	if !exists {
		return err
	}
	checkErr("Sensor data file not readable: "+dataFilePath, err)
	var goodFileTypes = map[string]string{".nc": "ok", ".csv": "ok", ".hd5": "ok"}
	dataFileType := strings.ToLower(path.Ext(dataFilePath))
	_, ok := goodFileTypes[dataFileType]
	if !ok {
		return errors.New("Cannot process source file type: " + dataFilePath)
	}
	return nil
}

func Initialize_IoTDbNcDataFile(isActive bool, programArgs []string) (NetCDF, error) {
	fileType := programArgs[2]                      // subtype of *.nc file.
	outputPath := GetOutputPath(programArgs[1], "") // path has no extension
	datasetName := path.Base(outputPath)            // Jan_clean
	isAccessibleSensorDataFile(programArgs[1])
	xcdf, err := ParseVariableFile(outputPath+varExtension, fileType, datasetName, datasetName, programArgs, isActive)
	checkErr("ParseVariableFile", err)
	//fmt.Println(xcdf.ToString(true)) // true => output variables
	err = xcdf.ReadCsvFile(GetSummaryFilename(programArgs[1]), false) // isDataset: no, is summary  REFACTOR: read from GraphDB?
	checkErr("ReadCsvFile ", err)
	// where do I read the nc data file?
	xcdf.TimeMeasurementName = programArgs[3]
	xcdf.XsvSummaryTypeMap()
	return xcdf, nil
}

// First reads the *.var file for meta-information and then the *.nc file (in same folder).
func ProcessNcSensorData(programArgs []string) {
	createSession := CreateIotSession(programArgs)
	xcdf, err := Initialize_IoTDbNcDataFile(createSession, programArgs)
	checkErr("Initialize_IoTDbNcDataFile: ", err)
	err = xcdf.ProcessTimeseries()
	checkErr("ProcessTimeseries(nc)", err)
}

// Return a NamedIndividual unit_of_measure from an abreviated key. All of these are specified at the uomPrefix URI.
// This expects you to manually add the map keys to the (last) Units column header in every *.var file.
// REFACTOR to get from website?
func GetNamedIndividualUnitMeasure(uom string) string {
	const uomPrefix = "http://www.ontology-of-units-of-measure.org/resource/om-2/"
	const varPrefix = "https://energyknowledgebase.com/topics/volt-ampere-reactive-var.asp"
	var unitsOfMeasure = map[string]string{
		"kW":       uomPrefix + "kilowatt",
		"kWh":      uomPrefix + "kilowattHour",
		"pascal":   uomPrefix + "pascal",
		"kelvin":   uomPrefix + "kelvin",
		"°C":       uomPrefix + "degreeCelsius",
		"°F":       uomPrefix + "degreeFahrenheit",
		"%rh":      uomPrefix + "PercentRelativeHumidity",
		"mb":       uomPrefix + "millibar",
		"degree":   uomPrefix + "degree",
		"lux":      uomPrefix + "lux",
		"km":       uomPrefix + "kilometre",
		"dV":       uomPrefix + "decivolt",
		"dA":       uomPrefix + "deciampere",
		"Hz":       uomPrefix + "hertz",
		"DPF":      "https://ctlsys.com/support/power_factor/",
		"APF":      "https://ctlsys.com/support/power_factor/",
		"VAR":      varPrefix,
		"VAR.hour": varPrefix,
		"VA":       "https://en.wikipedia.org/wiki/Volt-ampere",
		"VA.hour":  "https://en.wikipedia.org/wiki/Volt-ampere",
	}

	switch uom {
	case "km":
		return "### " + unitsOfMeasure[uom] +
			`uom:kilometre rdf:type owl:NamedIndividual ,
		saref:UnitOfMeasure ;
		rdfs:comment "1000 metres."@en ;
		rdfs:label "kilometre"@en .`

	case "degree":
		return "### " + unitsOfMeasure[uom] +
			`uom:degree rdf:type owl:NamedIndividual ,
		saref:UnitOfMeasure ;
		rdfs:comment "One unit of 360 degrees."@en ;
		rdfs:label "degree"@en .`

	case "lux":
		return "### " + unitsOfMeasure[uom] +
			`uom:lux rdf:type owl:NamedIndividual ,
		saref:IlluminanceUnit ;
		rdfs:comment "The lux is a unit of illuminance defined as lumen divided by square metre = candela times steradian divided by square metre."@en ;
		rdfs:label "lux"@en .`

	case "mb":
		return "### " + unitsOfMeasure[uom] +
			`uom:millibar rdf:type owl:NamedIndividual ,
		saref:PressureUnit ;
   		rdfs:comment "The millibar is a unit of pressure defined as 100 pascal."@en ;
   		rdfs:label "millibar"@en .`

	case "%rh":
		return "### " + unitsOfMeasure[uom] +
			`uom:RelativeHumidity rdf:type owl:NamedIndividual ,
		saref:Humidity ;
		rdfs:comment "A saref:Property related to some measurements that are characterized by a certain value that is measured in percent relative humidity."@en ;
		rdfs:label "percent relative humidity"@en .`

	case "°F":
		return "### " + unitsOfMeasure[uom] +
			`uom:degreeFahrenheit rdf:type owl:NamedIndividual ,
		saref:TemperatureUnit ;
		rdfs:comment "The degree Fahrenheit is a unit of temperature defined as 5.555556e-1 kelvin."@en ;
		rdfs:label "degrees Fahrenheit"@en .`

	case "°C":
		return "### " + unitsOfMeasure[uom] +
			`uom:degreeCelsius rdf:type owl:NamedIndividual ,
		saref:TemperatureUnit ;
		rdfs:comment "The degree Celsius is a unit of temperature defined as 1 kelvin."@en ;
		rdfs:label "degrees Celsius"@en .`

	case "kelvin":
		return "### " + unitsOfMeasure[uom] +
			`uom:kelvin rdf:type owl:NamedIndividual ,
		saref:TemperatureUnit ;
		rdfs:comment "The kelvin is a unit of temperature defined as 1/273.16 of the thermodynamic temperature of the triple point of water."@en ;
		rdfs:label "degrees kelvin"@en .`

	case "pascal":
		return "### " + unitsOfMeasure[uom] +
			`uom:pascal rdf:type owl:NamedIndividual ,
		saref:PressureUnit ;
		rdfs:comment "The pascal is a unit of pressure and stress defined as newton divided by square metre = joule divided by cubic metre = kilogram divided by metre second squared."@en ;
		rdfs:label "pascal"@en .`

	case "kW":
		return "### " + unitsOfMeasure[uom] +
			`uom:kilowatt rdf:type owl:NamedIndividual ,
		saref:PowerUnit ;
		rdfs:comment "The watt is a unit of power defined as joule divided by second = newton times metre divided by second = volt times ampere = kilogram times square metre divided by second to the power 3."@en ;
		rdfs:label "kilowatt"@en .`

	case "kWh":
		return "### " + unitsOfMeasure[uom] +
			`uom:kilowattHour rdf:type owl:NamedIndividual ,
		saref:EnergyUnit ;
		rdfs:comment "The kiloWatt Hour is a unit of energy equivalent to 1000 watts of power expended during one hour of time. An energy expenditure of 1 Wh represents 3600 joules "@en ;
		rdfs:label "kilowatt hour"@en .`
	}

	return unknown
}

var timeSeriesCommands = []string{"createdb", "createts", "dropts", "delete", "insert"}
var iotdbParameters IoTDbProgramParameters
var clientConfig *client.Config

// select this column of values from summary file as instance values.
var summaryColumnNames = []string{"field", "type", "sum", "min", "max", "min_length", "max_length", "mean", "stddev", "median", "mode", "cardinality", "units"} // , LastColumnName

// Replace embedded quote marks with backticks; return enclosed string. Only for data insert statements!
// Could format floats to constant width,precision. This function is called millions of times.
// REFACTOR why are dateTime dataType formats not being passed in?
func formatDataItem(s, dataType string) string {
	dt := strings.TrimSpace(strings.ToLower(dataType))
	if dt == "string" || dt == "unicode" || dt == "datetime" || len(dt) == 0 { // hex40? boolean?
		ss := strings.ReplaceAll(strings.ReplaceAll(s, "\"", "`"), "'", "`")
		return "'" + ss + "'" 
	} else {
		if len(s) == 0 {
			return "null"
		} else {
			return s // float
		}
	}
}

// Return quoted name and its alias. Alias: open brackets are replaced with underscore.
// Does not handle 2 aliases being the same. Return MeasurementName, MeasurementAlias.
// Some variable names must be aliased in SQL statements: {time, timestamp}
func StandardName(oldName string) (string, string) {
	newName := strings.TrimSpace(strings.Title(strings.TrimSpace(oldName))) // uppercase first letter of each word
	replacer := strings.NewReplacer("~", "", "!", "", "@", "", "#", "", "$", "", "%", "", "^", "", "&", "", "*", "", "/", "", "?", "", ".", "", ",", "", ":", "", ";", "", "|", "", "\\", "", "=", "", "+", "", ")", "", "}", "", "]", "", "(", "_", "{", "_", "[", "_")
	alias := strings.ReplaceAll(newName, " ", "")
	alias = replacer.Replace(alias)

	if strings.ToLower(newName) == "time" || strings.ToLower(newName) == "timestamp" {
		return newName, timeAlias
	}

	if newName != alias {
		return alias, newName
	} else {
		return newName, alias
	}
}

// Return IotDB datatype; encoding; compression. See https://iotdb.apache.org/UserGuide/V1.0.x/Data-Concept/Encoding.html#encoding-methods
// (client.TSDataType, client.TSEncoding, client.TSCompressionType)  Make all compression SNAPPY
func getClientStorage(dataColumnType string) (string, string, string) {
	sw := strings.ToLower(dataColumnType)
	switch sw {
	case "decimal", "double":
		return "DOUBLE", "GORILLA", "SNAPPY"
	case "unicode", "string", "datetime":
		return "TEXT", "PLAIN", "SNAPPY"
	case "integer", "int", "int32":
		return "INT32", "GORILLA", "SNAPPY"
	case "longint", "int64":
		return "INT64", "GORILLA", "SNAPPY"
	case "float":
		return "FLOAT", "GORILLA", "SNAPPY"
	case "boolean":
		return "BOOLEAN", "RLE", "SNAPPY"
	}
	return "TEXT", "PLAIN", "SNAPPY"
}

type IoTDbProgramParameters struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}

// First read environment variables: IOTDB_PASSWORD, IOTDB_USER, IOTDB_HOST, IOTDB_PORT; then read override parameters from command-line.
// Assigns iotdbParameters and returns client.Config. iotdbParameters is a superset of client.Config.
func configureIotdbAccess() *client.Config {
	iotdbParameters = IoTDbProgramParameters{
		Host:     os.Getenv("IOTDB_HOST"),
		Port:     os.Getenv("IOTDB_PORT"),
		User:     os.Getenv("IOTDB_USER"),
		Password: os.Getenv("IOTDB_PASSWORD"),
	}
	envFound := len(iotdbParameters.Host) > 0 && len(iotdbParameters.Port) > 0
	if !envFound {
		flag.StringVar(&iotdbParameters.Host, "host", "127.0.0.1", "--host=10.103.4.83")//<<<
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

// Assigns clientConfig.
func Init_IoTDB(testIotdbAccess bool) (string, bool) {
	fmt.Println("Initializing IoTDB client...")
	clientConfig = configureIotdbAccess()
	isOpen, err := filesystem.TestRemoteAddressPortsOpen(clientConfig.Host, []string{clientConfig.Port})
	connectStr := clientConfig.Host + ":" + clientConfig.Port
	if testIotdbAccess && !isOpen {
		fmt.Printf("%s%v%s", "Expected IoTDB to be available at "+connectStr+" but got ERROR: ", err, "\n")
		fmt.Printf("Please execute:  cd ~/iotdb && sbin/start-standalone.sh && sbin/start-cli.sh -h 127.0.0.1 -p 6667 -u root -pw root")
		return connectStr, false
	}
	return connectStr, true
}

///////////////////////////////////////////////////////////////////////////////////////////

type IoTDbCsvDataFile struct {
	IoTDbAccess
	Identifier          string                      `json:"identifier"`
	Description         string                      `json:"description"`
	DataFilePath        string                      `json:"datafilepath"`
	DataFileType        string                      `json:"datafiletype"`
	DatasetName         string                      `json:"datasetname"`
	TimeMeasurementName string                      `json:"timemeasurementname"` // index into Measurements map;
	Measurements        map[string]*MeasurementItem `json:"measurements"`
	Summary             [][]string                  `json:"summary"` // from summary file
	Dataset             [][]string                  `json:"dataset"` // actual data
}

// expect only 1 instance of 'datasetName' in iotdbDataFile.DataFilePath.
func Initialize_IoTDbCsvDataFile(isActive bool, programArgs []string) (IoTDbCsvDataFile, error) {
	datasetName := filepath.Base(programArgs[1]) // includes extension
	datasetName = strings.Replace(datasetName, csvExtension, "", 1)
	isAccessibleSensorDataFile(programArgs[1])
	ioTDbAccess := IoTDbAccess{ActiveSession: isActive}
	timeMeasurementName := programArgs[2]
	iotdbDataFile := IoTDbCsvDataFile{IoTDbAccess: ioTDbAccess, Description: datasetName, DataFilePath: programArgs[1], DatasetName: datasetName, TimeMeasurementName: timeMeasurementName}
	iotdbDataFile.TimeseriesCommands = GetTimeseriesCommands(programArgs)
	err := iotdbDataFile.ReadCsvFile(GetSummaryFilename(iotdbDataFile.DataFilePath), false) // isDataset: no, is summary  REFACTOR: read from GraphDB?
	checkErr("ReadCsvFile ", err)
	iotdbDataFile.XsvSummaryTypeMap()
	_, readDataFile := find(programArgs, "insert")
	if readDataFile > 1 {
		err = iotdbDataFile.ReadCsvFile(iotdbDataFile.DataFilePath, true) // isDataset: yes
		checkErr("ReadCsvFile ", err)
		iotdbDataFile.NormalizeValues()
	}
	return iotdbDataFile, nil
}

// Change tiny values to 0, not null.
func (iot *IoTDbCsvDataFile) NormalizeValues() {
	const minimum = 10e-10
	changed := 0
	for ndx1 := 0; ndx1 < len(iot.Dataset[0]); ndx1++ { // number of columns
		for ndx2 := 1; ndx2 < len(iot.Dataset); ndx2++ { // number of rows
			fval, err := strconv.ParseFloat(iot.Dataset[ndx2][ndx1], 64)
			if err == nil && fval < minimum {
				iot.Dataset[ndx2][ndx1] = "0"
				changed++
			}
		}
	}
	if changed > 0 {
		fmt.Print(changed)
		fmt.Print(" tiny values changed to 0 [< ")
		fmt.Print(minimum)
		fmt.Println("]")
	}
}

// search for either original or alias name
func (iot *IoTDbCsvDataFile) GetMeasurementItemFromName(name string) (*MeasurementItem, bool) {
	item, ok := iot.Measurements[name]
	if ok {
		return item, true
	}
	for _, item := range iot.Measurements {
		if name == item.MeasurementName || name == item.MeasurementAlias {
			return item, true
		}
	}
	return item, false
}

// Return all column values except header row. Return false if column name not found. Fill in default values.
// columnName={summaryColumnNames}
func (iot *IoTDbCsvDataFile) GetSummaryStatValues(columnName string) ([]string, bool) {
	_, columnIndex := find(summaryColumnNames, columnName)
	if columnIndex < 0 {
		return []string{}, false
	}
	stats := make([]string, len(iot.Measurements))

	for ndx := 0; ndx < len(iot.Measurements); ndx++ {
		for _, item := range iot.Measurements {
			if item.ColumnOrder == ndx && !item.Ignore && ndx < len(iot.Measurements)-1 {
				_, aliasName := StandardName(iot.Summary[ndx+1][0])
				item, _ := iot.GetMeasurementItemFromName(aliasName)
				stats[ndx] = strings.TrimSpace(iot.Summary[ndx+1][columnIndex])
				if strings.HasPrefix(strings.ToLower(item.MeasurementType), "int") || strings.HasPrefix(strings.ToLower(item.MeasurementType), "long") {
					index := strings.Index(stats[ndx], ".")
					if index >= 0 {
						stats[ndx] = stats[ndx][0:index]
					}
				}
				if len(stats[ndx]) == 0 {
					stats[ndx] = strings.TrimSpace(iot.Summary[ndx+1][1]) // this type is nearly always Unicode
				}
			}
		}
	}
	return stats, true
}

// Return string-formatted value from: columnName={summaryColumnNames}, fieldName={measurement names}
func (iot *IoTDbCsvDataFile) GetSummaryValue(columnName, fieldName string) string {
	columnNames, found1 := iot.GetSummaryStatValues(summaryColumnNames[0])
	columnValues, found2 := iot.GetSummaryStatValues(columnName)
	_, index := find(columnNames, fieldName) // exact match
	if !found1 || !found2 || index < 0 {
		return ""
	}
	return columnValues[index]
}

// iterate over Summary header row
func (iot *IoTDbCsvDataFile) GetColumnNumberFromName(columnName string) int {
	for ndx := 0; ndx < len(iot.Summary[0]); ndx++ {
		if strings.ToLower(iot.Summary[0][ndx]) == columnName {
			return ndx
		}
	}
	return -1
}

// Expects {Units, DatasetName} fields to have been appended to the summary file. Assign []Measurements. Expects Summary to be assigned. Use XSD data types.
// len() only returns the length of the "external" array.
func (iot *IoTDbCsvDataFile) XsvSummaryTypeMap() {
	iot.Measurements = make(map[string]*MeasurementItem, 0)
	// get units column
	unitsColumn := iot.GetColumnNumberFromName(unitsName)
	ndx1 := 0
	for ndx := 0; ndx < maxColumns; ndx++ { // iterate over summary file rows.
		endOfMeasurements := iot.Summary[ndx+1][0] == interpolated || strings.TrimSpace(iot.Summary[ndx+1][0]) == endOfFields
		if endOfMeasurements {
			iot.Identifier = iot.Summary[ndx+1][1]
			ndx1 = ndx
			break
		}
		dataColumnName, aliasName := StandardName(iot.Summary[ndx+1][0]) // dataColumnName is normalized for IotDB naming conventions.
		ignore := iot.Summary[ndx+1][0] == "time" || (iot.Summary[ndx+1][2] == "0" && iot.Summary[ndx+1][3] == "0" && iot.Summary[ndx+1][4] == "0") || dataColumnName == interpolated
		if ignore {
			fmt.Println("Ignoring empty data column " + dataColumnName)
		}
		theEnd := dataColumnName == interpolated || strings.TrimSpace(iot.Summary[ndx+1][0]) == endOfFields
		if !theEnd {
			mi := MeasurementItem{
				MeasurementName:  aliasName, // the CSV field names are often unusable.
				MeasurementAlias: dataColumnName,
				MeasurementType:  rowsXsdMap[iot.Summary[ndx+1][1]],
				MeasurementUnits: iot.Summary[ndx+1][unitsColumn],
				ColumnOrder:      ndx,
				Ignore:           ignore,
			}
			iot.Measurements[dataColumnName] = &mi // add to map using original name
		}
	}
	// add DatasetName timerseries in case data column names are the same for different sampling intervals.
	mi := MeasurementItem{
		MeasurementName:  LastColumnName,
		MeasurementAlias: LastColumnName,
		MeasurementType:  "string",
		MeasurementUnits: "unitless",
		ColumnOrder:      ndx1,
		Ignore:           false,
	}
	iot.Measurements[LastColumnName] = &mi
}

// Return list of ordered dataset column names as string. Does not include enclosing ()
func (iot *IoTDbCsvDataFile) FormattedColumnNames() string {
	var sb strings.Builder
	for ndx := 0; ndx < len(iot.Measurements); ndx++ {
		for _, item := range iot.Measurements {
			if item.ColumnOrder == ndx && !item.Ignore {
				sb.WriteString(item.MeasurementAlias + ",")
			}
		}
	}
	str := sb.String()[0:len(sb.String())-1] + " " // replace trailing comma
	return str
}

// Expects comma-separated files. Assigns Dataset or Summary.
func (iot *IoTDbCsvDataFile) ReadCsvFile(filePath string, isDataset bool) error {
	f, err := os.Open(filePath)
	checkErr("Unable to read csv file: ", err)
	defer f.Close()
	fmt.Println("Reading " + filePath)
	csvReader := csv.NewReader(f)
	if !isDataset {
		iot.Summary, err = csvReader.ReadAll()
	} else {
		iot.Dataset, err = csvReader.ReadAll()
	}
	checkErr("Unable to parse file as CSV for "+filePath, err)
	return err
}

// Command-line parameters: {drop create delete insert ...}. Always output dataset description.
// create time series root.datasets.etsi.household_data_60min_singleindex.DE_KN_industrial1_grid_import with datatype=FLOAT, encoding=GORILLA, compressor=SNAPPY;
// ProcessTimeseries is the only place where iot.IoTDbAccess.session is instantiated and clientConfig is used.
func (iot *IoTDbCsvDataFile) ProcessTimeseries() error {
	if iot.IoTDbAccess.ActiveSession {
		iot.IoTDbAccess.session = client.NewSession(clientConfig)
		if err := iot.IoTDbAccess.session.Open(false, 0); err != nil {
			checkErr("ProcessTimeseries(iot.IoTDbAccess.session.Open): ", err)
		}
		defer iot.IoTDbAccess.session.Close()
	}
	fmt.Println("Processing time series for CSV dataset " + iot.DatasetName + " ...")

	for _, command := range iot.TimeseriesCommands {
		switch command {
		case "createdb":
			sql := "CREATE DATABASE " + iot.Identifier
			_, err := iot.IoTDbAccess.session.ExecuteNonQueryStatement(sql)
			checkErr("ExecuteNonQueryStatement(createDBstatement)", err)
			fmt.Println(sql)

		case "dropts": // time series schema; uses single statement;
			sql := "DROP TIMESERIES " + IotDatasetPrefix(iot.Identifier, iot.DatasetName) + ".*"
			_, err := iot.IoTDbAccess.session.ExecuteNonQueryStatement(sql)
			checkErr("ExecuteNonQueryStatement(dropStatement)", err)
			for k := range iot.Measurements { // does not account for Ignore items.
				delete(iot.Measurements, k)
			}

		case "createts":
			// create aligned time series schema; single statement: CREATE ALIGNED TIMESERIES root.etsidata.device.household_data_1min_singleindex (utc_timestamp TEXT encoding=PLAIN compressor=SNAPPY,  etc);
			// Note: For a group of aligned timeseries, Iotdb does not support different compressions.
			// https://iotdb.apache.org/UserGuide/V1.0.x/Reference/SQL-Reference.html#schema-statement
			var sb strings.Builder
			sb.WriteString("CREATE ALIGNED TIMESERIES " + IotDatasetPrefix(iot.Identifier, iot.DatasetName) + "(")
			for ndx := 0; ndx < len(iot.Measurements); ndx++ {
				for _, item := range iot.Measurements {
					if item.ColumnOrder == ndx && !item.Ignore {
						dataType, encoding, compressor := getClientStorage(item.MeasurementType)
						// Note: It is not currently supported to set an alias, tag, and attribute for aligned timeseries.
						//<<<attributes := " ATTRIBUTES('datatype'='" + item.MeasurementType  + "') TAGS('units'='" + item.MeasurementUnits + "')"
						sb.WriteString(item.MeasurementAlias + " " + dataType + " encoding=" + encoding + " compressor=" + compressor + ",") // attributes +
					}
				}
			}
			sql := sb.String()[0:len(sb.String())-1] + ");" // replace trailing comma
			//fmt.Println(sql)
			_, err := iot.IoTDbAccess.session.ExecuteNonQueryStatement(sql)
			checkErr("ExecuteNonQueryStatement(createStatement)", err)
			fmt.Println("IOTDB TEST QUERY: show timeseries " + IotDatasetPrefix(iot.Identifier, iot.DatasetName) + ".*;")

		case "delete": // remove all data; retain schema; multiple commands.
			deleteStatements := make([]string, 0)
			for _, item := range iot.Measurements {
				deleteStatements = append(deleteStatements, "DELETE FROM "+IotDatasetPrefix(iot.Identifier, iot.DatasetName)+"."+item.MeasurementName+";")
			}
			_, err := iot.IoTDbAccess.session.ExecuteBatchStatement(deleteStatements) // (r *common.TSStatus, err error)
			checkErr("ExecuteBatchStatement(deleteStatements)", err)

		case "insert": // insert(append) data; retain schema; either single or multiple statements;
			// Automatically inserts long time column as first column (which should be UTC). Save in blocks.
			timeIndex := 0
			blockSize := getBlockSize(len(iot.Measurements))
			nBlocks := len(iot.Dataset)/blockSize + 1
			fmt.Printf("%s%d%s", "Writing ", nBlocks, " blocks: ")
			for block := 0; block < nBlocks; block++ {
				fmt.Print(".") //fmt.Printf("%s%d-%d\n", "block: ", startRow, endRow-1)
				var sb strings.Builder
				var insert strings.Builder
				insert.WriteString("INSERT INTO " + IotDatasetPrefix(iot.Identifier, iot.DatasetName) + " (time, " + iot.FormattedColumnNames() + ") ALIGNED VALUES ")
				startRow := blockSize*block + 1
				endRow := startRow + blockSize
				if block == nBlocks-1 {
					endRow = len(iot.Dataset)
				}
				for r := startRow; r < endRow; r++ {
					sb.Reset()
					startTime, err := filesystem.GetStartTimeFromLongint(iot.Dataset[r][timeIndex]) 
					if err != nil {
						fmt.Println(iot.Dataset[r][timeIndex])
						break
					}
					sb.WriteString("(" + strconv.FormatInt(startTime.UTC().Unix(), 10) + ",")
					for c := 0; c < len(iot.Dataset[r]); c++ {
						for _, item := range iot.Measurements {
							if item.ColumnOrder == c && !item.Ignore {
								sb.WriteString(formatDataItem(iot.Dataset[r][c], item.MeasurementType) + ",") 
							}
						}
					}
					sb.WriteString(formatDataItem(iot.DatasetName, "string") + ")")
					if r < endRow-1 {
						sb.WriteString(",")
					}
					insert.WriteString(sb.String())
				}
				_, err := iot.IoTDbAccess.session.ExecuteNonQueryStatement(insert.String() + ";") // (r *common.TSStatus, err error)
				checkErr("ExecuteNonQueryStatement(insertStatement)", err)
			}
			fmt.Println("\nIOTDB TEST QUERY: SELECT COUNT(*) FROM " + IotDatasetPrefix(iot.Identifier, iot.DatasetName) + ";")
		}

		fmt.Println("Timeseries <" + command + "> completed.")
	} // for

	return nil
}

// Data source file types determined by file extension: {.nc, .csv, .hd5}  Args[0] is program name.
func main() {
	//fmt.Println(ShowLastExternalBackup())
	sourceDataType := "help"
	if len(os.Args) > 1 {
		sourceDataType = strings.ToLower(path.Ext(os.Args[1]))
	}

	switch sourceDataType {
	case ".csv":
		ProcessCsvSensorData(os.Args)
	case ".nc": // this reads the var file too.
		ProcessNcSensorData(os.Args)
	//case ".hd5":
	default:
		fmt.Println("The commands to the netcdf program copy time series data from source files into the IoT database.")
		fmt.Println("Before running netcdf, run the 'xsv stats <dataFile.csv> --everything' program to place a csv summary* file in the same folder as the <dataFile.csv>.")
		fmt.Println("netcdf parameters: full path to csv or nc sensor data file, followed by timeMeasurementName, followed by an (optional) CDF file type {HDF5, netCDF-4, classic}, ")
		fmt.Println(" followed by one or more commands: create, insert, drop, delete, query, example.")
		os.Exit(0)
	}
}

/*func printDataSet(sds *client.SessionDataSet) []string {
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
}*/

/*
type AutoGenerated struct {
	Title           string   `json:"title"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	LongDescription string   `json:"long_description"`
	Documentation   string   `json:"documentation"`
	Version         string   `json:"version"`
	LastChanges     string   `json:"last_changes"`
	Keywords        []string `json:"keywords"`
	Contributors    []struct {
		Web   string `json:"web"`
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"contributors"`
	Sources []struct {
		Web    string `json:"web"`
		Name   string `json:"name"`
		Source string `json:"source"`
	} `json:"sources"`
	Licenses []struct {
		ID      string `json:"id"`
		Version string `json:"version"`
		Name    string `json:"name"`
		URL     string `json:"url"`
	} `json:"licenses"`
	External          bool   `json:"external"`
	GeographicalScope string `json:"geographical-scope"`
	IotdbGroupname    string `json:"iotdb-groupname"`
	Resources         []struct {
		Mediatype string `json:"mediatype"`
		Format    string `json:"format"`
		Path      string `json:"path"`
		Encoding  string `json:"encoding,omitempty"`
		Schema    string `json:"schema,omitempty"`
		Dialect   struct {
			CsvddfVersion  float64 `json:"csvddfVersion"`
			Delimiter      string  `json:"delimiter"`
			LineTerminator string  `json:"lineTerminator"`
			Header         bool    `json:"header"`
		} `json:"dialect,omitempty"`
	} `json:"resources"`
	Schemas struct {
		OneMin struct {
			PrimaryKey   string `json:"primaryKey"`
			MissingValue string `json:"missingValue"`
			Fields       []struct {
				Name              string `json:"name"`
				Description       string `json:"description"`
				Type              string `json:"type"`
				Format            string `json:"format,omitempty"`
				OpsdContentfilter bool   `json:"opsd-contentfilter,omitempty"`
				Unit              string `json:"unit,omitempty"`
			} `json:"fields"`
		} `json:"1min"`
		One5Min struct {
			PrimaryKey   string `json:"primaryKey"`
			MissingValue string `json:"missingValue"`
			Fields       []struct {
				Name              string `json:"name"`
				Description       string `json:"description"`
				Type              string `json:"type"`
				Format            string `json:"format,omitempty"`
				OpsdContentfilter bool   `json:"opsd-contentfilter,omitempty"`
				Unit              string `json:"unit,omitempty"`
			} `json:"fields"`
		} `json:"15min"`
		Six0Min struct {
			PrimaryKey   string `json:"primaryKey"`
			MissingValue string `json:"missingValue"`
			Fields       []struct {
				Name              string `json:"name"`
				Description       string `json:"description"`
				Type              string `json:"type"`
				Format            string `json:"format,omitempty"`
				OpsdContentfilter bool   `json:"opsd-contentfilter,omitempty"`
				Unit              string `json:"unit,omitempty"`
			} `json:"fields"`
		} `json:"60min"`
	} `json:"schemas"`
}
*/
