package graphdb

// Version 1.1: Run curl in-memory queries against the (local) GraphDB instance 'merged' (loaded from merged-ontology.ttl)
// Test the URL-encoded SPARQL query string from https://www.urlencoder.io/ 	Search for 'exported'
import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	fs "github.com/dgnabasik/acmsearchlib/filesystem"
)

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

// All ports must be open.
func testRemoteAddressPortsOpen(host string, ports []string) (bool, error) {
	for _, port := range ports {
		timeout := time.Second
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
		if err != nil {
			fmt.Println(err)
			return false, err
		}
		checkErr("testRemoteAddressPortsOpen(connection error)", err)
		if conn != nil {
			defer conn.Close()
			fmt.Println("Opened", net.JoinHostPort(host, port))
		}
	}
	return true, nil
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

// curl -G http://localhost:7200/repositories	downloads JSON file to /home/davidgnabasik/Downloads/repositories.srx
/*func main() {
	if len(os.Args) == 1 {
		fmt.Println("This program accepts an ontology file as input and for each Entity finds the best match in the current ")
		fmt.Println("GraphDB instance based upon a predicate similarity score. It then outputs the new ontology file.")
		fmt.Println("graphdb.json contains the program parameters except for the local ontology file.")
		fmt.Println("Example: ./similarity sarefcore.ttl")
		os.Exit(0)
	}

	stats := ProcessOntology(ontologyFile)
	similarEnities := ProcessEntities(ontologyFile, isAuto, graphDBparameters.SimilarityCutoff)
	DisplayProgramStats(graphDBparameters, stats)
	similarEnities = ChooseEntitiesToChange(similarEnities)
	fmt.Print("Total number of same-named entities: ")
	fmt.Println(len(similarEnities))
} */
