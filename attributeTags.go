package main    
// attributeTags.go program outputs IoTDB commands that add IoTDB timeseries ATTRIBUTES & TAGS. Use UPSERT to change values. 
// ExecuteAlterStatements() no longer has to be run.
// https://iotdb.apache.org/UserGuide/V1.0.x/Syntax-Conventions/KeyValue-Pair.html
// Setting an alias, tag, and attribute for an aligned timeseries is supported as of Nov. 3, 2023 (v1.2.2).
// CREATE timeseries root.turbine.d1.s1(temprature) WITH datatype = FLOAT, encoding = RLE, compression = SNAPPY, 'max_point_number' = '5' TAGS('tag1' = 'v1', 'tag2'= 'v2') ATTRIBUTES('attr1' = 'v1', 'attr2' = 'v2')
import (
    "errors"
    "fmt"
	"os"
	"github.com/apache/iotdb-client-go/client"
)

const (
	attributes  = " ADD ATTRIBUTES 'datatype'="
	tags        = " ADD TAGS 'units'="
    unicode     = "'unicode';"
    alter       = "ALTER timeseries "
)

func combed_iiitd() []string {
    statements := make([]string, 0)
    const prefix0 string = alter + "root.combed.iiitd."
	var Energyuse = []string{".Utctimestamp", ".Energyuse", ".DatasetName"}
	var Kilowatts = []string{".Utctimestamp", ".Kilowatts", ".DatasetName"}
	var datatypes = []string{"'longint';", "'float';", "'unicode';"}
	var energyUnits = []string{"'unixutc';", "'kBTU/ft^2/year';", "'unicode';"}
	var Amperes = []string{".Utctimestamp", ".Amperes", ".DatasetName"}
	var currentUnits = []string{"'unixutc';", "'A';", "'unicode';"}
	
	var EnergyuseList = []string{
	"lecture_Building_Total_Mains_Energy",
	"lecture_Floor_Total_Energy2",
	"lecture_Floor_Total_Energy0",
	"lecture_Floor_Total_Energy1",
	"lecture_AHU_Energy0",
	"AHU_Energy3",
	"Lifts_Energy",
	"lecture_AHU_Energy2",
	"lecture_AHU_Energy1",
	"AHU_Energy2",
	"AHU_Energy0", 
	"Building_Total_Mains_Energy",
	"AHU_Energy1",
	"Floor_Total_Energy1",
	"Floor_Total_Energy2",
	"Floor_Total_Energy3",
	"Floor_Total_Energy4",
	"Floor_Total_Energy5",
	"Light_Energy",
	"UPS_Sockets_Energy",
	"Power_Sockets_Energy", }

	var KilowattsList = []string{
	"Building_Total_Mains_Power",
	"Floor_Total_Power1",
	"lecture_Floor_Total_Power0",
	"Floor_Total_Power2",
	"Lifts_Power",
	"lecture_Floor_Total_Power2",
	"lecture_Floor_Total_Power1",
	"Floor_Total_Power5",
	"Floor_Total_Power3",
	"Floor_Total_Power4",
	"Light_Power",
	"UPS_Sockets_Power",
	"AHU_Power3",
	"AHU_Power2",
	"lecture_AHU_Power0",
	"lecture_AHU_Power1",
	"Power_Sockets_Power",
	"AHU_Power0",
	"AHU_Power1",
	"lecture_AHU_Power2",
	"lecture_Building_Total_Mains_Power",}

	for n0,_ := range EnergyuseList{
		for n1,_ := range Energyuse{
			statements = append(statements, prefix0+EnergyuseList[n0]+Energyuse[n1]+attributes+datatypes[n1])	
			statements = append(statements, prefix0+EnergyuseList[n0]+Energyuse[n1]+tags+energyUnits[n1])	
		}
	}
	
	for n0,_ := range KilowattsList{
		for n1,_ := range Kilowatts{
			statements = append(statements, prefix0+KilowattsList[n0]+Kilowatts[n1]+attributes+datatypes[n1])	
			statements = append(statements, prefix0+KilowattsList[n0]+Kilowatts[n1]+tags+energyUnits[n1])	
		}
	} 
	
	var AmperesList = []string{
		"Building_Total_Mains_Current",
		"lecture_Floor_Total_Current0",
		"Lifts_Current",
		"lecture_Floor_Total_Current2",
		"lecture_Floor_Total_Current1",
		"UPS_Sockets_Current",
		"Floor_Total_Current3",
		"Floor_Total_Current1",
		"Floor_Total_Current4",
		"Light_Current",
		"Floor_Total_Current2",
		"Floor_Total_Current5",
		"AHU_Current2",
		"AHU_Current1",
		"Power_Sockets_Current",
		"lecture_Building_Total_Mains_Current",
		"lecture_AHU_Current0",
		"lecture_AHU_Current1",
		"AHU_Current0",
		"AHU_Current3",
		"lecture_AHU_Current2", }

    for n0,_ := range AmperesList{
        for n1,_ := range Amperes{
            statements = append(statements, prefix0+AmperesList[n0]+Amperes[n1]+attributes+datatypes[n1])	
            statements = append(statements, prefix0+AmperesList[n0]+Amperes[n1]+tags+currentUnits[n1])	
        }
    }
    return statements

}

func AMP_Vancouver1() []string {
    statements := make([]string, 0)
	const prefix0 string = alter + "root.AMP.Vancouver."
	// These are the only datasets that have been entered so far. They all have the same timeseries names. Only Electricity_I has type FLOAT.
	var DatasetList  = []string{"Electricity_I.", "Electricity_P.", "Electricity_Q.", "Electricity_S."} 
	var dataSetTypes = []string{"'float';", "'int32';", "'int32';", "'int32';"}
	var dataSetUnits = []string{"'dA';", "'VAR';", "'VAR';", "'VAR';"}
	var TimeseriesList = []string{"TVE","CDE","UTE","EBE","MHE","UNE","OFE","CWE","DWE","WHE","B2E","EQE","BME","OUE","RSE","FGE","WOE","B1E","FRE","HTE","DNE","GRE","HPE"}

	for n0,_ := range DatasetList {
		statements = append(statements, prefix0+DatasetList[n0]+"UNIX_TS"+attributes+"'longint';")	
		statements = append(statements, prefix0+DatasetList[n0]+"UNIX_TS"+tags+"'unixutc';")	
		statements = append(statements, prefix0+DatasetList[n0]+"DatasetName"+attributes+unicode)	
		statements = append(statements, prefix0+DatasetList[n0]+"DatasetName"+tags+unicode)	
		for n1,_ := range TimeseriesList{
			statements = append(statements, prefix0+DatasetList[n0]+TimeseriesList[n1]+attributes+dataSetTypes[n0])	
			statements = append(statements, prefix0+DatasetList[n0]+TimeseriesList[n1]+tags+dataSetUnits[n0])	
		}
	}
    return statements
}

func AMP_Vancouver2() []string {
    statements := make([]string, 0)
	const prefix0 string = alter + "root.AMP.Vancouver."
	var DatasetList1  = []string{"Water_HTW.", "Water_WHW.", "Water_DWW."} 
    var TimeseriesList1 = []string{"DatasetName", "Unix_ts", "Counter", "Avg_rate", "Inst_rate"}
    var dataSetTypes1 = []string{unicode, "'longint';", "'float';", "'float';", "'float';"}
	var dataSetUnits1 = []string{unicode, "'unixutc';", "'liters';", "'liters/minute';", "'liters/minute';"} // Water_DWW does not have the last measurement.

    var DatasetList2  = []string{"NaturalGas_WHG.", "NaturalGas_FRG."} 
    var TimeseriesList2 = []string{"DatasetName", "Unix_ts", "Counter", "Avg_rate", "Inst_rate"}
    var dataSetTypes2 = []string{unicode, "'longint';", "'float';", "'float';", "'float';"}
	var dataSetUnits2 = []string{unicode, "'unixutc';", "'dm^3';", "'dm^3/hour';", "'dm^3/hour';"} // NaturalGas_FRG does not have the last measurement.

	for n0,_ := range DatasetList1 {
		for n1,_ := range TimeseriesList1 {
            statements = append(statements, prefix0+DatasetList1[n0]+TimeseriesList1[n1]+attributes+dataSetTypes1[n1])	
            statements = append(statements, prefix0+DatasetList1[n0]+TimeseriesList1[n1]+tags+dataSetUnits1[n1])	
        }
    }

	for n0,_ := range DatasetList2 {
		for n1,_ := range TimeseriesList2 {
            statements = append(statements, prefix0+DatasetList2[n0]+TimeseriesList2[n1]+attributes+dataSetTypes2[n1])	
            statements = append(statements, prefix0+DatasetList2[n0]+TimeseriesList2[n1]+tags+dataSetUnits2[n1])	
        }
    }
    return statements
}

func AMP_Vancouver3() []string {
    statements := make([]string, 0)
	const prefix0 string = alter + "root.AMP.Vancouver.Electricity_"
	var DatasetList = []string{"B1E.","TVE.","UTE.","OUE.","OFE.","HTE.","HPE.","GRE.","FRE.","FGE.","EQE.","EBE.","DWE.","DNE.","CWE.","CDE.","BME.","B2E.","WHE.","WOE.","RSE."} // no "MHE","UNE"
    var dataSetTypes = []string{"'int32';", "'int32';", "'int32';", "'int32';", "'int32';", "'int32';", "'float';","'float';","'float';","'float';","'float';" } // ATTRIBUTES
    var dataSetUnits = []string{"'VAR';", "'VAR.hour';", "'VAR';", "'VAR.hour';", "'VA';", "'VA.hour';", "'dV';", "'dA';", "'Hz';", "'DPF';", "'APF';" }    // TAGS
	var TimeseriesList = []string{ // does not include Unix_ts & DatasetName
        "RealPower_P",
        "RealEnergy_Pt",
        "ReactivePower_Q", 
        "ReactiveEnergy_Qt", 
        "ApparentPower_S",  // should be 'VA'
        "ApparentEnergy_St",// should be 'VA.hour'
        "LineVoltage_V", 
        "LineCurrent_I",
        "LineFrequency_f", 
        "Displacement_PF", 
        "Apparent_PF",  } 

    for n0,_ := range DatasetList {
		statements = append(statements, prefix0+DatasetList[n0]+"Unix_ts"+attributes+"'longint';")	
		statements = append(statements, prefix0+DatasetList[n0]+"Unix_ts"+tags+"'unixutc';")	
		statements = append(statements, prefix0+DatasetList[n0]+"DatasetName"+attributes+unicode)	
		statements = append(statements, prefix0+DatasetList[n0]+"DatasetName"+tags+unicode)	
		for n1,_ := range TimeseriesList{
			statements = append(statements, prefix0+DatasetList[n0]+TimeseriesList[n1]+attributes+dataSetTypes[n1])	
			statements = append(statements, prefix0+DatasetList[n0]+TimeseriesList[n1]+tags+dataSetUnits[n1])
		}
	}
    return statements
}

func ecobee() []string {
    statements := make([]string, 0)
	const prefix0 string = alter + "root.ecobee.household."

	var DatasetList = []string{  // 984
	"00248d5f9ecd01a008b95d6f5a79688db7f8344c", 
    "0031fe0263b18f5fd70c0e47892a5ad0daf5db2e", 
    "0086a19bfa211168e593e005a9436e9a3a20c05a", 
    "016bd8bc8d5fe75d56dc4faca6d2f1c7006569c7", 
    "018efe0684a343a852761d26465e8fb35f9605e3", 
    "019808cb633c44fe337886560229984eccd33293", 
    "01affd4535251d36798e77d8d4c9b8aa04e7a830", 
    "021903cb53abd34c6c9dbf023e6b86b529dfecd9", 
    "0264a3cb5be95c038997756d7c78bc4f63322279", 
    "029a49840e4e5006744c53547109938e5b7e840f", 
    "02af31b89b3e20c7e547cc2a578fdf1ef7f03fa0", 
    "02e6b274aa1e4299f11eb6c35782aa8459fead5d", 
    "03128ce9b7f3b0f05f9329e18b9c3c34086af11f", 
    "037bc3ba361dcc4c0efabcc5dd14e8bc03a1c845", 
    "03faf8d14ab9e8ffbda6c5c92d7db6680d435184", 
    "042a0fc857d89c9339116ebeb8e880fca088cdfe", 
    "0480b6a0849bc0bc4bdb7297b76efa909413a690", 
    "04c1688e1db57c17a8e446756470c9da66517210", 
    "04d643e30f4e945ae2aa8c2bd68f8aabb799a827", 
    "051db978e49eef2cacda77318e4d37dc64fb1214", 
    "0572d17663dcb3c97b77dcb60e1bf2264a864500", 
    "059c9439f3b98b114f740e7254ab05ec14c17ed8", 
    "05a192bbf8452cd5d2083f8322247b0ccbde4346", 
    "05d0a623d5a18bd8501b57134b7df5996a1f6bd7", 
    "05ea7541f542d6c002cf255b0912573b67ae7e05", 
    "05ed0cced4eca919f5ea98fea85eb8dd9da6c7c4", 
    "070c1daa1b0ca8689b747a7340e15c1d110c186d", 
    "073164298b8d29c2a5073b4e2d7381997eb39065", 
    "075f381257a76a7b200a3b27e6f0bb848f200a3b", 
    "0764f96250e17991b48a4c3d6c2700a465886115", 
    "0786550b4ebb66cc96d637f5edfd439e8dc1b1fb", 
    "079dbf1f9a8292a0c58bd10943a993f2cddbebb9", 
    "0813295b9225921d0cd79a9cce26c39877beeffd", 
    "0856d074b5403f48da22b8296fd47abfdd641283", 
    "085d122132c8c868e365711341b84ff982513fea", 
    "089a646f83cc38e0ed8596e8c97842c1a78dae1e", 
    "08b79eee18a7729a1eab9a5d7de655f0222859e6", 
    "0903cb2f431a08ae88c155aedab51cdddaa46588", 
    "092c6423bc0d0f9c88b3ae8594b294f42b15014e", 
    "095b77a01955fe42ad62acf2a0f0a27b2dc78d4d", 
    "097b6a00a386cddc9875708259e8e0da42288434", 
    "0989e11ec297ac4f75458f584368234adcc3e295", 
    "099b2f39b91e68e2f56fc828372e34f2c284e801", 
    "09cdee8fcba3b53b106a59c7d582010090d3b7cc", 
    "09e9641600e7f47be104d0a862bae97359b31b5f", 
    "0a3821e8043d8d0cdc26f293e4ded53cf5252754", 
    "0abee7c5442074dfa5592ad378cda30c9b619cb4", 
    "0b50750888a5272d1a1281be5f759e53cb6f48f2", 
    "0bfc76cb98e8bab850bd1f3178a8ad16a20b7bff", 
    "0c497fc8bf140a5e4c5230500f46039202533dc5", 
    "0c77ca33e7c3d8f024d634396bd3e35a6bc7836b", 
    "0c98cb812420df12138626fa8f8195e63d76ecf4", 
    "0d4b507ec7fbc050d9c8b4e0d1d39763f56825ab", 
    "0d53a915adfea6a65fcab7799e2eb2ca24a4aa31", 
    "0dd2b19ce244c578eef2efeb5947199d68c3dcec", 
    "0e11682dabb47af786ddf577a35cf847417b11ba", 
    "0e580082a387b676cf3da5534cf3fafd3a0a30a2", 
    "0e611e55b3d88695c728b12f19fe846d263b6010", 
    "0e704ce36b5780dad3fcc0ab811899717b197c73", 
    "0fef91bf17b4e08091bd1b0a5c67d3f1602cc7f3", 
    "10181f3c0e5f276b174aab8e4adc0d98ad8cbed4", 
    "105336e2648bb6c5c25bf0ff7444f15098fc23b4", 
    "108151f3715570b1ead10f6c9fe678c421f5a246", 
    "10f4767bbb3c90b9acb50ad49bde77675502e290", 
    "11544e87f7dd636e8be28cdbb471c7adf27500a0", 
    "11aebe03c73b26dc8fcd6b31f26b4b6318ba1227", 
    "12139e1ce991807444c5949a4acf15197014ebd1", 
    "12a3884ac08346f3b260c452557b518d7e288d31", 
    "12a5b7dbf0228865e89e285d58a7bbba24a19994", 
    "12c997ccf6d688db599a8ee74357f840314fc029", 
    "12eee8f533f07ddaf8f0aa19f022d4cb99119648", 
    "1308c3274e4039ddd75ed164e956f68df69eb491", 
    "138c6d18991a6057e91897c82d0b99f41dd72bfc", 
    "13f4645afa75ce036326c7162b496597c0db15c0", 
    "1433abf330426bc411b7638a5980f431adec79fe", 
    "14495446e98ebc7e5a928e859da085bd3fd2f8c5", 
    "148a643533d0f6f96cadeb47a10e21a185ff2de3", 
    "148af4b204762f88d6ae5289cb43cf29d825499e", 
    "152aae3e7e88e7cd44f3a645a03162d4ad93dc28", 
    "15793cbbe677f79df49e9decd5083393deeb1f5d", 
    "157c5ac5e50489f830a27e748d0fcb22c6695ecd", 
    "15aa67befddfd1962d97666234cdd7e2a0d3286b", 
    "15ecc3f9b8860faeb418f7e647ee5815301877be", 
    "18f547c09e08b3f02f4d97769c143c5471ea66aa", 
    "193591941af07ea2011d40f43a5f6f61fca5b4e2", 
    "194cc222008f5ad42dba217a8596c572e0e7a4fc", 
    "199c305996aba2474b3e6e4fc90ebb0cd8f69c32", 
    "19dd9220c9758178fca924081fff309005836d61", 
    "19f3db28c3fe69771cff4bfdea7d5840b8e9c102", 
    "1a202ddd67ed411c2c4702db56ea269141f07c91", 
    "1a4a24f131b9f3dae630c17a2bb5ae8b0d6e60e4", 
    "1a5a5549b4dee3f06f1c852a422185b37808ed38", 
    "1a5d41e18118bf7c2a68531b70dce70daa6d1253", 
    "1a7284c2f73d61910bf737bd9e3334ffe5e8ec8d", 
    "1abbd6036f5a7774f784fe5b5954fa0c5a2d4e32", 
    "1ae2f0b5236f49c2c1fc06825e55bcdf6fabc454", 
    "1af8476cec59d3415c8e02f9e1ad9c878f454282", 
    "1b24154de3844b617b1dae284314f87a638475b1", 
    "1b334fbcb5ebf43a7dac3b36bd31499bc6ca4524", 
    "1b5a150985b59bda06f744956d46daca9a72643d", 
    "1bb9e84eda1e18b449959cc13edf446290b81f8d", 
    "1bbff5d4878007fedccfe1c3f9db74a8ac013175", 
    "1bdebf3af1bd154d9e69e32fafd41124cab5133b", 
    "1bdeda0b28cc3af2e6118f9c5b58627d2cebe26c", 
    "1bf818f8d03bd1579320560d581fc23aa51fa42b", 
    "1c663548bb009350a6735052fbb59515bd0582cf", 
    "1c8d179cfe8f3c99d802d0a22924fd47aedc62fa", 
    "1cd769414b06d63b499d52a7a5bfb1cfc6a0e694", 
    "1d1350f3bad7723103d06c5ca5d7a1e96a9ba291", 
    "1d351304694b12dd18c61bfdb1134e2410d8619c", 
    "1d36c8597f31277a8aa9019cb05332e13bfd3c99", 
    "1d5350e1e800373fc7e81410a4c6882a9eb89fab", 
    "1d642131e3347c25ec68866f73fee2807bee43d3", 
    "1da65029bb9709eca79606697e1fa2865bee7cd9", 
    "1e3b2c2255c88a8959ac02dabf03630797d73cbc", 
    "1e84303a06ca02c8e2fc204748376538a0197118", 
    "1e9a38c7269a57c7da8f84f1f2ff5cc1ecf40943", 
    "1eff4404948aa8642391fbd7b32d823e91e10fce", 
    "1fc5e3eb478f6ec01891f5b4d4a54d5c3619a33c", 
    "1fd81ec92eb93cda017111dcbd030c72c4df985f", 
    "1fe4f60fb7e6a4001369883bf6fedea3b0dcc7b8", 
    "20d881309e1cfb17e12cd2de88b4d3a55498801e", 
    "20ef371235d72f826fad236e15fa268f09e906f6", 
    "21728933dfe083377219f99af9e2a3ba1c8f6260", 
    "21dbbfa0547ce1c2e1b5e08a4be088a9f75ceb77", 
    "23abb913f6c0076c331bb426546b7c844591aa90", 
    "23ce9aac75c6cfbbd483de5ac312da58af7f1144", 
    "23fb19b13c7ee30535e795964c9c060ddd15cb1d", 
    "24203c53f4f496e729c152c9e6269206f2937d3d", 
    "2427170502b45542cc4da7730c5819f351742b20", 
    "2434703e9463c7bb8c00fc1aa63650c110c046a2", 
    "243bc934faf940e50cce81df3d2c15e61962b4d0", 
    "244c53b7feb27f012c56783813c76e69ca450eec", 
    "246d765c408e59e509904a9c2b19e7d4174bb28f", 
    "24c41db211fff03aa07ce7d21e081d2e96520d35", 
    "2507a1033693f9bbf8533781e3a30bfed2f0720d", 
    "250da26e53ac7252b8eb8d4ab53139d06e93442d", 
    "2556aacdf40c9c733fcea0f73b20d4be7b12e697", 
    "256aec54451d8b80fa49f5a0a02b591fa68543f8", 
    "25b677637c6c50460988e5d0627cbecd66f715a7", 
    "25c16f64f075b01206f3218e775ec1198abeeac2", 
    "25cec3811f3eadc6602886d0bfea67706b0b521a", 
    "26529a236eb2fbed97103cfbb42a11a61dddbeff", 
    "275953e3199786bf5fd9671c3b075c696630872d", 
    "277020cc51e703ed4c25488eab734f368ba7f511", 
    "27a7f2d7c288bfd66713d80cabfbc58bb4266036", 
    "27c6ced2864f46ec4266968c70a356cf4a3246af", 
    "281e8a479e56679c452a8d5a0a64bf75082165a5", 
    "282c423d0c4bd4317cc69394f53215e8f81147a7", 
    "287a5b943d20da415cf95b38fbd5d30c4bd928ed", 
    "288331f865ff492d05cc91c93c29ceb04ceec085", 
    "288a1d0a5a03d61098d9e9917d6aa9c08501f5b7", 
    "2894c0267a34b818b6230a2c323f08f9ffcee208", 
    "289aab7b8812ad58092a9b413466a1fe36d1c0d6", 
    "28a2929464685987e73e82be9a135b7f3b46278d", 
    "28aa87af191ebfc1dd89d582bdcf3ce31cc2e042", 
    "28c75933c55c0d80bcc6d7ea47441b73159f3eac", 
    "28d592d208b353012e0feac93a56acaef8b7c01d", 
    "28ef5d83d78f0d3a02bac36f39bc9cedcd95b3b7", 
    "292d3952d42693915f7fdf2633b5bd409d779825", 
    "297406ca57dbdec6dc16e8ff27b3ae4bae90b2e3", 
    "29c94766d76ae5e31d3931a827be0a132d622af1", 
    "29fb9a9101a35a499b415dfe22ee4263795c9deb", 
    "2a5d32a3a48801d51a031d6aafb238446256fa99", 
    "2a6123ff1cc8d3e017ae411f86184a6a8d5abe7c", 
    "2a9695131f6e022486fc177d0ef58fd21a6bb83d", 
    "2aac3da31c8941497974ec5741c8ead0328c6cce", 
    "2ae450099baf9b59f4e68ef35924da90a22fefe9", 
    "2b49de0cb01f22e76e48509a9778b8e989f2dadd", 
    "2b9997f1e62a44637081abc12c465d1d18701e6e", 
    "2ba815f52392b90f615de7cde594ab68941f35b0", 
    "2c21f7f7b6750267c50e30a1880e7d799d17564a", 
    "2c62cf81710e636a1b219f93726eca143ed9f833", 
    "2c8ac82a96d5e5bca2fda7f9cc345f0f7df3865f", 
    "2c9c66944274189c656f8eb44ac32d3ef97ce125", 
    "2cceb7f426f847d09f9c4d15808e24884bb3dbf8", 
    "2cd9b4495c37909cec4e613feab8e83bb6db5eff", 
    "2d38850220ae12164a0f8936124e876ce5c99c33", 
    "2dab3d384b3686f19232d11e345df2baf64196e7", 
    "2e27c95e0ef6121b1c9cfc131bdc32a6a8ef827b", 
    "2e5253925556eb33f248d545e63b581dc4883d34", 
    "2e903f1bc5591aa5c2dfa70c552aaf3c14cc99c4", 
    "2ef54171332f5247da485b1d188b3ba6c790c2de", 
    "2fb9e1128bde3c9dea444a49995272f206b9a904", 
    "2fee820c1bf460e5e4fe023025ea57907517aa8c", 
    "30840e60a9b21de95d384f8605215be42520955e", 
    "3084a2513f97abbf18c32431fd73f9939d1f2867", 
    "30a9376c9e15af791b229a5ebaf65fa778cbae3e", 
    "30dea40b28d576c4572ba8c550717a6b110b14c3", 
    "31512d836e026f04e8cecb29df4e53b3bc6c28f0", 
    "31d69bc8608de8f8fc07bc2ce129c78345aee909", 
    "31f3ae4c1f30dbec8c3a56210869b20f6734d81c", 
    "3200f9aa03bfedc80533b273d6dc2de839f8343e", 
    "3205295dc1bfc8e9b1e8a390fb533423a7784021", 
    "326446986718118d9ee661fc2454553a05a2e925", 
    "32aa88ca3d31d2ffb5d935538eacb451a8dfd5ed", 
    "3330b36e84946c9914554a0aa703f7ec3e801580", 
    "337232f0c4796c0c9cdd74074d56236c64b7e996", 
    "33bf4cfaa32417b651b95765889ac3a2257c890d", 
    "34494e977223d70aaf836898aad0a90805ef0a5b", 
    "3716a7155c828e81a2b995bfcfe57061f62b5fef", 
    "37328623b2aec95389a27b9a1513977bb076d806", 
    "374a70b53685c4b6988718b39f39697a27f66742", 
    "374da9821f8c49b5649fb0a89fa59ff75f550649", 
    "37a943cec374e616124363228347691cd55caebf", 
    "37c6a46a1799f4dd32182c29d0866fe3eb61ec60", 
    "37fe990fd5f674d2cfb8cc6a61c94d1cd8b3d073", 
    "38785eabc4894e93cd52cc62a1f8bc2d631dae42", 
    "39554dbaffe43fc81d573ae1c661b5967b4a7fab", 
    "396c45ec6cbb879667558e3067c9ed14dba83786", 
    "399ecd4df444814f20baa88770c6910d76e6d4dc", 
    "39c991d5b944dc7615ac303bdf0524b46f5361f9", 
    "39deeb0a4368d23715b79d46b0fc94e15e4fe0e1", 
    "3a185dac0e2ae048c84cdd6b353cf3a00b2745c7", 
    "3a3dea005d9efece9b4590001963cf771502f01c", 
    "3bb854e778566d855b5b56e131bd19fd52e06b02", 
    "3bf1ef689e6e86aaf4f0d4b1e9ccb246abd911a2", 
    "3c4050b4572ceaf59b853656beb7b1c15d33955f", 
    "3d28bff0893aaa9ac9bbe407f7ca3aae0ba9f0c5", 
    "3d2aabf8e6fd404b9ead2803ec5f383599fc82d3", 
    "3d55516ea02d1c281e1fd594f3edeb3e6d00a0f0", 
    "3d5eccbd8e92ebf693b08aa1e93658fc21a6bf56", 
    "3db0f665a071b6e22b34f27e06078c02c52519f7", 
    "3e2ed4f58b62b926fe6e430b0f93791f1de9988f", 
    "3e554dbe8b8cb0ac2a39c4a523eed6e9c414d01d", 
    "3e71b8b933c67434dc3704a964c7ff2ed3246f6d", 
    "3e78e93f13e0cc3933c194414eb27a7500ade8bc", 
    "3e93689b9ecf59b1526eff3789d3f31335593203", 
    "3ea18703886d759d0d1a7cac2d20217887f1318f", 
    "3ed6dbe1e452ad82db3ab2e65a624a9520580e44", 
    "3edd4c4e7479992fd081b1869cb02be7b2eeca66", 
    "3f60f15577980fcf5de6a3ccda51d12cc9b39368", 
    "3fc5765ce5f0e0b9b962f6ce8862a012548f4539", 
    "3fd30c803609b3df19aaa03e5972170c8ff3af1b", 
    "3fdc25dd3ef73dffc613d22b27ad9fe688405e73", 
    "4055e23b8ef06cc04aaa7a75e5f16f284d79f6e3", 
    "40565d5d3fdad3e4e297065db1020440c1ea163c", 
    "4073d19cdb079acb57e8df88f588e51068f0eacc", 
    "4143ae9a7e295644412db0dfc6c4dab26b7f467c", 
    "41889b5047b907fed1a6e0ffa5005be907a91d89", 
    "4229576796a96d0115376a69215795ed09bfaaa0", 
    "4231d87ef238acb06d34c45696e2eb6557bca6d0", 
    "4234c00c716610fdd6ed369fe2745b59c0149312", 
    "423b2c3aae8b3d02c64162bad603f1d66ff38e58", 
    "42516c71bf14292012bf26a67b627f8f24175bbf", 
    "430492be5866eb99e3fb62b61881a406fa2d1a5f", 
    "4355b44054c75755b0f2fa5f6b8b50e76fa84241", 
    "43c451cc56a76a7fc18644b8bb4dc2fbc8bfa6b8", 
    "4418e808efbc5e12b2fed1650dea07943832d838", 
    "4453aa86e80238779093b05cb0e5fc8ea8477538", 
    "449cadb9595d2b66094b772feffca56bfe5504a0", 
    "45facc80e3e239c3bbe1b2e98617311839277b86", 
    "4615c60c2adc4594e4b8681cb38c72d716ce1d5c", 
    "46921bc0beb001f36442aa64984ba25c79aea5d8", 
    "46b2df7a951d1951dfdb9233196dbe737bdf06ef", 
    "470de420db4a8798d3f426835540a36e9bafebe4", 
    "4768cc6441d447efd1c1f9c58b82db5a23d13494", 
    "477fde0fd0dbb80282b91d4395b4361c1d03b1fc", 
    "47df226423fcfbfa9514a155cffdbf620e8e2b41", 
    "48320522d01098b44ee1f01945be6214ff799e31", 
    "48525570d19f19768eaa9dd5114fccae0272143c", 
    "48b1c2fdfeb6c67261d601a74421767e97027244", 
    "495d527bcc2ee274c97107fc3fb89c31a1bc072a", 
    "49e0c769b3e1f3484b9b97ae52beaa5b233c24fd", 
    "4a2b05c8bbbe70b3b910e6093240df3f052a6bf8", 
    "4a48049e9e1e696b374f72f0aed97b78f41e8379", 
    "4aa5a7aad78e6d23aac2066c8008a72c65601521", 
    "4acee82221432569d6f8e6d6670baf0b729c71be", 
    "4aef89feb17446571e35957dd35a07454f25a225", 
    "4b00f7e016c29f5bfac16660105cf812fab684b5", 
    "4b3304e94195cdff8bf480f05bc6232ca6d5b21e", 
    "4b531b7c49fe40228f0b78610d7cfc82138c91fb", 
    "4b5ef64446328a58c329c3cf65536fa74e947116", 
    "4bb29108a8d21504b72444e213a342f37a99dde9", 
    "4bfa2676b8f64a767939e4f6d2c27e3476b3640b", 
    "4c08054e66edf3631d009b21da3ce33d6ed998ed", 
    "4c1020cd1060ffc3f0753eb12e0b5bd7bdc428a4", 
    "4c2c2a21d5ce9c26bd9d83dbb0d54a414ef7b25d", 
    "4cedcc06d39b466ddf9024ca4e5395226ebdc75a", 
    "4cf6883014294ec2f8acde5ad8c56f07a76c2744", 
    "4d5229210f7057b556e6719dc8e700ca63f9000a", 
    "4d6d6f5f2867360f8348ac82b5be2990b136a82a", 
    "4d9819b90cd270284e3f5ddf966804abebeb1f53", 
    "4dc49fa2c8c8a0d7e0ffde0eb0d84710b8682451", 
    "4df7bb33a5775d41ca299f14d48f48b1a8cd593c", 
    "4dfbfe7becac59f04ed21ac280a5c1813432d60c", 
    "4e4ec6b73a9d38bd32493e165a92d0a446cb0c2a", 
    "4e641f0eb69d858c5557140910b32add3a561d70", 
    "4e753e612c52af44429671b34deeb6e4008eadc2", 
    "4e75deb4f6b2672eec34456733b5bdae68757b2c", 
    "4e9af9fe89906445429bd6f7e5aa1dcb9cf95750", 
    "4eb247ae008c5661dd6a0d153b116f1e6c2f21a4", 
    "4ec49f41e1c5ad16f9de908d9293c1c25ee177fb", 
    "4ed567bb79bf71cf3b66d520f983726a47a1958b", 
    "4edc6809c83d3e761e66aa1587433d085f829f87", 
    "4f116543cf72f519ce316abf67668a323e2d35a8", 
    "4f30566484037218deea6211ab039b2afbed0aa1", 
    "4f3311d34e15206c6f49965de5dd63dab95ee9e5", 
    "50722d8b9e817337837e29f4740792eabc6d0950", 
    "50880ab09b9e8193be97fcb173872c72ffb95f62", 
    "509f36d7ab23676b4b4c1e29bebb70a123375b42", 
    "510b1757bcd106e8482319ec3a7adeb8a2a5a086", 
    "51b5909a9f3d147da894507fb079985bdec51b86", 
    "51c47359af69ca1a0265b6e683d86c395fd3bf8a", 
    "51fcc79a8bd0e2ae9d69614517f6a439e5d30fb6", 
    "525dc90b751c9c19fb22acc98a6bea8b2f03d914", 
    "529dec636afc2f8cbd0ede56fa77ae84b4b0f788", 
    "52f63f06a2c8588cbc7ad0d64434f8dbc3a60744", 
    "5359492e514ca0c0879992ed3ef1b08656191d0a", 
    "5363d9dd711e035069d04f47b696e64a904653b9", 
    "539a825d8804a834249c6a75839edb4afa39bed8", 
    "5407fb65f91d232061fa5125e8e99b97905a8889", 
    "540e57c62f76b4b128bcefe3479f0fed1bf09fff", 
    "544d14687400d1c3d6fc617df7791f9ca18b9ba4", 
    "54685271cf20e19c66fc2195f917c5bb422f5923", 
    "54aea850550cdcd2560e191fbf9863a795cab4b4", 
    "5528129c9faf727bfd4cc69d063aa6327b99b6b4", 
    "556800ffcf5e1302c152072ccdfbce81992e1896", 
    "557c8904a5e56e77ab76ff94e0f1db172a0600b1", 
    "5582790c3d13323ffe5b461720ffa5bab5bdd73c", 
    "55f02ed8cc7c1c8e617fad0b2129297144505655", 
    "56046246619b4597926a04f1a19b098e2d2b86d4", 
    "565c42ad0fbd84b4f806387d9779f8ad5e768651", 
    "56aba2d13addeb70313f44105784d7e8ac180b5c", 
    "578f1e3d4d3f45a4afa24a9c977838f7779e7cd9", 
    "5792c7cf3cc0911fc35ca38564933a575e926a40", 
    "57931ffdcfd93d3b9984289edb5cb189479a47af", 
    "57a5b23e00ca326d2d27e43435954732f9d46c3d", 
    "5892da9c71f0dfd12b5aac8448f534b84b091721", 
    "590ceb02b45bd8841c4a02bfbcf7a6e4c2c7e344", 
    "5935b2c4fa885443123433e98034db16bb26a344", 
    "5979adb0223b722e92ec7a188d3e31bd2ccaee48", 
    "59932fc0cfa1745477359d6170bbd5ba2f56a7be", 
    "5a07a6212f13745a84d40652ffbe45d566785a77", 
    "5a0ee5e005e338a133c64fc3dc8c9cc918f689b5", 
    "5a46a20ec7c22541e5aeb2b01c5499ded430e025", 
    "5a4d7242f5be96f58cf9e3d00793d914c8dfe8b3", 
    "5a90e67184c927fc56e238e04a64f39f853d4215", 
    "5b0b6146dd8a1700f9c101e768cfe19a0cd20299", 
    "5b65b4d9407c93a321c7ef51336fde964f6ff5da", 
    "5b92b326fb7bf2604e18e5b3e64eecee111fb01a", 
    "5bd9dc1cb2d7f84446fb86db80fc92f5616eaaaa", 
    "5c42b02a41d8f136e069c982567c5fa11466acf4", 
    "5c4d52677cdf6c98754a07035357a79567d1213b", 
    "5c716b1afa3f5f4c03c4fb835e4c99366edca866", 
    "5cb661fde4fb3dae4758502d48121bfa994072ae", 
    "5cf094c45aab65916ebc5d428921d7f6cd158a85", 
    "5d47b2f400faad132798094a0788c4c1c268bc7b", 
    "5d55a5ededaf4a3dcafdeda0f45191d71ad90672", 
    "5d9d1017a1631bf1c39d2a7fcc13519448791b86", 
    "5dbc14d5a89d1d72c41009c708963b366015d014", 
    "5df756f4f2c9812b4e67736001967989c4395701", 
    "5e0437f10583b60c0b3f758644a3327ad6a2da1d", 
    "5e53e5a2984e99858d22116798786165a6f63d48", 
    "5e92b5b3124356b7c6ab78dff60c6dec962b7ffa", 
    "5e99c976b482de7d21f4ccf4af7ba45934ebf333", 
    "5ed85566d06aa9da16866ab9f9b8491190542507", 
    "5f9033d76f65da8a8463389c0048678e64708013", 
    "5fbe16f368f36267078916b81f924e73c0c0470b", 
    "608e5ce820d2dec588d148e34c03a150a289d670", 
    "60ae478791c39d26c3b049c6f2eaa7557ae7416f", 
    "60bdc8df51e99a46dba36820852b5591d92038f8", 
    "60d65b76223845593aac97c011d6aeae4e3978ff", 
    "612b22e29a8be6bc9b4dcfdea02a8b2f21087cbc", 
    "61a0bdc02dbf688d8e0843ae403ce1a6d3b1d74a", 
    "61b013ea16c0e26d6e58c2cf9ab0cf68908a3980", 
    "62dc184f1004f72020986af5185c100bf7a3192e", 
    "6388dec833805cd824420104ba2d7105ae89c031", 
    "63932c4ca7226eadb203799bdc4be11e60e03525", 
    "63d90391571f3e34f21d0c1c636cacb570138a15", 
    "63f88165a74a46629064e8b6dde3742a618bdf16", 
    "641ba3790bb159d68cd32bcf21106c774c0dd684", 
    "645e8f355d901e0eddfe3d0bd618ac7e265a233a", 
    "64eb8123425332b3597221ae96586b2d29073238", 
    "65de326ea362bc635280f0d037870a0d876393e2", 
    "6657ca78073ee8c8dbb89ae9e2144589838286b4", 
    "667bd6a45154cfabf09086c61178c5923f458d9c", 
    "66b724608f2db1bb475d512a66843b9a908aa787", 
    "66c985ff473d0766c435b8c30568c9526ad75f70", 
    "67199c6bc364985d00c764432d276cd7cc64dbc3", 
    "6756777b3a4d2291d49d72645ff4a72c1e948f8e", 
    "676be21b483939ead67ec02c443b3f78f86fe3c5", 
    "678f6f9f53e0e09d36b8a3d86d9d5391a2857edd", 
    "679a58cb02ebb4d498bdcf6c37a0171bcc30f418", 
    "67e83df80731309d249c606c1e5cdcca53ffde85", 
    "682db6b32db01c138c878b9d5501495167a1c669", 
    "68d655bc704363dbd19552db6ec79a0024ebd620", 
    "68f1921ef5ecb1ed1c520f16bba5e9f1a78bb6ff", 
    "68f3719e3656c4a47e3c7e97d54c662c889359d0", 
    "6929663acaa4cacae84d2a64be69b0ba61363b8c", 
    "692a51b3168c55447d213b0253c1a9bd5e4084fc", 
    "692aba427ece327a86b82af99aa27aafa76be1f1", 
    "695aa250cd36f5ad7a6c753860d5d2d99cde52de", 
    "695c9b415f6750814f2a7dec792fb32715b4f793", 
    "69c8d7f0087dab71e710fcfa078a5d8b7747d850", 
    "6a942a36458a7a7552a29db1d62efaecffd51e60", 
    "6a9d4987db1bf2167d8e6f50dd0aa5b1e3729d81", 
    "6aade298e2ca2f02956ddb599ec2dd0434382ab9", 
    "6ad6fa64a019c7c722e158b8f5c83935b1fce282", 
    "6b3becf2efe01fda0e5ff78422d37ebafc6252c6", 
    "6b5d215fcde16dd70efc175989fce658150e7ca6", 
    "6bc88c9ef04d75659330959ffa783099cf51da58", 
    "6bc8b9744ea239291542bf32818b859c998acdf1", 
    "6bfc3511a7f67035cc4bcf43af09a0662e11dc91", 
    "6c7faa24c5852d6a734e0fb174ad1ce994b934a6", 
    "6ccbbaec7c28d8e699eedea0de77314f39e80194", 
    "6cd74bc06ab3f75de5f65670bcf5e059cfba6e86", 
    "6ce2a7e58bcd6cb514d175fd37cd1281d43f6b65", 
    "6d196d3c0bb8007f78dec89d16afc01bb9fa921e", 
    "6d3c1e3ad1c2ac4b321967d7b84189850b3608d6", 
    "6d689ee1fb10888d9d677dc132240c4a36a6a879", 
    "6e79eaafa27ed04a91a49843cb5885265bf5da08", 
    "6ea24193880a19459b147c6c895d8473dc9d5e1e", 
    "6ef27ded179e1ab0714e2a93692750576cf7f02b", 
    "6efbfe237d9dc65a6cb4f40d8e8955320aae40ed", 
    "6f68e4a6f83105ef323b41f5d9c5b8bc9dba9697", 
    "6fca882f977e8fbead7f443a7592b8008e4a4df5", 
    "704b8f53ce95813e65cbfb80c4b4d720eb02aa22", 
    "7077d9497002c86e085928845f49e05eb7447d19", 
    "70d4656c917457350cb02439a8599073454a989f", 
    "70ee026e97250369a035e86cdd2cb8b6363c4062", 
    "7122c1df681c696fb42ece32d2b75d4eb0912f94", 
    "7126765f4bf5a86808fa130d706491cbb66a0a53", 
    "7184eb81f69b8d0e3a93d737039ef97ff41fbaae", 
    "71add3495982224e35b7482e51a9926e0ab5111c", 
    "71f9b1368134fc5b776e834b2b9d3ad1bb8b0457", 
    "72125f219860104af69731e528af94d0b0ce8b8b", 
    "72c36873c56d4cad56f1135fdcd805c97a883868", 
    "72e8117b2f86de8d420fdce557c76f1f6b501192", 
    "739c548d08e2874759eb4389b51222e1a82cc5c6", 
    "73b0ff81643d2c5e6811fc14ef28fccf4dcd629e", 
    "73c14df9e85d4f34a9b712d4c3433a3ee255a348", 
    "742d08f5a447b46dcb2c11320e39daf7b48345c7", 
    "743134f45035f595f340ccf76974c5ab29fa21ec", 
    "745cc425b9b233f55a3f5e9aa92c5b0947eb320b", 
    "7462033f403813c567e5e09bb9d4f601677e4243", 
    "74ab6ac22ed5f0b88dd4c299c2f090848f313c85", 
    "74b90d93fb5c7ca4c8957367d30be18c402ff295", 
    "74da9ea358564d193c298b3585808820aceefbe8", 
    "7509afaf5fa6284666655ff875d20c147571f50a", 
    "759381bff7ef9e8be2582be8f402eff8a6bbba03", 
    "75e9158e4d11033f5baeb1a71a12ff2b0d6b2bf9", 
    "7680f3f9cee4056ff6d9b2aa17e02b1514945e38", 
    "7680fd94b428c30949ed9d0d98a8b6dbf3ea501f", 
    "7685019d26c362603d0521702836b8139164383f", 
    "76c1869f09c926c4089d96e4c9d7a8b740a8de46", 
    "76f078ae0a296448d6ca15f4e1941cfe62cb5bf2", 
    "77a33250828c64d7bfdbe474388e5137eb462df7", 
    "77d7149bb75162ba79697efd0344abdaab7983a4", 
    "77ffc2ad44ee03c4078d2267b4159ed2c2bc2b46", 
    "78132f752dc068093f24178814c9f2f19ca750f7", 
    "781b9326075e4a23258dc56386a33ba4bd41018a", 
    "782a0f1406813288225dbc513b7fd79fe71138de", 
    "789de60bcd513ba0991a0957455fc323f41d6915", 
    "78e30f71d7847381ee35f9c0e6b6791b4ceb4820", 
    "78fbaccb8b9728a31f2270a12f47963512dc144c", 
    "797148b7e6166b3f209fd4c0e28879032e42dde1", 
    "79738edc48a244378f218e3bebd343f48f6f1d96", 
    "79b88292741727ebda470f2240b3b685e7361918", 
    "7a3866ac415bdf3f50d0fe08cfdc8caa422be1de", 
    "7a643c54a304c813123c59ba1c9be987ce4c9d93", 
    "7aafed6e8b8021af4eae9d3e22948e745cc2457b", 
    "7ae50896960439cee2e8b90203ff951de0bdad5d", 
    "7aeda94a8e72481fc70f610d12adc9e65163cb37", 
    "7af991d0ecf8a50462800313800cb5481bf5427a", 
    "7b529c3153ec337ea7a3bbf7acd5fb85477137f9", 
    "7b72741179485c4edb31c91b395321410a20a334", 
    "7b80de7a59982189c63edaaa84c1ea1f6520f52c", 
    "7bf689f879bcff5166a2a8d9455022c24d819a2f", 
    "7c8c0fad61df010949700bd438513de526363f5f", 
    "7cb8f868bfd1edc1483116c26de7ebbbf156bbe7", 
    "7cbcd3e82126fe2ad62cdfae4f95b4feb831417d", 
    "7ceb6c3e1c5efa63bb23655799045535866d8cbe", 
    "7d25e3034f5521495a67ca7f01e968a58fb30bb2", 
    "7d4c4532ef28d375eb9e3c7337b9c1678695f77f", 
    "7d735aa4ee71b3128c5642b9fb1f69b1719473d2", 
    "7da53a28577796c9e05d58193d1d0eab7942fcc6", 
    "7de66579da95c19d99bd3bf6461d38951efb41f2", 
    "7df2d632a74c004899faeebe646d1adec0db1bea", 
    "7dfe3a062e974367a0670830fdd0d67ab451e688", 
    "7dff646eee5767da91ecf2e6fec9285ef006327f", 
    "7e9b9a5eb412969fa77e5f01f9e87e913779b1eb", 
    "7f3dee94aa1575dc38633a7f2709900b22b5b8a2", 
    "7f6532d1a1ec725d98416c37afc8800f9b6e1f95", 
    "7fc80666fab6a451cbda1606724eb362257b9522", 
    "801f0080997649f75f2395ede997e5f8dd9ddc78", 
    "8060dad72ba20e41e7c4e366b2c3dc3cecd1d451", 
    "8072d237a87557c8a89e9319cfa890d691a4be02", 
    "80d15ac14382a681f6df8e90bc61e75b88361ba8", 
    "80ddc5b9981b22e6eb3650086a76541dd2942485", 
    "82406381aa7669a8b106410caa9e5cf12f81afe5", 
    "827694ec8a6be84d54e62cbc3c5f63c229b97f78", 
    "829fca517b7bae93ad90827c34db32131d98b422", 
    "8300714491933bbfc8580fd5bf4ce554fcc65d48", 
    "839856e82cf763aa6b049d8df6b9f6e348994c62", 
    "839ac15c32b0736d5a5b6a9fd4138370fda4a427", 
    "84020b6726854e117d82b192df3daa5826c642c8", 
    "840bdc7f3592183ab996e94f87ff195761b78e99", 
    "84127b54e15004056f64475452705573e1ad69f1", 
    "846316a35a6871593ef4d143f219adb1774c54e1", 
    "846e638f09be05254c4de1629385bc2fa1bf97e2", 
    "84ea0ded2290ef2d4e509787b1d84252ab7e40ad", 
    "84eb9a4d63bc3f2d022cf04b454b70261a78ba04", 
    "854f3d4bf64f9871e93db21338ff86bb490067d4", 
    "85522c653fbb31c258b7e704bf8dcc8c4a5b24cb", 
    "85a12cb7a06fd43651f7100c6103f5c940ee302f", 
    "85a259ad871a5ba49e764494fc9965591ca26bd3", 
    "86c89a6018e2da25ec2fa125204095e807883fca", 
    "86daffb39d79173179845b220fe5c2ac8f667191", 
    "871fd86588178bc0380b258d1d033136e9f8bcf2", 
    "875a3cdc77e2df015da4300113db9732bf93feb9", 
    "87efc401af016a6bacf72ed4b9c1d1c02ed09378", 
    "880f9098081bb5da9ea51d792ab11ff50ce50c85", 
    "886cd272f22116497a91d7e077d875cdc56a451e", 
    "887df94c224c26000fa822f8322d983db93f48fb", 
    "8976d5ff40772b2ac65c2899511735099ff0eaa4", 
    "89b4b9da76548630020d4cdaf09cbbbb571763cb", 
    "89e99f5d0b641659ad82403b937037cd9d70803b", 
    "89f34156ee1c3a622338fffa9d578015567b084c", 
    "8a1aef8ca5fe64a259aae0ff05575bc16bdaf900", 
    "8a21140d014657a4178c2d3c3a42580285ce617f", 
    "8a286cf21d1623252fbb5314e2cbc095ba5681fe", 
    "8a2c27a62c904032afc1005b0599644ad069c082", 
    "8a3abc11a9d542837d1cf6c8ffd5e74600bae610", 
    "8ac95dd77589dffdf92245870106d2b4dc48592f", 
    "8bc8949978fa149112aeb47e711a932c9a4d3801", 
    "8bcf6460ed62459c8aef604062dce341c704fc89", 
    "8c00595585a1a78c68b559941d288305247e8e25", 
    "8c7b1451ae818ef1034f213037ccfd3eaa30b385", 
    "8c9124725f26ad5a4eb5da226468a1c39272996e", 
    "8c9d8dec56cbf2dc9d72aed13b626a519dff1529", 
    "8d022a06776b803936630d0738df14334e916d72", 
    "8d0d5eb2616df52a7a1063e06273680b901eb6e1", 
    "8d80031d3e9c723e5c634538c15eb0d9e72c06ce", 
    "8dc6bedcaee8930a396972b12a138712e3e8d170", 
    "8dd570997692aec5b47c3cb4ce1d3d750581fa49", 
    "8dd59ba3cfc57d54a191695ab44aa08f4a9f0f2b", 
    "8e143b4f0aac24487f483d850db86c5ef91a2c3a", 
    "8e40ec6d049d9d715af2cd1e163dcefc321f8d86", 
    "8e672948e44b620407aa1e8dd5108c936f07ab3a", 
    "8e6ee2be8e92d0b4aa02d46bdf23ec51ae6ca5bf", 
    "8ee17a0b72d80a88494fe3001238868571e6807e", 
    "8ff8a09563ab73f9bde40eb25b4c86d6e09ef2b0", 
    "8ffee741d25bb4d66b39cd9e7daf9ef149f37a22", 
    "9037007659854bcce76361ca002afe8a7bbcc9f6", 
    "90c13f92b8b91a309b7f846d23266de306a58969", 
    "90e3e532a1987795c276b2acfb7a76900ec13f54", 
    "911050326b5d1dc22355c885a13d2f34d9bd3d5b", 
    "9126ac2e81d2dbf2e078ce25b982b9fdd8fec1e3", 
    "9127eed017b4a8765d9df2ce815abcfa96c486b3", 
    "91c838e3f021718c404ecdb1fc4a02a201cbc341", 
    "928128025298802ad3109b12ba9319c7080d44b9", 
    "934ddf938c89a13a59d36bc028ff3c5795f43a40", 
    "9362251feada05deded398ec36c70a7facb485dc", 
    "937cafcc88913bc76252656403218456747ba0c1", 
    "939379134b349c0021deae8d32fdbbbeea4e2a31", 
    "9394f4d79fce9b17e246da0fb8ef851e1f719c91", 
    "93c09bf13359b8e057e044296684c387346d4737", 
    "93cae7e52daf3e7425ad29d60e6f562a6c40eeb5", 
    "93e5e06a6ad20ba17ed653094ea461232ec39153", 
    "94b8472cd436a0f109b210ea02c408cc4d96b10d", 
    "94bc5d3409cc28b6fbf1b6294cbb9721c636975f", 
    "95013a09d6eecf3521687378aaddebb7866fb0b6", 
    "9507028ae830144fe8362f92dd218fa2bbd1b86e", 
    "9523a71aee99646bf0b336d9157d8b606379ab78", 
    "958dfd89cf3a3d2be0a26115e2d64dc7b1432ed8", 
    "967d48400da0de504928a10429e686281cabe338", 
    "9695f5507551a2422c65ab251fbb26e26596c770", 
    "97500b0c4497acaf2b6458460a743b611ad42ae1", 
    "9756691b953c0b5f597d9500f3a256647f3c720b", 
    "9775b654f42215e3b34b6a14030d2dba87c5885c", 
    "97e74659051d77357d377e15fe840c82c6db4867", 
    "981f8afdfacc2d9d6c159fee576a6d8d4859c07f", 
    "982bf8bf40519f75754c081d60b6d0565c2271f6", 
    "988ddbac16dd8367e14711f00bfc891382c6dabe", 
    "99205450f3343f99a4001c03c23048e8849e2d18", 
    "9983e9bb6ebab4a8736e7d2de01d8d22d1285ccd", 
    "999048244d53884a021d28cdbfc0a56da93d5aa4", 
    "99a4ab8d18853d5d647f2a36403122539652706f", 
    "99bc691a46c5e3d5bf250e368f124ced6b8b0e9e", 
    "99d14fb5b0015e5865a970626cfee90537cc16bd", 
    "9a021df59bf5af19ee5cf10284aaa1cb903391fa", 
    "9a58584f6c9c0adf27ebc705601930031eadbe43", 
    "9a6060912c91d6dd1991455d32b7831265c42c5d", 
    "9ac141c6209a9e0f954c7ebf239ba2cc322d7e8c", 
    "9b21816745063f4e258c3714dd5a87b263e4afdb", 
    "9c1cfa1e275c9ebf90f5a6cb1507382244b915ce", 
    "9c772d6bd9554b6e23f8748bcf06ccc36e4ea3c9", 
    "9c8d056b1a7a8be02fdaf55c310a424b1be7d97d", 
    "9c8d106c797179c4116105fb2468f23fa16dafc9", 
    "9c9caf927b4ad39fd6a45a32378197391841b2a0", 
    "9cbbaddf9f4f51e3a54a1950e4419f8a994449b0", 
    "9cbf100dbf6a4e7d8b5535fab97405944a29008d", 
    "9ce2fc9f8fb90473f004f7975c2e2b85f8f2bd6d", 
    "9ce85092ef9fa7beec8de9a35e3256c63859e71f", 
    "9d222ecaaf3ac0c8d75c85461c76ac27b15690e9", 
    "9d2e4da1068fac71a487c3b55558d4bbb5627750", 
    "9d7a74ed2165e133a68cd775acba3b58291536f1", 
    "9dd564ae8450aefe40585e336270f99c2401a417", 
    "9e2104eaa873ccf82f23d10a378ac77aca9a52fd", 
    "9e539eef404fe3d8c62c693dd0cd6f97ff0fbbeb", 
    "9e5687eeac865771d7e059426045431c14730a68", 
    "9e786bd2db3bf893e7f86cb66f2fa97fb0cd51e9", 
    "9f2456bb3c3f7996c5a60580de99abe08eb2f1c1", 
    "9f3db9369b4df392930c316277a6ab2e0be3c0dc", 
    "9f94aa4747b04dc58e05401475645bf770158e2e", 
    "a0667c30c3404642ee1ed39318cc1d4371140a0a", 
    "a0a5280b09ee943b4f4e1ccc5c657e39def6242b", 
    "a0ac49d63a1299776c35f0b3c8d122a65f345457", 
    "a0e5a03f169e483354e68d255d7e4dbea01843c5", 
    "a11bd3f149859bd6be96b75202266cb8452f0577", 
    "a1355257bba77af6b9d238311d86bcd4487defe9", 
    "a1a1bd958f2edd7587be79690bb1298eabe265f9", 
    "a1f907eb94353fdb150443bcc004ed65cbc11480", 
    "a21c953644c4975a141fe509357aae6be54dfb20", 
    "a21cdcd90d5ac63349c34b57223d5aec5c176091", 
    "a24495eebdfb7c3edc44a1c27684a523eee592bb", 
    "a2ad2263f6974fcd2a22c1c825abcc7690926749", 
    "a2d48e9a93dc992f770d83aedd406f96b732fa37", 
    "a30fc4aef03a0aaeedd5ad2fb33cedcfb4f7917a", 
    "a31beecb4c377e73a30ba2216ec4c46d71f6f2ed", 
    "a35a82773da2ae98eef44794473f2737191ac12d", 
    "a3696918e7d4dfaca99b181102237a0153d0b311", 
    "a3add799cd4c1c9f2e3c5d644908d581005fbab5", 
    "a3d79c2cb9458d3e49e6d1993c38edd934c6de69", 
    "a3f5cac1b37b20f8da29579742bcee6c8e737af0", 
    "a41b13b5bf3bc0c0a5685df30a34e6a4d98140b2", 
    "a43a6330dc5316b7bd7e578274ea6d3fbf1c3f60", 
    "a482837bc15e78fb837c590e6cb83456363a8b96", 
    "a4876a47c6170569dcd5c4b3601d969b268563ec", 
    "a4999d8ba998a69ce802799d25e324a70bf1a7f1", 
    "a49af942b433e2d109c5abee801265af7290855f", 
    "a4a3fb75c643f1d991aae43e399ca4504fb874d2", 
    "a4beb37921396bc3292c5553550bcc223da946e8", 
    "a51952b3178b9035839b67a4c9800caf33952d5b", 
    "a580c8bf72fd42452c0fb6acb35f57bb508ec77f", 
    "a5d9a17f442472da1d25f8499742baab315fdb6a", 
    "a69e76cf1fb84ad975f6f08b88beccb9da924a69", 
    "a6c3b3b8f5751bf7275db6d88d001b2602f342e9", 
    "a6edd9478e9733af498fe8d05190b4dbc1a0a41a", 
    "a6f0c33d9ef043c62ed797fd3720a304be754d22", 
    "a711f8ac18d3745316581a713cf29fc4d6dbfba0", 
    "a74d3ebaac49b0a6ae6f97c01bf8cb570081b4aa", 
    "a76a161da4c1ece6bd334129783e1987259c61f6", 
    "a7925b50345e8e7d3ae54e1bb2c27e13f79292a5", 
    "a7d407fed7e161a5eb3f95f148dd08770fb11808", 
    "a8532f51ebff36f99caf30c1c6d7b6b52d9b1cee", 
    "a898352c4c4207aa99cf99d5d6153f4ea31bc2f9", 
    "a89aca4b83adadcfd99ddb49d7a801249e93223c", 
    "a8e43ae6c7f462f646c9de2de657aebde182acae", 
    "a8e4e99dd7b0b2f72d1feee9305038956a46fd41", 
    "a942acd022ba5e24aa618f0fda0169c86c8c6fd7", 
    "a9c199fbf79a083f15bf53dd3ff35e486b9b0cda", 
    "aa05e7f28c29c1aebb1c8714cebd824639cf2850", 
    "aa6281aac6c7f32aee9bb93de1b7629ed8de0136", 
    "aa8f6e0092aa84993560ed25a41711b2132bdc95", 
    "aaa712be8b4627275d3e0facb7955c2b5e24cf76", 
    "aace95b556c7bd2f6bf031eac1316cce7a3b370c", 
    "aae7e958c876d20721d6cd8cb6df96281c376ccb", 
    "ab33df5b9b38955dd524db7c7812440ab1b41fa4", 
    "abadc122f4bf3d86834131c59d587badde358203", 
    "ac1bc13824ce3960c23ae0f83e30c14922d41abf", 
    "acbd7f288f3f8f2713863909d5c2d0259c25eb11", 
    "ad39f0171a3f245381ee21cc47d4aa61f5f3b636", 
    "ad64baaa2732462270a0114f2b384a7ef4195a5e", 
    "ad8ad9eb1cca713c6e17e2480918df705a9027a9", 
    "ada67507c9b21e6fb576f7bc323c0e986df6e992", 
    "adc601e576d30a1d0af4a919e483debfbfa8545f", 
    "aeb1245074d77189573ec9b9d29e2a1b0bd3a6a7", 
    "aeb4ed253d169a89076f734257215739ad921008", 
    "aed0e7a03399c56e95614a3a7006924c3e0b97a4", 
    "afe3974d3ddf8992feab153c1bbe1309106b904c", 
    "b03bc5bd0b6b1f61108fc681d12cadc4e05c6efd", 
    "b04de478f8c3545fc4a12f5637adf3b1b952ab61", 
    "b0531baca3b85d0da92c97e5ce7c48ac5c616d2b", 
    "b0bee85df00eb1a75d63b4a3202f3c7f5370b951", 
    "b12a576c6688021b4aef112764e01c212e7eac9b", 
    "b15f409c51de657ea9b0e31762e3f945cb416428", 
    "b172f3109bd4e144e408ae660649582806e801d3", 
    "b18ecd214e28e58b75932cbe094c729db0523379", 
    "b227718ada66849b1ac6ccc11ec8be4b13249c48", 
    "b254e258b6dad2d2f2d6592d99edb118784be7e3", 
    "b2a27e2ed70203f61c9cfe6c50f1f64dccfaf469", 
    "b2bd578c1355a761a46c5752413ff4c17934f169", 
    "b2fa229c72d29595f519aa2822f7bacfb46529ce", 
    "b300c8b4ce29091eeaa5443eb20099c509959d5d", 
    "b3c39499a744db303fb196978be3db27f38adef3", 
    "b437af5f0611b96255f19289d88d7733ac8c5a99", 
    "b47c175e94fade63c3415ecfc49a4826d8ed8351", 
    "b4b5d7cef54f7efc5fe4dcacfabb6875ee573da9", 
    "b4bf65c61dbc80f1eaf9adda0ef04c9dacc9f80f", 
    "b510c2b4f4a2cff8a0cba279a4f3b9fd8490fb59", 
    "b52bed44ca2b30b4dbdd190d9061a9d8d314b594", 
    "b57dcba90ed02de419ca5882f51bed92881143d5", 
    "b59d158bdc80273924c588b398c0b9374d261f74", 
    "b5ee3d548cf6940ffc6933664768209a60c2fc38", 
    "b626ea55bf1fa0b93a9819fa8bd9e9f61465bb75", 
    "b63829594dd70c7531919ebb92a097132472242b", 
    "b6677ba081b203e5c3214c89b8c1723e5a7478b6", 
    "b69159ac52bf678246c2301b92debe4753d7f0c5", 
    "b6b9f852041b3c85c3734e9fc06e2b6289a70253", 
    "b6ce8d58a18a270f80b48ca4f49a13e3ab5d5f1a", 
    "b6d41554ae88dd9e4da01ae02c01654a3c605bae", 
    "b6fe48a28b66cdd8357bbad70f1c9685c67d1f05", 
    "b70eede4f4c2971a12a76c141fe5c75b275047fc", 
    "b7702091ab5a7697ea199e00811e1d7c3ef9286b", 
    "b820d0b22e062484012a306fba31c1c1fdd27b58", 
    "b83c9fa195eddd0afaa3fb9d3f76f02a1da58710", 
    "b860a0b8b99e663094e15af97ec8e13385615996", 
    "b874e5fe83914e638487a18db3936f8584342037", 
    "b8ef7d8e57aa6b49e22024b2eb634b8c4f89e303", 
    "b90e6d175b22496ed948e26c3319487dd1e3c9c5", 
    "b920fd0c121b2376bcd041fb6018bd2d6354f67b", 
    "ba44182af395af67f5157d40a15c49024f0b74a4", 
    "baa1a00dee3481b103df9b431d84f6ffb65fcf0e", 
    "bbc7c7a51fff3c1bae8cc3980a72195ea841c5e1", 
    "bc281d7edeab4ff3ea1c84143815515e3493a8a8", 
    "be3b32efc243327a7afbeabd36c0fb4e906ecd1e", 
    "be632966631700a34f0d69ec6d63647cc03a1ac0", 
    "be85f91a5ca54b6c5e68743b6ffb97bb117a0dd1", 
    "bef26f5d32e8921c1826ee8def7a2c6ef8cd5f24", 
    "bfcf83f84121855abea28708ec9feb1556f3fcbb", 
    "bfcfc3d4953a2a04831c1410f9b9953227cb8d31", 
    "c03c1e6d7adce6c339884b1d8fc4973433e97107", 
    "c04e745e5084fbaad6e9349ec4eb9aa9f07588a0", 
    "c08a7e398eb607f1060bb5aff8f0108213d78b21", 
    "c0d48ccbe55cc94acdebe3b4b5f35dd85e92b26a", 
    "c0f53755d7213cec28a261e672a9a832ec84ea4d", 
    "c10fd4b2fe2af2e69a4b8d12836bdaf93a0e65a9", 
    "c123e6041e43c04974f3510452f196fef78ac488", 
    "c12d7a742725c60169cf32600d4096dad554f109", 
    "c13ba0262780488ba1206ccca67b0b6c685a3bf0", 
    "c16759aa3d90570bc51d3f67d808bb6ab99a8d27", 
    "c1a27a869a0068979b166ed06b0b4f88d8640033", 
    "c1bf7ba396dcd8d2cf31deb65a30cd0ecd48287e", 
    "c1cd95af8703d82a5cbe413db82c0f3a94c76bad", 
    "c1fd2d2d2e972390793d3ce306c5b0047b2176e1", 
    "c2c3a5856b6b6a25933e1012570f703f40f0f2e0", 
    "c3550c14545df6382be6da12bb2aab14500b1bf9", 
    "c35ba027ae010c26090c89e20ea8a688f82c9673", 
    "c3838cc185a04acad156f8d194f9941d01d9319b", 
    "c389d669ca8045fbf73063e289070cd1320101b1", 
    "c3d4b65d24fa52d7dd10421546e5c4ead7c5575b", 
    "c3e46dbd01e0e442de0c5ce89898cce5456c0f55", 
    "c44020acb65309bb9fe9e595b07d0c196603af5e", 
    "c520d5375278c622eb846c01e1f3ef6c7d4a9bef", 
    "c5482eef7c3539dd05a4c9796bda163fa9bd64a8", 
    "c5bb480d52300e16a73dfff363441b65868473f4", 
    "c63258b2126d7f3a5d9f070aa5c386c7920a41c0", 
    "c63d97280dc8ecb7e6291b827bc76a336513ebef", 
    "c6dfb1f5aa761ddf58da67d7904126c62763262f", 
    "c741c8b624171be616b86a72070a48e43e42576d", 
    "c7ac0ce45168f93ef87c29a9d37d4fadc1683766", 
    "c7ed25789649872520840c4be828bbbf2dc1f5b6", 
    "c80b4341b645ffa026c8a054341c4c0337c35122", 
    "c84e4af50df609c223b90d918e2a3db98342bf07", 
    "c8675fc0ce9a6b485e45540edb9b171268192da7", 
    "c8b9e50408065ab64094ae929f00eb596413b901", 
    "c8c0f51b6023da6201a8760a7d3b1bc5d3d5115a", 
    "c96eea503b215a34d00013088bce3b63f885241b", 
    "c97dbe9cc97d036c0660fead36ee8520f872292e", 
    "c983b00f9db41e2fed445b26a131d4b002f860d7", 
    "c9962b7e0569578741d50644d4f590dc1e6dd233", 
    "ca2fd0d3ce91164be6585d82420259210718a1dd", 
    "caa5ca0dd51adf3c1d814301538113782b273232", 
    "cab0916c4667afa160a82f3437a48742fdc0fc47", 
    "cab6684f858c5bf4512127f5ebf578cecf3976b8", 
    "caec6d1041d10466b789ddc5276bfb4243b14b9e", 
    "cb0950afeb6e5e093ccafb388e54186919a0e904", 
    "cb56cd97279973f2a4ff7d50db61f955a555e249", 
    "cba3a9c1454c41e67c6f5393aba5a0c254109803", 
    "cbb150dcbf001adfe276f721b3704298d9a5c192", 
    "cc6993b491bf7f87a2f67dee207a80e9ebdd299d", 
    "cc92f7b03843b016980e4eb998d7fe59ccd7219b", 
    "cca02722f6da8f6d55cb94c8454a2500e2ef77cb", 
    "cca901614b3fb946817fea8110f0594483c48ad6", 
    "cd4fb2d58e4e8240d1c1ed6fb10a6dfa34aa793b", 
    "cd97632e9cd74d6d6f6ff3e6f8a5501881a3f9fd", 
    "ce240e67b6735f7cfae48651d435eb0285e45af4", 
    "ce2bba4242e55296d263ed4a4153f1d40b503177", 
    "ce3997c83dee7db34e3f35cb52ae1349b34301a8", 
    "cf78a5d8a79881fb05ec3f0915edd3c27bfc0a7e", 
    "cfa5f9f8335043b40fb81cd49d5e6a2e432002bc", 
    "cfad73bfc0fdfe15c1da416eb9ae439d2e96aa1e", 
    "cfe4f297837db2b72a27af3261776fa78d3da182", 
    "d003008c80a727fac6313607c7cf7b85cfbbf125", 
    "d0125888b96b1b08eba41457e9e78f361455652b", 
    "d0152be7a07cadf27eaa39dbc6287526acd4eeeb", 
    "d0a71b27dd075ec1a72af93126107f697d84c3aa", 
    "d0d2bd262585ce87d7f413378cb4762cad9ac781", 
    "d0ecf2cfbd20f9c760b83b31011601a7bbcf5736", 
    "d0f42baeb19d703a8866897e817abad149f774cc", 
    "d11f11a30ff72c4d36ebfab95db2edb674c61f49", 
    "d1a5e747f33ba05c1fc8e489e90bd7ac012f83d3", 
    "d21ce59ff6ab3cf2969a4caa5df28245c5b79257", 
    "d254cca6a538dca8134e99b46f921e76c35be2e3", 
    "d2597e76d647806f403534926f961e4f183fc56b", 
    "d2f2fcaa6ace09000394e2a5f54375164f060236", 
    "d30ec7dda4dc3a9206cd1e0bbde46bfb044f544a", 
    "d31a7a17ff94ef99724f42eb39d671adb253132d", 
    "d32a7ab50809ca99994b4bc1bba64c818e2f1808", 
    "d33b89b3ee96723fbee50e39f87a0f9531f662e5", 
    "d35f4ca95a5187865356828317b9499708876d2d", 
    "d3788b91e86a75fafb9b0682a73ec5039a333632", 
    "d3a444f433cefeacf054a772b78bbf25104e4162", 
    "d436c03f51782f404147e1a02ea82399d13db0b0", 
    "d487ab0725a99d1d3db560876351aaf78d432a3a", 
    "d4bf71aa6709f9e5966cf265e3c8bab2c20ddf27", 
    "d4c167202e11855fc052bcfa4318fea5778ca678", 
    "d4dc8a7021fe705e5df3aaa079da5b74d2ebda42", 
    "d53132aaf83bd8a9869883177077d8721dd686e2", 
    "d60b468f4aecdd39f5b942faf4ae4420d2184a19", 
    "d660892ea137727346096cba5765ec13762fb2f9", 
    "d66566fdf4ccf11ff85b9f13d26810d8f5847382", 
    "d6753e2c7717cda160070bed00183cb722951f5f", 
    "d6bbd110a7f2b6bd493efe6ea2e85eab57c0a2ad", 
    "d6fe826d81d2597e3552cfc204fdedf87d9a4819", 
    "d7ea4cb49b59076f77b682070ae162bcee394a3e", 
    "d8333dbadec4ffa74849d30d1481bfd8eb31459b", 
    "d8585149fc98ec2ba4288731fb04f482146d75f2", 
    "d90264ad1ac2bca1856f3e416310a5faf51aa357", 
    "d90a3c79f2806816bc248b20aa7d96ac40dcaa1e", 
    "d916aed930aabdcaa2f8634d8e86dedc729002bc", 
    "d9b0ecaf17abb73c58eee993a97b6835d263bc34", 
    "d9d77f1aa3604bed04ea621623d553fc323396e0", 
    "da09897f6b67c4511ee33c658ddbdfe3afd082e3", 
    "da1194137046bdf0be965b1f2bf7ae6b61c47f47", 
    "da419115b223768d9b694b2b16113505c5cafe1e", 
    "dab2e4d8af18612a6b957e9adde14fc0b7d0a33d", 
    "db09dc0f31a50bb5fa87ef3b279fb0f6a9079926", 
    "dba3a15f58cd9cb0e6aaf0b31a62c085039896f3", 
    "dc101ca0eb5dc1961857ff898567d6b403d9c633", 
    "dc384a085213ca5e48113f4c64c72db0e03c2f3b", 
    "dcbff6c8cc6b98ca5985089959c36121c0606304", 
    "dcc8f8ac5e6d42db3ceec1bcbaecd20799c00c14", 
    "ddd253ccfb1b0df8f7f2669510685a6582b8a45e", 
    "dded0c052c7b95feb0ec74833b7401c971a0618c", 
    "de20a6b94f18bb19ecfc3a6a2a58afab31fbcf85", 
    "de242b1dfd57ea77084b23659cbea6e91f753c66", 
    "de2ba3a6081134e478e76c4522cc842657abaa9d", 
    "de426655ea431c3ab65f02443bc5533005e937fc", 
    "de461287cdb9549abb0b7082a55ee38851316300", 
    "de6664d82d66f042ac76d3846d77e213c4e00ec1", 
    "de736e0057494ef21988d1f0c8bedb68510cfe10", 
    "de74ecde8bfd81c1a48fc18d08eba8428943026c", 
    "deb485968dbb46c65ba833f63a8b108fd349cbe7", 
    "ded2a8c957ac6cc039cef618e0b7cc69cf0c9b13", 
    "def2dcb86c5fe51792fd3578c1bbaea12c82576f", 
    "df2c4fa56ddfb52f922f8d2d52f88abde7742e81", 
    "df7bc7432bdc0f914f0adff2eccbdcb3ee1e0643", 
    "dfd9c3f7d2df81216e1cd1af62c4ede82aa9555b", 
    "e0af8dc1a0d38067ed9a1ec6edc0c927016f36a0", 
    "e0f0e999ab6c28b6889f27af6781829914ae340b", 
    "e11c6d81e4180ecdb823807cf7d1b10b4bf8930a", 
    "e12b7147f0a5502a5fc48223780d9bccc6b7752a", 
    "e14ffcc21ca71a16d04a8ba96a98bc3a13c4952f", 
    "e172f41eb318c4b7982d4a2fc8f6eee290a1d425", 
    "e195ef3dfcabe8e70690eea64a6fb4cb6d902de4", 
    "e1dffc3256a5312e05cc3b8c0f0daa9b7df4c81b", 
    "e261845478349eeeab049bb2808d65ea4404eabc", 
    "e2c72fb00c09acdeec5c8a6770d9b1ca695a41c3", 
    "e2df8f7c237dea336d9ad0ae2157806af88c7c51", 
    "e2e2331176cce4938a1fbe70991ddb8f588b68e2", 
    "e2eefbdef21d5f752d838973626ca3ed9a880d9f", 
    "e2f2f8e46f732fd6b5639d6a9bc63befff0a7d3b", 
    "e30870e33572a67f89f0dbd9a63eca2af0e4f7a7", 
    "e34138116e7b2ae1f13f289c9ef231bbc22d7836", 
    "e378d2644ea27061990a69844f8cb63f30cd80b8", 
    "e38f69c8f092c09898fd35bc861753424f1e728a", 
    "e433952b0da5e080ead1808268034dba37597839", 
    "e449a6ef6fe3139663181f215e969d3393390039", 
    "e44bf8831c28eee0cf4e746a9e9c618c68dffb5b", 
    "e4608f3911078900b6cfadef199a7d1293cef91c", 
    "e4c10bb8264fb095b88c7233e0426a41a5927ab5", 
    "e5434937229322b1ec7f633956ff572b16226d88", 
    "e55ba798e2c27541aa978c4310439679a78b55ff", 
    "e564005d5d723a00f78e4fee960476807d5040b5", 
    "e578606d7171f276843655b3d14327f7f939694d", 
    "e599011cb32f87f2746241e0394bdcb843527766", 
    "e5ca70a8cbc973718fdb85cb5d12f2d5740da4eb", 
    "e5ddd2f6e4cae88fd488a793a932acf615a0efe6", 
    "e6188b1eecfae681591cedb2bc7748175e15daeb", 
    "e72029cab625e845b0da6a1a7071f8a7e7cc51cb", 
    "e72ef8f74008eec0e03a2859179e5ca6e0ba61b1", 
    "e7acb846c2bf74d596ea23c12a016192086edfb4", 
    "e7b3f165b09c6b24bbbd56a85a6bd20e13295306", 
    "e7c905ee1fbe5154b47da85e3cdc3a1a3957909f", 
    "e7db7ddb4a5b47f33b0932f4488896b3b87ae66f", 
    "e82cf2d187a99a8414929074e2dc603d4e7be624", 
    "e90cd73cbbb4a94d6ea26485c8e9e2731ecbc878", 
    "e949be3a6cd9dfb637dea569287bb4fddcc7f06c", 
    "e97bbcef17897fe63eede23413b96dcf3252b6f3", 
    "e981f78540e82d095355e561c19ecde2d859f722", 
    "e9a5aedf8b0543f5f2418afc4583d8f025adc467", 
    "ea02adee968077cb9c04bc63354feae4b49a2b13", 
    "ea1f5fb52022818095112b54b42ff3d1b390043d", 
    "ea917de513600a774337bad9a8ba9fbe73d34eb4", 
    "eab3218ccb253c9483b04367c6574c60fa43af15", 
    "eac9abc0cf55cb83b8a19b25c5c0d49dcb70bf92", 
    "eaf5ccf14f7d8acca4d442830a7bb228bb89c960", 
    "eb9c37902324ca223e1e997f65ef0bddbc85c2e3", 
    "ebb6e28a1d4fc404fc8607d29418bfa0bf39ace7", 
    "ec185965c5a17f0031a5dcbdc0f2d4a4040f2254", 
    "ec6f8c20c6a4146307a1044b413e2ff426bbfd46", 
    "ecbd60f0707250a9d166d7ff6b2634577ed86d4c", 
    "ed04aa7844afb394158b439398508569fa2f24a9", 
    "ed16a9b8c68fc47f18688dfbb2df449b7f3f7f19", 
    "ed2b748e6a4c29dcab99dbf5ebdfcf8de12b9752", 
    "ed4ccb67dbd995f15b00ccc0623478bddb17bff0", 
    "ed8581690eeff282b3fd1b7910fcd3f5ae6f845c", 
    "ed91276ff56435ce5da0e48b9aab036698ba5994", 
    "edb659cf1efb39ed23d9a0b7a5b73ec7598007ee", 
    "ee37dfc4d6e45887d38fb44677ef7d3b3558b66b", 
    "ee42de4cfdb7dad46528a1f49f32147acb9acc7d", 
    "ee4dc498a2660e3324af7006df69ca31d3c53316", 
    "ee62cfef615958b9345a20a9b6a8a9c51ce73372", 
    "eed061c3b03cbb2fa3d86c3580a9a0022612ed30", 
    "eefd4240404ff1032b0710ed438999a5dd89850c", 
    "eefed09ec834ad33e21365c138a6e545c50987e9", 
    "ef54d891eafdc9630654e27138df38a1b0fe289f", 
    "ef72ad9eff4768c8c014d12bf58d5dfc1c49df69", 
    "ef84ec1449633f1045bf8a6b77c559857f993206", 
    "efc6b90959f9651e0a8e6eee6e08955486f93e1a", 
    "efef0ec1cf5ea34027b13701b12c151f17f9501b", 
    "f0130df1865aaaccf6558a4f9bf5ae501a77640f", 
    "f057ad45dd5ef4b57a6db149c3cb99d99813578b", 
    "f118022ce4e50acc3889586ff1c091e70e29308a", 
    "f1180cfcbdb989d7890950c01c38db85c93b2c8a", 
    "f17826dce7c323c15e8f1e91cb7543f10b09520d", 
    "f1b2da369c741119b78fc6251e6431297c923d3a", 
    "f1c85dfa81b900231e3cbccfa14c40a1bbd08ab1", 
    "f23609f4811d86382b46bd98f7f0662550c58ca3", 
    "f26d93b54499eb678a90116c204ff0ba04c21d15", 
    "f2a0f3a3a0009f669840f6c32ecae05fbce38f09", 
    "f2ebf27e835ba72216535731777460e771e6fefc", 
    "f2f4c0c134d7cd458cb5001f85b1b495b53146b3", 
    "f38e0ec06adeb2cf09d190783902f27a235c521a", 
    "f3c7a9ac2363b4508fdc090fa26b0d936ea7fd92", 
    "f3cf63aed6c725ac15f88a1607b55a8d1296f624", 
    "f410f41a63af9e1a5ededfa10ab2be94bce28bc0", 
    "f431f9bac82386022590b6528ae5c98f00670815", 
    "f43727897bb8073c3e91c80e01e6f10697d79f62", 
    "f44202c0775faad9d9818f766dcd865d7fea9513", 
    "f44e559e19b50362e56e6c7ffc0875a6049c8b57", 
    "f4d7a8fe9df2d538ddcf55a4211629dacac1539c", 
    "f519697301e1d3d2ab939b2010ce998ec9ec4141", 
    "f5520476c9a3519cb22b1f612c9fbad0730e6a1c", 
    "f5fd8e98fdfaf0d186d7548c4a061384c37398d4", 
    "f614f9f5756b9f09f0a33acc8455d3dae49c9cff", 
    "f619a25e5f36454a1e4aee1a631349075fb850d9", 
    "f6718407c0ee69b960930c9469b483ca87434582", 
    "f7bbe511f7917826783b0ffece9d5b37d7a786e5", 
    "f8476dfe88d3bceff6a2438a36a9c31ec44407c7", 
    "f88b2f592accff083eee6aac884aa1ef54b1e350", 
    "f8bef7c249bdf09744365254f78e701fbc348b43", 
    "f8c8588a784a30d74c150dbcb90f85cc9def4a8c", 
    "f8ecbfe293f4aefab059b5a5d65a89e298531480", 
    "f92bb1113a8bf8aed60764ea8d7e5bf6e064ffed", 
    "f944ac4791c2d236c4b6fd0201306064892a61d1", 
    "f9458300b346c517dc9720e92bbe1eeb8c8d3e55", 
    "f98311dfc36cf692b190992fdc8a2193e1692d06", 
    "fa8db9041e29ecc685057810717c7f216928d594", 
    "fa935f4077b932e1eb8e612f33aad4bb8ffd4ddc", 
    "fb1714e466c7c9a29bbe943598c8a64446c3875e", 
    "fb286e8a122f251df9217bfcd3354aef87048c7e", 
    "fb2889a73a5b69523d314ca71d47ab145e45f867", 
    "fb3d0ba85aeb5f555b41ec997f32732f2f6d7ea7", 
    "fb90343047b9657d980484f3b4141b1924b8936b", 
    "fc1bd98c159826638d31829bcf876138c9223225", 
    "fc3ba5eb020a36342d2a94b2a569402bec60e5ce", 
    "fc91321bd5fbd3c2cd8b0af6c770af3ad491f020", 
    "fcdc32ff59bbc051c5e15e7241571063a19b6ce0", 
    "fd255c9fe7d5afd1de6ec3fc44237100931fff10", 
    "fd3e2513c84b9d11b412a58803383571428cc7db", 
    "fd7e3f88351372787f4101ec6e2f860b7750ca53", 
    "fdcca4100cbbef1b6983b9cee3d610d39982daa1", 
    "fdf9e50c21043219b135b61996f196ddc67270ae", 
    "fdfebd00ceb552e2856d54a0655701f35a5fefde", 
    "fe00cfc8a3b4a979daac0ca57c0386bca8309c72", 
    "fe1afcf00b1ccb0b24567b5b7315cc134c55e81b", 
    "fe3131f06a59b1db2e3de0b88b865fb29b724c53", 
    "fe5365820385edcb92400c5458b8b8b8171e70c5", 
    "fec073e58c7e1b79271a1b0b5f66989d6e36593c",}

	var TimeseriesList = []string{ //32
	"Id",
	"Time1",
	"State",
	"HVAC_Mode",
	"Event",
	"Schedule",
	"DatasetName",

	"Indoor_AverageTemperature",
	"Indoor_CoolSetpoint",
	"Indoor_HeatSetpoint",
	"Indoor_Humidity",
	"HeatingEquipmentStage1_RunTime",
	"HeatingEquipmentStage2_RunTime",
	"CoolingEquipmentStage1_RunTime",
	"CoolingEquipmentStage2_RunTime",
	"HeatPumpsStage1_RunTime",
	"HeatPumpsStage2_RunTime",
	"Fan_RunTime",
	"Thermostat_Temperature",
	"Thermostat_DetectedMotion",
	"RemoteSensor1_Temperature",
	"RemoteSensor1_DetectedMotion",
	"RemoteSensor2_Temperature",
	"RemoteSensor2_DetectedMotion",
	"RemoteSensor3_Temperature",
	"RemoteSensor3_DetectedMotion",
	"RemoteSensor4_Temperature",
	"RemoteSensor4_DetectedMotion",
	"RemoteSensor5_Temperature",
	"RemoteSensor5_DetectedMotion",
	"Outdoor_Temperature",
	"Outdoor_Humidity",}

	var dataSetTypes = []string{"'unicode';", "'dateTime';", "'unicode';", "'unicode';", "'unicode';", "'unicode';", "'unicode';", 
	"'double';", "'double';", "'double';", "'double';", "'double';", "'double';", "'double';", "'double';", "'double';", "'double';", "'double';", "'double';", "'double';", "'double';", "'double';", "'double';", "'double';", "'double';", "'double';", "'double';", "'double';", "'double';", "'double';", "'double';", "'double';", "'double';"}

	var dataSetUnits = []string{"'hex40';","'yyyy-MM-dd hh:mm:ss';","'unicode';","'unicode';","'unicode';","'unicode';","'unicode';",
	"'°F';","'°F';","'°F';","'%rh';","'seconds';","'seconds';","'seconds';","'seconds';","'seconds';","'seconds';","'seconds';","'seconds';","'°F';","'boolean';","'°F';","'boolean';","'°F';","'boolean';","'°F';","'boolean';","'°F';","'boolean';","'°F';","'boolean';","'°F';","'%rh';"}
	
	for n0,_ := range DatasetList{
		for n1,_ := range TimeseriesList {
			statements = append(statements, prefix0+DatasetList[n0]+"."+TimeseriesList[n1]+attributes+dataSetTypes[n1])	
			statements = append(statements, prefix0+DatasetList[n0]+"."+TimeseriesList[n1]+tags+dataSetUnits[n1])	
		}
	}
    return statements
}

func homec_weather() []string {
    statements := make([]string, 0)
    const prefix0 = alter + "root.homec.weather.HomeC."
	var TimeseriesList = []string{
        "Time1",
        "Dishwasher_KW",
        "GarageDoor_KW",
        "LivingRoom_KW",
        "Temperature",
        "Kitchen38_KW",
        "WindSpeed",
        "Use_KW",
        "HomeOffice_KW",
        "Barn_KW",
        "PrecipIntensity",
        "Furnace1_KW",
        "Humidity",
        "PrecipProbability",
        "Kitchen12_KW",
        "Well_KW",
        "Kitchen14_KW",
        "Solar_KW",
        "DewPoint",
        "Fridge_KW",
        "Pressure",
        "Gen_KW",
        "WineCellar_KW",
        "Furnace2_KW",
        "Microwave_KW",
        "HouseOverall_KW",
        "ApparentTemperature",
        "Visibility",
        "DatasetName",
        "Summary",
        "Icon",
        "CloudCover",
        "WindBearing",
    }
	var dataSetTypes = []string{"'longint';", "'float';","'float';","'float';","'float';","'float';","'float';","'float';","'float';","'float';","'float';","'float';","'float';","'float';","'float';","'float';","'float';","'float';","'float';","'float';","'float';","'float';","'float';","'float';","'float';","'float';","'float';","'float';","'string';","'string';","'string';","'string';","'integer';",}
    var dataSetUnits = []string{"'unixutc';", "'kW';","'kW';","'kW';","'kW';","'kW';","'kW';","'kW';","'kW';","'kW';","'kW';","'kW';","'kW';","'kW';","'kW';","'kW';","'kW';","'kW';","'kW';","'°F';","'knots';","'metershour';","'%rh';","'percent';","'°C';","'mb';","'°F';","'km';","'unicode';","'unicode';","'unicode';","'unicode';","'degree';", }
    for n1,_ := range TimeseriesList {
        statements = append(statements, prefix0+TimeseriesList[n1]+attributes+dataSetTypes[n1])	
        statements = append(statements, prefix0+TimeseriesList[n1]+tags+dataSetUnits[n1])	
    }
    return statements
}

func  toniot_synthetic() []string {
    statements := make([]string, 0)
    const prefix0 = alter + " root.toniot.synthetic." 
	var DatasetList  = []string{"IoT_Weather.", "IoT_Thermostat.", "IoT_Modbus.", "IoT_Motion_Light."} 
    // IoT_Weather
	var timeseries0 = []string{"Utc_timestamp", "Datetime", "Temperature", "Pressure", "Humidity", "Label", "Type", "DatasetName", }
    var dataSetTypes0 = []string{"'longint';", "'dateTime';", "'float';", "'float';", "'float';", "'integer';", "'unicode';", "'string';", }
	var dataSetUnits0 = []string{"'unixutc';", "'yyyy-MM-ddThh:mm:ssZ';", "'°C';", "'mb';", "'%rh';", "'unitless';", "'unicode';", "'unicode';", }
    for n1,_ := range timeseries0 {
        statements = append(statements, prefix0+DatasetList[0]+timeseries0[n1]+attributes+dataSetTypes0[n1])
        statements = append(statements, prefix0+DatasetList[0]+timeseries0[n1]+tags+dataSetUnits0[n1])
    }
    
    // IoT_Thermostat
	var timeseries1 = []string{"Utc_timestamp", "Datetime", "Current_temperature", "Thermostat_status", "Label", "Type", "DatasetName", }
    var dataSetTypes1 = []string{"'longint';", "'dateTime';", "'float';", "'integer';", "'integer';", "'unicode';", "'unicode';", }
	var dataSetUnits1 = []string{"'unixutc';", "'yyyy-MM-ddThh:mm:ssZ';", "'°C';", "'unitless';", "'unitless';", "'unicode';", "'unicode';", }
    for n1,_ := range timeseries1 {
        statements = append(statements, prefix0+DatasetList[1]+timeseries1[n1]+attributes+dataSetTypes1[n1])
        statements = append(statements, prefix0+DatasetList[1]+timeseries1[n1]+tags+dataSetUnits1[n1])	
    }
    
    // IoT_Modbus
	var timeseries2 = []string{"Utc_timestamp", "Datetime", "FC1_Read_Input_Register", "FC2_Read_Discrete_Value", "FC3_Read_Holding_Register", "FC4_Read_Coil", "Label", "Type", "DatasetName", }
    var dataSetTypes2 = []string{"'longint';", "'dateTime';", "'integer';", "'integer';", "'integer';", "'integer';", "'integer';", "'unicode';", "'unicode';", }
	var dataSetUnits2 = []string{"'unixutc';", "'yyyy-MM-ddThh:mm:ssZ';", "'unitless';", "'unitless';", "'unitless';", "'unitless';", "'unitless';", "'unicode';", "'unicode';", }
    for n1,_ := range timeseries2 {
        statements = append(statements, prefix0+DatasetList[2]+timeseries2[n1]+attributes+dataSetTypes2[n1])	
        statements = append(statements, prefix0+DatasetList[2]+timeseries2[n1]+tags+dataSetUnits2[n1])	
    }
    
    // IoT_Motion_Light
	var timeseries3 = []string{"Utc_timestamp", "Datetime", "Motion_status", "Light_status", "Label", "Type", "DatasetName", }
    var dataSetTypes3 = []string{"'longint';", "'dateTime';", "'integer';", "'unicode';", "'integer';", "'unicode';", "'unicode';", }
	var dataSetUnits3 = []string{"'unixutc';", "'yyyy-MM-ddThh:mm:ssZ';", "'unitless';", "'unicode';", "'unitless';", "'unicode';", "'unicode';", }
    for n1,_ := range timeseries3 {
        statements = append(statements, prefix0+DatasetList[3]+timeseries3[n1]+attributes+dataSetTypes3[n1])	
        statements = append(statements, prefix0+DatasetList[3]+timeseries3[n1]+tags+dataSetUnits3[n1])	
    }
    
    return statements
}

func opsd_household() []string {
    statements := make([]string, 0)
    const prefix0 = alter + "root.opsd_household." 
	var DatasetList  = []string{"1min_singleindex.", "15min_singleindex.", "60min_singleindex."} 
	var TimeseriesList = []string{"Utc_timestamp", "Cet_cest_timestamp", "DatasetName", "DE_KN_industrial1_grid_import", "DE_KN_industrial1_pv_1", "DE_KN_industrial1_pv_2", "DE_KN_industrial2_grid_import", "DE_KN_industrial2_pv", "DE_KN_industrial2_storage_charge", 
    "DE_KN_industrial2_storage_decharge", "DE_KN_industrial3_area_offices", "DE_KN_industrial3_area_room_1", "DE_KN_industrial3_area_room_2", "DE_KN_industrial3_area_room_3", "DE_KN_industrial3_area_room_4", "DE_KN_industrial3_compressor", 
    "DE_KN_industrial3_cooling_aggregate", "DE_KN_industrial3_cooling_pumps", "DE_KN_industrial3_dishwasher", "DE_KN_industrial3_ev", "DE_KN_industrial3_grid_import", "DE_KN_industrial3_machine_1", "DE_KN_industrial3_machine_2", 
    "DE_KN_industrial3_machine_3", "DE_KN_industrial3_machine_4", "DE_KN_industrial3_machine_5", "DE_KN_industrial3_pv_facade", "DE_KN_industrial3_pv_roof", "DE_KN_industrial3_refrigerator", "DE_KN_industrial3_ventilation", 
    "DE_KN_public1_grid_import", "DE_KN_public2_grid_import", "DE_KN_residential1_dishwasher", "DE_KN_residential1_freezer", "DE_KN_residential1_grid_import", "DE_KN_residential1_heat_pump", "DE_KN_residential1_pv", 
    "DE_KN_residential1_washing_machine", "DE_KN_residential2_circulation_pump", "DE_KN_residential2_dishwasher", "DE_KN_residential2_freezer", "DE_KN_residential2_grid_import", "DE_KN_residential2_washing_machine", 
    "DE_KN_residential3_circulation_pump", "DE_KN_residential3_dishwasher", "DE_KN_residential3_freezer", "DE_KN_residential3_grid_export", "DE_KN_residential3_grid_import", "DE_KN_residential3_pv", "DE_KN_residential3_refrigerator", 
    "DE_KN_residential3_washing_machine", "DE_KN_residential4_dishwasher", "DE_KN_residential4_ev", "DE_KN_residential4_freezer", "DE_KN_residential4_grid_export", "DE_KN_residential4_grid_import", "DE_KN_residential4_heat_pump", 
    "DE_KN_residential4_pv", "DE_KN_residential4_refrigerator", "DE_KN_residential4_washing_machine", "DE_KN_residential5_dishwasher", "DE_KN_residential5_grid_import", "DE_KN_residential5_refrigerator", "DE_KN_residential5_washing_machine", 
    "DE_KN_residential6_circulation_pump", "DE_KN_residential6_dishwasher", "DE_KN_residential6_freezer", "DE_KN_residential6_grid_export", "DE_KN_residential6_grid_import", "DE_KN_residential6_pv", "DE_KN_residential6_washing_machine", }

	dataSetTypes := make([]string, len(TimeseriesList))
    dataSetTypes[0] = "'dateTime';"
    dataSetTypes[1] = "'dateTime';"
    dataSetTypes[2] = "'string';"
    for n := 3; n < len(TimeseriesList); n++ {
        dataSetTypes[n] = "'float';"
    }

	dataSetUnits := make([]string, len(TimeseriesList))
    dataSetUnits[0] = "'yyyy-MM-ddThh:mm:ssZ';"
    dataSetUnits[1] = "'yyyy-MM-ddThh:mm:ssZ';"
    dataSetUnits[2] = "'unicode';"
    for n := 3; n < len(TimeseriesList); n++ {
        dataSetUnits[n] = "'kWh';"
    }

	for n0,_ := range DatasetList {
        for n1,_ := range TimeseriesList {
            statements = append(statements, prefix0+DatasetList[n0]+TimeseriesList[n1]+attributes+dataSetTypes[n1])	
            statements = append(statements, prefix0+DatasetList[n0]+TimeseriesList[n1]+tags+dataSetUnits[n1])	
        }
    }

    return statements
}

func opsd_timeseries() []string {
    statements := make([]string, 0)
	const prefix0 string = alter + "root.opsd.timeseries."
	var DatasetList  = []string{"opsd_time_series_15min.", "opsd_time_series_30min.", "opsd_time_series_60min."} 
    
    var TimeseriesList15 = []string{"Utc_timestamp", "Cet_cest_timestamp", "DatasetName", 
    "AT_load_actual_entsoe_transparency", "AT_load_forecast_entsoe_transparency", "AT_price_day_ahead", "AT_solar_generation_actual", "AT_wind_onshore_generation_actual", "BE_load_actual_entsoe_transparency", 
    "BE_load_forecast_entsoe_transparency", "DE_load_actual_entsoe_transparency", "DE_load_forecast_entsoe_transparency", "DE_solar_capacity", "DE_solar_generation_actual", "DE_solar_profile", 
    "DE_wind_capacity", "DE_wind_generation_actual", "DE_wind_profile", "DE_wind_offshore_capacity", "DE_wind_offshore_generation_actual", "DE_wind_offshore_profile", "DE_wind_onshore_capacity", 
    "DE_wind_onshore_generation_actual", "DE_wind_onshore_profile", "DE_50hertz_load_actual_entsoe_transparency", "DE_50hertz_load_forecast_entsoe_transparency", "DE_50hertz_solar_generation_actual", 
    "DE_50hertz_wind_generation_actual", "DE_50hertz_wind_offshore_generation_actual", "DE_50hertz_wind_onshore_generation_actual", "DE_LU_load_actual_entsoe_transparency", "DE_LU_load_forecast_entsoe_transparency", 
    "DE_LU_solar_generation_actual", "DE_LU_wind_generation_actual", "DE_LU_wind_offshore_generation_actual", "DE_LU_wind_onshore_generation_actual", "DE_amprion_load_actual_entsoe_transparency", 
    "DE_amprion_load_forecast_entsoe_transparency", "DE_amprion_solar_generation_actual", "DE_amprion_wind_onshore_generation_actual", "DE_tennet_load_actual_entsoe_transparency", "DE_tennet_load_forecast_entsoe_transparency", 
    "DE_tennet_solar_generation_actual", "DE_tennet_wind_generation_actual", "DE_tennet_wind_offshore_generation_actual", "DE_tennet_wind_onshore_generation_actual", "DE_transnetbw_load_actual_entsoe_transparency", 
    "DE_transnetbw_load_forecast_entsoe_transparency", "DE_transnetbw_solar_generation_actual", "DE_transnetbw_wind_onshore_generation_actual", "HU_load_actual_entsoe_transparency", "HU_load_forecast_entsoe_transparency", 
    "HU_solar_generation_actual", "HU_wind_onshore_generation_actual", "LU_load_actual_entsoe_transparency", "LU_load_forecast_entsoe_transparency", "NL_load_actual_entsoe_transparency", "NL_load_forecast_entsoe_transparency", 
    "NL_solar_generation_actual", "NL_wind_generation_actual", "NL_wind_offshore_generation_actual", "NL_wind_onshore_generation_actual",}
    
    dataSetTypes15 := make([]string, len(TimeseriesList15))
    dataSetTypes15[0] = "'dateTime';"
    dataSetTypes15[1] = "'dateTime';"
    dataSetTypes15[2] = "'string';"
    for n := 3; n < len(TimeseriesList15); n++ {
        dataSetTypes15[n] = "'float';"
    }

	dataSetUnits15 := make([]string, len(TimeseriesList15))
    dataSetUnits15[0] = "'yyyy-MM-ddThh:mm:ssZ';"
    dataSetUnits15[1] = "'yyyy-MM-ddThh:mm:ssZ';"
    dataSetUnits15[2] = "'unicode';"
    for n := 3; n < len(TimeseriesList15); n++ {
        dataSetUnits15[n] = "'MW';"
    }
    dataSetUnits15[5]  = "'EUR';"
    dataSetUnits15[14] = "'percent';"
    dataSetUnits15[17] = "'percent';"
    dataSetUnits15[20] = "'percent';"
    dataSetUnits15[23] = "'percent';"
    
    for n1,_ := range TimeseriesList15 {
        statements = append(statements, prefix0+DatasetList[0]+TimeseriesList15[n1]+attributes+dataSetTypes15[n1])	
        statements = append(statements, prefix0+DatasetList[0]+TimeseriesList15[n1]+tags+dataSetUnits15[n1])	
    }

    var TimeseriesList30 = []string{"Utc_timestamp", "Cet_cest_timestamp", "DatasetName", 
    "CY_load_actual_entsoe_transparency", 
    "CY_load_forecast_entsoe_transparency", 
    "CY_wind_onshore_generation_actual", 
    "GB_GBN_load_actual_entsoe_transparency", 
    "GB_GBN_load_forecast_entsoe_transparency", 
    "GB_GBN_solar_capacity", 
    "GB_GBN_solar_generation_actual", 
    "GB_GBN_solar_profile", 
    "GB_GBN_wind_capacity", 
    "GB_GBN_wind_generation_actual", 
    "GB_GBN_wind_profile", 
    "GB_GBN_wind_offshore_capacity", 
    "GB_GBN_wind_offshore_generation_actual", 
    "GB_GBN_wind_offshore_profile", 
    "GB_GBN_wind_onshore_capacity", 
    "GB_GBN_wind_onshore_generation_actual", 
    "GB_GBN_wind_onshore_profile", 
    "GB_NIR_load_actual_entsoe_transparency", 
    "GB_NIR_load_forecast_entsoe_transparency", 
    "GB_NIR_solar_capacity", 
    "GB_NIR_wind_onshore_capacity", 
    "GB_NIR_wind_onshore_generation_actual", 
    "GB_UKM_load_actual_entsoe_transparency", 
    "GB_UKM_load_forecast_entsoe_transparency", 
    "GB_UKM_solar_capacity", 
    "GB_UKM_solar_generation_actual", 
    "GB_UKM_wind_capacity", 
    "GB_UKM_wind_generation_actual", 
    "GB_UKM_wind_offshore_capacity", 
    "GB_UKM_wind_offshore_generation_actual", 
    "GB_UKM_wind_onshore_capacity", 
    "GB_UKM_wind_onshore_generation_actual", 
    "IE_load_actual_entsoe_transparency", 
    "IE_load_forecast_entsoe_transparency", 
    "IE_wind_onshore_generation_actual", 
    "IE_sem_load_actual_entsoe_transparency", 
    "IE_sem_load_forecast_entsoe_transparency", 
    "IE_sem_price_day_ahead", 
    "IE_sem_wind_onshore_generation_actual", }
    
    dataSetTypes30 := make([]string, len(TimeseriesList30))
    dataSetTypes30[0] = "'dateTime';"
    dataSetTypes30[1] = "'dateTime';"
    dataSetTypes30[2] = "'string';"
    for n := 3; n < len(TimeseriesList30); n++ {
        dataSetTypes30[n] = "'float';"
    }

	dataSetUnits30 := make([]string, len(TimeseriesList30))
    dataSetUnits30[0] = "'yyyy-MM-ddThh:mm:ssZ';"
    dataSetUnits30[1] = "'yyyy-MM-ddThh:mm:ssZ';"
    dataSetUnits30[2] = "'unicode';"
    for n := 3; n < len(TimeseriesList30); n++ {
        dataSetUnits30[n] = "'MW';"
    }
    dataSetUnits30[10] = "'percent';"
    dataSetUnits30[13] = "'percent';"
    dataSetUnits30[16] = "'percent';"
    dataSetUnits30[19] = "'percent';"
    dataSetUnits30[40] = "'EUR';"

    for n1,_ := range TimeseriesList30 {
        statements = append(statements, prefix0+DatasetList[1]+TimeseriesList30[n1]+attributes+dataSetTypes30[n1])	
        statements = append(statements, prefix0+DatasetList[1]+TimeseriesList30[n1]+tags+dataSetUnits30[n1])	
    }

    var TimeseriesList60 = []string{"Utc_timestamp", "Cet_cest_timestamp", "DatasetName", 
    "AT_load_actual_entsoe_transparency", 
    "AT_load_forecast_entsoe_transparency", 
    "AT_price_day_ahead", 
    "AT_solar_generation_actual", 
    "AT_wind_onshore_generation_actual", 
    "BE_load_actual_entsoe_transparency", 
    "BE_load_forecast_entsoe_transparency", 
    "BE_solar_generation_actual", 
    "BE_wind_generation_actual", 
    "BE_wind_offshore_generation_actual", 
    "BE_wind_onshore_generation_actual", 
    "BG_load_actual_entsoe_transparency", 
    "BG_load_forecast_entsoe_transparency", 
    "BG_solar_generation_actual", 
    "BG_wind_onshore_generation_actual", 
    "CH_load_actual_entsoe_transparency", 
    "CH_load_forecast_entsoe_transparency", 
    "CH_solar_capacity", 
    "CH_solar_generation_actual", 
    "CH_wind_onshore_capacity", 
    "CH_wind_onshore_generation_actual", 
    "CY_load_actual_entsoe_transparency", 
    "CY_load_forecast_entsoe_transparency", 
    "CY_wind_onshore_generation_actual", 
    "CZ_load_actual_entsoe_transparency", 
    "CZ_load_forecast_entsoe_transparency", 
    "CZ_solar_generation_actual", 
    "CZ_wind_onshore_generation_actual", 
    "DE_load_actual_entsoe_transparency", 
    "DE_load_forecast_entsoe_transparency", 
    "DE_solar_capacity", 
    "DE_solar_generation_actual", 
    "DE_solar_profile", 
    "DE_wind_capacity", 
    "DE_wind_generation_actual", 
    "DE_wind_profile", 
    "DE_wind_offshore_capacity", 
    "DE_wind_offshore_generation_actual", 
    "DE_wind_offshore_profile", 
    "DE_wind_onshore_capacity", 
    "DE_wind_onshore_generation_actual", 
    "DE_wind_onshore_profile", 
    "DE_50hertz_load_actual_entsoe_transparency", 
    "DE_50hertz_load_forecast_entsoe_transparency", 
    "DE_50hertz_solar_generation_actual", 
    "DE_50hertz_wind_generation_actual", 
    "DE_50hertz_wind_offshore_generation_actual", 
    "DE_50hertz_wind_onshore_generation_actual", 
    "DE_LU_load_actual_entsoe_transparency", 
    "DE_LU_load_forecast_entsoe_transparency", 
    "DE_LU_price_day_ahead", 
    "DE_LU_solar_generation_actual", 
    "DE_LU_wind_generation_actual", 
    "DE_LU_wind_offshore_generation_actual", 
    "DE_LU_wind_onshore_generation_actual", 
    "DE_amprion_load_actual_entsoe_transparency", 
    "DE_amprion_load_forecast_entsoe_transparency", 
    "DE_amprion_solar_generation_actual", 
    "DE_amprion_wind_onshore_generation_actual", 
    "DE_tennet_load_actual_entsoe_transparency", 
    "DE_tennet_load_forecast_entsoe_transparency", 
    "DE_tennet_solar_generation_actual", 
    "DE_tennet_wind_generation_actual", 
    "DE_tennet_wind_offshore_generation_actual", 
    "DE_tennet_wind_onshore_generation_actual", 
    "DE_transnetbw_load_actual_entsoe_transparency", 
    "DE_transnetbw_load_forecast_entsoe_transparency", 
    "DE_transnetbw_solar_generation_actual", 
    "DE_transnetbw_wind_onshore_generation_actual", 
    "DK_load_actual_entsoe_transparency", 
    "DK_load_forecast_entsoe_transparency", 
    "DK_solar_capacity", 
    "DK_solar_generation_actual", 
    "DK_wind_capacity", 
    "DK_wind_generation_actual", 
    "DK_wind_offshore_capacity", 
    "DK_wind_offshore_generation_actual", 
    "DK_wind_onshore_capacity", 
    "DK_wind_onshore_generation_actual", 
    "DK_1_load_actual_entsoe_transparency", 
    "DK_1_load_forecast_entsoe_transparency", 
    "DK_1_price_day_ahead", 
    "DK_1_solar_generation_actual", 
    "DK_1_wind_generation_actual", 
    "DK_1_wind_offshore_generation_actual", 
    "DK_1_wind_onshore_generation_actual", 
    "DK_2_load_actual_entsoe_transparency", 
    "DK_2_load_forecast_entsoe_transparency", 
    "DK_2_price_day_ahead", 
    "DK_2_solar_generation_actual", 
    "DK_2_wind_generation_actual", 
    "DK_2_wind_offshore_generation_actual", 
    "DK_2_wind_onshore_generation_actual", 
    "EE_load_actual_entsoe_transparency", 
    "EE_load_forecast_entsoe_transparency", 
    "EE_solar_generation_actual", 
    "EE_wind_onshore_generation_actual", 
    "ES_load_actual_entsoe_transparency", 
    "ES_load_forecast_entsoe_transparency", 
    "ES_solar_generation_actual", 
    "ES_wind_onshore_generation_actual", 
    "FI_load_actual_entsoe_transparency", 
    "FI_load_forecast_entsoe_transparency", 
    "FI_wind_onshore_generation_actual", 
    "FR_load_actual_entsoe_transparency", 
    "FR_load_forecast_entsoe_transparency", 
    "FR_solar_generation_actual", 
    "FR_wind_onshore_generation_actual", 
    "GB_GBN_load_actual_entsoe_transparency", 
    "GB_GBN_load_forecast_entsoe_transparency", 
    "GB_GBN_price_day_ahead", 
    "GB_GBN_solar_capacity", 
    "GB_GBN_solar_generation_actual", 
    "GB_GBN_solar_profile", 
    "GB_GBN_wind_capacity", 
    "GB_GBN_wind_generation_actual", 
    "GB_GBN_wind_profile", 
    "GB_GBN_wind_offshore_capacity", 
    "GB_GBN_wind_offshore_generation_actual", 
    "GB_GBN_wind_offshore_profile", 
    "GB_GBN_wind_onshore_capacity", 
    "GB_GBN_wind_onshore_generation_actual", 
    "GB_GBN_wind_onshore_profile", 
    "GB_NIR_load_actual_entsoe_transparency", 
    "GB_NIR_load_forecast_entsoe_transparency", 
    "GB_NIR_solar_capacity", 
    "GB_NIR_wind_onshore_capacity", 
    "GB_NIR_wind_onshore_generation_actual", 
    "GB_UKM_load_actual_entsoe_transparency", 
    "GB_UKM_load_forecast_entsoe_transparency", 
    "GB_UKM_solar_capacity", 
    "GB_UKM_solar_generation_actual", 
    "GB_UKM_wind_capacity", 
    "GB_UKM_wind_generation_actual", 
    "GB_UKM_wind_offshore_capacity", 
    "GB_UKM_wind_offshore_generation_actual", 
    "GB_UKM_wind_onshore_capacity", 
    "GB_UKM_wind_onshore_generation_actual", 
    "GR_load_actual_entsoe_transparency", 
    "GR_load_forecast_entsoe_transparency", 
    "GR_solar_generation_actual", 
    "GR_wind_onshore_generation_actual", 
    "HR_load_actual_entsoe_transparency", 
    "HR_load_forecast_entsoe_transparency", 
    "HR_solar_generation_actual", 
    "HR_wind_onshore_generation_actual", 
    "HU_load_actual_entsoe_transparency", 
    "HU_load_forecast_entsoe_transparency", 
    "HU_solar_generation_actual", 
    "HU_wind_onshore_generation_actual", 
    "IE_load_actual_entsoe_transparency", 
    "IE_load_forecast_entsoe_transparency", 
    "IE_wind_onshore_generation_actual", 
    "IE_sem_load_actual_entsoe_transparency", 
    "IE_sem_load_forecast_entsoe_transparency", 
    "IE_sem_price_day_ahead", 
    "IE_sem_wind_onshore_generation_actual", 
    "IT_load_actual_entsoe_transparency", 
    "IT_load_forecast_entsoe_transparency", 
    "IT_solar_generation_actual", 
    "IT_wind_onshore_generation_actual", 
    "IT_BRNN_price_day_ahead", 
    "IT_BRNN_wind_onshore_generation_actual", 
    "IT_CNOR_load_actual_entsoe_transparency", 
    "IT_CNOR_load_forecast_entsoe_transparency", 
    "IT_CNOR_price_day_ahead", 
    "IT_CNOR_solar_generation_actual", 
    "IT_CNOR_wind_onshore_generation_actual", 
    "IT_CSUD_load_actual_entsoe_transparency", 
    "IT_CSUD_load_forecast_entsoe_transparency", 
    "IT_CSUD_price_day_ahead", 
    "IT_CSUD_solar_generation_actual", 
    "IT_CSUD_wind_onshore_generation_actual", 
    "IT_FOGN_price_day_ahead", 
    "IT_FOGN_solar_generation_actual", 
    "IT_FOGN_wind_onshore_generation_actual", 
    "IT_GR_price_day_ahead", 
    "IT_NORD_load_actual_entsoe_transparency", 
    "IT_NORD_load_forecast_entsoe_transparency", 
    "IT_NORD_price_day_ahead", 
    "IT_NORD_solar_generation_actual", 
    "IT_NORD_wind_onshore_generation_actual", 
    "IT_NORD_AT_price_day_ahead", 
    "IT_NORD_CH_price_day_ahead", 
    "IT_NORD_FR_price_day_ahead", 
    "IT_NORD_SI_price_day_ahead", 
    "IT_PRGP_price_day_ahead", 
    "IT_PRGP_solar_generation_actual", 
    "IT_PRGP_wind_onshore_generation_actual", 
    "IT_ROSN_price_day_ahead", 
    "IT_ROSN_solar_generation_actual", 
    "IT_ROSN_wind_onshore_generation_actual", 
    "IT_SACO_AC_price_day_ahead", 
    "IT_SACO_DC_price_day_ahead", 
    "IT_SARD_load_actual_entsoe_transparency", 
    "IT_SARD_load_forecast_entsoe_transparency", 
    "IT_SARD_price_day_ahead", 
    "IT_SARD_solar_generation_actual", 
    "IT_SARD_wind_onshore_generation_actual", 
    "IT_SICI_load_actual_entsoe_transparency", 
    "IT_SICI_load_forecast_entsoe_transparency", 
    "IT_SICI_price_day_ahead", 
    "IT_SICI_solar_generation_actual", 
    "IT_SICI_wind_onshore_generation_actual", 
    "IT_SUD_load_actual_entsoe_transparency", 
    "IT_SUD_load_forecast_entsoe_transparency", 
    "IT_SUD_price_day_ahead", 
    "IT_SUD_solar_generation_actual", 
    "IT_SUD_wind_onshore_generation_actual", 
    "LT_load_actual_entsoe_transparency", 
    "LT_load_forecast_entsoe_transparency", 
    "LT_solar_generation_actual", 
    "LT_wind_onshore_generation_actual", 
    "LU_load_actual_entsoe_transparency", 
    "LU_load_forecast_entsoe_transparency", 
    "LV_load_actual_entsoe_transparency", 
    "LV_load_forecast_entsoe_transparency", 
    "LV_wind_onshore_generation_actual", 
    "ME_load_actual_entsoe_transparency", 
    "ME_load_forecast_entsoe_transparency", 
    "ME_wind_onshore_generation_actual", 
    "NL_load_actual_entsoe_transparency", 
    "NL_load_forecast_entsoe_transparency", 
    "NL_solar_generation_actual", 
    "NL_wind_generation_actual", 
    "NL_wind_offshore_generation_actual", 
    "NL_wind_onshore_generation_actual", 
    "NO_load_actual_entsoe_transparency", 
    "NO_load_forecast_entsoe_transparency", 
    "NO_wind_onshore_generation_actual", 
    "NO_1_load_actual_entsoe_transparency", 
    "NO_1_load_forecast_entsoe_transparency", 
    "NO_1_price_day_ahead", 
    "NO_1_wind_onshore_generation_actual", 
    "NO_2_load_actual_entsoe_transparency", 
    "NO_2_load_forecast_entsoe_transparency", 
    "NO_2_price_day_ahead", 
    "NO_2_wind_onshore_generation_actual", 
    "NO_3_load_actual_entsoe_transparency", 
    "NO_3_load_forecast_entsoe_transparency", 
    "NO_3_price_day_ahead", 
    "NO_3_wind_onshore_generation_actual", 
    "NO_4_load_actual_entsoe_transparency", 
    "NO_4_load_forecast_entsoe_transparency", 
    "NO_4_price_day_ahead", 
    "NO_4_wind_onshore_generation_actual", 
    "NO_5_load_actual_entsoe_transparency", 
    "NO_5_load_forecast_entsoe_transparency", 
    "NO_5_price_day_ahead", 
    "NO_5_wind_onshore_generation_actual", 
    "PL_load_actual_entsoe_transparency", 
    "PL_load_forecast_entsoe_transparency", 
    "PL_solar_generation_actual", 
    "PL_wind_onshore_generation_actual", 
    "PT_load_actual_entsoe_transparency", 
    "PT_load_forecast_entsoe_transparency", 
    "PT_solar_generation_actual", 
    "PT_wind_generation_actual", 
    "PT_wind_offshore_generation_actual", 
    "PT_wind_onshore_generation_actual", 
    "RO_load_actual_entsoe_transparency", 
    "RO_load_forecast_entsoe_transparency", 
    "RO_solar_generation_actual", 
    "RO_wind_onshore_generation_actual", 
    "RS_load_actual_entsoe_transparency", 
    "RS_load_forecast_entsoe_transparency", 
    "SE_load_actual_entsoe_transparency", 
    "SE_load_forecast_entsoe_transparency", 
    "SE_wind_capacity", 
    "SE_wind_offshore_capacity", 
    "SE_wind_onshore_capacity", 
    "SE_wind_onshore_generation_actual", 
    "SE_1_load_actual_entsoe_transparency", 
    "SE_1_load_forecast_entsoe_transparency", 
    "SE_1_price_day_ahead", 
    "SE_1_wind_onshore_generation_actual", 
    "SE_2_load_actual_entsoe_transparency", 
    "SE_2_load_forecast_entsoe_transparency", 
    "SE_2_price_day_ahead", 
    "SE_2_wind_onshore_generation_actual", 
    "SE_3_load_actual_entsoe_transparency", 
    "SE_3_load_forecast_entsoe_transparency", 
    "SE_3_price_day_ahead", 
    "SE_3_wind_onshore_generation_actual", 
    "SE_4_load_actual_entsoe_transparency", 
    "SE_4_load_forecast_entsoe_transparency", 
    "SE_4_price_day_ahead", 
    "SE_4_wind_onshore_generation_actual", 
    "SI_load_actual_entsoe_transparency", 
    "SI_load_forecast_entsoe_transparency", 
    "SI_solar_generation_actual", 
    "SI_wind_onshore_generation_actual", 
    "SK_load_actual_entsoe_transparency", 
    "SK_load_forecast_entsoe_transparency", 
    "SK_solar_generation_actual", 
    "SK_wind_onshore_generation_actual", 
    "UA_load_actual_entsoe_transparency", 
    "UA_load_forecast_entsoe_transparency", }

    dataSetTypes60 := make([]string, len(TimeseriesList60))
    dataSetTypes60[0] = "'dateTime';"
    dataSetTypes60[1] = "'dateTime';"
    dataSetTypes60[2] = "'string';"
    for n := 3; n < len(TimeseriesList60); n++ {
        dataSetTypes60[n] = "'float';"
    }

	dataSetUnits60 := make([]string, len(TimeseriesList60))
    dataSetUnits60[0] = "'yyyy-MM-ddThh:mm:ssZ';"
    dataSetUnits60[1] = "'yyyy-MM-ddThh:mm:ssZ';"
    dataSetUnits60[2] = "'unicode';"
    for n := 3; n < len(TimeseriesList60); n++ {
        dataSetUnits60[n] = "'MW';"
    }
    dataSetUnits60[35] = "'percent';"
    dataSetUnits60[38] = "'percent';"
    dataSetUnits60[41] = "'percent';"
    dataSetUnits60[44] = "'percent';"
    dataSetUnits60[116] = "'percent';"
    dataSetUnits60[119] = "'percent';"
    dataSetUnits60[122] = "'percent';"
    dataSetUnits60[125] = "'percent';"
    dataSetUnits60[158] = "'EUR';"
    dataSetUnits60[164] = "'EUR';"
    dataSetUnits60[168] = "'EUR';"
    dataSetUnits60[173] = "'EUR';"
    dataSetUnits60[176] = "'EUR';"
    dataSetUnits60[179] = "'EUR';"
    dataSetUnits60[182] = "'EUR';"
    dataSetUnits60[185] = "'EUR';"
    dataSetUnits60[186] = "'EUR';"
    dataSetUnits60[187] = "'EUR';"
    dataSetUnits60[188] = "'EUR';"
    dataSetUnits60[189] = "'EUR';"
    dataSetUnits60[192] = "'EUR';"
    dataSetUnits60[195] = "'EUR';"
    dataSetUnits60[196] = "'EUR';"
    dataSetUnits60[199] = "'EUR';"
    dataSetUnits60[204] = "'EUR';"
    dataSetUnits60[209] = "'EUR';"
    dataSetUnits60[235] = "'EUR';"
    dataSetUnits60[239] = "'EUR';"
    dataSetUnits60[243] = "'EUR';"
    dataSetUnits60[247] = "'EUR';"
    dataSetUnits60[251] = "'EUR';"
    dataSetUnits60[277] = "'EUR';"
    dataSetUnits60[281] = "'EUR';"
    dataSetUnits60[285] = "'EUR';"
    dataSetUnits60[289] = "'EUR';"

    for n1,_ := range TimeseriesList60 {
        statements = append(statements, prefix0+DatasetList[2]+TimeseriesList60[n1]+attributes+dataSetTypes60[n1])	
        statements = append(statements, prefix0+DatasetList[2]+TimeseriesList60[n1]+tags+dataSetUnits60[n1])	
    }

    return statements
}

func executeStatements(description string, statements []string) { 
	ioTDbAccess := IoTDbAccess{ActiveSession: true} //isActive
    ioTDbAccess.session = client.NewSession(clientConfig)
    if err := ioTDbAccess.session.Open(false, 0); err != nil {
        checkErr("executeStatements(iot.IoTDbAccess.session.Open): ", err)
    }
    defer ioTDbAccess.session.Close()
    fmt.Print(description)
    for n := 0; n < len(statements); n++ {
        _, err := ioTDbAccess.session.ExecuteNonQueryStatement(statements[n])
        checkErr("ExecuteNonQueryStatement(createDBstatement)", err)
        fmt.Print(".")    
    }
    fmt.Println()
}

func ExecuteAlterStatements() { 
    fmt.Println("Alter statements for ATTRIBUTES & TAGS last applied 2023-11-06.")
    return
	iotdbConnection, ok := Init_IoTDB(true)
    if !ok {
        checkErr("ExecuteAlterStatements(Init_IoTDB): ", errors.New(iotdbConnection))
        os.Exit(1)
    }

    //executeStatements("Altering root.combed.iiitd", combed_iiitd())
	//executeStatements("Altering root.ecobee", ecobee())
	//executeStatements("Altering root.AMP.Vancouver(1)", AMP_Vancouver1())
	//executeStatements("Altering root.AMP.Vancouver(2)", AMP_Vancouver2())
	//executeStatements("Altering root.AMP.Vancouver(3)", AMP_Vancouver3())
    //executeStatements("Altering root.homec.weather", homec_weather())
    //executeStatements("Altering root.toniot.synthetic", toniot_synthetic())
    //executeStatements("Altering root.opsd.household", opsd_household())
    //executeStatements("Altering root.opsd.timeseries", opsd_timeseries())
}

