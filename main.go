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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/dgnabasik/netcdf/graphdb"
	"github.com/dgnabasik/netcdf/iotdb"

	"github.com/apache/iotdb-client-go/client"
	fs "github.com/dgnabasik/acmsearchlib/filesystem"
)

const ( // these do not include trailing >
	SarefEtsiOrg  = "<https://saref.etsi.org/"
	DataSetPrefix = SarefEtsiOrg + "datasets/examples/"
)

var NetcdfFileFormats = []string{"classic", "netCDF", "netCDF-4", "HDF5"}

///////////////////////////////////////////////////////////////////////////////////////////

// This struct is returned from FindClosestSarefEntity().
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
	var xsdDatatypeMap = map[string]string{"string": "string", "int": "integer", "int64": "integer", "float": "float", "double": "double", "decimal": "double", "byte": "boolean"} // map cdf to xsd datatypes.
	output := make([]string, 256)
	output[0] = "@prefix s4data: " + DataSetPrefix + "> ."
	output[1] = "@prefix example: " + DataSetPrefix + cdf.Identifier + "/> ."

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

	eva, _ := FindClosestSarefEntity(cdf.Identifier) // error

	output[16] = DataSetPrefix + cdf.Identifier + "#> a dctype:Dataset ;"
	output[17] = `dcterms:title "` + cdf.Title + `"@en ;`
	output[18] = `dcterms:description "` + cdf.Description + `"@en .`
	output[19] = "dcterms:license <https://forge.etsi.org/etsi-software-license> ;"
	output[20] = "dcterms:conformsTo <https://saref.etsi.org/core/v3.1.1/> ;"
	output[21] = "dcterms:conformsTo <https://saref.etsi.org/saref4core/v3.1.1/> ;" //<<< SarefEtsiOrg + parent class

	output[21] = "### " + DataSetPrefix + cdf.Identifier
	output[22] = "s4data:" + cdf.Identifier + " rdf:type owl:Class ;"
	output[23] = `rdfs:comment "` + cdf.Description + `"@en ;`
	output[24] = `rdfs:label "` + cdf.Identifier + `"@en .`
	output[25] = `rdfs:subClassOf ` + eva.ParentClassIRI() + " ,"
	// Add standard Measurement properties
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
	for _, v := range cdf.Variables {
		index++
		output[index] = "[ rdf:type owl:Restriction ;"
		index++
		output[index] = "owl:onProperty :has" + v.StandardName + " ;" // .Units ???
		index++
		output[index] = "owl:allValuesFrom xsd:" + xsdDatatypeMap[v.ReturnType]
		index++
		output[index] = "] ; ."
	}
	//<<<< first review all existing examples.
	return output
}

// Access merged repository in local GraphDB; use merged_sim_ndx predicate-similarity index.
// Because the repository is simply an aggregate of the 13 SAREF ontologies, have to run the curl query 13 times.
// An exact match will return a score of about 1.0. This is the only function that calls the graphdb package.
// # $1=https://saref.etsi.org/core/AbsolutePosition
// curl -G -H "Accept:application/sparql-results+json" -d query=PREFIX%20%3A%3Chttp%3A%2F%2Fwww.ontotext.com%2Fgraphdb%2Fsimilarity%2F%3E%0APREFIX%20inst%3A%3Chttp%3A%2F%2Fwww.ontotext.com%2Fgraphdb%2Fsimilarity%2Finstance%2F%3E%0APREFIX%20psi%3A%3Chttp%3A%2F%2Fwww.ontotext.com%2Fgraphdb%2Fsimilarity%2Fpsi%2F%3E%0ASELECT%20%3Fentity%20%3Fscore%20%7B%0A%3Fsearch%20a%20inst%3Amerged_sim_ndx%20%3B%0Apsi%3AsearchEntity%20%3C" + $1 + "%3E%3B%0Apsi%3AsearchPredicate%20%3Chttp%3A%2F%2Fwww.ontotext.com%2Fgraphdb%2Fsimilarity%2Fpsi%2Fany%3E%3B%0A%3AsearchParameters%20%22-numsearchresults%209%22%3B%0Apsi%3AentityResult%20%3Fresult%20.%0A%3Fresult%20%3Avalue%20%3Fentity%20%3B%0A%3Ascore%20%3Fscore%20.%20%7D%0A http://localhost:7200/repositories/merged
func FindClosestSarefEntity(varName string) (EntityVariableAlias, error) {
	eva := EntityVariableAlias{EntityName: varName}
	isOpen, graphdbURL := graphdb.Init_GraphDB()
	if !isOpen {
		return eva, errors.New("Unable to access GraphDB at " + graphdbURL)
	}
	var sarefExtensions = make([]string, 0)
	for _, value := range graphdb.SarefMap {
		sarefExtensions = append(sarefExtensions, value)
	}
	eva.ParseEntityVariable() // Assigns eva.NameTokens.
	similarOutputs := make([]graphdb.Similarity, 0)
	for _, token := range eva.NameTokens {
		for _, saref := range sarefExtensions {
			entityIRI := saref + token
			similars := graphdb.ExtractSimilarEntities(entityIRI) // []graphdb.Similarity
			similarOutputs = append(similarOutputs, similars...)
		}
	}
	if len(similarOutputs) > 0 {
		eva.SarefEntityIRI = map[string]float64{similarOutputs[0].Uri: similarOutputs[0].Score}
	}
	return eva, nil
}

///////////////////////////////////////////////////////////////////////////////////////////

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

// Critical error aborts program.
func checkErr(title string, err error) {
	if err != nil {
		fmt.Print(title + ": ")
		fmt.Println(err)
		os.Exit(1)
	}
}

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

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

///////////////////////////////////////////////////////////////////////////////////////////

// Source file types determined by file extension: {.nc, .csv, .hd5}
func LoadSensorDataIntoDatabase(programArgs []string) { // [0] is program name
	ok, iotdbConnection, config := iotdb.Init_IoTDB()
	if !ok {
		log.Fatal(errors.New(iotdbConnection))
	}
	if len(programArgs) < 2 {
		fmt.Println("Required program parameters: path to csv sensor data file plus any timeseries command: {drop create delete insert example}.")
		fmt.Println("The csv summary file produced by 'xsv stats <dataFile.csv> --everything' should already exist in the same folder as <dataFile.csv>,")
		fmt.Println("including a description.txt file. All timeseries are placed under the database prefix: " + DatasetPrefix)
		// xsv stats /home/david/Documents/digital-twins/opsd.household/household_data_1min_singleindex.csv --everything >/home/david/Documents/digital-twins/opsd.household/summary_household_data_1min_singleindex.csv
	}
	iotdbDataFile, err := iotdb.Initialize_IoTDbDataFile(programArgs)
	checkError(err)
	session = client.NewSession(config)
	if err := session.Open(false, 0); err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	switch iotdbDataFile.DataFileType {
	case ".nc":

	case ".csv": // use xsv tool to output csv file of column types.
		err = iotdbDataFile.ProcessTimeseries()
		checkErr("ProcessTimeseries(csv)", err)

	case ".hd5":
	}
}

func main() {
	if len(os.Args) < 3 {

		fmt.Println("Specify 1) path to single *.var file, and 2) CDF file type {HDF5, netCDF-4, classic}.")
		fmt.Println("3) Optionally specify the unique dataset identifier; e.g. root.datasets.etsi.Entity_clean_5min")
		os.Exit(1)
	}
	fmt.Println(ShowLastExternalBackup())
	inputFile := os.Args[1]
	fileType := os.Args[2]
	identifier := ""
	if len(os.Args) > 3 {
		identifier = os.Args[3]
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
