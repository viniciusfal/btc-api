package main

import (
	"encoding/xml"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type Btcs struct {
	XMLName        xml.Name `xml:"btcs"`
	VersaoApp      string   `xml:"versaoApp,attr"`
	DataGeracao    string   `xml:"dataGeracao,attr"`
	DataIni        string   `xml:"DataIni,attr"`
	DataFim        string   `xml:"DataFim,attr"`
	CodFuncionario string   `xml:"CodFuncionario,attr"`
	NFuncionario   string   `xml:"NFuncionario,attr"`
	CodEmpresa     string   `xml:"CodEmpresa,attr"`
	Btc            []Btc    `xml:"btc"`
}

type Btc struct {
	Doc       string    `xml:"doc"`
	Matdmtu   string    `xml:"matdmtu"`
	Data      string    `xml:"data"`
	Nome      string    `xml:"nome"`
	CodigoTD  string    `xml:"codigoTD"`
	Operacoes Operacoes `xml:"operacoes"`
}

type Operacoes struct {
	Operacao []Operacao `xml:"operacao"`
}

type Operacao struct {
	CodigoEmpresa    string      `xml:"codigoEmpresa"`
	Veiculo          string      `xml:"veiculo"`
	Linha            string      `xml:"linha"`
	RoletaInicial    string      `xml:"roletaInicial"`
	RoletaFinal      string      `xml:"roletaFinal"`
	TotalPassageiros string      `xml:"totalPassageiros"`
	TarifaAtual      string      `xml:"tarifaAtual"`
	Receita          string      `xml:"Receita"`
	Passageiros      Passageiros `xml:"passageiros"`
	Datainicio       string      `xml:"datainicio"`
	Datafim          string      `xml:"datafim"`
	Coletas          Coletas     `xml:"coletas"`
}

type Passageiros struct {
	Passageiro []Passageiro `xml:"passageiro"`
}

type Passageiro struct {
	Tipo        string `xml:"tipo"`
	VlUnitario  string `xml:"vlUnitario"`
	Qtd         string `xml:"qtd"`
	QtdCreditos string `xml:"qtdCreditos"`
	Idoso       int    `xml:"idoso"`
}

type Coletas struct {
	Recebido      string `xml:"recebido"`
	GirosPagantes string `xml:"girosPagantes"`
	GirosCartoes  string `xml:"girosCartoes"`
	Engolidos     string `xml:"engolidos"`
}

type GroupedData struct {
	CodigoTD        string
	Empresa         string
	CNPJ            string
	NomeLinha       string
	PrefixoANTT     string
	Sentido         string
	Veiculo         string
	LocalOrigem     string
	LocalDestino    string
	Data            time.Time
	Datainicio      time.Time
	Nome            string
	Linha           string
	Placa           string
	Pagantes        int
	Idoso           int
	PasseLivre      int
	JovemBaixaRenda int
}

var local1 string
var local2 string
var nomeLinha string
var prefixoANTT string

func main() {
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://dadosdedemanda.vercel.app"}, // Permitir requisições do frontend
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Disposition"}, // Para permitir download de arquivos
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.POST("/upload", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Arquivo não enviado"})
			return
		}

		filepath := "btc.xml"
		if err := c.SaveUploadedFile(file, filepath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar arquivo"})
			return
		}

		excelPath, err := ProcessXML(filepath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Header("Content-Disposition", "attachment; filename=output.xlsx")
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.File(excelPath)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "3333" // Porta padrão local
	}
	router.Run(":" + port)

}

func ProcessXML(filePath string) (string, error) {
	limb := Access()
	plate := PlacaV()

	file, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	var btcs Btcs
	err = xml.Unmarshal(file, &btcs)
	if err != nil {
		return "", err
	}

	groupedData := make(map[string]*GroupedData)
	linhaCount := make(map[string]int)

	for _, btc := range btcs.Btc {
		for _, operacao := range btc.Operacoes.Operacao {
			key := btc.CodigoTD + "_" + operacao.Datainicio // Chave única para agrupar

			// Converte os valores para somar

			totalPassageiros, _ := strconv.Atoi(operacao.TotalPassageiros)

			d, err := time.Parse("2006-01-02", btc.Data)
			if err != nil {
				return "", err
			}

			h, err := time.Parse("2006-01-02 15:04:05", operacao.Datainicio)
			if err != nil {
				return "", err
			}

			linhaCount[operacao.Linha]++

			sentido := "GO-DF"
			if linhaCount[operacao.Linha]%2 == 0 {
				sentido = "DF-GO"
			}

			if linha, existe := limb[operacao.Linha]; existe {
				if sentido == "GO-DF" {
					local1 = linha.Local1
					local2 = linha.Local2
				} else {
					local1 = linha.Local2
					local2 = linha.Local1
				}
				nomeLinha = linha.Linha
				prefixoANTT = linha.CodANTT
			}

			var placa string

			if plateV, exist := plate[operacao.Veiculo]; exist {
				placa = plateV.Placa
			}

			if _, exists := groupedData[key]; !exists {
				groupedData[key] = &GroupedData{
					Empresa:         "Amazonia Inter",
					CNPJ:            "12.647.487/0001-88",
					NomeLinha:       nomeLinha,
					PrefixoANTT:     prefixoANTT,
					Sentido:         sentido,
					Data:            d,
					Nome:            btc.Nome,
					CodigoTD:        btc.CodigoTD,
					Linha:           key,
					LocalOrigem:     local1,
					LocalDestino:    local2,
					Placa:           placa,
					Pagantes:        totalPassageiros,
					Datainicio:      h,
					Idoso:           0,
					PasseLivre:      0,
					JovemBaixaRenda: 0,
				}
			}

			for _, passageiro := range operacao.Passageiros.Passageiro {
				qtd, _ := strconv.Atoi(passageiro.Qtd)

				if passageiro.Tipo == "6" {
					groupedData[key].Idoso += qtd
					groupedData[key].Pagantes -= qtd
				}

				if passageiro.Tipo == "5" {
					groupedData[key].PasseLivre += qtd
					groupedData[key].Pagantes -= qtd
				}
			}
		}
	}

	// Criando o arquivo CSV
	excelPath := "output.xlsx"
	f := excelize.NewFile()

	sheetName := "Dados de Demanda"
	f.SetSheetName("Sheet1", sheetName)

	headerStyle, err := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#059669"},
			Pattern: 1,
		},
		Font: &excelize.Font{
			Bold:  true,
			Color: "#ffffff",
			Size:  12,
		},
		Alignment: &excelize.Alignment{
			Vertical:        "center",
			JustifyLastLine: true,
		},
	})

	if err != nil {
		log.Fatal("Erro ao criar estilo de cabeçalho:", err)
	}

	// Escrevendo o cabeçalho
	headers := []string{
		"Empresa", "CNPJ", "Nome da Linha", "Prefixo", "Codigo", "Sentido", "Local de Origem", "Local de Destino", "Dia", "Horário", "Placa", "Pagantes",
		"Idosos", "Passe Livre", "Jovem de Baixa renda",
	}

	for i, h := range headers {
		col := string(rune('A' + i))
		f.SetCellValue(sheetName, col+"1", h)

		f.SetCellStyle(sheetName, col+"1", col+"1", headerStyle)

		f.SetRowHeight(sheetName, 1, 32)

	}

	row := 2

	// Escrevendo os dados agrupados
	for _, data := range groupedData {
		pagantesRestantes := data.Pagantes

		for pagantesRestantes > 0 {
			pagantesNaLinha := pagantesRestantes / 2
			if pagantesRestantes < 95 {
				pagantesNaLinha = pagantesRestantes
			}

			values := []interface{}{
				data.Empresa, data.CNPJ, data.NomeLinha, data.PrefixoANTT, data.Linha, data.Sentido, data.LocalOrigem, data.LocalDestino,
				data.Data.Format("02-01-2006"), data.Datainicio.Format("15:04:05"), data.Placa,
				strconv.Itoa(pagantesNaLinha),
				strconv.Itoa(data.Idoso),
				strconv.Itoa(data.PasseLivre),
				strconv.Itoa(data.JovemBaixaRenda),
			}

			for i, v := range values {
				col := string(rune('A' + i))
				f.SetCellValue(sheetName, col+strconv.Itoa(row), v)
			}

			pagantesRestantes -= pagantesNaLinha
			row++

			if data.Sentido == "DF-GO" {
				data.Sentido = "GO-DF"
				data.LocalOrigem = local1
				data.LocalDestino = local2
			} else {
				data.Sentido = "DF-GO"
				data.LocalOrigem = local2
				data.LocalDestino = local1
			}

			data.Datainicio = data.Datainicio.Add(2*time.Hour + 13*time.Minute)
			data.Idoso = data.Idoso / 2
			data.PasseLivre = data.PasseLivre / 2
			data.JovemBaixaRenda = 0

		}
	}

	if err := f.SaveAs(excelPath); err != nil {
		log.Fatal("Erro ao salvar arquivo Excel:", err)
	}

	return excelPath, nil
}
