package main

type Linhas struct {
	Cod     string
	Local1  string
	Local2  string
	Linha   string
	CodANTT string
	Km      string
	Lat1    string // Latitude do Local1
	Lng1    string // Longitude do Local1
	Lat2    string // Latitude do Local2
	Lng2    string // Longitude do Local2
}

type Cars struct {
	Placa string
}

func Access() map[string]Linhas {
	lines := map[string]Linhas{
		"9901": {Cod: "9901", Local1: "Rodoviária Interestadual de Formosa de Goiás (Via BR-020)", Local2: "Rodoviária de Planaltina - DF (Jardim Roriz)", Linha: "Planaltina-DF - Formosa-GO", CodANTT: "12-0338-70", Km: "50.9", Lat1: "-15.5508008", Lng1: "-47.3375733", Lat2: "-15.61845461", Lng2: "-47.65430421"},
		"9902": {Cod: "9902", Local1: "Rodoviária Interestadual de Formosa de Goiás", Local2: "Rodoviária de Planaltina - DF (Via BR-479 / Vale do Amanhecer)", Linha: "Planaltina-DF - Formosa-GO", CodANTT: "12-0338-70", Km: "", Lat1: "-15.5508008", Lng1: "-47.3375733", Lat2: "-15.61845461", Lng2: "-47.65430421"},
		"9903": {Cod: "9903", Local1: "Rodoviária Interestadual de Formosa de Goiás (Via BR-020)", Local2: "Posto Itiquira", Linha: "Formosa-GO - Posto Itiquira", CodANTT: "12-0338-70", Km: "80.8", Lat1: "-15.5508008", Lng1: "-47.3375733", Lat2: "", Lng2: ""},
		"9904": {Cod: "9904", Local1: "Rodoviária Interestadual de Formosa de Goiás (Via BR-020)", Local2: "Brasília-DF", Linha: "Formosa-GO - Brasília-DF", CodANTT: "12-0338-70", Km: "80.8", Lat1: "-15.5508008", Lng1: "-47.3375733", Lat2: "-15.7936645", Lng2: "-47.8829638"},
		"1001": {Cod: "1001", Local1: "Rodoviária de Planaltina de Goiás", Local2: "Rodoviária do Plano Piloto", Linha: "Planaltina-GO - Brasilia-DF", CodANTT: "12-0730-70", Km: "70.6", Lat1: "-15.43488062", Lng1: "-47.6108282", Lat2: "-15.7936645", Lng2: "-47.8829638"},
		"1002": {Cod: "1002", Local1: "Rodoviária de Planaltina de Goiás", Local2: "L2 Norte - Sul (Terminal Asa Sul)", Linha: "Planaltina-GO - Brasilia-DF", CodANTT: "12-0730-70", Km: "82.1", Lat1: "-15.43488062", Lng1: "-47.6108282", Lat2: "-15.8362131", Lng2: "-47.9328304"},
		"1003": {Cod: "1003", Local1: "Rodoviária de Planaltina de Goiás", Local2: "Eixo Norte e Sul (Terminal Asa Sul) - Executivo", Linha: "Planaltina-GO - Brasilia-DF", CodANTT: "12-0730-70", Km: "71.1", Lat1: "-15.43488062", Lng1: "-47.6108282", Lat2: "-15.8362131", Lng2: "-47.9328304"},
		"1054": {Cod: "1054", Local1: "Mutirão", Local2: "Eixo Norte e Sul / Terminal Asa Sul", Linha: "Planaltina-GO - Brasilia-DF", CodANTT: "12-0730-70", Km: "74.4", Lat1: "-15.42378904", Lng1: "-47.62148262", Lat2: "-15.8362131", Lng2: "-47.9328304"},
		"1055": {Cod: "1055", Local1: "Bairro São Fransciso", Local2: "Eixo Norte e Sul (Terminal Asa Sul)", Linha: "Planaltina-GO - Brasilia-DF", CodANTT: "12-0730-70", Km: "64.5", Lat1: "-15.43488062", Lng1: "-47.6108282", Lat2: "-15.8362131", Lng2: "-47.9328304"},
		"1056": {Cod: "1056", Local1: "Bairro Imigrantes", Local2: "Eixo Norte e Sul (Terminal Asa Sul)", Linha: "Planaltina-GO - Brasilia-DF", CodANTT: "12-0730-70", Km: "69.6", Lat1: "-15.43488062", Lng1: "-47.6108282", Lat2: "-15.8362131", Lng2: "-47.9328304"},
		"1057": {Cod: "1057", Local1: "Planaltina-GO (São José)", Local2: "Eixo W Norte e Sul / Terminal Asa Sul", Linha: "Planaltina-GO - Brasilia-DF", CodANTT: "12-0730-70", Km: "", Lat1: "-15.43488062", Lng1: "-47.6108282", Lat2: "-15.8362131", Lng2: "-47.9328304"},
		"1058": {Cod: "1058", Local1: "Planaltina de Goiás (Setor Oeste e Sul)", Local2: "Eixo Norte e Sul (Terminal da Asa Sul)", Linha: "Planaltina-GO - Brasilia-DF", CodANTT: "12-0730-70", Km: "68.3", Lat1: "-15.45837295", Lng1: "-47.6227588", Lat2: "-15.8362131", Lng2: "-47.9328304"},
		"1059": {Cod: "1059", Local1: "Mutirão (Via Feira)", Local2: "Rodoviária do Plano Piloto (Via Eixo Norte)", Linha: "Planaltina-GO - Brasilia-DF", CodANTT: "12-0730-70", Km: "64.9", Lat1: "-15.42378904", Lng1: "-47.62148262", Lat2: "-15.7936645", Lng2: "-47.8829638"},
		"1060": {Cod: "1060", Local1: "Bairro Nara (Via Setor Norte)", Local2: "Terminal Asa Sul (Eixo Norte e Sul)", Linha: "Planaltina-GO - Brasilia-DF", CodANTT: "12-0730-70", Km: "71.5", Lat1: "-15.43488062", Lng1: "-47.6108282", Lat2: "-15.8362131", Lng2: "-47.9328304"},
		"1061": {Cod: "1061", Local1: "Planaltina-GO (Brasilinha 17)", Local2: "Eixo W Norte e Sul/T.A.S. ", Linha: "Planaltina-GO - Brasilia-DF", CodANTT: "12-0730-70", Km: "63.4", Lat1: "-15.43488062", Lng1: "-47.6108282", Lat2: "-15.8362131", Lng2: "-47.9328304"},
		"1062": {Cod: "1062", Local1: "Planaltina-GO", Local2: "Eixo W Norte e Sul / Terminal Asa Sul", Linha: "Planaltina-GO - Brasilia-DF", CodANTT: "12-0730-70", Km: "", Lat1: "-15.43488062", Lng1: "-47.6108282", Lat2: "-15.8362131", Lng2: "-47.9328304"},
		"1073": {Cod: "1073", Local1: "Rodoviária de Planaltina de Goiás", Local2: "Eixo Norte e Sul (Terminal Asa Sul)", Linha: "Planaltina-GO - Brasilia-DF", CodANTT: "12-0730-70", Km: "76.3", Lat1: "-15.43488062", Lng1: "-47.6108282", Lat2: "-15.8362131", Lng2: "-47.9328304"},
		"1074": {Cod: "1074", Local1: "Rodoviária de Planaltina de Goiás", Local2: "W3 Norte e Sul (Terminal Asa Sul)", Linha: "Planaltina-GO - Brasilia-DF", CodANTT: "12-0730-70", Km: "77.0", Lat1: "-15.43488062", Lng1: "-47.6108282", Lat2: "-15.8362131", Lng2: "-47.9328304"},
		"1102": {Cod: "1102", Local1: "Rodoviária de Planaltina de Goiás", Local2: "Lago Norte", Linha: "Planaltina-GO - Brasilia-DF", CodANTT: "12-0730-70", Km: "71.9", Lat1: "-15.43488062", Lng1: "-47.6108282", Lat2: "-15.72153869", Lng2: "-47.87672546"},
		"1301": {Cod: "1301", Local1: "Rodoviária de Planaltina de Goiás", Local2: "SIA-SAAN (SOF Sul)", Linha: "Planaltina-GO - Brasilia-DF", CodANTT: "12-0730-70", Km: "85.5", Lat1: "-15.43488062", Lng1: "-47.6108282", Lat2: "-15.8362131", Lng2: "-47.9328304"},
		"1322": {Cod: "1322", Local1: "Mutirão", Local2: "Setor Gráfico (Eixo Norte)", Linha: "Planaltina-GO - Brasilia-DF", CodANTT: "12-0730-70", Km: "80.9", Lat1: "-15.42378904", Lng1: "-47.62148262", Lat2: "-15.8362131", Lng2: "-47.9328304"},
		"1323": {Cod: "1323", Local1: "Rodoviária de Planaltina de Goiás", Local2: "Sudoeste (W3 Norte - Terminal da Asa Sul)", Linha: "Planaltina-GO - Brasilia-DF", CodANTT: "12-0730-70", Km: "81.0", Lat1: "-15.43488062", Lng1: "-47.6108282", Lat2: "-15.8362131", Lng2: "-47.9328304"},
		"1324": {Cod: "1324", Local1: "Planaltina de Goiás", Local2: "Noroeste / Setor Gráfico", Linha: "Planaltina-GO - Brasilia-DF", CodANTT: "12-0730-70", Km: "94.8", Lat1: "-15.43488062", Lng1: "-47.6108282", Lat2: "-15.8362131", Lng2: "-47.9328304"},
		"1326": {Cod: "1326", Local1: "Rodoviária de Planaltina de Goiás", Local2: "Noroeste", Linha: "Planaltina-GO - Brasilia-DF", CodANTT: "12-0730-70", Km: "83.5", Lat1: "-15.43488062", Lng1: "-47.6108282", Lat2: "-15.8362131", Lng2: "-47.9328304"},
		"1327": {Cod: "1327", Local1: "Mutirão", Local2: "Noroeste", Linha: "Planaltina-GO - Brasilia-DF", CodANTT: "12-0730-70", Km: "78.5", Lat1: "-15.42378904", Lng1: "-47.62148262", Lat2: "-15.8362131", Lng2: "-47.9328304"},
		"1901": {Cod: "1901", Local1: "Rodoviária de Planaltina de Goiás", Local2: "Rodoviária de Sobradinho I", Linha: "Planaltina-GO - Sobradinho-DF", CodANTT: "12-0730-70", Km: "53.7", Lat1: "-15.43488062", Lng1: "-47.6108282", Lat2: "-15.64965962", Lng2: "-47.78525909"},
		"1902": {Cod: "1902", Local1: "Rodoviária de Planaltina de Goiás", Local2: "Grande Colorado", Linha: "Planaltina-GO - Brasilia-DF", CodANTT: "12-0730-70", Km: "57.3", Lat1: "-15.43488062", Lng1: "-47.6108282", Lat2: "-15.68902632", Lng2: "-47.85773071"},
		"1950": {Cod: "1950", Local1: "Rodoviária de Planaltina de Goiás", Local2: "Rodoviária de Planaltina DF (Via Estância)", Linha: "Planaltina-GO - Planaltina-DF", CodANTT: "12-1070-70", Km: "40.1", Lat1: "-15.43488062", Lng1: "-47.6108282", Lat2: "-15.61845461", Lng2: "-47.65430421"},
		"1952": {Cod: "1952", Local1: "Rodoviária de Planaltina de Goiás", Local2: "Rodoviária de Planaltina - DF (Via Roriz)", Linha: "Planaltina-GO - Planaltina-DF", CodANTT: "12-1070-70", Km: "40.8", Lat1: "-15.43488062", Lng1: "-47.6108282", Lat2: "-15.61845461", Lng2: "-47.65430421"},
		"1953": {Cod: "1953", Local1: "Rodoviária de Planaltina de Goiás (Via Centro Feira)", Local2: "Morro da Capelinha (Via DF-128)", Linha: "Planaltina-GO - Morro da Capelinha", CodANTT: "12-0730-70", Km: "40.6", Lat1: "-15.43488062", Lng1: "-47.6108282", Lat2: "", Lng2: ""},
	}

	return lines
}

func PlacaV() map[string]Cars {
	placas := map[string]Cars{
		"1001":   {Placa: "JHX-0E23"},
		"1002":   {Placa: "JHX-4G03"},
		"1003":   {Placa: "JHX-0D23"},
		"1004":   {Placa: "JHX-0D03"},
		"1005":   {Placa: "FVW-2B32"},
		"1006":   {Placa: "FYP-4C15"},
		"1007":   {Placa: "FIV-1H01"},
		"1008":   {Placa: "JHX-5A03"},
		"1009":   {Placa: "JHX-0D53"},
		"1010":   {Placa: "JHX-4E43"},
		"1011":   {Placa: "JHJ-4F62"},
		"1012":   {Placa: "JHX-0E03"},
		"1014":   {Placa: "JHX-0D93"},
		"1015":   {Placa: "JHJ-4F82"},
		"1016":   {Placa: "JHX-4J03"},
		"1017":   {Placa: "JHX-0D73"},
		"1018":   {Placa: "JHJ-4F22"},
		"1019":   {Placa: "JHJ-5G22"},
		"1020":   {Placa: "FWC-6J06"},
		"1021":   {Placa: "JHX-5093"},
		"1024":   {Placa: "JHX-4J63"},
		"1025":   {Placa: "JHJ-7B62"},
		"1026":   {Placa: "JHX-0D43"},
		"1027":   {Placa: "JHX-4J33"},
		"1028":   {Placa: "JHX-4I93"},
		"1029":   {Placa: "JHX-4D83"},
		"1030":   {Placa: "JHX-5A83"},
		"1031":   {Placa: "JHX-4G13"},
		"1032":   {Placa: "JHX-5A53"},
		"1033":   {Placa: "JHX-5A63"},
		"1034":   {Placa: "JHX-0C83"},
		"1035":   {Placa: "JHX-4F33"},
		"1036":   {Placa: "JHX-4D73"},
		"1037":   {Placa: "JHX-4463"},
		"1038":   {Placa: "JHX-0213"},
		"1039":   {Placa: "JHX-0383"},
		"1040":   {Placa: "JHX-4423"},
		"1041":   {Placa: "JHX-4563"},
		"1042":   {Placa: "JHX-4543"},
		"1043":   {Placa: "JHX-0C43"},
		"1044":   {Placa: "JHX-0193"},
		"1045":   {Placa: "JHX-5023"},
		"1046":   {Placa: "JHX-4523"},
		"1047":   {Placa: "JHX-0253"},
		"1048":   {Placa: "JHX-0C23"},
		"1049":   {Placa: "JHJ-7292"},
		"1050":   {Placa: "JHJ-7282"},
		"1051":   {Placa: "JHJ-5642"},
		"1052":   {Placa: "JHJ-6462"},
		"1053":   {Placa: "JHJ-4672"},
		"1054":   {Placa: "JHJ-7372"},
		"1055":   {Placa: "JHJ-5522"},
		"1056":   {Placa: "JHJ-4502"},
		"1057":   {Placa: "JHJ-4602"},
		"1058":   {Placa: "JHJ-4592"},
		"1059":   {Placa: "JHX-4953"},
		"1060":   {Placa: "JHX-5123"},
		"1061":   {Placa: "JHX-4393"},
		"1062":   {Placa: "JHX-4883"},
		"1063":   {Placa: "JHX-4973"},
		"1064":   {Placa: "JHX-5073"},
		"1066":   {Placa: "JHX-5103"},
		"1067":   {Placa: "JHX-4923"},
		"1068":   {Placa: "JHX-0363"},
		"1340":   {Placa: "ECM-5243"},
		"101001": {Placa: "LUJ-8G12"},
		"101002": {Placa: "LMX-5F24"},
		"101003": {Placa: "LUF-9D66"},
		"101004": {Placa: "LMX-2J75"},
		"101005": {Placa: "LMY-0E77"},
	}

	return placas
}
