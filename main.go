package main

// Define TimeseriesDataset as a queryable set of zero or more sequences of Timeseries measurements.
// Define TimeseriesDataStream as a sequence of measurements all with the same type of unit collected at periodic intervals but may include missing values.
// IotDB Data Model: https://iotdb.apache.org/UserGuide/Master/Data-Concept/Data-Model-and-Terminology.html
// https://saref.etsi.org/saref4data/v1.0.1/datasets/
/* Datatype properties (attributes) relate individuals to literal data whereas object properties relate individuals to other individuals.
   Time series as a sequence of measurements, where each measurement is defined as an object with a value, a named event, and a metric.
   A time series object binds a metric to a resource.
   Use queries to get statistical Maximum, Minimum, Mean, StandardDeviation, Median, Mode.
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
	//"bytes"
	//"context"
	//"database/sql/driver"
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"

	//"math/rand"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/apache/iotdb-client-go/client"
	"github.com/fhs/go-netcdf/netcdf"
	//"github.com/dgnabasik/netcdf/graphdb"
	//"github.com/dgnabasik/netcdf/iotdb"
	//"github.com/relvacode/iso8601"
)

/* iso8601.Time can be used as a drop-in replacement for time.Time with JSON responses
type ExternalAPIResponse struct {
	Timestamp *iso8601.Time
}
t, err := iso8601.ParseString("2020-01-02T16:20:00")  */

const ( // these do not include trailing >
	HomeDirectory          = "/home/david/" // davidgnabasik
	CurrentVersion         = "v1.0.1/"
	SarefExtension         = "saref4data/"
	s4data                 = "s4data:"
	SarefEtsiOrg           = "https://saref.etsi.org/"
	DataSetPrefix          = SarefEtsiOrg + SarefExtension + CurrentVersion + "datasets/"
	timeseriesDescription  = "/description.txt"
	serializationExtension = ".ttl" //trig<<< Classic Turtle does not support named graphs. Must output in TRiG format. https://en.wikipedia.org/wiki/TriG_(syntax)
	sparqlExtension        = ".sparql"
	jsonExtension          = ".json"
	csvExtension           = ".csv"
	varExtension           = ".var"
	ncExtension            = ".nc"
	crlf                   = "\n"
	timeAlias              = "time1"
	blockSize              = 163840 //=20*8192 	131072=16*8192
)

var xsdDatatypeMap = map[string]string{"string": "string", "int": "integer", "integer": "integer", "int64": "integer", "float": "float", "double": "double", "decimal": "double", "byte": "boolean"} // map cdf to xsd datatypes.
var NetcdfFileFormats = []string{"classic", "netCDF", "netCDF-4", "HDF5"}
var DiscreteDistributions = []string{"discreteUniform", "discreteBernoulli", "discreteBinomial", "discretePoisson"}
var ContinuousDistributions = []string{"continuousNormal", "continuousStudent_t_test", "continuousExponential", "continuousGamma", "continuousWeibull"}

/*
Discrete uniform distribution: All outcomes are equally likely.
Bernoulli Distribution: Single-trial with two possible outcomes.
Binomial Distribution: A sequence of Bernoulli events.
Poisson Distribution: The probability that an event may or may not occur.
Normal Distribution: Symmetric distribution of values around the mean.
Student t-Test Distribution: Small sample size approximation of a normal distribution.
Exponential distribution: Model elapsed time between two events.
Gamma distribution: Describes the time to wait for a fixed number of events.
Weibull Distribution: Describes a waiting time for one event, if that event becomes more or less likely with time.
*/

// Contains an entire month's worth of Entity data for every Variable where each row is indexed by {HouseIndex+LongtimeIndex}.
type EntityCleanData struct {
	HouseIndex    []string   `json:"houseindex"`
	LongtimeIndex []string   `json:"longtimeindex"`
	Data          [][]string `json:"data"`
}

// EntityCleanData initializer.
func MakeEntityCleanData(rows int, vars []MeasurementVariable) EntityCleanData {
	ecd := EntityCleanData{}
	cols := len(vars)
	ecd.Data = make([][]string, rows) // make a slice of rows slices
	for i := 0; i < rows; i++ {
		ecd.Data[i] = make([]string, cols) // make a slice of cols in each of rows slices
	}
	return ecd
}

///////////////////////////////////////////////////////////////////////////////////////////

// Separate struct if we want slice of these in container class.
// Keep these field names different from container structs to avoid confusion.
type IoTDbAccess struct {
	session       client.Session
	Sql           string   `json:"sql"`
	ActiveSession bool     `json:"activesession"`
	QueryResults  []string `json:"queryresults"`
}

// Assign iotAccess.QueryResults; return []TimeseriesProfile
// |Timeseries|Alias|Database|DataType|Encoding|Compression|Tags|Attributes|Deadband|DeadbandParameters|
func (iotAccess *IoTDbAccess) GetTimeseriesList(datasetName string) []TimeseriesProfile {
	var timeout int64 = 1000
	const blockSize = 11
	timeseriesList := make([]TimeseriesProfile, 0)
	timeseriesItem := TimeseriesProfile{}
	iotAccess.Sql = "show timeseries " + IotDatasetPrefix("") + datasetName + ".*;"
	sessionDataSet, err := iotAccess.session.ExecuteQueryStatement(iotAccess.Sql, &timeout)
	if err == nil {
		lines := printDataSet(sessionDataSet)
		for ndx, str := range lines { // first 10 lines are column headers, then 3 blank lines between each timeseries.
			if ndx > (blockSize - 1) {
				index := ndx % blockSize
				line := strings.TrimSpace(str)
				switch index {
				case 0:
					timeseriesItem = TimeseriesProfile{}
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

	// format/assign iotAccess.QueryResults
	const cwidth0 = 48
	const cwidth1 = 8
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

///////////////////////////////////////////////////////////////////////////////////////////

type TimeseriesProfile struct {
	Timeseries         string
	Alias              string
	Database           string
	DataType           string
	Encoding           string
	Compression        string
	Tags               string
	Attributes         string
	Deadband           string
	DeadbandParameters string
}

///////////////////////////////////////////////////////////////////////////////////////////

// FileExists Returns false if directory.
func FileExists(filePath string) (bool, error) {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false, nil
	}
	return !info.IsDir(), err
}

// ReadTextLines reads a whole file into memory and returns a slice of its lines. Applys .ToLower(). Skips empty lines if normalizeText is true.
func ReadTextLines(filePath string, normalizeText bool) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("ReadTextLines: %+v\n", err)
		return nil, err
	}
	fi, err := file.Stat()
	if err != nil {
		log.Printf("ReadTextLines: %+v\n", err)
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
			str := strings.ToLower(strings.TrimSpace(scanner.Text()))
			if len(str) > 0 {
				lines = append(lines, str)
			}
		} else {
			lines = append(lines, scanner.Text())
		}
	}
	fmt.Println("Read from file " + filePath)
	return lines, scanner.Err()
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
	if err != nil {
		log.Printf("WriteTextLines: %+v\n", err)
		return err
	}
	defer file.Close()

	contents := strings.Join(lines, "\n")
	if _, err := file.WriteString(contents); err != nil {
		log.Printf("WriteTextLines: %+v\n", err)
	}
	fmt.Println("Wrote to file " + filePath)
	return err
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

func GetTimeseriesCommands(programArgs []string) []string {
	timeseriesCommands := make([]string, 0)
	for ndx := range programArgs {
		_, index := find(timeSeriesCommands, strings.ToLower(programArgs[ndx]))
		if index >= 0 {
			timeseriesCommands = append(timeseriesCommands, strings.ToLower(programArgs[ndx]))
		}
	}
	return timeseriesCommands
}

// The Ecobee id vale act as a device
func IotDatasetPrefix(device string) string {
	prefix := "root.etsidata."
	if device == "" {
		return prefix + "device."
	}
	return prefix + device
}

///////////////////////////////////////////////////////////////////////////////////////////

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

// Generic NetCDF container. Not stored in IotDB.
type NetCDF struct {
	IoTDbAccess
	Identifier       string                          `json:"identifier"` // Unique ID to distinguish among different datasets.
	NetcdfType       string                          `json:"netcdftype"` // /usr/bin/ncdump -k cdf.nc ==> NetcdfFileFormats
	Dimensions       map[string]int                  `json:"dimensions"`
	Title            string                          `json:"title"`
	Description      string                          `json:"description"`
	Conventions      string                          `json:"conventions"`
	Institution      string                          `json:"institution"`
	Code_url         string                          `json:"code_url"`
	Location_meaning string                          `json:"location_meaning"`
	Datastream_name  string                          `json:"datastream_name"`
	Input_files      string                          `json:"input_files"`
	History          string                          `json:"history"`
	Measurements     map[string]*MeasurementVariable `json:"measurements"`
	HouseIndices     []string                        `json:"houseindices"`    // these unique 2 indices are specific to Ecobee datasets.
	LongtimeIndices  []string                        `json:"longtimeindices"` // Different months will have slightly different HouseIndices!
	// these are not in the source file.
	TimeseriesCommands []string   `json:"timeseriescommands"` // command-line parameters
	DataFilePath       string     `json:"datafilepath"`
	SummaryFilePath    string     `json:"summaryfilepath"`
	DatasetName        string     `json:"datasetname"`
	Summary            [][]string `json:"summary"` // from summary file
	Dataset            [][]string `json:"dataset"` // actual data
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

// Output struct as JSON.
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

// Write the validating SPARQL construct/query to a file.
func (cdf NetCDF) Format_SparqlQuery() []string {
	output := make([]string, 0)
	/*
		# queries

		select (?eventLabel as ?Event_Label) (?value as ?Value) where {
		?salesRegion salesRegion:code ?SalesRegionCode.
		   ?timeSeries timeSeries:appliesTo ?salesRegion.
		   ?timeSeries timeSeries:metric ?metric.
		   ?measure measurement:metric ?metric.
		   ?metric metric:code ?MetricCode.
		   ?metric metric:units ?currency.
		?currency currency:mask ?format.
		   ?measure measurement:metricEvent ?event.
		   ?measure measurement:value ?unformattedValue.
		   bind (fn:formatNumber(?unformattedValue,?format) as ?value
		   ?event rdfs:label ?eventLabel.
		   ?event event:startDateTime ?startTime.
		} order by ?startTime,
		{salesRegionCode:"PNW",
		 metricCode:"RevDef121"}

		select (sum(?value) as ?Sum) (avg(?value) as ?Average) where {
		?salesRegion salesRegion:code ?SalesRegionCode.
		   ?timeSeries timeSeries:appliesTo ?salesRegion.
		   ?timeSeries timeSeries:metric ?metric.
		   ?measure measurement:metric ?metric.
		   ?metric metric:code ?MetricCode.
		   ?metric metric:units ?currency.
		   ?measure measurement:metricEvent ?event.
		   ?measure measurement:value ?unformattedValue.
		   bind (fn:formatNumber(?unformattedValue,?format) as ?value
		   ?event rdfs:label ?eventLabel.
		   ?event event:startDateTime ?startTime.
		} order by ?startTime,
		{SalesRegionCode:"PNW",
		 MetricCode:"RevDef121"}

		 select
		 ?SalesRegionCode (sum(?value) as ?Sum) (avg(?value) as ?Average) where {
		   ?salesRegion salesRegion:code ?SalesRegionCode.
		   ?timeSeries timeSeries:appliesTo ?salesRegion.
		   ?timeSeries timeSeries:metric ?metric.
		   ?measure measurement:metric ?metric.
		   ?metric metric:code $metricCode.
		   ?metric metric:units ?currency.
		   ?currency currency:mask ?format.
		   ?measure measurement:metricEvent ?event.
		   ?measure measurement:value ?unformattedValue.
		   bind (fn:formatNumber(?unformattedValue,?format) as ?value
		   ?event rdfs:label ?eventLabel.
		   ?event event:startDateTime ?startTime.
		} order by ?SalesRegionCode ?startTime
		group by ?SalesRegionCode,
		{metricCode:"RevDef121"}
	*/
	return output
}

// accept either original or alias names.
func (cdf NetCDF) GetMeasurementVariableFromName(name string) (MeasurementVariable, bool) {
	item, ok := cdf.Measurements[name]
	if ok {
		return *item, true
	}
	// check alias names
	for _, item := range cdf.Measurements {
		if strings.ToLower(name) == strings.ToLower(item.MeasurementAlias) || strings.ToLower(name) == strings.ToLower(item.MeasurementItem.MeasurementName) {
			return *item, true
		}
	}
	return MeasurementVariable{}, false
}

// Return all column values except header row. Return false if wrong column name. Fill in default values.
func (cdf NetCDF) GetSummaryStatValues(columnName string) ([]string, bool) {
	_, columnIndex := find(summaryColumnNames, columnName)
	if columnIndex < 0 {
		return []string{}, false
	}
	stats := make([]string, len(cdf.Measurements))
	for ndx := 0; ndx < len(cdf.Measurements)-1; ndx++ {
		dataColumnName, _ := StandardName(cdf.Summary[ndx+1][0])
		item, _ := cdf.GetMeasurementVariableFromName(dataColumnName) // MeasurementVariable
		stats[ndx] = strings.TrimSpace(cdf.Summary[ndx+1][columnIndex])
		if strings.HasPrefix(strings.ToLower(item.MeasurementType), "int") {
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

// Make the dataset its own Class and loadable into GraphDB. Produce specific DatatypeProperty ontology from dataset. Special handling: "dateTime", "XMLLiteral", "anyURI".
func (cdf NetCDF) Format_Ontology() []string {
	baseline := getBaselineOntology(cdf.Identifier, cdf.Title, cdf.Description)
	output := strings.Split(baseline, crlf)
	output = append(output, `### specific time series DatatypeProperties`)
	ndx := 0
	for _, v := range cdf.Measurements {
		str := `[ rdf:type owl:Restriction ;` + crlf + `owl:onProperty ` + s4data + `has` + v.MeasurementItem.MeasurementName + ` ;` + crlf + `owl:allValuesFrom xsd:` + xsdDatatypeMap[strings.ToLower(v.MeasurementItem.MeasurementType)] + crlf
		if ndx < len(cdf.Measurements) {
			str += `] ,`
		} else {
			str += `] ;`
		}
		ndx++
		output = append(output, str)
	}

	output = append(output, ` rdfs:comment "`+cdf.Description+`"@en ;`)
	output = append(output, ` rdfs:label "`+cdf.Identifier+`"@en .`)

	// append single NamedIndividual.
	output = append(output, `### externel references`+crlf)
	externStr := getExternalReferences()
	output = append(output, strings.Split(externStr, crlf)...)

	// new common Classes: StartTimeseries, StopTimeseries
	uniqueID := GetUniqueInstanceID()
	output = append(output, `### class instances`+crlf)
	output = append(output, `ex:StartTimeseries`+uniqueID)
	output = append(output, `rdf:type `+s4data+`StartTimeseries ;`)
	output = append(output, `rdf:label "StartTimeseries`+uniqueID+`"^^xsd:string .`+crlf)
	output = append(output, `ex:StopTimeseries`+uniqueID)
	output = append(output, `rdf:type `+s4data+`StopTimeseries ;`)
	output = append(output, `rdf:label "StopTimeseries`+uniqueID+`"^^xsd:string .`+crlf)
	output = append(output, `### internel references`+crlf)

	values, ok := cdf.GetSummaryStatValues("mean")
	output = append(output, `<`+DataSetPrefix+cdf.Identifier+uniqueID+`> rdf:type owl:NamedIndividual , `)
	output = append(output, `  saref:Measurement , saref:Time , saref:UnitOfMeasure , s4envi:FrequencyUnit , s4envi:FrequencyMeasurement ;`+crlf)

	if ok {
		for ndx := 0; ndx < len(cdf.Measurements); ndx++ {
			for _, v := range cdf.Measurements { //
				if v.ColumnOrder == ndx && !v.Ignore {
					str := s4data + v.MeasurementName + ` "` + values[ndx] + `"^^xsd:` + xsdDatatypeMap[strings.ToLower(v.MeasurementItem.MeasurementType)]
					if v.ColumnOrder < len(cdf.Measurements)-1 {
						str += ` ;`
					} else {
						str += ` .`
					}
					output = append(output, str)
					break
				}
			}
		}
	}
	output = append(output, "")
	return output
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

// Expects {Units, DatasetName} fields to have been appended to the summary file. Assign []Measurements. Expects Summary to be assigned. Use XSD data types.
// len() only returns the length of the "external" array.
func (cdf *NetCDF) XsvSummaryTypeMap() {
	rowsXsdMap := map[string]string{"Unicode": "string", "Float": "float", "Integer": "integer", "Longint": "int64", "Double": "double"}
	cdf.Measurements = make(map[string]*MeasurementVariable, 0)
	// get units column
	unitsColumn := 0
	for ndx := 0; ndx < len(cdf.Summary[0]); ndx++ { // iterate over summary header row
		if strings.ToLower(cdf.Summary[0][ndx]) == "units" {
			unitsColumn = ndx
			break
		}
	}
	ndx1 := 0
	dimMap := cdf.getDimensionMap()
	for ndx := 0; ndx < 256; ndx++ { // iterate over summary file rows.
		if cdf.Summary[ndx+1][0] == "interpolated" || len(cdf.Summary[ndx+1][0]) < 2 {
			ndx1 = ndx
			break
		}
		dataColumnName, aliasName := StandardName(cdf.Summary[ndx+1][0]) // NOTE!! does not include cdf.Summary[ndx+1][0] == "time" ||
		ignore := cdf.Summary[ndx+1][2] == "0" && cdf.Summary[ndx+1][2] == "0" && cdf.Summary[ndx+1][4] == "0"
		if ignore {
			fmt.Println("Ignoring empty data column " + dataColumnName)
		}
		theEnd := dataColumnName == "interpolated"
		if !theEnd {
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

// If there are no values at all in the block, IoTDB does not write the measurement, so get mismatch between number of INSERT field names and VALUES (e.g., HeatingEquipmentStage3_RunTime)
// Ids are consecutive: data rows = 8838720; 990 distinct id values; ==> 8928 rows per id-device
func (cdf *NetCDF) CopyCsvTimeseriesDataIntoIotDB() error {
	fileToRead := cdf.DataFilePath + "/csv/" + cdf.DatasetName + csvExtension
	err := cdf.ReadCsvFile(fileToRead, true) // isDataset: yes
	checkErr("cdf.ReadCsvFile ", err)
	nBlocks := cdf.Dimensions["id"]     //=990	 len(cdf.Dataset)/(blockSize) + 1
	blockSize := cdf.Dimensions["time"] // scope override
	timeIndex := 1                      //<<< need to calculate
	fmt.Printf("%s%d%s", "Writing ", nBlocks, " blocks: ")
	for block := 0; block < nBlocks; block++ {
		fmt.Print(block + 1)
		fmt.Print(" ")
		var sb strings.Builder
		var insert strings.Builder
		insert.WriteString("INSERT INTO " + IotDatasetPrefix(cdf.DatasetName+"."+cdf.HouseIndices[block]) + " (time," + cdf.FormattedColumnNames() + ") ALIGNED VALUES ")
		startRow := blockSize*block + 1
		endRow := startRow + blockSize
		if block == nBlocks-1 {
			endRow = len(cdf.Dataset)
		}
		for r := startRow; r < endRow; r++ {
			sb.Reset()
			startTime, err := getStartTimeFromString(cdf.Dataset[r][timeIndex])
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
		//WriteStringToFile(HomeDirectory+"Downloads/block" + strconv.Itoa(block) + ".txt", insert.String())
	}
	fmt.Println()
	return err
}

// Assume time series have been created; erase existing data; insert data. Assigns cdf.Dataset. INCOMPLETE! Converted *.nc files to CSV files. See CopyCSVTimeseriesDataIntoIotDB().
// mapNetcdfGolangTypes: "byte": "int8", "ubyte": "uint8", "char": "string", "short": "int16", "ushort": "uint16", "int": "int32", "uint": "uint32", "int64": "int64", "uint64": "uint64", "float": "float32", "double": "float64"
func (cdf *NetCDF) CopyNcTimeseriesDataIntoIotDB() error {
	iotPrefix := IotDatasetPrefix(cdf.DatasetName)
	const createMsg string = " time series not found -- run this program with the create parameter first " // also get this if no data in column
	fileToRead := cdf.DataFilePath + "/" + cdf.DatasetName + ncExtension
	nc, err := netcdf.OpenFile(fileToRead, netcdf.NOWRITE)
	if err != nil {
		checkErr("Could not access "+fileToRead, err)
	}
	defer nc.Close()

	// Read every NetCDF variable to construct an aligned time series.  For Jan_clean: id = 990; time = 8928 => datasetSize := 8838720
	for ndx := 0; ndx < len(cdf.Measurements); ndx++ {
		for _, item := range cdf.Measurements { // LastColumnName values do not exist in any *.nc data file.
			/*skipThis := item.MeasurementName == LastColumnName || strings.ToLower(item.MeasurementName) == "time" || strings.ToLower(item.MeasurementAlias) == timeAlias || strings.ToLower(item.MeasurementAlias) == "id"
			if skipThis {
				//fmt.Println("Skipping " + item.MeasurementAlias)
				continue
			}*/
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
				case "unicode", "string":
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

// Similar to the iot version but not the same. REFACTOR
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
		case "drop": // time series schema; uses single statement;
			sql := "DROP TIMESERIES " + IotDatasetPrefix(cdf.DatasetName) + ".*"
			_, err := cdf.IoTDbAccess.session.ExecuteNonQueryStatement(sql)
			checkErr("ExecuteNonQueryStatement(dropStatement)", err)
			for k := range cdf.Measurements {
				delete(cdf.Measurements, k)
			}

		case "create":
			// create aligned time series schema; single statement: CREATE ALIGNED TIMESERIES root.etsidata.household_data_1min_singleindex.<id> (utc_timestamp TEXT encoding=PLAIN compressor=SNAPPY,  etc);
			// Note: For a group of aligned timeseries, Iotdb does not support different compressions.
			// https://iotdb.apache.org/UserGuide/V1.0.x/Reference/SQL-Reference.html#schema-statement
			var sb strings.Builder
			var sql string
			// Use each id as a 'device'; read from var file; are unique.
			for id := 0; id < len(cdf.HouseIndices); id++ {
				sb.Reset()
				sb.WriteString("CREATE ALIGNED TIMESERIES " + IotDatasetPrefix(cdf.DatasetName+"."+cdf.HouseIndices[id]) + "(")
				for ndx := 0; ndx < len(cdf.Measurements); ndx++ {
					for _, v := range cdf.Measurements {
						if v.ColumnOrder == ndx && !v.Ignore {
							dataType, encoding, compressor := getClientStorage(v.MeasurementItem.MeasurementType)
							sb.WriteString(v.MeasurementAlias + " " + dataType + " encoding=" + encoding + " compressor=" + compressor + ",")
						}
					}
				}
				sql = sb.String()[0:len(sb.String())-1] + ");" // replace trailing comma
				_, err := cdf.IoTDbAccess.session.ExecuteNonQueryStatement(sql)
				checkErr("ExecuteNonQueryStatement(createStatement)", err)
			}
			fmt.Print("Created 'id-Devices' for " + cdf.DatasetName + ": ")
			fmt.Println(len(cdf.HouseIndices))

		case "delete": // remove all data; retain schema; multiple commands.
			deleteStatements := make([]string, 0)
			for _, item := range cdf.Measurements {
				deleteStatements = append(deleteStatements, "DELETE FROM "+IotDatasetPrefix(cdf.DatasetName)+item.MeasurementName+";")
			}
			_, err := cdf.IoTDbAccess.session.ExecuteBatchStatement(deleteStatements) // (r *common.TSStatus, err error)
			checkErr("ExecuteBatchStatement(deleteStatements)", err)

		case "insert": // insert(append) data; retain schema; either single or multiple statements;
			// Automatically inserts long time column as first column (which should be UTC). Save in blocks.
			err := cdf.CopyCsvTimeseriesDataIntoIotDB()
			checkErr("ExecuteNonQueryStatement(insertStatements)", err)

		case "query":
			cdf.GetTimeseriesList(cdf.DatasetName)
			outputPath := GetOutputPath(cdf.DataFilePath, ".sql")
			err := WriteTextLines(cdf.QueryResults, outputPath, false)
			checkErr("WriteTextLines(query)", err)

		case "example": // serialize saref class file:
			ttlLines := cdf.Format_Ontology()
			outputPath := GetOutputPath(cdf.DataFilePath, serializationExtension)
			err := WriteTextLines(ttlLines, outputPath, false)
			checkErr("WriteTextLines(example)", err)

		}
		fmt.Println("Timeseries <" + command + "> completed.")
	} // for

	return nil
}

// Return NetCDF struct by parsing Jan_clean.var file that is output from /usr/bin/ncdump -c Jan_clean.nc. Var files are specific to *.nc datasets.
func ParseVariableFile(varFile, filetype, dataSetIdentifier string, description, programArgs []string, isActive bool) (NetCDF, error) {
	lines, err := ReadTextLines(varFile, false)
	checkErr("Could not access "+varFile, err)
	datasetPathName := filepath.Dir(programArgs[1]) // does not include trailing slash
	ioTDbAccess := IoTDbAccess{ActiveSession: isActive}
	xcdf := NetCDF{IoTDbAccess: ioTDbAccess, Description: strings.Join(description, " "), DataFilePath: datasetPathName, DatasetName: dataSetIdentifier, NetcdfType: filetype}
	xcdf.TimeseriesCommands = GetTimeseriesCommands(programArgs)
	xcdf.SummaryFilePath = datasetPathName + "/" + summaryPrefix + filepath.Base(varFile) + csvExtension
	xcdf.SummaryFilePath = strings.Replace(xcdf.SummaryFilePath, varExtension, "", 1)
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
	xcdf.LongtimeIndices, lineIndex = parseDimensionIndices(xcdf.Dimensions["time"], lineIndex, lines)
	return xcdf, nil
}

///////////////////////////////////////////////////////////////////////////////////////////

// Excluded "rdf:type owl:Restriction owl:onProperty saref:measurementMadeBy owl:someValuesFrom saref:Device" because don't know Device.
func getBaselineOntology(identifier, title, description string) string {
	return `@prefix ` + s4data + ` <` + DataSetPrefix + `> .` + crlf +
		`@prefix ex: <` + DataSetPrefix + identifier + serializationExtension + `> .` + crlf +
		`@prefix foaf: <http://xmlns.com/foaf/spec/#> .` + crlf +
		`@prefix geosp: <http://www.opengis.net/ont/geosparql#> .` + crlf +
		`@prefix obo: <http://purl.obolibrary.org/obo/> .` + crlf +
		`@prefix org: <https://schema.org/> .` + crlf +
		`@prefix owl: <http://www.w3.org/2002/07/owl#> .` + crlf +
		`@prefix org: <https://schema.org/> .` + crlf +
		`@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .` + crlf +
		`@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .` + crlf +
		`@prefix saref: <` + SarefEtsiOrg + `core/> .` + crlf +
		// include all referenced extensions here without a version number:
		`@prefix s4ehaw: <` + SarefEtsiOrg + `saref4ehaw/> .` + crlf +
		`@prefix s4envi: <` + SarefEtsiOrg + `saref4envi/> .` + crlf +
		`@prefix s4auto: <` + SarefEtsiOrg + `saref4auto/> .` + crlf +
		`@prefix ssn: <http://www.w3.org/ns/ssn/> .` + crlf +
		`@prefix time: <http://www.w3.org/2006/time#> .` + crlf +
		`@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .` + crlf +
		`@prefix dcterms: <http://purl.org/dc/terms/> .` + crlf +
		`@prefix dctype: <http://purl.org/dc/dcmitype/> .` + crlf +
		`<` + SarefEtsiOrg + SarefExtension + CurrentVersion + `> rdf:type owl:Ontology ;` + crlf +
		`    owl:versionInfo "v3.1.1" ;` + crlf +
		`    owl:versionIRI <https://saref.etsi.org/core/v3.1.1/> ;` + crlf +
		`dcterms:title "` + title + `"@en ;` + crlf +
		`dcterms:description "` + description + `"@en ;` + crlf +
		`dcterms:license <https://forge.etsi.org/etsi-software-license> ;` + crlf +
		`dcterms:conformsTo <` + SarefEtsiOrg + `core/v3.1.1/> ;` + crlf + // include every referenced extension!
		`dcterms:conformsTo <` + SarefEtsiOrg + `saref4ehaw/v1.1.2/> ;` + crlf +
		`dcterms:conformsTo <` + SarefEtsiOrg + `saref4envi/v1.1.2/> ;` + crlf +
		`dcterms:conformsTo <` + SarefEtsiOrg + `saref4auto/v1.1.2/> ;` + crlf +
		`dcterms:conformsTo <` + SarefEtsiOrg + SarefExtension + CurrentVersion + `> .` + // period at end of block
		crlf + crlf +
		// extension class declarations for domains, ranges, rdfs:isDefinedBy
		`###  ` + SarefEtsiOrg + `core/Time` + crlf +
		`saref:Time rdf:type owl:Class .` + crlf +
		crlf +
		`###  ` + SarefEtsiOrg + `core/Measurement` + crlf +
		`saref:Measurement rdf:type owl:Class .` + crlf +
		crlf +
		`###  ` + SarefEtsiOrg + `core/UnitOfMeasure` + crlf +
		`saref:UnitOfMeasure rdf:type owl:Class .` + crlf +
		crlf +
		`###  ` + SarefEtsiOrg + `saref4ehaw/MeasurementFunction` + crlf +
		`s4ehaw:MeasurementFunction rdf:type owl:Class .` + crlf +
		crlf +
		`###  ` + SarefEtsiOrg + `saref4ehaw/Data` + crlf +
		`s4ehaw:Data rdf:type owl:Class .` + crlf +
		crlf +
		`###  ` + SarefEtsiOrg + `saref4envi/FrequencyUnit` + crlf +
		`s4envi:FrequencyUnit rdf:type owl:Class .` + crlf +
		crlf +
		`###  ` + SarefEtsiOrg + `saref4envi/FrequencyMeasurement` + crlf +
		`s4envi:FrequencyMeasurement rdf:type owl:Class .` + crlf +
		crlf +
		`###  ` + SarefEtsiOrg + `saref4auto/Confidence` + crlf +
		`s4auto:Confidence rdf:type owl:Class .` + crlf +
		crlf +
		// new common Classes
		`###  ` + SarefEtsiOrg + SarefExtension + `StartTimeseries` + crlf +
		`` + s4data + `StartTimeseries rdf:type owl:Class ;` + crlf +
		` rdfs:comment "The start time of a time series shall be present."@en ;` + crlf +
		` rdfs:label "start time series"@en ;` + crlf +
		` rdfs:subClassOf <http://www.w3.org/2006/time#TemporalEntity> ; .` + crlf +
		crlf +
		`###  ` + SarefEtsiOrg + SarefExtension + `StopTimeseries` + crlf +
		`` + s4data + `StopTimeseries rdf:type owl:Class ;` + crlf +
		` rdfs:comment "The stop time of a time series shall be present."@en ;` + crlf +
		` rdfs:label "stop time series"@en ;` + crlf +
		` rdfs:subClassOf <http://www.w3.org/2006/time#TemporalEntity> ; .` + crlf +
		crlf +
		// new common ObjectProperty
		`###  ` + SarefEtsiOrg + SarefExtension + `hasEquation` + crlf +
		`` + s4data + `hasEquation rdf:type owl:ObjectProperty ;` + crlf +
		` rdfs:comment "A relationship indicating that the entire time series dataset is represented by a type of equation such as {linear, quadratic, polynomial, exponential, radical, trigonometric, or partial differential}."@en ;` + crlf +
		` rdfs:label "has equation"@en .` + crlf +
		crlf +
		`###  ` + SarefEtsiOrg + SarefExtension + `hasDistribution` + crlf +
		`` + s4data + `hasDistribution rdf:type owl:ObjectProperty ;` + crlf +
		` rdfs:comment "A relationship indicating that the entire time series dataset is accurately represented by a type of discrete or continuous distribution such as {Uniform, Bernoulli, Binomial, Poisson; Normal, Student_t_test, Exponential, Gamma, Weibull}."@en ;` + crlf +
		` rdfs:label "has distribution"@en .` + crlf +
		crlf +
		`###  ` + SarefEtsiOrg + SarefExtension + `isPriorTo` + crlf +
		`` + s4data + `isPriorTo rdf:type owl:ObjectProperty ;` + crlf +
		` rdfs:comment "A relationship indicating that the time series dataset acts as a Bayesian prior to another dataset."@en ;` + crlf +
		` rdfs:label "is Bayesian prior to "@en .` + crlf +
		crlf +
		`###  ` + SarefEtsiOrg + SarefExtension + `isPosteriorTo` + crlf +
		`` + s4data + `isPosteriorTo rdf:type owl:ObjectProperty ;` + crlf +
		` rdfs:comment "A relationship indicating that the time series dataset acts as a Bayesian posterior to another dataset."@en ;` + crlf +
		` rdfs:label "is Bayesian posterior to "@en .` + crlf +
		crlf +
		`###  ` + SarefEtsiOrg + SarefExtension + `isComparableTo` + crlf +
		`` + s4data + `isComparableTo rdf:type owl:ObjectProperty ;` + crlf +
		` rdfs:comment "A relationship indicating that the time series dataset can be logically compared to another dataset. Necessary condition: type of Units and type of Distribution must agree. Sufficient condition: type of Equation must agree."@en ;` + crlf +
		` rdfs:label "is comparable to "@en .` + crlf +
		crlf +
		// non-core ObjectProperty extension references:
		`###  ` + SarefEtsiOrg + `saref4ehaw/hasTimeSeriesMeasurement` + crlf + // s4ehaw:
		`s4ehaw:hasTimeSeriesMeasurement rdf:type owl:ObjectProperty ;` + crlf +
		` rdfs:domain s4ehaw:Data ;` + crlf +
		` rdfs:range s4ehaw:TimeseriesMeasurement ;` + crlf +
		` rdfs:comment "Data has time series measurements, a sequence taken at successive equally spaced points in time."@en ;` + crlf +
		` rdfs:label "has time series measurement"@en .` + crlf +
		crlf +
		// new common DatatypeProperties
		`###  ` + SarefEtsiOrg + SarefExtension + `isRaw` + crlf +
		`` + s4data + `isRaw rdf:type owl:DatatypeProperty ;` + crlf +
		` rdfs:range xsd:boolean ;` + crlf +
		` rdfs:comment "The time series has (not) been cleaned or curated."@en ;` + crlf +
		` rdfs:label "is raw data set"@en .` + crlf +
		crlf +
		`###  ` + SarefEtsiOrg + SarefExtension + `isAlignedTimeseries` + crlf +
		`` + s4data + `isAlignedTimeseries rdf:type owl:DatatypeProperty ;` + crlf +
		` rdfs:range xsd:string ;` + crlf +
		` rdfs:comment "The name of the time sequence that the time series is aligned with."@en ;` + crlf +
		` rdfs:label "is aligned time series"@en .` + crlf +
		crlf +
		`###  ` + SarefEtsiOrg + SarefExtension + `hasSamplingPeriodValue` + crlf +
		`` + s4data + `hasSamplingPeriodValue rdf:type owl:DatatypeProperty ;` + crlf +
		` rdfs:range xsd:float ;` + crlf +
		` rdfs:comment "The sampling period in seconds."@en ;` + crlf +
		` rdfs:label "has sampling period value"@en .` + crlf +
		crlf +
		`###  ` + SarefEtsiOrg + SarefExtension + `hasUpperLimitValue` + crlf +
		`` + s4data + `hasUpperLimitValue rdf:type owl:DatatypeProperty ;` + crlf +
		` rdfs:range xsd:float ;` + crlf +
		` rdfs:comment "The highest value in the time series."@en ;` + crlf +
		` rdfs:label "has upper limit value"@en .` + crlf +
		crlf +
		`###  ` + SarefEtsiOrg + SarefExtension + `hasLowerLimitValue` + crlf +
		`` + s4data + `hasLowerLimitValue rdf:type owl:DatatypeProperty ;` + crlf +
		` rdfs:range xsd:float ;` + crlf +
		` rdfs:comment "The lowest value in the time series."@en ;` + crlf +
		` rdfs:label "has lower limit value"@en .` + crlf +
		crlf +
		`###  ` + SarefEtsiOrg + SarefExtension + `hasNumericPrecision` + crlf +
		`` + s4data + `hasNumericPrecision rdf:type owl:DatatypeProperty ;` + crlf +
		` rdfs:range xsd:integer ;` + crlf +
		` rdfs:comment "Indicates the number of trailing significant digits in the measurement."@en ;` + crlf +
		` rdfs:label "has numeric precision"@en .` + crlf +
		crlf +
		// non-core ObjectProperty extension references:
		`###  ` + SarefEtsiOrg + SarefExtension + `hasFrequencyMeasurement` + crlf + // s4envi
		`` + s4data + `hasFrequencyMeasurement rdf:type owl:ObjectProperty ;` + crlf +
		` rdfs:isDefinedBy <` + SarefEtsiOrg + `saref4envi/hasFrequencyMeasurement> ;` + crlf +
		` rdfs:comment "The relation between a device and the frequency in which it makes measurements."@en ;` + crlf +
		` rdfs:label "has frequency measurement"@en .` + crlf +
		crlf +
		`###  ` + SarefEtsiOrg + SarefExtension + `hasConfidence` + crlf + // s4auto
		`` + s4data + `hasConfidence rdf:type owl:ObjectProperty ;` + crlf +
		` rdfs:isDefinedBy <` + SarefEtsiOrg + `saref4auto/hasConfidence> ;` + crlf +
		` rdfs:comment "A relation between an estimated measurement (saref:Measurement class) and its confidence (s4auto:Confidence)"@en ;` + crlf +
		` rdfs:label "has confidence"@en .` + crlf +
		crlf +
		// define the dataset Class derived from various saref Classes but NOT s4ehaw:TimeSeriesMeasurement because that demands rdf:Seq or rdf:List.
		`### ` + DataSetPrefix + identifier + crlf +
		`` + s4data + `` + identifier + ` rdf:type owl:Class ;` + crlf +
		` rdfs:subClassOf saref:Measurement , saref:Time , saref:UnitOfMeasure , s4envi:FrequencyUnit , s4envi:FrequencyMeasurement ,` + crlf +
		// common Measurement properties;
		`[` + crlf +
		`  rdf:type owl:Restriction ;` + crlf +
		`  owl:minQualifiedCardinality "1"^^xsd:nonNegativeInteger ;` + crlf +
		`  owl:onClass ` + s4data + `StartTimeseries ;` + crlf +
		`  owl:onProperty saref:hasTime ;` + crlf +
		`] ,` + crlf +
		`[ rdf:type owl:Restriction ;` + crlf +
		`  owl:maxQualifiedCardinality "1"^^xsd:nonNegativeInteger ;` + crlf +
		`  owl:onClass ` + s4data + `StopTimeseries ;` + crlf +
		`  owl:onProperty saref:hasTime ;` + crlf +
		`] ,` + crlf +
		// core Measurement properties
		`[ rdf:type owl:Restriction ;` + crlf +
		`  owl:onProperty saref:hasTime ;` + crlf +
		`  owl:allValuesFrom saref:Time` + crlf +
		`] ,` + crlf +
		`[ rdf:type owl:Restriction ;` + crlf +
		`  owl:onProperty saref:hasMeasurement ;` + crlf +
		`  owl:allValuesFrom saref:Measurement ` + crlf +
		`] ,` + crlf +
		`[ rdf:type owl:Restriction ;` + crlf +
		`  owl:onProperty saref:relatesToMeasurement ;` + crlf +
		`  owl:allValuesFrom saref:Measurement` + crlf +
		`] ,` + crlf +
		`[ rdf:type owl:Restriction ;` + crlf +
		`  owl:onProperty saref:isMeasuredIn ;` + crlf +
		`  owl:allValuesFrom saref:UnitOfMeasure` + crlf +
		`] ,` + crlf +
		`[ rdf:type owl:Restriction ;` + crlf +
		`  owl:onProperty saref:relatesToProperty ;` + crlf +
		`  owl:allValuesFrom saref:Property` + crlf +
		`] ,` + crlf +
		`[ rdf:type owl:Restriction ;` + crlf +
		`  owl:onProperty saref:isMeasuredIn ;` + crlf +
		`  owl:qualifiedCardinality "1"^^xsd:nonNegativeInteger ;` + crlf +
		`  owl:onClass saref:UnitOfMeasure:` + crlf +
		`] ,` + crlf +
		`[ rdf:type owl:Restriction ;` + crlf +
		`  owl:onProperty saref:hasTimestamp ;` + crlf +
		`  owl:allValuesFrom xsd:dateTime` + crlf +
		`] ,` + crlf +
		`[ rdf:type owl:Restriction ;` + crlf +
		`  owl:onProperty saref:hasValue ;` + crlf +
		`  owl:qualifiedCardinality "1"^^xsd:nonNegativeInteger ;` + crlf +
		`  owl:onDataRange xsd:float` + crlf +
		`] ,` + crlf +
		// extension Measurement properties
		`[ rdf:type owl:Restriction ;` + crlf + // s4auto
		`  owl:onProperty s4auto:hasConfidence ;` + crlf +
		`  owl:someValuesFrom s4auto:Confidence` + crlf +
		`] ,` + crlf
}

// ObjectProperties are appended with uniqueID.
func getExternalReferences() string {
	return `saref:Measurement a owl:Class .` + crlf +
		`saref:Time a owl:Class .` + crlf +
		`saref:UnitOfMeasure a owl:Class .` + crlf +
		`s4envi:FrequencyUnit a owl:Class .` + crlf +
		`s4envi:FrequencyMeasurement a owl:Class .` + crlf +
		`s4ehaw:MeasurementFunction a owl:Class .` + crlf +
		`s4ehaw:Data a owl:Class .` + crlf +
		`s4envi:FrequencyUnit a owl:Class .` + crlf +
		`s4envi:FrequencyMeasurement a owl:Class .` + crlf +
		`s4auto:Confidence a owl:Class .` + crlf +
		`s4ehaw:hasTimeSeriesMeasurement a owl:ObjectProperty .` + crlf // this is the only external ObjectProperty
}

// Expects nginx server running at http://localhost:80/datasets
// Upload JSON descriptor files, turtle files, SPARQL query files.
func UploadFile(destinationUrl string, lines []string) error {
	//<<<
	return nil
}

// Not used.
func readJsonFileInto_NetCDF(filename string) (NetCDF, error) {
	funcName := "readJsonFileInto_NetCDF"
	file, err := os.Open(filename)
	checkErr(funcName, err)
	defer file.Close()

	byteValue, err := io.ReadAll(file)
	checkErr(funcName, err)

	var data NetCDF
	err = json.Unmarshal(byteValue, &data)
	checkErr(funcName, err)

	return data, nil
}

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
	exists, _ := FileExists(fp)
	if !exists {
		return "The database has never been backed up to an external drive."
	}
	lines, _ := ReadTextLines(fp, false)
	msg := strings.Replace(lines[0], "               ", "Last backup at ", 1)
	return msg
}

func GetOutputPath(filePath, ext string) string {
	return filePath[:len(filePath)-len(filepath.Ext(filePath))] + ext
}

func CreateIotSession(programArgs []string) bool {
	createIotSession := !(len(programArgs) == 3 && programArgs[2] == timeSeriesCommands[0])
	if createIotSession {
		iotdbConnection, ok := Init_IoTDB(createIotSession)
		if !ok {
			checkErr("Init_IoTDB: ", errors.New(iotdbConnection))
		}
	}
	return createIotSession
}

// Produce *.var file using: /usr/bin/ncdump -k cdf.nc  &&  /usr/bin/ncdump -c Jan_clean.nc
func LoadCsvSensorDataIntoDatabase(programArgs []string) {
	createIotSession := CreateIotSession(programArgs)
	iotdbDataFile, err := Initialize_IoTDbCsvDataFile(createIotSession, programArgs)
	checkErr("Initialize_IoTDbCsvDataFile: ", err)
	err = iotdbDataFile.ProcessTimeseries()
	checkErr("ProcessTimeseries(csv)", err)
}

func isAccessibleSensorDataFile(dataFilePath string) error {
	exists, err := FileExists(dataFilePath)
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
	datasetPathName := programArgs[1]                // *.nc file
	fileType := programArgs[2]                       // subtype of *.nc file.
	outputPath := GetOutputPath(datasetPathName, "") // path has no extension
	datasetName := path.Base(outputPath)             // Jan_clean
	description, _ := ReadTextLines(filepath.Dir(datasetPathName)+timeseriesDescription, false)
	isAccessibleSensorDataFile(datasetPathName)

	xcdf, err := ParseVariableFile(outputPath+varExtension, fileType, datasetName, description, programArgs, isActive)
	checkErr("ParseVariableFile", err)
	//fmt.Println(xcdf.ToString(true)) // true => output variables

	// move UploadFile() statements.
	err = xcdf.ReadCsvFile(xcdf.SummaryFilePath, false) // isDataset: no, is summary
	checkErr("ReadCsvFile ", err)
	//err = UploadFile(DataSetPrefix+outputCsvFile, summaryLines)
	//checkErr("UploadFile: "+DataSetPrefix+datasetName+serializationExtension, err)

	xcdf.XsvSummaryTypeMap()

	ontology := xcdf.Format_Ontology()
	err = WriteTextLines(ontology, outputPath+serializationExtension, false)
	checkErr("WriteTextLines("+serializationExtension+")", err)
	err = UploadFile(DataSetPrefix+datasetName+serializationExtension, ontology)
	checkErr("UploadFile: "+DataSetPrefix+datasetName+serializationExtension, err)

	/*sparqlQuery := xcdf.Format_SparqlQuery()
	err = WriteTextLines(sparqlQuery, outputPath+sparqlExtension, false)
	checkErr("WriteTextLines(sparql)", err)
	err = UploadFile(DataSetPrefix+datasetName+sparqlExtension, sparqlQuery)
	checkErr("UploadFile: "+DataSetPrefix+datasetName+sparqlExtension, err)*/

	return xcdf, nil
}

// First reads the *.var file for meta-information and then the *.nc file (in same folder).
func LoadNcSensorDataIntoDatabase(programArgs []string) {
	createSession := CreateIotSession(programArgs)
	xcdf, err := Initialize_IoTDbNcDataFile(createSession, programArgs)
	checkErr("Initialize_IoTDbNcDataFile: ", err)
	err = xcdf.ProcessTimeseries()
	checkErr("ProcessTimeseries(nc)", err)
}

func LoadHd5SensorDataIntoDatabase(programArgs []string) {
	// <<<
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
		LoadCsvSensorDataIntoDatabase(os.Args)
	case ".nc": // this reads the var file too.
		LoadNcSensorDataIntoDatabase(os.Args)
	case ".hd5":
		LoadHd5SensorDataIntoDatabase(os.Args)
	default:
		fmt.Println("Required *.csv parameters: path to csv sensor data file plus any time series command: {drop create delete insert example}.")
		fmt.Println("The csv summary file produced by 'xsv stats <dataFile.csv> --everything' should already exist in the same folder as <dataFile.csv>,")
		fmt.Println("including a description.txt file. All time series data are placed under the database prefix: " + IotDatasetPrefix(""))
		fmt.Println("Required *.nc parameters: 1) path to single *.nc & *.var files, and 2) CDF file type {HDF5, netCDF-4, classic}.")
		fmt.Println("  Optionally specify the unique dataset identifier; e.g. Entity_clean_5min")
		os.Exit(0)
	}
}

///////////////////////////////////////////////////////////////////////////////////////////

const (
	//sarefConst      = "saref:"
	//owlClass        = "owl:Class"
	//subClassOfmatch = "rdfs:subClassOf"
	//etsiOrg         = "` + SarefEtsiOrg + `"
	unknown        = "???"
	zero           = 0.0
	graphDbPrefix  = "PREFIX%20%3A%3Chttp%3A%2F%2Fwww.ontotext.com%2Fgraphdb%2Fsimilarity%2F%3E%0APREFIX%20inst%3A%3Chttp%3A%2F%2Fwww.ontotext.com%2Fgraphdb%2Fsimilarity%2Finstance%2F%3E%0APREFIX%20psi%3A%3Chttp%3A%2F%2Fwww.ontotext.com%2Fgraphdb%2Fsimilarity%2Fpsi%2F%3E%0A"
	graphDbPostfix = "%3E%3B%0Apsi%3AsearchPredicate%20%3Chttp%3A%2F%2Fwww.ontotext.com%2Fgraphdb%2Fsimilarity%2Fpsi%2Fany%3E%3B%0A%3AsearchParameters%20%22-numsearchresults%208%22%3B%0Apsi%3AentityResult%20%3Fresult%20.%0A%3Fresult%20%3Avalue%20%3Fentity%20%3B%0A%3Ascore%20%3Fscore%20.%20%7D%0A"
)

// json returned from GraphDB curl query.
type JsonSimilarity struct {
	Head struct {
		Vars []string `json:"vars"`
	} `json:"head"`
	Results struct {
		Bindings []struct {
			Entity struct {
				XMLLang string `json:"xml:lang,omitempty"` // last element only
				Type    string `json:"type"`
				Value   string `json:"value"`
			} `json:"entity,omitempty"`
			Score struct {
				Datatype string `json:"datatype"`
				Type     string `json:"type"`
				Value    string `json:"value"`
			} `json:"score"`
		} `json:"bindings"`
	} `json:"results"`
}

type Similarity struct { // exported
	IsMarked   bool    `json:"ismarked"`
	Uri        string  `json:"uri"`
	Entity     string  `json:"entity"`
	Match      string  `json:"match"`
	Score      float64 `json:"score"`
	SubClassOf string  `json:"subclassof"` // rdfs:subClassOf : exclude _:node*  Can be [].
	Comment    string  `json:"comment"`    // rdfs:comment
	Label      string  `json:"label"`      // rdfs:label
}

// {Entity:Entity name, Match:Extension name, Score:Entity type} Not all fields displayed.
func (sim Similarity) ToString() string {
	const sep = "\n"
	return "EntityName : " + sim.Entity + sep +
		"Entity URI : " + sim.Uri + sep +
		//"Match      : " + Match + sep +
		"Score      : " + strconv.FormatFloat(sim.Score, 'f', 4, 64) + sep +
		"Comment    : " + sim.Comment + sep +
		"SubClassOf : " + sep + "  " + strings.Join(strings.Split(sim.SubClassOf, " "), sep+"  ")
}

// use as maps
var SimilarEntities = []string{"owl:Class", "owl:NamedIndividual", "owl:ObjectProperty", "owl:DatatypeProperty"}

var OwlElements = []string{"owl:NamedIndividual", "owl:AllDifferent", "owl:allValuesFrom", "owl:AnnotationProperty", "owl:backwardCompatibleWith", "owl:cardinality", "owl:Class", "owl:complementOf",
	"owl:DataRange", "owl:DatatypeProperty", "owl:DeprecatedClass", "owl:DeprecatedProperty", "owl:differentFrom", "owl:disjointWith", "owl:distinctMembers", "owl:equivalentClass",
	"owl:equivalentProperty", "owl:FunctionalProperty", "owl:hasValue", "owl:imports", "owl:incompatibleWith", "owl:intersectionOf", "owl:InverseFunctionalProperty", "owl:inverseOf",
	"owl:maxCardinality", "owl:minCardinality", "owl:Nothing", "owl:ObjectProperty", "owl:oneOf", "owl:onProperty", "owl:Ontology", "owl:OntologyProperty", "owl:priorVersion",
	"owl:Restriction", "owl:sameAs", "owl:someValuesFrom", "owl:SymmetricProperty", "owl:Thing", "owl:TransitiveProperty", "owl:unionOf", "owl:versionInfo"}

// does not include rdf:_1, rdf_2, etc
var RdfElements = []string{"rdf:type", "rdf:Property", "rdf:XMLLiteral", "rdf:langString", "rdf:HTML", "rdf:nil", "rdf:List", "rdf:Statement",
	"rdf:subject", "rdf:predicate", "rdf:object", "rdf:first", "rdf:rest", "rdf:Seq", "rdf:Bag", "rdf:Alt", "rdf:value"}

var RdfsElements = []string{"rdfs:Resource", "rdfs:Class", "rdfs:Literal", "rdfs:Datatype", "rdfs:range", "rdfs:domain", "rdfs:subClassOf",
	"rdfs:subPropertyOf", "rdfs:label", "rdfs:comment", "rdfs:Container", "rdfs:ContainerMembershipProperty", "rdfs:seeAlso", "rdfs:isDefinedBy"}

var SarefMap = map[string]string{ // exported
	"s4agri": SarefEtsiOrg + "saref4agri/",
	"s4auto": SarefEtsiOrg + "saref4auto/",
	"s4bldg": SarefEtsiOrg + "saref4bldg/",
	"s4city": SarefEtsiOrg + "saref4city/",
	"s4ehaw": SarefEtsiOrg + "saref4ehaw/",
	"s4ener": SarefEtsiOrg + "saref4ener/",
	"s4envi": SarefEtsiOrg + "saref4envi/",
	"s4inma": SarefEtsiOrg + "saref4inma/",
	"s4lift": SarefEtsiOrg + "saref4lift/",
	"s4syst": SarefEtsiOrg + "saref4syst/",
	"s4watr": SarefEtsiOrg + "saref4watr/",
	"s4wear": SarefEtsiOrg + "saref4wear/",
	"saref":  SarefEtsiOrg + "core/",
}

var NamespaceMap = map[string]string{
	"cpsv":       "http://purl.org/vocab/cpsv#",
	"dc":         "http://purl.org/dc/elements/1.1/",
	"dcterms":    "http://purl.org/dc/terms/",
	"foaf":       "http://xmlns.com/foaf/0.1/",
	"geo":        "http://www.opengis.net/ont/geosparql#",
	"geosp":      "http://www.opengis.net/ont/geosparql#",
	"owl":        "http://www.w3.org/2002/07/owl#",
	"prov":       "http://www.w3.org/ns/prov#",
	"rdf":        "http://www.w3.org/1999/02/22-rdf-syntax-ns#",
	"rdfs":       "http://www.w3.org/2000/01/rdf-schema#",
	"schema":     "http://schema.org/",
	"sf":         "http://www.opengis.net/ont/sf#",
	"skos":       "http://www.w3.org/2004/02/skos/core#",
	"sosa":       "http://www.w3.org/ns/sosa/",
	"ssn-system": "http://www.w3.org/ns/ssn/systems/",
	"ssn":        "http://www.w3.org/ns/ssn/",
	"time":       "http://www.w3.org/2006/time#",
	"vann":       "http://purl.org/vocab/vann/",
	"voaf":       "http://purl.org/vocommons/voaf#",
	"wgs84":      "http://www.w3.org/2003/01/geo/wgs84_pos#",
	"xml":        "http://www.w3.org/XML/1998/namespace/",
	"xsd":        "http://www.w3.org/2001/XMLSchema#",
}

type GraphDbProgramParameters struct {
	TtlFileDirectory     string  `json:"ttlFileDirectory"`
	SimilarityCutoff     float64 `json:"similarityCutoff"`
	DefaultDbInstanceUrl string  `json:"defaultDbInstanceUrl"`
	DefaultUserMode      string  `json:"defaultUserMode"`
}

func (progParams GraphDbProgramParameters) DisplayProgramStats(stats []string) {
	fmt.Println()
	fmt.Println("Program Parameters::")
	fmt.Println(progParams)
	fmt.Println()
	for i := 0; i < len(stats); i++ {
		fmt.Println(stats[i])
	}
	fmt.Println()
}

const (
	ttlFileDirectory     = HomeDirectory + "Documents/ontologies.ttl/"
	defaultUserMode      = "auto"
	similarityCutoff     = 0.92
	defaultDbInstanceUrl = "http://localhost:7200/repositories/merged" // nmap -p 7200 localhost
)

var graphDBparameters GraphDbProgramParameters

func assignProgramParametersDefaults(err error) {
	fmt.Println(err)
	fmt.Println("Unable to read configuration file ./graphdb.json ... using defaults.")
	graphDBparameters = GraphDbProgramParameters{
		TtlFileDirectory:     ttlFileDirectory,
		DefaultUserMode:      defaultUserMode,
		SimilarityCutoff:     similarityCutoff,
		DefaultDbInstanceUrl: defaultDbInstanceUrl,
	}
	graphDBparameters.DisplayProgramStats([]string{})
}

// Return default GraphDbProgramParameters values if file not found. Expects graphdb package folder is directly beneath main go file.
// Do not use global init() because you can't control when it is run -- could be before or after main().
func assignProgramParameters() {
	file, err := os.ReadFile(HomeDirectory + "Documents/digital-twins/netcdf/graphdb/graphdb.json")
	if err != nil {
		assignProgramParametersDefaults(err)
		return
	}
	err = json.Unmarshal([]byte(file), &graphDBparameters)
	if err != nil {
		assignProgramParametersDefaults(err)
	}
}

// is GraphDB available? (does not test for merged repository)
func Init_GraphDB() (string, bool) {
	assignProgramParameters()
	tokens := strings.Split(graphDBparameters.DefaultDbInstanceUrl, ":")
	host := strings.ReplaceAll(tokens[1], "/", "") // "localhost"
	tokens = strings.Split(tokens[2], "/")
	ports := []string{tokens[0]}
	isOpen, _ := testRemoteAddressPortsOpen(host, ports)
	if !isOpen {
		fmt.Println("GraphDB is not available at " + graphDBparameters.DefaultDbInstanceUrl)
		return graphDBparameters.DefaultDbInstanceUrl, false
	}
	return graphDBparameters.DefaultDbInstanceUrl, true
}

type saveOutput struct {
	savedOutput []byte
}

func (so *saveOutput) Write(p []byte) (n int, err error) {
	so.savedOutput = append(so.savedOutput, p...)
	return os.Stdout.Write(p)
}

// curl -G -H "Accept:application/sparql-results+json" -d query=PREFIX%20%3A%3Chttp%3A%2F%2Fwww.ontotext.com%2Fgraphdb%2Fsimilarity%2F%3E%0APREFIX%20inst%3A%3Chttp%3A%2F%2Fwww.ontotext.com%2Fgraphdb%2Fsimilarity%2Finstance%2F%3E%0APREFIX%20psi%3A%3Chttp%3A%2F%2Fwww.ontotext.com%2Fgraphdb%2Fsimilarity%2Fpsi%2F%3E%0ASELECT%20%3Fentity%20%3Fscore%20%7B%0A%3Fsearch%20a%20inst%3Amerged_sim_ndx%20%3B%0Apsi%3AsearchEntity%20%3Chttps://saref.etsi.org/core/Device%3E%3B%0Apsi%3AsearchPredicate%20%3Chttp%3A%2F%2Fwww.ontotext.com%2Fgraphdb%2Fsimilarity%2Fpsi%2Fany%3E%3B%0A%3AsearchParameters%20%22-numsearchresults%204%22%3B%0Apsi%3AentityResult%20%3Fresult%20.%0A%3Fresult%20%3Avalue%20%3Fentity%20%3B%0A%3Ascore%20%3Fscore%20.%20%7D%0A http://localhost:7200/repositories/merged
// Assumes a predicate-similarity index called 'merged_sim_ndx' exists on the GraphDB repository (see query in line above).
// Uses a numsearchresults value of 8. Does not use score cutoff.  Form URL-encoded query. Return blank string upon error.
// AVOID calling queryGraphDB() within another query -- no inner-loop queries, else get "missing parameter in query; invalid character ...""
func queryGraphDB(query string) (JsonSimilarity, string) {
	var so saveOutput
	cmd := exec.Command("curl", "-G", "-H", "Accept:application/sparql-results+json", "-d", query, graphDBparameters.DefaultDbInstanceUrl)
	cmd.Stdout = &so
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	checkErr("queryGraphDB.1 failed with ", err)
	var jsonSimilarity JsonSimilarity
	err = json.Unmarshal([]byte(so.savedOutput), &jsonSimilarity)
	checkErr("queryGraphDB.2 failed with ", err)
	return jsonSimilarity, graphDBparameters.DefaultDbInstanceUrl
}

func formatFloat(str string) float64 {
	f, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return zero
	}
	return f
}

// Returns empty struct upon error: SELECT * WHERE {<` + SarefEtsiOrg + `saref4wear/CrowdProperty> ?entity ?score .}
// GET o.value FOR p.value: "http://www.w3.org/2000/01/rdf-schema#comment"		(one)
// GET o.value FOR p.value: "http://www.w3.org/2000/01/rdf-schema#subClassOf"	(multiple as comma-separated string)
// GET o.value FOR p.value: "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"	{owl#Class|owl#NamedIndividual|etc}
// GET o.value FOR p.value: "http://www.w3.org/2000/01/rdf-schema#label"		(one)
func fetchEntityProperties(entityUri string) Similarity { // url.QueryEscape() .Replace
	query := "query=" + graphDbPrefix + "SELECT%20%2A%20WHERE%20%7B%3C" + entityUri + "%3E%20%3Fentity%20%3Fscore%20.%7D%20"
	jsonSimilarity, _ := queryGraphDB(query)
	toks := strings.Split(entityUri, "/")
	sim := Similarity{IsMarked: true, Uri: entityUri, Entity: strings.ReplaceAll(toks[4], "]", ""), Match: toks[3], Score: zero, SubClassOf: "", Comment: unknown}
	for ndx := 0; ndx < len(jsonSimilarity.Results.Bindings); ndx++ {
		uri := jsonSimilarity.Results.Bindings[ndx].Entity.Value
		if uri == "http://www.w3.org/2000/01/rdf-schema#label" {
			sim.Label = jsonSimilarity.Results.Bindings[ndx].Score.Value
		}
		if uri == "http://www.w3.org/2000/01/rdf-schema#comment" {
			sim.Comment = jsonSimilarity.Results.Bindings[ndx].Score.Value
		}
		if uri == "http://www.w3.org/1999/02/22-rdf-syntax-ns#type" {
			sim.Score = formatFloat(jsonSimilarity.Results.Bindings[ndx].Score.Value)
		}
		if uri == "http://www.w3.org/2000/01/rdf-schema#subClassOf" && !strings.HasPrefix(jsonSimilarity.Results.Bindings[ndx].Score.Value, "node") {
			sim.SubClassOf = sim.SubClassOf + jsonSimilarity.Results.Bindings[ndx].Score.Value + " "
		}
	}
	return sim
}

// Expect entity to be one of []SimilarEntities.  Include inferred results.

// curl -G -H Accept:application/sparql-results+json -d query=PREFIX%20owl%3A%20%3Chttp%3A%2F%2Fwww.w3.org%2F2002%2F07%2Fowl%23%3E%0ASELECT%20%3Fentity%0AWHERE%20%7B%0A%20%20%20%20%3Fentity%20a%20owl%3ANamedIndividual%20.%20%0A%7D%0Aorder%20by%20%3Fentity http://localhost:7200/repositories/merged
func getEntityList(entity, query string) []Similarity {
	jsonSimilarity, _ := queryGraphDB(query)
	output := make([]Similarity, 0)
	entityUri := unknown
	for ndx := 0; ndx < len(jsonSimilarity.Results.Bindings); ndx++ {
		uri := jsonSimilarity.Results.Bindings[ndx].Entity.Value
		if strings.HasPrefix(uri, SarefEtsiOrg) {
			toks := strings.Split(uri, "/")
			entityName := strings.ReplaceAll(toks[4], "]", "") // don't know why DatatypeProperties has trailing ].
			output = append(output, Similarity{IsMarked: false, Uri: entityUri, Entity: entityName, Match: toks[3], Score: formatFloat(entity), SubClassOf: unknown, Comment: unknown})
		}
	}
	return output
}

// generic array search version
func Find[A any](items []A, predicate func(A) bool) (value A, found bool) {
	for _, v := range items {
		if predicate(v) {
			return v, true
		}
	}
	return
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

// Search list for every embedded string match, but return first token in string.
// Could get the parent class of each and build a tree.
func getEntities(lines []string, match string) []string {
	output := make([]string, 0)
	for _, v := range lines {
		if strings.Contains(v, match) {
			if !strings.Contains(v, "[") {
				tokens := strings.Fields(v)
				output = append(output, tokens[0])
			}
		}
	}
	return output
}

// abort
func checkErr(title string, err error) {
	if err != nil {
		fmt.Print(title + ": ")
		fmt.Println(err)
		log.Fatal(err)
	}
}

// generic version
func concatSlice[T any](first []T, second []T) []T {
	n := len(first)
	return append(first[:n:n], second...)
}

// Sort by descending value.
func formatMap(header string, strIntMap map[string]int) []string {
	const spaces = 30

	keys := make([]string, 0, len(strIntMap))
	for key := range strIntMap {
		keys = append(keys, key)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		return strIntMap[keys[i]] > strIntMap[keys[j]]
	})

	output := make([]string, len(strIntMap)+1)
	output[0] = header
	count := 1
	for _, k := range keys {
		output[count] = fmt.Sprintf("%s%d", k+strings.Repeat(" ", spaces-len(k)), strIntMap[k])
		count++
	}
	return output
}

// Not generic.
func removeDuplicateStr(strSlice []string) []string {
	allKeys := make(map[string]bool)
	list := []string{}
	for _, item := range strSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

///////////////////////////////////////////////////////////////////////////////////////////

// This struct is returned from GetClosestSarefEntity().
type EntityVariableAlias struct {
	EntityName     string             `json:"Entityname"`
	NameTokens     []string           `json:"nametokens"`
	SarefEntityIRI map[string]float64 `json:"sarefentityiri"`
}

func (eva EntityVariableAlias) ParentClassIRI() string {
	for key := range eva.SarefEntityIRI {
		return key
	}
	return ""
}

func (eva EntityVariableAlias) ToString() string {
	const sep = "\n"
	str := "Name  : " + eva.EntityName + sep + "Tokens: " + strings.Join(eva.NameTokens, "  ") + sep + "IRI   : "
	for k, v := range eva.SarefEntityIRI {
		str += k + " : " + strconv.FormatFloat(v, 'f', 4, 64) + sep
	}
	return str
}

// Parse words from variable names: camelCase, underscores, ignore digits. Entity index names use lowercase. Assigns eva.NameTokens.
func (eva *EntityVariableAlias) ParseEntityVariable() {
	re := regexp.MustCompile(`[A-Z][^A-Z]*`)
	tokens := strings.Split(eva.EntityName, "_")
	camels := make([]string, 0)
	for ndx := range tokens {
		for inner := 0; inner <= 9; inner++ {
			sn := strconv.Itoa(inner)
			if strings.HasSuffix(tokens[ndx], sn) {
				tokens[ndx] = strings.Replace(tokens[ndx], sn, "", 1)
			}
		}
		matchCase := re.FindAllString(tokens[ndx], -1) // Split by uppercase; disallow single-letter names.
		for outer := 0; outer < len(matchCase); outer++ {
			if len(matchCase[outer]) > 1 {
				camels = append(camels, matchCase[outer])
			}
		}
	}
	// remove duplicates but retain camelCase vars.
	tokens = append(tokens, camels...)
	tokens = removeDuplicateStr(tokens)
	eva.NameTokens = make([]string, len(tokens))
	copy(eva.NameTokens, tokens) // dst, src
}

// Access merged repository in local GraphDB; use merged_sim_ndx predicate-similarity index.
// Because the repository is simply an aggregate of the 13 SAREF ontologies, have to run the curl query 13 times.
// curl -G -H "Accept:application/sparql-results+json" -d query=PREFIX%20%3A%3Chttp%3A%2F%2Fwww.ontotext.com%2Fgraphdb%2Fsimilarity%2F%3E%0APREFIX%20inst%3A%3Chttp%3A%2F%2Fwww.ontotext.com%2Fgraphdb%2Fsimilarity%2Finstance%2F%3E%0APREFIX%20psi%3A%3Chttp%3A%2F%2Fwww.ontotext.com%2Fgraphdb%2Fsimilarity%2Fpsi%2F%3E%0ASELECT%20%3Fentity%20%3Fscore%20%7B%0A%3Fsearch%20a%20inst%3Amerged_sim_ndx%20%3B%0Apsi%3AsearchEntity%20%3C" + $1 + "%3E%3B%0Apsi%3AsearchPredicate%20%3Chttp%3A%2F%2Fwww.ontotext.com%2Fgraphdb%2Fsimilarity%2Fpsi%2Fany%3E%3B%0A%3AsearchParameters%20%22-numsearchresults%209%22%3B%0Apsi%3AentityResult%20%3Fresult%20.%0A%3Fresult%20%3Avalue%20%3Fentity%20%3B%0A%3Ascore%20%3Fscore%20.%20%7D%0A http://localhost:7200/repositories/merged
func GetClosestEntity(varName string) (EntityVariableAlias, error) {
	eva := EntityVariableAlias{EntityName: varName}
	graphdbURL, isOpen := Init_GraphDB()
	if !isOpen {
		return eva, errors.New("Unable to access GraphDB at " + graphdbURL)
	}
	var sarefExtensions = make([]string, 0)
	for _, value := range SarefMap {
		sarefExtensions = append(sarefExtensions, value)
	}
	eva.ParseEntityVariable() // Assigns eva.NameTokens.
	similarOutputs := make([]Similarity, 0)
	for _, token := range eva.NameTokens {
		for _, saref := range sarefExtensions {
			entityIRI := saref + token
			similars := ExtractSimilarEntities(entityIRI) // []Similarity
			similarOutputs = append(similarOutputs, similars...)
		}
	}
	if len(similarOutputs) > 0 {
		eva.SarefEntityIRI = map[string]float64{similarOutputs[0].Uri: similarOutputs[0].Score}
	} else {
		eva.SarefEntityIRI = map[string]float64{"owl:Thing": 1.0} // default
	}
	return eva, nil
}

///////////////////////////////////////////////////////////////////////////////////////////

// Automatically extract values.
func ExtractSimilarEntities(entityIRI string) []Similarity { // exported
	query := "query=" + graphDbPrefix + "SELECT%20%3Fentity%20%3Fscore%20%7B%0A%3Fsearch%20a%20inst%3Amerged_sim_ndx%20%3B%0Apsi%3AsearchEntity%20%3C" +
		url.QueryEscape(entityIRI) + graphDbPostfix
	jsonSimilarity, _ := queryGraphDB(query)
	tokens := strings.Split(entityIRI, "/")
	entityName := tokens[len(tokens)-1]
	similars := ExtractSimilarityResults(entityName, graphDBparameters.SimilarityCutoff, jsonSimilarity)
	sort.Sort(SimilaritySorter_Score(similars))
	return similars
}

// Extract values greater than similarityCutoff for manual choice. Selection rules go here.
func ExtractSimilarityResults(entityName string, cutOffValue float64, jsonSimilarity JsonSimilarity) []Similarity {
	output := make([]Similarity, 0)
	for ndx := 0; ndx < len(jsonSimilarity.Results.Bindings); ndx++ {
		uri := jsonSimilarity.Results.Bindings[ndx].Entity.Value
		if formatFloat(jsonSimilarity.Results.Bindings[ndx].Score.Value) >= cutOffValue {
			sim := fetchEntityProperties(uri)
			output = append(output, Similarity{
				IsMarked:   false,
				Uri:        uri,
				Entity:     entityName,
				Match:      jsonSimilarity.Results.Bindings[ndx].Entity.Value,
				Score:      formatFloat(jsonSimilarity.Results.Bindings[ndx].Score.Value),
				SubClassOf: sim.SubClassOf,
				Comment:    sim.Comment})
		}
	}
	return output
}

// input saref4agri.ttl; output saref4agri
func getExtensionFromFilename(fileName string) string {
	extension := ""
	if strings.Contains(fileName, "core") {
		return "core"
	}
	toks := strings.Split(path.Base(extension), ".")
	return toks[0]
}

// Fetch the highest similarity values for the Entity. extension is {"core", "saref4agri", "saref4auto", ...}
func ProcessEntities(fileName string, isAuto bool, cutOff float64) []Similarity {
	sourceLines, err := ReadTextLines(fileName, false)
	checkErr("ProcessEntities.1", err)
	output := make([]Similarity, 0)
	extension := getExtensionFromFilename(fileName)
	for _, entity := range SimilarEntities {
		sourceEntities := getEntities(sourceLines, entity)
		for _, entityName := range sourceEntities { // saref:entity
			toks := strings.Split(entityName, ":")
			Entity := SarefEtsiOrg + extension + "/" + toks[1]
			query := graphDbPrefix + Entity + graphDbPostfix
			jsonSimilarity, _ := queryGraphDB(query)
			results := ExtractSimilarityResults(Entity, cutOff, jsonSimilarity)
			output = append(output, results...)
		}
	}
	return output
}

// In-place map update. Ignores comments and text within double quotes! Does not handle multi-line quoted text.
func updateMapElements(line string, strMatches []string, strIntMap map[string]int) {
	if strings.HasPrefix(line, "#") {
		return
	}
	re, _ := regexp.Compile("gm")
	xline := re.ReplaceAllString(line, "")
	for _, element := range strMatches {
		if strings.Contains(xline, " "+element+" ") {
			strIntMap[element]++
		}
	}
}

// Collect rdfs: & owl: stats.
func ProcessOntology(ontologyFile string) []string {
	lines, _ := ReadTextLines(ontologyFile, false)
	owlMap := make(map[string]int)
	rdfMap := make(map[string]int)
	rdfsMap := make(map[string]int)
	for _, line := range lines {
		updateMapElements(line, OwlElements, owlMap)
		updateMapElements(line, RdfElements, rdfMap)
		updateMapElements(line, RdfsElements, rdfsMap)
	}

	o1 := formatMap("    OWL Statement Count for "+ontologyFile, owlMap)
	r1 := formatMap("    RDF Statement Count::", rdfMap)
	r2 := formatMap("    RDFS Statement Count::", rdfsMap)
	return concatSlice(o1, concatSlice(r1, r2))
}

type SimilaritySorter_Entity []Similarity       // Case-insensitive sort of Entity name.
func (a SimilaritySorter_Entity) Len() int      { return len(a) }
func (a SimilaritySorter_Entity) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SimilaritySorter_Entity) Less(i, j int) bool {
	return strings.Compare(strings.ToLower(a[i].Entity), strings.ToLower(a[j].Entity)) < 0
}

type SimilaritySorter_Score []Similarity       // Descending sort by Score
func (a SimilaritySorter_Score) Len() int      { return len(a) }
func (a SimilaritySorter_Score) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SimilaritySorter_Score) Less(i, j int) bool {
	return a[i].Score > a[j].Score
}

// case-insensitive version; return subset of []list.
func getEmbeddedMatches(sim Similarity, list []Similarity) []Similarity {
	subList := make([]Similarity, 0)
	for ndx := range list {
		found := list[ndx].Entity == "has"+sim.Entity || list[ndx].Entity == "is"+sim.Entity
		if found && list[ndx].Entity != sim.Entity {
			subList = append(subList, sim) // will insert duplicates
			subList = append(subList, list[ndx])
		}
	}
	return subList
}

// case-insensitive version.
func isEntityNameMatch(sim1, sim2 Similarity) bool {
	return strings.EqualFold(sim1.Entity, sim2.Entity)
}

// Read merged-ontology.ttl and show duplicate entities based upon case-sensitive Entity names.
// ?? Get single class: SELECT * WHERE {<` + SarefEtsiOrg + `saref4wear/CrowdProperty> ?entity ?score .}
// ?? SELECT%20%2A%20WHERE%20%7B%3Chttps%3A%2F%2Fsaref.etsi.org%2Fsaref4wear%2FCrowdProperty%3E%20%3Fentity%20%3Fscore%20.%7D%20
func DisplayDuplicateEntities() {
	fmt.Println("\nTo aid in identifying modelling discrepancies, this is the list of same-named SAREF Entities: ")
	output := make([]Similarity, 0)
	for _, entityType := range SimilarEntities {
		query := "query=PREFIX%20owl%3A%20%3Chttp%3A%2F%2Fwww.w3.org%2F2002%2F07%2Fowl%23%3E%0ASELECT%20%3Fentity%0AWHERE%20%7B%0A%20%20%20%20%3Fentity%20a%20" +
			strings.ReplaceAll(entityType, ":", "%3A") + "%20.%20%0A%7D%0Aorder%20by%20%3Fentity"
		eList := getEntityList(entityType, query)
		output = append(output, eList...)
	}

	// identify duplicates:
	sort.Sort(SimilaritySorter_Entity(output)) // in-place sort; case-sensitive so does NOT catch Frequency && frequency.
	for ndx := range output {
		if ndx > 0 && isEntityNameMatch(output[ndx], output[ndx-1]) {
			output[ndx-1].IsMarked = true
			output[ndx].IsMarked = true
		}
	}

	// insert embedded(substring) matches into separate list
	embedded := make([]Similarity, 0)
	for ndx := range output {
		sList := getEmbeddedMatches(output[ndx], output)
		embedded = append(embedded, sList...)
	}
	embedded = removeDuplicateSimilarityValues(embedded)

	// now fetchEntityProperties and output results.
	for ndx := range output {
		if output[ndx].IsMarked {
			output[ndx] = fetchEntityProperties(output[ndx].Uri)
			fmt.Println(output[ndx].ToString())
			if ndx > 0 && ndx < len(output) && isEntityNameMatch(output[ndx], output[ndx-1]) && !isEntityNameMatch(output[ndx], output[ndx+1]) {
				fmt.Println()
			}
		}
	}
	fmt.Println("\nEmbedded matches: ")
	for ndx := range embedded {
		fmt.Println(embedded[ndx].ToString())
	}
}

///////////////////////////////////////////////////////////////////////////////////////////

type JsonClassSuperClass struct {
	Head struct {
		Vars []string `json:"vars"`
	} `json:"head"`
	Results struct {
		Bindings []struct {
			Entity struct {
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"entity,omitempty"`
			SuperClass struct {
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"superclass"`
		} `json:"bindings"`
	} `json:"results"`
}

type ClassSuperClass struct { // all shortened names
	Entity     string   `json:"entity"`
	SuperClass []string `json:"superclass"`
}

func queryGraphSuperClass(query string) JsonClassSuperClass {
	var so saveOutput
	cmd := exec.Command("curl", "-G", "-H", "Accept:application/sparql-results+json", "-d", query, graphDBparameters.DefaultDbInstanceUrl)
	cmd.Stdout = &so
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	checkErr("queryGraphSuperClass.1 failed with ", err)
	var jsonClassSuperClass JsonClassSuperClass
	err = json.Unmarshal([]byte(so.savedOutput), &jsonClassSuperClass)
	if err != nil {
		fmt.Printf("\nerror decoding response: %v", err)
		if e, ok := err.(*json.SyntaxError); ok {
			fmt.Printf("\nsyntax error at byte offset %d", e.Offset)
		}
	}
	checkErr("\nqueryGraphSuperClass.2 failed with ", err)
	return jsonClassSuperClass
}

// Uses GraphDB SPARQL query GetClassByState which gets all classes and their superclasses. FIlter out non-SAREF entities. The query is ordered by entityName.
func GetClassByState() []ClassSuperClass {
	classSuperClass := make([]ClassSuperClass, 0)
	query := "PREFIX%20owl%3A%20%3Chttp%3A%2F%2Fwww.w3.org%2F2002%2F07%2Fowl%23%3E%0APREFIX%20rdfs%3A%20%3Chttp%3A%2F%2Fwww.w3.org%2F2000%2F01%2Frdf-schema%23%3E%0ASELECT%20DISTINCT%20%3Fentity%20%3Fsuperclass%0AWHERE%20%7B%0A%20%20%20%20%7B%20%3Fentity%20a%20owl%3AClass%20.%20%0A%20%20%20%20FILTER%20%28isURI%28%3Fentity%29%29%20%0A%20%20%20%20FILTER%20%28isURI%28%3Fsuperclass%29%29%20%0A%20%20%20%20%7D%20%0A%20%20%20%20UNION%20%0A%20%20%20%20%7B%20%3Findividual%20a%20%3Fentity%20.%20FILTER%20%28isURI%28%3Fentity%29%29%20FILTER%20%28isURI%28%3Fsuperclass%29%29%20%7D%20.%0A%20%20%20%20OPTIONAL%20%7B%20%3Fentity%20rdfs%3AsubClassOf%20%3Fsuperclass%20%7D%20.%0A%7D%20ORDER%20BY%20%3Fentity"
	jsonClassSuperClass := queryGraphSuperClass(query)
	var oldEntityName string
	////superClasses := make([]string, 0)
	for ndx := 0; ndx < len(jsonClassSuperClass.Results.Bindings); ndx++ {
		baseClassURI := jsonClassSuperClass.Results.Bindings[ndx].Entity.Value
		superClassURI := jsonClassSuperClass.Results.Bindings[ndx].SuperClass.Value
		if strings.HasPrefix(baseClassURI, SarefEtsiOrg) {
			baseClassTokens := strings.Split(baseClassURI, "/")
			superClassTokens := strings.Split(superClassURI, "/")
			baseClassEntity := baseClassTokens[3] + ":" + baseClassTokens[4]
			superClassEntity := superClassTokens[3] + ":" + superClassTokens[4]
			if baseClassEntity == oldEntityName && baseClassEntity != superClassEntity { // add superclass except if self
				//superClasses = append(superClasses, superClassEntity)
				fmt.Println(baseClassEntity + "  ::  " + superClassEntity)
			} else {
				fmt.Println()
				//classSuperClass = append(classSuperClass, ClassSuperClass{Entity: entityName, SuperClass: toks[3]}) // <<<
			}
			oldEntityName = baseClassEntity
		}
	}
	return classSuperClass
}

func removeIndex(s []Similarity, index int) []Similarity {
	return append(s[:index], s[index+1:]...)
}

func removeDuplicateSimilarityValues(simSlice []Similarity) []Similarity {
	keys := make(map[Similarity]bool)
	simList := []Similarity{}
	for _, entry := range simSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			simList = append(simList, entry)
		}
	}
	return simList
}

// allow manual choice here
func ChooseEntitiesToChange(similarEnities []Similarity) []Similarity {
	fmt.Println("Enter the number of the line to make that change in the ontology.")
	changes := make([]Similarity, 0)
	var choice = -1
	for choice != 0 {
		fmt.Println("\nEntity in Source Ontology            Best Match Found in Graph Database              Similarity score")
		fmt.Println("0) Quit; don't make any more changes.")
		for ndx := range similarEnities {
			fmt.Print(ndx + 1)
			fmt.Print(") ")
			fmt.Println(similarEnities[ndx])
		}
		fmt.Print("Enter your choice: ")
		_, err := fmt.Scanf("%d", &choice)
		if err != nil {
			choice = 0
		}
		// move element from []similarEnities to []changes
		if choice >= 1 && choice <= len(similarEnities) {
			changes = append(changes, similarEnities[choice-1])
			similarEnities = removeIndex(similarEnities, choice-1)
		}
	}
	return changes
}

// /////////////////////////////////////////////////////////////////////////////////////////
// ISO 8601 format: use package iso8601 since The built-in RFC3333 time layout in Go is too restrictive to support any ISO8601 date-time.
const (
	summaryPrefix  = "summary_"
	OutputLocation = SarefEtsiOrg + "datasets/examples/" //<<< wrong
	TimeFormat     = "2006-01-02T15:04:05Z"              // yyyy-MM-ddThh:mm:ssZ UTC RFC3339 format. Do not save timezone.
	TimeFormat1    = "2006-01-02 15:04:05"
	TimeFormatNano = "2006-01-02T15:04:05.000Z07:00" // this is the preferred milliseconds version.
	LastColumnName = "DatasetName"
)

//var formattedUnits = []string{"longtime", "yyyy-MM-ddThh:mm:ssZ", "unicode", "unixutc", meters/hour", "knots", "percent" } "unitless"=>"unicode"

// Return a NamedIndividual unit_of_measure from an abreviated key. All of these are specified at the uomPrefix URI.
// This expects you to manually add the map keys to the (last) Units column header in every *.var file.
func GetNamedIndividualUnitMeasure(uom string) string {
	const uomPrefix = "http://www.ontology-of-units-of-measure.org/resource/om-2/"
	var unitsOfMeasure = map[string]string{
		"kW":     uomPrefix + "kilowatt",
		"kWh":    uomPrefix + "kilowattHour",
		"pascal": uomPrefix + "pascal",
		"kelvin": uomPrefix + "kelvin",
		"°C":     uomPrefix + "degreeCelsius",
		"°F":     uomPrefix + "degreeFahrenheit",
		"%rh":    uomPrefix + "PercentRelativeHumidity",
		"mb":     uomPrefix + "millibar",
		"degree": uomPrefix + "degree",
		"lux":    uomPrefix + "lux",
		"km":     uomPrefix + "kilometre"}

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

var timeSeriesCommands = []string{"example", "drop", "create", "delete", "insert", "query"}
var iotdbParameters IoTDbProgramParameters
var clientConfig *client.Config

// select this column of values from summary file as instance values.
var summaryColumnNames = []string{"field", "type", "sum", "min", "max", "min_length", "max_length", "mean", "stddev", "median", "mode", "cardinality", "units"} // , LastColumnName

// All ports must be open.
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

// For dates different more than 1 day apart.
func TimeDifference(a, b time.Time) (year, month, day, hour, min, sec int) {
	if a.Location() != b.Location() {
		b = b.In(a.Location())
	}
	if a.After(b) {
		a, b = b, a
	}
	y1, M1, d1 := a.Date()
	y2, M2, d2 := b.Date()

	h1, m1, s1 := a.Clock()
	h2, m2, s2 := b.Clock()

	year = int(y2 - y1)
	month = int(M2 - M1)
	day = int(d2 - d1)
	hour = int(h2 - h1)
	min = int(m2 - m1)
	sec = int(s2 - s1)

	// Normalize negative values
	if sec < 0 {
		sec += 60
		min--
	}
	if min < 0 {
		min += 60
		hour--
	}
	if hour < 0 {
		hour += 24
		day--
	}
	if day < 0 {
		// days in month:
		t := time.Date(y1, M1, 32, 0, 0, 0, 0, time.UTC)
		day += 32 - t.Day()
		month--
	}
	if month < 0 {
		month += 12
		year--
	}
	return
}

// Replace embedded quote marks with backticks; return enclosed string.
// Could format floats to constant width,precision. This function is called millions of times.
func formatDataItem(s, dataType string) string {
	if dataType == "string" {
		ss := strings.ReplaceAll(strings.ReplaceAll(s, "\"", "`"), "'", "`")
		return "'" + ss + "'"
	} else {
		if len(s) == 0 {
			return "null"
		} else {
			return s
		}
	}
}

// Return quoted name and its alias. Alias: open brackets are replaced with underscore.
// Does not handle 2 aliases being the same. Return MeasurementName, MeasurementAlias.
// Some variable names must be aliased in SQL statements: {time, }
func StandardName(oldName string) (string, string) {
	//newName := strings.TrimSpace(cases.Title(language.Und, cases.NoLower).String(oldName)) // uppercase first letter
	newName := strings.TrimSpace(oldName)
	replacer := strings.NewReplacer("~", "", "!", "", "@", "", "#", "", "$", "", "%", "", "^", "", "&", "", "*", "", "/", "", "?", "", ".", "", ",", "", ":", "", ";", "", "|", "", "\\", "", "=", "", "+", "", ")", "", "}", "", "]", "", "(", "_", "{", "_", "[", "_")
	alias := strings.ReplaceAll(newName, " ", "")
	alias = replacer.Replace(alias)

	if strings.ToLower(newName) == "time" {
		return newName, timeAlias
	}

	if newName != alias {
		return alias, newName
	} else {
		return newName, alias
	}
}

// someTime can be either a long or a readable dateTime string.
func getStartTimeFromLongint(someTime string) (time.Time, error) {
	isLong, err := strconv.ParseInt(someTime, 10, 64)
	if err == nil {
		return time.Unix(isLong, 0), err
	}
	t, err := time.Parse(TimeFormat, someTime)
	if err != nil {
		return time.Unix(isLong, 0), err
	}
	return t, nil
}

// Process: 2017-01-18 10:40:00
func getStartTimeFromString(someTime string) (time.Time, error) {
	t, err := time.Parse(TimeFormat1, someTime)
	if err != nil {
		return t, err
	}
	return t, nil
}

// Return IotDB datatype; encoding; compression. See https://iotdb.apache.org/UserGuide/V1.0.x/Data-Concept/Encoding.html#encoding-methods
// (client.TSDataType, client.TSEncoding, client.TSCompressionType)  Make all compression SNAPPY
func getClientStorage(dataColumnType string) (string, string, string) {
	sw := strings.ToLower(dataColumnType)
	switch sw {
	case "decimal", "double":
		return "DOUBLE", "GORILLA", "SNAPPY"
	case "unicode", "string":
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

func StandardDate(dt time.Time) string {
	var a [20]byte
	var b = a[:0]                        // Using the a[:0] notation converts the fixed-size array to a slice type represented by b that is backed by this array.
	b = dt.AppendFormat(b, time.RFC3339) // AppendFormat() accepts type []byte. The allocated memory a is passed to AppendFormat().
	return string(b[0:10])
}

func GetUniqueInstanceID() string {
	dateTime := StandardDate(time.Now())
	dateTime = strings.ReplaceAll(dateTime, "-", "")
	return `_` + dateTime
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

type IoTDbCsvDataFile struct {
	IoTDbAccess
	Description        string                     `json:"description"`
	DataFilePath       string                     `json:"datafilepath"`
	DataFileType       string                     `json:"datafiletype"`
	SummaryFilePath    string                     `json:"summaryfilepath"`
	DatasetName        string                     `json:"datasetname"`
	Measurements       map[string]MeasurementItem `json:"measurements"`       //<<< make pointer! one less than the number of rows in summary file.
	TimeseriesCommands []string                   `json:"timeseriescommands"` // command-line parameters
	Summary            [][]string                 `json:"summary"`            // from summary file
	Dataset            [][]string                 `json:"dataset"`            // actual data
}

// expect only 1 instance of 'datasetName' in iotdbDataFile.DataFilePath.
func Initialize_IoTDbCsvDataFile(isActive bool, programArgs []string) (IoTDbCsvDataFile, error) {
	datasetPathName := filepath.Dir(programArgs[1]) // does not include trailing slash
	datasetName := filepath.Base(programArgs[1])    // includes extension
	datasetName = strings.Replace(datasetName, csvExtension, "", 1)
	description, _ := ReadTextLines(datasetPathName+timeseriesDescription, false)
	isAccessibleSensorDataFile(programArgs[1])
	ioTDbAccess := IoTDbAccess{ActiveSession: isActive}
	iotdbDataFile := IoTDbCsvDataFile{IoTDbAccess: ioTDbAccess, Description: strings.Join(description, " "), DataFilePath: programArgs[1], DatasetName: datasetName}
	iotdbDataFile.TimeseriesCommands = GetTimeseriesCommands(programArgs)
	iotdbDataFile.SummaryFilePath = datasetPathName + "/" + summaryPrefix + filepath.Base(programArgs[1])
	err := iotdbDataFile.ReadCsvFile(iotdbDataFile.SummaryFilePath, false) // isDataset: no, is summary
	checkErr("ReadCsvFile ", err)
	iotdbDataFile.XsvSummaryTypeMap()
	//fmt.Println(iotdbDataFile.OutputDescription(true))
	//<<< read sensor data from IotDB if available, else CSV file.
	err = iotdbDataFile.ReadCsvFile(iotdbDataFile.DataFilePath, true) // isDataset: yes
	checkErr("ReadCsvFile ", err)
	return iotdbDataFile, nil
}

// Include percentage of missing data, which can be got from a SELECT count(*) from root.datasets.etsi.household_data_1min_singleindex;
func (iot *IoTDbCsvDataFile) OutputDescription(displayColumnInfo bool) string {
	var sb strings.Builder
	sb.WriteString(crlf)
	sb.WriteString("Description     : " + iot.Description + crlf)
	sb.WriteString("Data Set Name   : " + iot.DatasetName + crlf)
	sb.WriteString("Data Path       : " + iot.DataFilePath + crlf)
	sb.WriteString("n Data Types    : " + strconv.Itoa(len(iot.Measurements)) + crlf)
	sb.WriteString("n Measurements  : " + strconv.Itoa(len(iot.Dataset)-1) + crlf)
	//sb.WriteString("Data start time : " + iot.Dataset[1][0] + crlf)
	//sb.WriteString("Data end time   : " + iot.Dataset[len(iot.Dataset)-1][0] + crlf)
	if displayColumnInfo {
		for ndx := 0; ndx < len(iot.Measurements); ndx++ {
			for _, item := range iot.Measurements {
				if item.ColumnOrder == ndx && !item.Ignore {
					sb.WriteString("  " + item.ToString() + crlf)
				}
			}
		}
	}
	return sb.String()
}

// search for either original or alias name
func (iot *IoTDbCsvDataFile) GetMeasurementItemFromName(name string) (MeasurementItem, bool) {
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

// Return all column values except header row. Return false if wrong column name. Fill in default values.
func (iot *IoTDbCsvDataFile) GetSummaryStatValues(columnName string) ([]string, bool) {
	_, columnIndex := find(summaryColumnNames, columnName)
	if columnIndex < 0 {
		return []string{}, false
	}
	stats := make([]string, len(iot.Measurements))
	for ndx := 0; ndx < len(iot.Measurements); ndx++ {
		dataColumnName, _ := StandardName(iot.Summary[ndx+1][0]) // aliasName
		item, _ := iot.GetMeasurementItemFromName(dataColumnName)
		stats[ndx] = strings.TrimSpace(iot.Summary[ndx+1][columnIndex])
		if strings.HasPrefix(strings.ToLower(item.MeasurementType), "int") {
			index := strings.Index(stats[ndx], ".")
			if index >= 0 {
				stats[ndx] = stats[ndx][0:index]
			}
		}
		if len(stats[ndx]) == 0 {
			stats[ndx] = strings.TrimSpace(iot.Summary[ndx+1][1]) // this type is nearly always Unicode
		}
	}
	return stats, true
}

// Expects {Units, DatasetName} fields to have been appended to the summary file. Assign []Measurements. Expects Summary to be assigned. Use XSD data types.
// len() only returns the length of the "external" array.
func (iot *IoTDbCsvDataFile) XsvSummaryTypeMap() {
	rowsXsdMap := map[string]string{"Unicode": "string", "Float": "float", "Integer": "integer", "Longint": "longint", "Double": "double"}
	iot.Measurements = make(map[string]MeasurementItem, 0)
	// get units column
	unitsColumn := 0
	for ndx := 0; ndx < len(iot.Summary[0]); ndx++ { // iterate over summary header row
		if strings.ToLower(iot.Summary[0][ndx]) == "units" {
			unitsColumn = ndx
			break
		}
	}
	ndx1 := 0
	for ndx := 0; ndx < 256; ndx++ { // iterate over summary file rows.
		if iot.Summary[ndx+1][0] == "interpolated" || len(iot.Summary[ndx+1][0]) < 2 {
			ndx1 = ndx
			break
		}
		dataColumnName, aliasName := StandardName(iot.Summary[ndx+1][0]) // dataColumnName is normalized for IotDB naming conventions.
		ignore := iot.Summary[ndx+1][0] == "time" || iot.Summary[ndx+1][2] == "0" && iot.Summary[ndx+1][2] == "0" && iot.Summary[ndx+1][4] == "0"
		if ignore {
			fmt.Println("Ignoring empty data column " + dataColumnName)
		}
		theEnd := dataColumnName == "interpolated"
		if !theEnd {
			mi := MeasurementItem{
				MeasurementName:  aliasName, // the CSV field names are often unusable.
				MeasurementAlias: dataColumnName,
				MeasurementType:  rowsXsdMap[iot.Summary[ndx+1][1]],
				MeasurementUnits: iot.Summary[ndx+1][unitsColumn],
				ColumnOrder:      ndx,
				Ignore:           ignore,
			}
			iot.Measurements[dataColumnName] = mi // add to map using original name
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
	iot.Measurements[LastColumnName] = mi
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

// Expects comma-separated files. Assigns Dataset or Summary. REFACTOR
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

// Produce DatatypeProperty ontology from summary & dataset. Write to filesystem, then upload to website.
// Make the dataset its own Class and loadable into GraphDB. Special handling: "dateTime", "XMLLiteral", "anyURI"
func (iot *IoTDbCsvDataFile) Format_Ontology() []string {
	baseline := getBaselineOntology(iot.DatasetName, iot.DatasetName, iot.Description)
	output := strings.Split(baseline, crlf)
	output = append(output, `### specific time series DatatypeProperties`)
	ndx := 0
	for _, v := range iot.Measurements {
		ndx++
		str := `[ rdf:type owl:Restriction ;` + crlf + `owl:onProperty ` + s4data + `has` + v.MeasurementName + ` ;` + crlf + `owl:allValuesFrom xsd:` + xsdDatatypeMap[strings.ToLower(v.MeasurementType)] + crlf
		if ndx < len(iot.Measurements) {
			str += `] ,`
		} else {
			str += `] ;`
		}
		if !v.Ignore {
			output = append(output, str)
		}
	}
	output = append(output, ` rdfs:comment "`+iot.Description+`"@en ;`)
	output = append(output, ` rdfs:label "`+iot.DatasetName+`"@en .`+crlf)
	output = append(output, `### externel references`+crlf)
	externStr := getExternalReferences()
	output = append(output, strings.Split(externStr, crlf)...)

	// new common Classes: StartTimeseries, StopTimeseries
	uniqueID := GetUniqueInstanceID()
	output = append(output, `### class instances`+crlf)
	output = append(output, `ex:StartTimeseries`+uniqueID)
	output = append(output, `rdf:type `+s4data+`StartTimeseries ;`)
	output = append(output, `rdf:label "StartTimeseries`+uniqueID+`"^^xsd:string .`+crlf)
	output = append(output, `ex:StopTimeseries`+uniqueID)
	output = append(output, `rdf:type `+s4data+`StopTimeseries ;`)
	output = append(output, `rdf:label "StopTimeseries`+uniqueID+`"^^xsd:string .`+crlf)
	output = append(output, `### internel references`+crlf)

	values, ok := iot.GetSummaryStatValues("mean")
	output = append(output, `<`+DataSetPrefix+iot.DatasetName+uniqueID+`> rdf:type owl:NamedIndividual , `)
	output = append(output, `  saref:Measurement , saref:Time , saref:UnitOfMeasure , s4envi:FrequencyUnit , s4envi:FrequencyMeasurement ;`+crlf)
	if ok {
		for ndx := 0; ndx < len(iot.Measurements); ndx++ {
			for _, v := range iot.Measurements {
				if v.ColumnOrder == ndx && !v.Ignore {
					str := s4data + v.MeasurementName + `  "` + values[ndx] + `"^^xsd:` + xsdDatatypeMap[strings.ToLower(v.MeasurementType)]
					if v.ColumnOrder < len(iot.Measurements)-1 {
						str += ` ;`
					} else {
						str += ` .`
					}
					output = append(output, str)
				}
			}
		}
	}
	return output
}

// Command-line parameters: {drop create delete insert ...}. Always output dataset description.
// create time series root.datasets.etsi.household_data_60min_singleindex.DE_KN_industrial1_grid_import with datatype=FLOAT, encoding=GORILLA, compressor=SNAPPY;
// ProcessTimeseries() is the only place where iot.IoTDbAccess.session is instantiated and clientConfig is used.
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
		case "drop": // time series schema; uses single statement;
			sql := "DROP TIMESERIES " + IotDatasetPrefix("") + iot.DatasetName + ".*"
			_, err := iot.IoTDbAccess.session.ExecuteNonQueryStatement(sql)
			checkErr("ExecuteNonQueryStatement(dropStatement)", err)
			for k := range iot.Measurements { // does not account for Ignore items.
				delete(iot.Measurements, k)
			}

		case "create":
			// create aligned time series schema; single statement: CREATE ALIGNED TIMESERIES root.etsidata.device.household_data_1min_singleindex (utc_timestamp TEXT encoding=PLAIN compressor=SNAPPY,  etc);
			// Note: For a group of aligned timeseries, Iotdb does not support different compressions.
			// https://iotdb.apache.org/UserGuide/V1.0.x/Reference/SQL-Reference.html#schema-statement
			var sb strings.Builder
			sb.WriteString("CREATE ALIGNED TIMESERIES " + IotDatasetPrefix("") + iot.DatasetName + "(")
			for _, item := range iot.Measurements {
				if !item.Ignore {
					dataType, encoding, compressor := getClientStorage(item.MeasurementType)
					sb.WriteString(item.MeasurementAlias + " " + dataType + " encoding=" + encoding + " compressor=" + compressor + ",")
				}
			}
			sql := sb.String()[0:len(sb.String())-1] + ");" // replace trailing comma
			_, err := iot.IoTDbAccess.session.ExecuteNonQueryStatement(sql)
			checkErr("ExecuteNonQueryStatement(createStatement)", err)

		case "delete": // remove all data; retain schema; multiple commands.
			deleteStatements := make([]string, 0)
			for _, item := range iot.Measurements {
				deleteStatements = append(deleteStatements, "DELETE FROM "+IotDatasetPrefix("")+iot.DatasetName+"."+item.MeasurementName+";")
			}
			_, err := iot.IoTDbAccess.session.ExecuteBatchStatement(deleteStatements) // (r *common.TSStatus, err error)
			checkErr("ExecuteBatchStatement(deleteStatements)", err)

		case "insert": // insert(append) data; retain schema; either single or multiple statements;
			// Automatically inserts long time column as first column (which should be UTC). Save in blocks.
			timeIndex := 0
			nBlocks := len(iot.Dataset)/(blockSize) + 1
			fmt.Printf("%s%d%s", "Writing ", nBlocks, " blocks: ")
			for block := 0; block < nBlocks; block++ {
				fmt.Print(block + 1)
				fmt.Print(" ") //fmt.Printf("%s%d-%d\n", "block: ", startRow, endRow-1)
				var sb strings.Builder
				var insert strings.Builder
				insert.WriteString("INSERT INTO " + IotDatasetPrefix("") + iot.DatasetName + " (time," + iot.FormattedColumnNames() + ") ALIGNED VALUES ")
				startRow := blockSize*block + 1
				endRow := startRow + blockSize
				if block == nBlocks-1 {
					endRow = len(iot.Dataset)
				}
				for r := startRow; r < endRow; r++ {
					sb.Reset()
					startTime, err := getStartTimeFromLongint(iot.Dataset[r][timeIndex])
					if err != nil {
						fmt.Println(iot.Dataset[r][timeIndex])
						break
					}
					sb.WriteString("(" + strconv.FormatInt(startTime.UTC().Unix(), 10) + ",")
					for c := 0; c < len(iot.Dataset[r])-1; c++ { // skip DatasetName; <<< use Ignore.
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
				//WriteStringToFile(HomeDirectory+"Downloads/block"+strconv.Itoa(block)+".txt", insert.String())
			}
			fmt.Println()

		case "query":
			iot.GetTimeseriesList(iot.DatasetName)
			outputPath := GetOutputPath(iot.DataFilePath, ".sql")
			err := WriteTextLines(iot.QueryResults, outputPath, false)
			checkErr("WriteTextLines(query)", err)

		case "example": // serialize saref class file:
			ttlLines := iot.Format_Ontology()
			outputPath := GetOutputPath(iot.DataFilePath, serializationExtension)
			err := WriteTextLines(ttlLines, outputPath, false)
			checkErr("WriteTextLines(example)", err)

		}
		fmt.Println("Timeseries <" + command + "> completed.")
	} // for

	return nil
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
