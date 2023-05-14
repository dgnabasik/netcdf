package main

// Produce json file, DatatypeProperty ontology file, and SPARQL query files file from ncdump outputs.
// Use Named Graphs (Identifier+Title+DatastreamName) as Publish/Subscribe topics.
// /usr/bin/ncdump -k cdf.nc			==> get file type {classic, netCDF-4, others...}
// /usr/bin/ncdump -c Jan_clean.nc		==> gives header + indexed {id, time} data
// https://docs.unidata.ucar.edu/nug/current/index.html		Golang supports compressed file formats {rardecode(rar), gz/gzip, zlib, lzw, bzip2}
// Curated data: Each data row is indexed by {a house ID, a time value}. How to import the data into GraphDB (as a Named Graph)?
// nc files: ./github.com/go-native-netcdf/netcdf/*.nc	./github.com/netcdf-c/*.nc	./Documents/digital-twins/Entity/*.nc
// h5 files: ./Documents/digital-twins/AMP/AMPds2.h5	./github.com/netcdf-c/nc_test4/*.h5	./github.com/nci-doe-data-sharing/flaskProject/mt-cnn/mt_cnn_model.h5 ./github.com/go-native-netcdf/netcdf/hdf5/testdata
// csv files: many.
import (
	//"bytes"
	//"context"
	//"database/sql/driver"
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
	fs "github.com/dgnabasik/acmsearchlib/filesystem"
	//"github.com/dgnabasik/netcdf/graphdb"
	//"github.com/dgnabasik/netcdf/iotdb"
)

const ( // these do not include trailing >
	SarefEtsiOrg  = "<https://saref.etsi.org/"
	DataSetPrefix = SarefEtsiOrg + "datasets/examples/"
	IotDataPrefix = "root.datasets.etsi."
)

var NetcdfFileFormats = []string{"classic", "netCDF", "netCDF-4", "HDF5"}

// Contains an entire month's worth of Entity data for every Variable where each row is indexed by {HouseIndex+LongtimeIndex}.
type EntityCleanData struct {
	HouseIndex    []string   `json:"houseindex"`
	LongtimeIndex []string   `json:"longtimeindex"`
	Data          [][]string `json:"data"`
}

// EntityCleanData initializer.
func MakeEntityCleanData(rows int, vars []CDFvariable) EntityCleanData {
	ecd := EntityCleanData{}
	cols := len(vars)
	ecd.Data = make([][]string, rows) // initialize a slice of rows slices
	for i := 0; i < rows; i++ {
		ecd.Data[i] = make([]string, cols) // initialize a slice of cols in each of rows slices
	}
	return ecd
}

///////////////////////////////////////////////////////////////////////////////////////////

// Expects the parameters to every Variable to be all the (2) dimensions (except for the dimension variables).
type CDFvariable struct {
	StandardName   string `json:"standardname"`
	LongName       string `json:"longname"`
	Units          string `json:"units"`
	ReturnType     string `json:"returntype"`
	DimensionIndex int    `json:"dimensionindex"` // {0,1,2} Default 0 signifies the Variable is not a Dimension.
	FillValue      string `json:"fillvalue"`
	Comment        string `json:"comment"`
	Calendar       string `json:"calendar"`
}

// Generic NetCDF container. Not stored in IotDB.
type NetCDF struct {
	Identifier       string         `json:"identifier"` // Unique ID to distinguish among different datasets; use as Named Graph.
	NetcdfType       string         `json:"netcdftype"` // /usr/bin/ncdump -k cdf.nc ==> NetcdfFileFormats
	Dimensions       map[string]int `json:"dimensions"`
	Title            string         `json:"title"`
	Description      string         `json:"description"`
	Conventions      string         `json:"conventions"`
	Institution      string         `json:"institution"`
	Code_url         string         `json:"code_url"`
	Location_meaning string         `json:"location_meaning"`
	Datastream_name  string         `json:"datastream_name"`
	Input_files      string         `json:"input_files"`
	History          string         `json:"history"`
	Variables        []CDFvariable  `json:"variables"`
	HouseIndices     []string       `json:"houseindices"`    // these unique 2 indices are specific to Entity datasets.
	LongtimeIndices  []string       `json:"longtimeindices"` // Different months will have slightly different HouseIndices!
}

// Some variable names must be aliased in SQL statements: {time, }
func (cdf NetCDF) StandardNameAlias(standardName string) string {
	if standardName == "time" {
		return "longtime"
	}
	return standardName
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
	sb.WriteString("Variables      : ")
	sb.WriteString(strconv.Itoa(len(cdf.Variables)) + sep)

	if outputVariables {
		for _, tv := range cdf.Variables {
			sb.WriteString(" StandardName: " + tv.StandardName + "; ReturnType: " + tv.ReturnType + "; ")
			if len(tv.LongName) > 0 {
				sb.WriteString(" LongName: " + tv.LongName + ";")
			}
			if len(tv.Units) > 0 {
				sb.WriteString(" Units: " + tv.Units + ";")
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

// CREATE TIMESERIES root.datasets.etsi.Entity_clean_5min.HVAC_Mode with datatype=INT32, encoding=RLE;
// prefix: root.datasets.etsi.Entity_clean_5min
func (cdf NetCDF) Format_DBcreate() []string {
	var cdfTypeMap = map[string]string{"string": "TEXT", "double": "DOUBLE", "int": "INT32", "int64": "INT64", "bool": "BOOLEAN"} // map CDF to IotDB datatypes.
	var encodeMap = map[string]string{"string": "PLAIN", "double": "RLE", "int": "RLE", "int64": "RLE", "bool": "PLAIN"}
	lines := make([]string, 0)
	for _, tv := range cdf.Variables { // Msg: 700: Error occurred while parsing SQL to physical plan: line 1:55 no viable alternative at input '.time'
		str := "CREATE TIMESERIES " + cdf.Identifier + "." + cdf.StandardNameAlias(tv.StandardName) + " with datatype=" + cdfTypeMap[tv.ReturnType] + ", encoding=" + encodeMap[tv.ReturnType] + ";"
		lines = append(lines, str)
	}
	return lines
}

// return list of dataset column names.
func (cdf NetCDF) FormattedColumnNames() string {
	var sb strings.Builder
	sb.WriteString("(")
	for ndx := range cdf.Variables {
		sb.WriteString(cdf.StandardNameAlias(cdf.Variables[ndx].StandardName) + ",")
	}
	str := sb.String()[0:len(sb.String())-1] + ")"
	return str
}

// General data Statistics. Not stored in IotDB. <<<REFACTOR with xsv outputs.
type DataStatistic struct {
	StandardName      string         `json:"standardname"`
	NumberOfValues    int            `json:"numberofvalues"`
	NumDistinctValues int            `json:"numdistinctvalues"`
	MinimumValue      string         `json:"minimumvalue"`
	MaximumValue      string         `json:"maximumvalue"`
	FrequencyMap      map[string]int `json:"frequencymap"`
}

// TODO: Read the rest of the data from the *.nc files.
// type DataStatistic struct { NumDistinctValues int    `json:"numdistinctvalues"`  insert into map.
func (cdf NetCDF) Format_DBinsert() ([]string, []DataStatistic) {
	lines := make([]string, 0)
	var sb strings.Builder
	dataStats := make([]DataStatistic, len(cdf.Variables))                               //<<< cdf.FormattedColumnNames()
	columnNames := "(Time," + cdf.StandardNameAlias(cdf.Variables[0].StandardName) + ")" // constant first column (Time,id)
	sb.WriteString("INSERT INTO " + cdf.Identifier + columnNames + " VALUES ")           // That's right -- the column names don't match the actual data columns.
	dv1min := "~"
	dv1max := " "
	dv2min := "~"
	dv2max := " "

	for NDX := 0; NDX < 1; NDX++ { // cdf.Variables <<<
		tv := cdf.Variables[NDX]
		dataStatistic := DataStatistic{StandardName: tv.StandardName, NumberOfValues: cdf.Dimensions[tv.StandardName]}
		for ndx := 0; ndx < len(cdf.HouseIndices); ndx++ {
			dv1 := strings.TrimSpace(cdf.HouseIndices[ndx])
			dv2 := strings.TrimSpace(cdf.LongtimeIndices[ndx])
			sb.WriteString("(" + dv2 + ",'" + dv1 + "')") // (longtime,id) actual values
			if dv1 < dv1min {
				dv1min = dv1
			}
			if dv1 > dv1max {
				dv1max = dv1
			}
			if dv2 < dv2min {
				dv2min = dv2
			}
			if dv2 > dv2max {
				dv2max = dv2
			}
			if ndx < len(cdf.HouseIndices)-1 {
				sb.WriteString(", ")
			} else {
				sb.WriteString("; ")
			}
		}
		fmt.Println(dv2min)
		fmt.Println(dv2max)
		lines = append(lines, sb.String())
		dataStats = append(dataStats, dataStatistic)
		//fmt.Println(dataStatistic)
	}

	return lines, dataStats
}

func (cdf NetCDF) Format_SparqlQuery() []string {
	output := make([]string, 0)
	//<<<
	return output
}

// Make the dataset its own Class and loadable into GraphDB as Named Graph. Produce DatatypeProperty ontology from dataset.
// Special handling: "dateTime", "XMLLiteral", "anyURI"
func (cdf NetCDF) Format_TurtleOntology() []string {
	const crlf = `\n`
	fmt.Println(">>>>>>NetCDF.Format_TurtleOntology")
	var xsdDatatypeMap = map[string]string{"string": "string", "int": "integer", "int64": "integer", "float": "float", "double": "double", "decimal": "double", "byte": "boolean"} // map cdf to xsd datatypes.

	baseline := `@prefix s4data: ` + DataSetPrefix + `> .` + crlf +
		`@prefix example: ` + DataSetPrefix + cdf.Identifier + `/> .` + crlf +
		`@prefix foaf: <http://xmlns.com/foaf/spec/#> .` + crlf +
		`@prefix geosp: <http://www.opengis.net/ont/geosparql#> .` + crlf +
		`@prefix obo: <http://purl.obolibrary.org/obo/> .` + crlf +
		`@prefix org: <https://schema.org/> .` + crlf +
		`@prefix owl: <http://www.w3.org/2002/07/owl#> .` + crlf +
		`@prefix org: <https://schema.org/> .` + crlf +
		`@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .` + crlf +
		`@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .` + crlf +
		`@prefix saref: ` + SarefEtsiOrg + `core/> .` + crlf +
		`@prefix ssn: <http://www.w3.org/ns/ssn/> .` + crlf +
		`@prefix time: <http://www.w3.org/2006/time#> .` + crlf +
		`@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .` + crlf +
		`@prefix dcterms: <http://purl.org/dc/terms/> .` + crlf +
		`@prefix dctype: <http://purl.org/dc/dcmitype/> .` + crlf +

		DataSetPrefix + cdf.Identifier + `#> a dctype:Dataset ;` + crlf +
		`dcterms:title "` + cdf.Title + `"@en ;` + crlf +
		`dcterms:description "` + cdf.Description + `"@en .` + crlf +
		`dcterms:license <https://forge.etsi.org/etsi-software-license> ;` + crlf +
		`dcterms:conformsTo <https://saref.etsi.org/core/v3.1.1/> ;` + crlf +
		`dcterms:conformsTo <https://saref.etsi.org/saref4core/v3.1.1/> ;` + crlf +
		crlf +
		// new common Classes
		`###  https://saref.etsi.org/saref4data/StartTimeseries` + crlf +
		`s4data:StartTimeseries rdf:type owl:Class ;` + crlf +
		` rdfs:comment "The start time of a timeseries shall be present."@en ;` + crlf +
		` rdfs:label "start timeseries"@en ;` + crlf +
		` rdfs:subClassOf <http://www.w3.org/2006/time#TemporalEntity> ; .` + crlf +
		crlf +
		`###  https://saref.etsi.org/saref4data/StopTimeseries` + crlf +
		`s4data:StopTimeseries rdf:type owl:Class ;` + crlf +
		` rdfs:comment "The stop time of a timeseries shall be present."@en ;` + crlf +
		` rdfs:label "stop timeseries"@en ;` + crlf +
		` rdfs:subClassOf <http://www.w3.org/2006/time#TemporalEntity> ; .` + crlf +
		crlf +
		// new common ObjectProperty
		`###  https://saref.etsi.org/saref4data/hasEquation` + crlf +
		`s4data:hasEquation rdf:type owl:ObjectProperty ;` + crlf +
		` rdfs:comment "A relationship indicating that the entire timeseries dataset is represented by an equation of type linear, quadratic, polynomial, exponential, radical, trigonometric, or partial differential."@en ;` + crlf +
		` rdfs:label "has equation"@en .` + crlf +
		crlf +
		`###  https://saref.etsi.org/saref4data/isPriorTo` + crlf +
		`s4data:isPriorTo rdf:type owl:ObjectProperty ;` + crlf +
		` rdfs:comment "A relationship indicating that the timeseries dataset acts as a Bayesian prior to another dataset."@en ;` + crlf +
		` rdfs:label "is Bayesian prior to "@en .` + crlf +
		crlf +
		`###  https://saref.etsi.org/saref4data/isPosteriorTo` + crlf +
		`s4data:isPosteriorTo rdf:type owl:ObjectProperty ;` + crlf +
		` rdfs:comment "A relationship indicating that the timeseries dataset acts as a Bayesian posterior to another dataset."@en ;` + crlf +
		` rdfs:label "is Bayesian posterior to "@en .` + crlf +
		crlf +
		`###  https://saref.etsi.org/saref4data/isComparableTo` + crlf +
		`s4data:isComparableTo rdf:type owl:ObjectProperty ;` + crlf +
		` rdfs:comment "A relationship indicating that the timeseries dataset can be logically compared to another dataset. Necessary condition: Units must agree. Sufficient condition: equation types must agree."@en ;` + crlf +
		` rdfs:label "is Bayesian posterior to "@en .` + crlf +
		crlf +
		// new common DatatypeProperty
		`###  https://saref.etsi.org/saref4data/isAlignedWithTimeseries` + crlf +
		`s4data:isAlignedWithTimeseries rdf:type owl:DatatypeProperty ;` + crlf +
		` rdfs:range xsd:string ;` + crlf +
		` rdfs:comment "The name of the time sequence that the timeseries is aligned with."@en ;` + crlf +
		` rdfs:label "is aligned with timeseries"@en .` + crlf +
		crlf +
		`###  https://saref.etsi.org/saref4data/hasSamplingPeriodValue` + crlf +
		`s4data:hasSamplingPeriodValue rdf:type owl:DatatypeProperty ;` + crlf +
		` rdfs:range xsd:float ;` + crlf +
		` rdfs:comment "The sampling period in seconds."@en ;` + crlf +
		` rdfs:label "has samplingP period value"@en .` + crlf +
		crlf +
		`###  https://saref.etsi.org/saref4data/hasUpperLimitValue` + crlf +
		`s4data:hasUpperLimitValue rdf:type owl:DatatypeProperty ;` + crlf +
		` rdfs:range xsd:float ;` + crlf +
		` rdfs:comment "The highest value in the timeseries."@en ;` + crlf +
		` rdfs:label "has upper limit value"@en .` + crlf +
		crlf +
		`###  https://saref.etsi.org/saref4data/hasLowerLimitValue` + crlf +
		`s4data:hasLowerLimitValue rdf:type owl:DatatypeProperty ;` + crlf +
		` rdfs:range xsd:float ;` + crlf +
		` rdfs:comment "The lowest value in the timeseries."@en ;` + crlf +
		` rdfs:label "has lower limit value"@en .` + crlf +
		crlf +
		`###  https://saref.etsi.org/saref4data/hasNumericPrecisionValue` + crlf +
		`s4data:hasNumericPrecisionValue rdf:type owl:DatatypeProperty ;` + crlf +
		` rdfs:range xsd:integer ;` + crlf +
		` rdfs:comment "Indicates the number of leading and trailing significant digits in the measurement."@en ;` + crlf +
		` rdfs:label "available ram"@en .` + crlf +
		crlf +

		// define the dataset Class derieved from saref:Measurement:
		`### ` + DataSetPrefix + cdf.Identifier + crlf +
		`s4data:` + cdf.Identifier + ` rdf:type owl:Class ;` + crlf +
		` rdfs:subClassOf saref:Measurement ,` + crlf +
		` rdfs:comment "` + cdf.Description + `"@en ;` + crlf +
		` rdfs:label "` + cdf.Identifier + `"@en .` + crlf +
		// Add standard Measurement properties
		` rdfs:subClassOf [` + crlf +
		`  rdf:type owl:Restriction ;` + crlf +
		`  owl:minQualifiedCardinality "1"^^xsd:nonNegativeInteger ;` + crlf +
		`  owl:onClass s4data:StartTimeseries ;` + crlf +
		`  owl:onProperty saref:hasTime ;` + crlf +
		` ] ;` + crlf +
		` rdfs:subClassOf [` + crlf +
		`  rdf:type owl:Restriction ;` + crlf +
		`  owl:maxQualifiedCardinality "1"^^xsd:nonNegativeInteger ;` + crlf +
		`  owl:onClass s4data:EndTimeseries ;` + crlf +
		`  owl:onProperty saref:hasTime ;` + crlf +
		` ] ;` + crlf +
		` [ rdf:type owl:Restriction ;` + crlf +
		`  owl:onProperty saref:hasTime ;` + crlf +
		`  owl:allValuesFrom saref:Time` + crlf +
		` ] ,` + crlf +
		` [ a owl:Restriction ;` + crlf +
		`  owl:onProperty saref:hasMeasurement ;` + crlf +
		`  owl:allValuesFrom saref:Measurement ` + crlf +
		` ] ; ` + crlf +
		` [ rdf:type owl:Restriction ;` + crlf +
		`  owl:onProperty saref:relatesToMeasurement ;` + crlf +
		`  owl:allValuesFrom saref:Measurement` + crlf +
		` ] ;` + crlf +
		` [ rdf:type owl:Restriction ;` + crlf +
		`  owl:onProperty saref:measurementMadeBy ;` + crlf +
		`  owl:someValuesFrom saref:Device` + crlf +
		` ] ,` + crlf +
		` [ rdf:type owl:Restriction ;` + crlf +
		`  owl:onProperty saref:hasConfidence ;` + crlf +
		`  owl:someValuesFrom saref:Confidence` + crlf +
		` ] ,` + crlf +
		` [ rdf:type owl:Restriction ;` + crlf +
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
		` ] ; .` + crlf

	output := strings.Split(baseline, crlf)
	// add specific field names as DatatypeProperties:
	for _, v := range cdf.Variables {
		eva, _ := GetClosestEntity(cdf.Identifier) // graphdb.EntityVariableAlias
		fmt.Println(eva.ParentClassIRI())
		output = append(output, ` rdfs:subClassOf `+eva.ParentClassIRI()+` ,`+crlf+`[ rdf:type owl:Restriction ;`+crlf+`owl:onProperty :has`+v.StandardName+` ;`+crlf+`owl:allValuesFrom xsd:`+xsdDatatypeMap[v.ReturnType]+crlf+`] ; .`+crlf+crlf)
	}

	return output
}

///////////////////////////////////////////////////////////////////////////////////////////

// Expects nginx server running at http://localhost:80/datasets
// Upload JSON descriptor files, turtle files, SPARQL query files.
func UploadFile(destinationUrl string, lines []string) error {
	//<<<
	return nil
}

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
	if strings.HasSuffix(s, " ;") {
		s = s[0 : len(s)-2]
	}
	return s
}

// Parse Jan_clean.var file that is output from /usr/bin/ncdump -c Jan_clean.nc. Var files are specific to *.nc datasets.
func ParseVariableFile(fname, filetype, identifier string) (NetCDF, error) {
	netcdf := NetCDF{}
	lines, err := fs.ReadTextLines(fname, false)
	if err != nil {
		fmt.Println(err)
		return netcdf, err
	}
	lineIndex := 0
	tokens := strings.Split(lines[lineIndex], " ")
	netcdf.NetcdfType = filetype
	netcdf.Identifier = tokens[1]
	if len(identifier) > 0 {
		netcdf.Identifier = identifier
	}
	netcdf.Dimensions = make(map[string]int, 0)
	netcdf.Variables = make([]CDFvariable, 0)
	lineIndex++

	if strings.Contains(lines[lineIndex], "dimensions:") {
		variables := strings.Contains(lines[lineIndex], "variables:")
		for !variables {
			lineIndex++
			tokens := strings.Split(strings.TrimSpace(lines[lineIndex]), " ")
			size, e := strconv.Atoi(tokens[2])
			if e == nil {
				netcdf.Dimensions[tokens[0]] = size
			} else {
				fmt.Println("Error converting Dimension " + tokens[0])
			}
			variables = strings.Contains(lines[lineIndex+1], "variables:")
		}
		lineIndex++
	}

	// Index each dimension
	dimensionMap := make(map[string]int, len(netcdf.Dimensions))
	index := 1
	for k := range netcdf.Dimensions {
		dimensionMap[k] = index
		index++
	}

	if strings.Contains(lines[lineIndex], "variables:") {
		offset := 1
		data := strings.Contains(lines[lineIndex], "data:")
		for !data {
			lineIndex++
			if len(strings.TrimSpace(lines[lineIndex])) == 0 {
				break
			}
			tokens := strings.Split(strings.TrimSpace(lines[lineIndex]), " ")
			standardname := strings.Split(tokens[1], "(")[0]
			tmpVar := CDFvariable{ReturnType: tokens[0], StandardName: standardname}
			val, ok := dimensionMap[tmpVar.StandardName]
			if ok {
				tmpVar.DimensionIndex = val
			}
			thisVariable := strings.Contains(lines[lineIndex], tmpVar.StandardName)
			for thisVariable {
				lineIndex++
				tokens = strings.Split(strings.TrimSpace(lines[lineIndex]), "=")
				if strings.Contains(lines[lineIndex], ":_FillValue") {
					tmpVar.FillValue = prettifyString(tokens[offset])
				}
				if strings.Contains(lines[lineIndex], ":units") {
					tmpVar.Units = prettifyString(tokens[offset])
				}
				if strings.Contains(lines[lineIndex], ":standard_name") {
					tmpVar.StandardName = prettifyString(tokens[offset])
				}
				if strings.Contains(lines[lineIndex], ":long_name") {
					tmpVar.LongName = prettifyString(tokens[offset])
				}
				if strings.Contains(lines[lineIndex], ":comment") {
					tmpVar.Comment = prettifyString(tokens[offset])
				}
				if strings.Contains(lines[lineIndex], ":calendar") {
					tmpVar.Calendar = prettifyString(tokens[offset])
				}
				thisVariable = strings.Contains(lines[lineIndex+1], tmpVar.StandardName+":")
			}
			netcdf.Variables = append(netcdf.Variables, tmpVar)
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
			netcdf.Title = tokens[offset]
		}
		if strings.Contains(lines[lineIndex], ":description") {
			netcdf.Description = tokens[offset]
		}
		if strings.Contains(lines[lineIndex], ":conventions") {
			netcdf.Conventions = tokens[offset]
		}
		if strings.Contains(lines[lineIndex], ":institution") {
			netcdf.Institution = tokens[offset]
		}
		if strings.Contains(lines[lineIndex], ":code_url") {
			netcdf.Code_url = tokens[offset]
		}
		if strings.Contains(lines[lineIndex], ":location_meaning") {
			netcdf.Location_meaning = tokens[offset]
		}
		if strings.Contains(lines[lineIndex], ":datastream_name") {
			netcdf.Datastream_name = tokens[offset]
		}
		if strings.Contains(lines[lineIndex], ":input_files") {
			netcdf.Input_files = tokens[offset]
		}
		if strings.Contains(lines[lineIndex], ":history") {
			netcdf.History = tokens[offset]
		}
		lineIndex++
	}

	lineIndex = lineIndex + 2
	//netcdf.HouseIndices = make([]string, netcdf.Dimensions["id"])
	//netcdf.LongtimeIndices = make([]string, netcdf.Dimensions["time"])
	netcdf.HouseIndices, lineIndex = parseDimensionIndices(netcdf.Dimensions["id"], lineIndex, lines)
	netcdf.LongtimeIndices, lineIndex = parseDimensionIndices(netcdf.Dimensions["time"], lineIndex, lines)
	lineIndex++

	/* dimIndex := 0
	for lineIndex < len(lines) {
		if len(strings.TrimSpace(lines[lineIndex])) == 0 {
			break
		}
		tokens := strings.Split(strings.TrimSpace(lines[lineIndex]), ",")
		if len(tokens) == 1 { // last element terminated by semi-colon; no comma.
			tokens = strings.Split(strings.TrimSpace(lines[lineIndex]), ";")
		}
		for ndx := 0; ndx < len(tokens)-1; ndx++ {
			netcdf.HouseIndices[dimIndex] = prettifyString(tokens[ndx])
			if strings.Contains(netcdf.HouseIndices[dimIndex], "=") { // remove variable names
				tok2 := strings.Split(netcdf.HouseIndices[dimIndex], "=")
				netcdf.HouseIndices[dimIndex] = tok2[1]
			}
			dimIndex++
		}
		lineIndex++
	}

	lineIndex++
	dimIndex = 0
	for lineIndex < len(lines) {
		if len(strings.TrimSpace(lines[lineIndex])) == 0 {
			break
		}
		tokens := strings.Split(strings.TrimSpace(lines[lineIndex]), ",")
		if len(tokens) == 1 { // last element terminated by semi-colon; no comma.
			tokens = strings.Split(strings.TrimSpace(lines[lineIndex]), ";")
		}
		for ndx := 0; ndx < len(tokens)-1; ndx++ {
			netcdf.LongtimeIndices[dimIndex] = prettifyString(tokens[ndx])
			if strings.Contains(netcdf.LongtimeIndices[dimIndex], "=") { // remove variable names
				tok2 := strings.Split(netcdf.LongtimeIndices[dimIndex], "=")
				netcdf.LongtimeIndices[dimIndex] = tok2[1]
			}
			dimIndex++
		}
		lineIndex++
	} */

	return netcdf, nil
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
				output[dimIndex] = tok2[1]
			}
			dimIndex++
		}
		lineIndex++
	}
	return output, lineIndex

}

func ShowLastExternalBackup() string {
	fp := "/home/david/lastExternalBackup.txt"
	exists, _ := fs.FileExists(fp)
	if !exists {
		return "The database has never been backed up to an external drive."
	}
	lines, _ := fs.ReadTextLines("/home/david/lastExternalBackup.txt", false)
	msg := strings.Replace(lines[0], "               ", "Last backup at ", 1)
	return msg
}

///////////////////////////////////////////////////////////////////////////////////////////

// This is the only function that calls iotdb. use xsv tool to output csv file of column types:
// xsv stats /home/david/Documents/digital-twins/opsd.household/household_data_1min_singleindex.csv --everything >/home/david/Documents/digital-twins/opsd.household/summary_household_data_1min_singleindex.csv
func LoadCsvSensorDataIntoDatabase(programArgs []string) {
	ok, iotdbConnection := Init_IoTDB()
	if !ok {
		log.Fatal(errors.New(iotdbConnection))
	}
	iotdbDataFile, err := Initialize_IoTDbDataFile(programArgs)
	checkErr("Initialize_IoTDbDataFile: ", err)
	//fmt.Println(iotdbDataFile.OutputDescription(true))
	err = iotdbDataFile.ProcessTimeseries()
	checkErr("ProcessTimeseries(csv)", err)
}

func LoadNcSensorDataIntoDatabase(programArgs []string) {
	inputFile := programArgs[1]
	fileType := programArgs[2]
	identifier := ""
	if len(programArgs) > 3 {
		identifier = programArgs[3]
	}

	netcdf, err := ParseVariableFile(inputFile+".var", fileType, identifier)
	checkErr("ParseVariableFile", err)
	netcdf.ToString(true) // true => output variables

	json, err := netcdf.Format_Json()
	checkErr("netcdf.Format_Json", err)
	err = fs.WriteTextLines(json, inputFile+".json", false)
	checkErr("fs.WriteTextLines(json)", err)
	err = UploadFile(DataSetPrefix+inputFile+".json", json)
	checkErr("UploadFile: "+inputFile+".json", err)

	ontology := netcdf.Format_TurtleOntology()
	err = fs.WriteTextLines(ontology, inputFile+".ttl", false)
	checkErr("fs.WriteTextLines(ttl)", err)
	err = UploadFile(DataSetPrefix+inputFile+".ttl", ontology)
	checkErr("UploadFile: "+inputFile+".ttl", err)

	sparqlQuery := netcdf.Format_SparqlQuery()
	err = fs.WriteTextLines(sparqlQuery, inputFile+".sparql", false)
	checkErr("fs.WriteTextLines(ttl)", err)
	err = UploadFile(DataSetPrefix+inputFile+".sparql", sparqlQuery)
	checkErr("UploadFile: "+inputFile+".sparql", err)
}

func LoadHd5SensorDataIntoDatabase(programArgs []string) {
	// <<<
}

// Data source file types determined by file extension: {.nc, .csv, .hd5}  Args[0] is program name.
func main() {
	fmt.Println(ShowLastExternalBackup())
	sourceDataType := "help"
	if len(os.Args) > 1 {
		sourceDataType = strings.ToLower(path.Ext(os.Args[1]))
	}

	switch sourceDataType {
	case ".csv":
		LoadCsvSensorDataIntoDatabase(os.Args)
	case ".var":
		LoadNcSensorDataIntoDatabase(os.Args)
	case ".hd5":
		LoadHd5SensorDataIntoDatabase(os.Args)
	default:
		fmt.Println("Required *.csv parameters: path to csv sensor data file plus any timeseries command: {drop create delete insert example}.")
		fmt.Println("The csv summary file produced by 'xsv stats <dataFile.csv> --everything' should already exist in the same folder as <dataFile.csv>,")
		fmt.Println("including a description.txt file. All timeseries data are placed under the database prefix: " + IotDataPrefix)
		fmt.Println("Required *.var parameters: 1) path to single *.var file, and 2) CDF file type {HDF5, netCDF-4, classic}.")
		fmt.Println("  3) Optionally specify the unique dataset identifier; e.g. Entity_clean_5min")
		fmt.Println("Required *.hd5 parameters: 1) path to single *.var file, and 2) CDF file type {HDF5, netCDF-4, classic}.")
		fmt.Println("  3) Optionally specify the unique dataset identifier; e.g. Entity_clean_5min")
		os.Exit(0)
	}
}

///////////////////////////////////////////////////////////////////////////////////////////

const (
	sarefConst      = "saref:"
	owlClass        = "owl:Class"
	subClassOfmatch = "rdfs:subClassOf"
	etsiOrg         = "https://saref.etsi.org/"
	unknown         = "???"
	zero            = 0.0
	graphDbPrefix   = "PREFIX%20%3A%3Chttp%3A%2F%2Fwww.ontotext.com%2Fgraphdb%2Fsimilarity%2F%3E%0APREFIX%20inst%3A%3Chttp%3A%2F%2Fwww.ontotext.com%2Fgraphdb%2Fsimilarity%2Finstance%2F%3E%0APREFIX%20psi%3A%3Chttp%3A%2F%2Fwww.ontotext.com%2Fgraphdb%2Fsimilarity%2Fpsi%2F%3E%0A"
	graphDbPostfix  = "%3E%3B%0Apsi%3AsearchPredicate%20%3Chttp%3A%2F%2Fwww.ontotext.com%2Fgraphdb%2Fsimilarity%2Fpsi%2Fany%3E%3B%0A%3AsearchParameters%20%22-numsearchresults%208%22%3B%0Apsi%3AentityResult%20%3Fresult%20.%0A%3Fresult%20%3Avalue%20%3Fentity%20%3B%0A%3Ascore%20%3Fscore%20.%20%7D%0A"
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
		//"Match      : " + fs.Match + sep +
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

const SarefPrefix = "https://saref.etsi.org/"

var SarefMap = map[string]string{ // exported
	"s4agri": SarefPrefix + "saref4agri/",
	"s4auto": SarefPrefix + "saref4auto/",
	"s4bldg": SarefPrefix + "saref4bldg/",
	"s4city": SarefPrefix + "saref4city/",
	"s4ehaw": SarefPrefix + "saref4ehaw/",
	"s4ener": SarefPrefix + "saref4ener/",
	"s4envi": SarefPrefix + "saref4envi/",
	"s4inma": SarefPrefix + "saref4inma/",
	"s4lift": SarefPrefix + "saref4lift/",
	"s4syst": SarefPrefix + "saref4syst/",
	"s4watr": SarefPrefix + "saref4watr/",
	"s4wear": SarefPrefix + "saref4wear/",
	"saref":  SarefPrefix + "core/",
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
	ttlFileDirectory     = "/home/davidgnabasik/Documents/ontologies.ttl/"
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
	file, err := os.ReadFile("/home/david/Documents/digital-twins/netcdf/graphdb/graphdb.json")
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
func Init_GraphDB() (bool, string) {
	assignProgramParameters()
	tokens := strings.Split(graphDBparameters.DefaultDbInstanceUrl, ":")
	host := strings.ReplaceAll(tokens[1], "/", "") // "localhost"
	tokens = strings.Split(tokens[2], "/")
	ports := []string{tokens[0]}
	isOpen, _ := testRemoteAddressPortsOpen(host, ports)
	if !isOpen {
		fmt.Println("GraphDB is not available at " + graphDBparameters.DefaultDbInstanceUrl)
		return false, graphDBparameters.DefaultDbInstanceUrl
	}
	return true, graphDBparameters.DefaultDbInstanceUrl
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

// Returns empty struct upon error: SELECT * WHERE {<https://saref.etsi.org/saref4wear/CrowdProperty> ?entity ?score .}
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
		if strings.HasPrefix(uri, etsiOrg) {
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

// non-generic version looks for first embedded string match.
// return empty string if not found.
func find(lines []string, target string) string {
	for ndx, v := range lines {
		if strings.Contains(v, target) {
			return lines[ndx]
		}
	}
	return ""
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
	isOpen, graphdbURL := Init_GraphDB()
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
	sourceLines, err := fs.ReadTextLines(fileName, false)
	checkErr("ProcessEntities.1", err)
	output := make([]Similarity, 0)
	extension := getExtensionFromFilename(fileName)
	for _, entity := range SimilarEntities {
		sourceEntities := getEntities(sourceLines, entity)
		for _, entityName := range sourceEntities { // saref:entity
			toks := strings.Split(entityName, ":")
			Entity := etsiOrg + extension + "/" + toks[1]
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
	lines, _ := fs.ReadTextLines(ontologyFile, false)
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
// ?? Get single class: SELECT * WHERE {<https://saref.etsi.org/saref4wear/CrowdProperty> ?entity ?score .}
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

/* *********************************************************************************************************** */

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
		if strings.HasPrefix(baseClassURI, etsiOrg) {
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

///////////////////////////////////////////////////////////////////////////////////////////

const (
	//SarefEtsiOrg   = "<https://saref.etsi.org/" // does not include trailing >
	OutputLocation = SarefEtsiOrg + "datasets/examples/"
	DatasetPrefix  = "root.datasets.etsi."
	TimeFormat     = "2006-01-02T15:04:05Z" // yyyy-MM-ddThh:mm:ssZ not quite RFC3339 format. What about timezone?
	LastColumnName = "DatasetName"
)

//var recommendedUnits = []string{"yyyy-MM-ddThh:mm:ssZ,RFC3339", "unicode,string", "unixutc,long", "m/h,meters/hour", "knots,knots","percent,percent" }

// Manually add the map keys to the (last) units column header. All of these are specified at uomPrefix URI.
// If th value contains a comma, call GetGetNamedIndividualFormat() instead.
func GetNamedIndividualUnitMeasure(uom string) string {
	const uomPrefix = "http://www.ontology-of-units-of-measure.org/resource/om-2/"
	var unitsOfMeasure = map[string]string{
		"kW":     uomPrefix + "kilowatt",
		"kWh":    uomPrefix + "kilowattHour",
		"pascal": uomPrefix + "pascal",
		"kelvin": uomPrefix + "kelvin",
		"°C":     uomPrefix + "degreeCelsius",
		"°F":     uomPrefix + "degreeFahrenheit",
		"%rh":    uomPrefix + "RelativeHumidity",
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
	return `???`
}

var timeSeriesCommands = []string{"drop", "create", "delete", "insert", "example"}
var goodFileTypes = map[string]string{".nc": "ok", ".csv": "ok", ".hd5": "ok"}
var iotdbParameters IoTDbProgramParameters
var session client.Session
var clientConfig *client.Config

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

func Init_IoTDB() (bool, string) {
	fmt.Println("Initializing IoTDB client...")
	clientConfig = configureIotdbAccess()
	isOpen, err := testRemoteAddressPortsOpen(clientConfig.Host, []string{clientConfig.Port})
	connectStr := clientConfig.Host + ":" + clientConfig.Port
	if !isOpen {
		fmt.Printf("%s%v%s", "Expected IoTDB to be available at "+connectStr+" but got ERROR: ", err, "\n")
		fmt.Printf("Please execute:  cd ~/iotdb && sbin/start-standalone.sh && sbin/start-cli.sh -h 127.0.0.1 -p 6667 -u root -pw root")
		return false, connectStr
	}
	return true, connectStr
}

// mapped to original column name
type MeasurementItem struct {
	MeasurementName  string `json:"measurementname"`  // Original name always single-quoted
	MeasurementAlias string `json:"measurementalias"` // Names that fit IotDB format; see StandardName() => [0-9 a-z A-Z _ ]
	MeasurementType  string `json:"measurementtype"`  // XSD data type
	MeasurementUnits string `json:"measurementunits"`
	ColumnOrder      int    `json:"columnorder"` // Column order from data file
}

func (mi MeasurementItem) ToString() string {
	return mi.MeasurementName + " : " + mi.MeasurementAlias + " : " + mi.MeasurementType + " : " + mi.MeasurementUnits
}

type IoTDbDataFile struct {
	session            client.Session             // for (iot *IoTDbDataFile) methods.
	Description        string                     `json:"description"`
	DataFilePath       string                     `json:"datafilepath"`
	DataFileType       string                     `json:"datafiletype"`
	SummaryFilePath    string                     `json:"summaryfilepath"`
	DatasetName        string                     `json:"datasetname"`
	Measurements       map[string]MeasurementItem `json:"measurements"`
	TimeseriesCommands []string                   `json:"timeseriescommands"` // command-line parameters
	Summary            [][]string                 `json:"summary"`            // from summary file
	Dataset            [][]string                 `json:"dataset"`            // actual data
}

func Initialize_IoTDbDataFile(programArgs []string) (IoTDbDataFile, error) {
	datasetPathName := filepath.Base(programArgs[1]) // does not include trailing slash
	datasetName := datasetPathName[:len(datasetPathName)-len(filepath.Ext(datasetPathName))]
	description, _ := fs.ReadTextLines(filepath.Dir(programArgs[1])+"/description.txt", false)
	iotdbDataFile := IoTDbDataFile{Description: strings.Join(description, " "), DataFilePath: programArgs[1], DatasetName: datasetName}
	exists, err := fs.FileExists(iotdbDataFile.DataFilePath)
	if !exists {
		return iotdbDataFile, err
	}
	checkErr("Sensor data file not readable: "+iotdbDataFile.DataFilePath, err)

	iotdbDataFile.DataFileType = strings.ToLower(path.Ext(iotdbDataFile.DataFilePath))
	_, ok := goodFileTypes[iotdbDataFile.DataFileType]
	if !ok {
		return iotdbDataFile, errors.New("Cannot process source file type: " + iotdbDataFile.DataFilePath)
	}
	iotdbDataFile.TimeseriesCommands = make([]string, 0)
	for ndx := range programArgs {
		if find(timeSeriesCommands, strings.ToLower(programArgs[ndx])) != "" {
			iotdbDataFile.TimeseriesCommands = append(iotdbDataFile.TimeseriesCommands, strings.ToLower(programArgs[ndx]))
		}
	}
	// expect only 1 instance of 'datasetName' in iotdbDataFile.DataFilePath.
	iotdbDataFile.SummaryFilePath = strings.Replace(iotdbDataFile.DataFilePath, datasetPathName, "summary_"+datasetPathName, 1)
	iotdbDataFile.ReadCsvFile(iotdbDataFile.SummaryFilePath, false) // isDataset: no, is summary
	iotdbDataFile.XsvSummaryTypeMap()
	// read sensor data
	iotdbDataFile.ReadCsvFile(iotdbDataFile.DataFilePath, true) // isDataset: yes
	return iotdbDataFile, nil
}

// Include percentage of missing data, which can be got from a SELECT count(*) from root.datasets.etsi.household_data_1min_singleindex;
func (iot *IoTDbDataFile) OutputDescription(displayColumnInfo bool) string {
	const crlf = "\n"
	var sb strings.Builder
	sb.WriteString(crlf)
	sb.WriteString("Description     : " + iot.Description + crlf)
	sb.WriteString("Data Set Name   : " + iot.DatasetName + crlf)
	sb.WriteString("Data Path       : " + iot.DataFilePath + crlf)
	sb.WriteString("n Data Types    : " + strconv.Itoa(len(iot.Measurements)) + crlf)
	sb.WriteString("n Measurements  : " + strconv.Itoa(len(iot.Dataset)-1) + crlf)
	sb.WriteString("Data start time : " + iot.Dataset[1][0] + crlf)
	sb.WriteString("Data end time   : " + iot.Dataset[len(iot.Dataset)-1][0] + crlf)
	if displayColumnInfo {
		for _, item := range iot.Measurements {
			sb.WriteString("  " + item.ToString() + crlf)
		}
	}
	return sb.String()
}

// Expects comma-separated files. Assigns Dataset or Summary.
func (iot *IoTDbDataFile) ReadCsvFile(filePath string, isDataset bool) {
	f, err := os.Open(filePath)
	checkErr("Unable to read csv file "+filePath, err)
	defer f.Close()
	fmt.Println("Reading " + filePath)
	csvReader := csv.NewReader(f)
	if !isDataset {
		iot.Summary, err = csvReader.ReadAll()
	} else {
		iot.Dataset, err = csvReader.ReadAll()
	}
	checkErr("Unable to parse file as CSV for "+filePath, err)
}

// Return quoted name and its alias. Alias: open brackets are replaced with underscore.
// Need to handle 2 aliases being the same.  Handles some IotDB reserved keywords.
func StandardName(oldName string) (string, string) { // return MeasurementName, MeasurementAlias
	newName := oldName
	if strings.ToLower(newName) == "time" {
		newName = "time1"
	}
	replacer := strings.NewReplacer("~", "", "!", "", "@", "", "#", "", "$", "", "%", "", "^", "", "&", "", "*", "", "/", "", "?", "", ".", "", ",", "", ":", "", ";", "", "|", "", "\\", "", "=", "", "+", "", ")", "", "}", "", "]", "", "(", "_", "{", "_", "[", "_")
	alias := strings.ReplaceAll(newName, " ", "")
	alias = replacer.Replace(alias)
	if newName != alias {
		return alias, newName
	} else {
		return newName, alias
	}
}

// Expects {Units, DatasetName} fields to have been appended to the summary file. Assign []Measurements. Expects Summary to be assigned. Use XSD data types.
// []string{"field", "type", "sum", "min", "max", "min_length", "max_length", "mean", "stddev", "median", "mode", "cardinality", "Units", "DatasetName"}
// len() only returns the length of the "external" array.
func (iot *IoTDbDataFile) XsvSummaryTypeMap() {
	rowsXsdMap := map[string]string{"Unicode": "string", "Float": "float", "Integer": "integer"}
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
		dataColumnName, aliasName := StandardName(iot.Summary[ndx+1][0]) // skip header row
		theEnd := dataColumnName == "interpolated"
		if !theEnd {
			mi := MeasurementItem{
				MeasurementName:  dataColumnName,
				MeasurementAlias: aliasName,
				MeasurementType:  rowsXsdMap[iot.Summary[ndx+1][1]],
				MeasurementUnits: iot.Summary[ndx+1][unitsColumn],
				ColumnOrder:      ndx,
			}
			iot.Measurements[dataColumnName] = mi // add to map
		}
	}
	// add DatasetName timerseries in case data column names are the same for different sampling intervals.
	mi := MeasurementItem{
		MeasurementName:  LastColumnName,
		MeasurementAlias: LastColumnName,
		MeasurementType:  "string",
		MeasurementUnits: "unicode,string",
		ColumnOrder:      ndx1,
	}
	iot.Measurements[LastColumnName] = mi
}

// Return list of ordered dataset column names as string. Does not include enclosing ()
func (iot *IoTDbDataFile) FormattedColumnNames() string {
	var sb strings.Builder
	for ndx := 0; ndx < len(iot.Measurements); ndx++ {
		for _, item := range iot.Measurements {
			if item.ColumnOrder == ndx {
				sb.WriteString(item.MeasurementName + ",")
			}
		}
	}
	str := sb.String()[0:len(sb.String())-1] + " " // replace trailing comma
	return str
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

// COUNT TIMESERIES root.datasets.etsi.household_data_60min_singleindex.DatasetName;
// Each ALIGNED insert statement inserts one row.  Dataset includes header row so subtract 1.
// Automatically inserts long time column as first column (which should be UTC)
// Avoid returning huge result.
func (iot *IoTDbDataFile) SaveInsertStatements() { // []string {
	var sb strings.Builder
	insertTimeseriesStatements := make([]string, len(iot.Dataset)-1)
	for r := 1; r < len(iot.Dataset); r++ {
		sb.Reset()
		sb.WriteString("INSERT INTO " + DatasetPrefix + iot.DatasetName + " (time," + iot.FormattedColumnNames() + ") ALIGNED VALUES (")
		startTime, err := time.Parse(TimeFormat, iot.Dataset[r][0])
		if err != nil {
			fmt.Println(err)
		}
		sb.WriteString(strconv.FormatInt(startTime.UTC().Unix(), 10) + ",")
		for c := 0; c < len(iot.Dataset[r])-1; c++ { // skip 'interpolated'
			for _, item := range iot.Measurements {
				if item.ColumnOrder == c {
					sb.WriteString(formatDataItem(iot.Dataset[r][c], item.MeasurementType) + ",")
				}
			}
		}
		sb.WriteString(formatDataItem(LastColumnName, "string") + ");")
		insertTimeseriesStatements[r-1] = sb.String()
	}
	err := fs.WriteTextLines(insertTimeseriesStatements, iot.DataFilePath+".iotdb.multiple.insert", false)
	checkErr("SaveInsertStatements("+iot.DataFilePath+".iotdb.multiple.insert"+")", err)
}

// someTime can be either a long or a readable dateTime string.
func getStartTime(someTime string) (time.Time, error) {
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

// Return IotDB datatype; encoding; compression. See https://iotdb.apache.org/UserGuide/V1.0.x/Data-Concept/Encoding.html#encoding-methods
// (client.TSDataType, client.TSEncoding, client.TSCompressionType)
func getClientStorage(dataColumnType string) (string, string, string) {
	sw := strings.ToLower(dataColumnType)
	switch sw {
	case "boolean":
		return "BOOLEAN", "RLE", "SNAPPY"
	case "string":
		return "TEXT", "PLAIN", "SNAPPY"
	case "integer":
		return "INT64", "GORILLA", "SNAPPY"
	case "long":
		return "INT64", "GORILLA", "SNAPPY"
	case "float":
		return "FLOAT", "GORILLA", "SNAPPY"
	case "double":
		return "DOUBLE", "GORILLA", "SNAPPY"
	case "decimal":
		return "DOUBLE", "GORILLA", "SNAPPY"
	}
	return "TEXT", "PLAIN", "SNAPPY"
}

// Produce DatatypeProperty ontology from summary & dataset. Write to filesystem, then upload to website.
// Make the dataset its own Class and loadable into GraphDB as Named Graph. Special handling: "dateTime", "XMLLiteral", "anyURI"
func (iot *IoTDbDataFile) Format_TurtleOntology() []string {
	fmt.Println(">>>>>>IoTDbDataFile.Format_TurtleOntology")                                                                                                                       //<<<
	var xsdDatatypeMap = map[string]string{"string": "string", "int": "integer", "int64": "integer", "float": "float", "double": "double", "decimal": "double", "byte": "boolean"} // map cdf to xsd datatypes.
	output := make([]string, 256)                                                                                                                                                  // best guess
	output[0] = "@prefix s4data: " + OutputLocation + "> ."
	output[1] = "@prefix example: " + OutputLocation + iot.DatasetName + "/> ."

	output[2] = "@prefix foaf: <http://xmlns.com/foaf/spec/#> ."
	output[3] = "@prefix geosp: <http://www.opengis.net/ont/geosparql#> ."
	output[4] = "@prefix obo: <http://purl.obolibrary.org/obo/> ."
	output[5] = "@prefix org: <https://schema.org/> ."
	output[6] = "@prefix owl: <http://www.w3.org/2002/07/owl#> ."
	output[7] = "@prefix org: <https://schema.org/> ."
	output[8] = "@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> ."
	output[9] = "@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> ."
	output[10] = "@prefix saref: " + SarefEtsiOrg + "core/> ."
	output[11] = "@prefix ssn: <http://www.w3.org/ns/ssn/> ."
	output[12] = "@prefix time: <http://www.w3.org/2006/time#> ."
	output[13] = "@prefix xsd: <http://www.w3.org/2001/XMLSchema#> ."
	output[14] = "@prefix dcterms: <http://purl.org/dc/terms/> ."
	output[15] = "@prefix dctype: <http://purl.org/dc/dcmitype/> ."

	output[16] = OutputLocation + iot.DatasetName + "#> a dctype:Dataset ;"
	output[17] = `dcterms:title "` + iot.Description + `"@en ;`
	output[18] = `dcterms:description "` + iot.DatasetName + `"@en .`
	output[19] = "dcterms:license <https://forge.etsi.org/etsi-software-license> ;"
	output[20] = "dcterms:conformsTo <https://saref.etsi.org/core/v3.1.1/> ;"
	output[21] = "dcterms:conformsTo <https://saref.etsi.org/saref4core/v3.1.1/> ;"

	output[21] = "### " + OutputLocation + iot.DatasetName
	output[22] = "s4data:" + iot.DatasetName + " rdf:type owl:Class ;"
	output[23] = `rdfs:comment "` + iot.Description + `"@en ;`
	output[24] = `rdfs:label "` + iot.DatasetName + `"@en .`

	eva, _ := GetClosestEntity("variableName") // (EntityVariableAlias, error)
	output[25] = `rdfs:subClassOf ` + eva.ParentClassIRI() + " ,"
	output[26] = "[ rdf:type owl:Restriction ;"
	output[27] = "owl:onProperty saref:measurementMadeBy ;"
	output[28] = "owl:someValuesFrom saref:Device"
	output[29] = "] ,"
	output[30] = "[ rdf:type owl:Restriction ;"
	output[31] = "owl:onProperty saref:hasConfidence ;"
	output[32] = "owl:someValuesFrom saref:Confidence"
	output[33] = "] ,"
	output[34] = "[ rdf:type owl:Restriction ;"
	output[35] = "owl:onProperty saref:isMeasuredIn ;"
	output[36] = "owl:allValuesFrom saref:UnitOfMeasure"
	output[37] = "] ,"
	output[38] = "[ rdf:type owl:Restriction ;"
	output[39] = "owl:onProperty saref:relatesToProperty ;"
	output[40] = "owl:allValuesFrom saref:Property"
	output[41] = "] ,"
	output[42] = "[ rdf:type owl:Restriction ;"
	output[43] = "owl:onProperty saref:isMeasuredIn ;"
	output[44] = `owl:qualifiedCardinality "1"^^xsd:nonNegativeInteger ;`
	output[45] = "owl:onClass saref:UnitOfMeasure:"
	output[46] = "] ,"
	output[47] = "[ rdf:type owl:Restriction ;"
	output[48] = "owl:onProperty saref:relatesToProperty ;"
	output[49] = `owl:qualifiedCardinality "1"^^xsd:nonNegativeInteger ;`
	output[50] = "owl:onClass saref:Property"
	output[51] = "] ,"
	output[52] = "[ rdf:type owl:Restriction ;"
	output[53] = "owl:onProperty saref:hasTimestamp ;"
	output[54] = "owl:allValuesFrom xsd:dateTime"
	output[55] = "] ,"
	output[56] = "[ rdf:type owl:Restriction ;"
	output[57] = "owl:onProperty saref:hasValue ;"
	output[58] = `owl:qualifiedCardinality "1"^^xsd:nonNegativeInteger ;`
	output[59] = "owl:onDataRange xsd:float"
	output[60] = "] ; ."
	index := 60
	// add specific field names as DatatypeProperties.
	for _, item := range iot.Measurements {
		index++
		output[index] = "[ rdf:type owl:Restriction ;"
		index++
		output[index] = "owl:onProperty :has" + item.MeasurementName + " ;" // item.MeasurementAlias, item.MeasurementUnits
		index++
		output[index] = "owl:allValuesFrom xsd:" + xsdDatatypeMap[item.MeasurementType]
		index++
		output[index] = "] ; ."
	}
	return output
}

// Command-line parameters: {drop create delete insert ...}. Always output dataset description.
// create timeseries root.datasets.etsi.household_data_60min_singleindex.DE_KN_industrial1_grid_import with datatype=FLOAT, encoding=GORILLA, compressor=SNAPPY;
// ProcessTimeseries() is the only place where iot.session is instantiated and clientConfig is used.
func (iot *IoTDbDataFile) ProcessTimeseries() error {
	iot.session = client.NewSession(clientConfig)
	if err := iot.session.Open(false, 0); err != nil {
		log.Fatal(err)
	}
	defer iot.session.Close()
	fmt.Println("Processing timeseries for dataset " + iot.DatasetName + " ...")

	for _, command := range iot.TimeseriesCommands {
		switch command {
		case "drop": // timeseries schema; uses single statement;
			sql := "DROP TIMESERIES " + DatasetPrefix + iot.DatasetName + ".*"
			fmt.Println(sql)
			_, err := iot.session.ExecuteNonQueryStatement(sql)
			checkErr("ExecuteNonQueryStatement(dropStatement)", err)
			for k := range iot.Measurements {
				delete(iot.Measurements, k)
			}

		case "create": // create aligned timeseries schema; single statement: CREATE ALIGNED TIMESERIES root.datasets.etsi.household_data_1min_singleindex (utc_timestamp TEXT encoding=PLAIN compressor=SNAPPY,  etc);
			var sb strings.Builder
			sb.WriteString("CREATE ALIGNED TIMESERIES " + DatasetPrefix + iot.DatasetName + "(")
			for _, item := range iot.Measurements {
				dataType, encoding, compressor := getClientStorage(item.MeasurementType)
				sb.WriteString(item.MeasurementName + " " + dataType + " encoding=" + encoding + " compressor=" + compressor + ",")
			}
			sql := sb.String()[0:len(sb.String())-1] + ");" // replace trailing comma
			_, err := iot.session.ExecuteNonQueryStatement(sql)
			checkErr("ExecuteNonQueryStatement(createStatement)", err)

		case "delete": // remove all data; retain schema; multiple commands.
			deleteStatements := make([]string, 0)
			for _, item := range iot.Measurements {
				deleteStatements = append(deleteStatements, "DELETE FROM "+DatasetPrefix+iot.DatasetName+"."+item.MeasurementName+";")
			}
			_, err := iot.session.ExecuteBatchStatement(deleteStatements) // (r *common.TSStatus, err error)
			checkErr("ExecuteBatchStatement(deleteStatements)", err)

		case "insert": // insert(append) data; retain schema; either single or multiple statements;
			// iot.SaveInsertStatements()   // can write multiple statements to disk for review, but execute single statement since somewhat faster:
			// Automatically inserts long time column as first column (which should be UTC). Avoid returning huge result. Save in blocks.
			const blockSize = 163840 // safe
			nBlocks := len(iot.Dataset)/(blockSize) + 1
			for block := 0; block < nBlocks; block++ {
				var sb strings.Builder
				var insert strings.Builder
				insert.WriteString("INSERT INTO " + DatasetPrefix + iot.DatasetName + " (time," + iot.FormattedColumnNames() + ") ALIGNED VALUES ")
				startRow := 1
				if block > 0 {
					startRow = blockSize*block + 1
				}
				endRow := startRow + blockSize
				if block == nBlocks-1 {
					endRow = len(iot.Dataset)
				}
				fmt.Printf("%s%d-%d\n", "block: ", startRow, endRow-1)
				for r := startRow; r < endRow; r++ {
					sb.Reset()
					startTime, err := getStartTime(iot.Dataset[r][0])
					if err != nil {
						fmt.Println(iot.Dataset[r][0]) //<<< Integer type not picked up.
						break
					}
					sb.WriteString("(" + strconv.FormatInt(startTime.UTC().Unix(), 10) + ",")
					for c := 0; c < len(iot.Dataset[r])-1; c++ {
						for _, item := range iot.Measurements {
							if item.ColumnOrder == c {
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
				//insertStatement := []string{insert.String() + ";"}
				/*err := fs.WriteTextLines(insertStatement, iot.DataFilePath+".iotdb.insert", false)
				checkErr("WriteTextLines((insertStatement)", err) */
				_, err := iot.session.ExecuteNonQueryStatement(insert.String() + ";") // (r *common.TSStatus, err error)
				checkErr("ExecuteNonQueryStatement(insertStatement)", err)
			}
			fmt.Println()

		case "example": // output saref ttl class file:
			ttlLines := iot.Format_TurtleOntology()
			err := fs.WriteTextLines(ttlLines, iot.DataFilePath+".ttl", false)
			checkErr("WriteTextLines(ttl.output", err)

		} // switch
		fmt.Println("Timeseries " + command + " completed.")
	} // for

	return nil
}
