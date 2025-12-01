package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
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
	Empresa             string
	PrefixoANTT         string
	Linha               string
	Sentido             string
	DataInicioViagem    time.Time
	HoraInicioViagem    string
	HoraFinalViagem     string
	QtePaxPagantes      int
	Idoso               int
	PasseLivre          int
	QteOutrasGratuidade int
	QteTotalPax         int
	QtePagoDinheiro     int
	QtePagoEletronico   int
	DistanciaViagem     int
	TempoViagem         string
	VelocidadeMedia     float64
	LtAberturaViagem    string
	LgAberturaViagem    string
	LtFechamentoViagem  string
	LgFechamentoViagem  string
	VeiculoNumero       string
	CPFRodoviario       string
}

var (
	cpfCache      = make(map[string]string)
	cpfCacheLock  sync.RWMutex
	dbPool        *sql.DB
	dbPoolOnce    sync.Once
	dbPoolInitErr error
)

// getDBConnection retorna o pool de conexões com o banco de dados PostgreSQL
func getDBConnection() (*sql.DB, error) {
	dbPoolOnce.Do(func() {
		// Tentar múltiplas variáveis de ambiente na ordem de prioridade
		var databaseURL string
		envVars := []string{"DATABASE_URL", "POSTGRES_URL", "DATABASE_PUBLIC_URL"}

		for _, envVar := range envVars {
			databaseURL = os.Getenv(envVar)
			if databaseURL != "" {
				log.Printf("Variável de ambiente encontrada: %s", envVar)
				break
			}
		}

		if databaseURL == "" {
			dbPoolInitErr = fmt.Errorf("nenhuma variável de ambiente de banco encontrada (DATABASE_URL, POSTGRES_URL, DATABASE_PUBLIC_URL)")
			log.Printf("ERRO: %v", dbPoolInitErr)
			return
		}

		// Log da URL de conexão (sem senha para segurança)
		urlForLog := databaseURL
		if strings.Contains(urlForLog, "@") {
			parts := strings.Split(urlForLog, "@")
			if len(parts) > 0 {
				// Ocultar senha na URL
				userPass := strings.Split(parts[0], "://")
				if len(userPass) > 1 {
					userParts := strings.Split(userPass[1], ":")
					if len(userParts) > 1 {
						urlForLog = userPass[0] + "://" + userParts[0] + ":***@" + parts[1]
					}
				}
			}
		}
		log.Printf("URL de conexão (senha oculta): %s", urlForLog)

		// Adicionar parâmetro SSL se não estiver presente na URL
		// Railway geralmente fornece a URL completa, mas se não tiver sslmode, usamos 'prefer' (mais flexível)
		if !strings.Contains(databaseURL, "sslmode=") {
			separator := "?"
			if strings.Contains(databaseURL, "?") {
				separator = "&"
			}
			databaseURL = databaseURL + separator + "sslmode=prefer"
			log.Printf("Parâmetro SSL adicionado à connection string (sslmode=prefer)")
		} else {
			log.Printf("URL já contém parâmetros SSL, usando configuração original")
		}

		log.Printf("Tentando conectar ao banco de dados...")
		var err error
		dbPool, err = sql.Open("postgres", databaseURL)
		if err != nil {
			dbPoolInitErr = fmt.Errorf("erro ao abrir conexão com banco: %w", err)
			log.Printf("ERRO ao abrir conexão: %v", dbPoolInitErr)
			return
		}

		log.Printf("Testando conexão com Ping...")
		if err = dbPool.Ping(); err != nil {
			dbPool.Close()
			dbPool = nil
			dbPoolInitErr = fmt.Errorf("erro ao conectar com banco (Ping falhou): %w", err)
			log.Printf("ERRO no Ping: %v", dbPoolInitErr)
			return
		}

		log.Printf("Conexão com banco de dados estabelecida com sucesso!")

		// Configurar pool de conexões
		dbPool.SetMaxOpenConns(25)
		dbPool.SetMaxIdleConns(5)
		dbPool.SetConnMaxLifetime(5 * time.Minute)
	})

	if dbPoolInitErr != nil {
		return nil, dbPoolInitErr
	}

	if dbPool == nil {
		return nil, fmt.Errorf("pool de conexões não inicializado")
	}

	return dbPool, nil
}

// getCPFByCodIdentificador busca o CPF na tabela pessoa usando o código identificador
func getCPFByCodIdentificador(codIdentificador string) (string, error) {
	// Verificar cache primeiro
	cpfCacheLock.RLock()
	if cpf, exists := cpfCache[codIdentificador]; exists {
		cpfCacheLock.RUnlock()
		return cpf, nil
	}
	cpfCacheLock.RUnlock()

	// Buscar no banco de dados
	db, err := getDBConnection()
	if err != nil {
		log.Printf("ERRO ao obter conexão com banco para código %s: %v", codIdentificador, err)
		// Se não conseguir conectar, retornar string vazia (não bloquear processamento)
		return "", nil
	}

	if db == nil {
		log.Printf("ERRO: conexão com banco é nil para código %s", codIdentificador)
		// Se db for nil, retornar string vazia
		return "", nil
	}

	var cpf sql.NullString
	query := "SELECT cpf FROM pessoa WHERE cod_identificador = $1"
	log.Printf("Consultando CPF para código identificador: %s", codIdentificador)
	err = db.QueryRow(query, codIdentificador).Scan(&cpf)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("CPF não encontrado para código identificador: %s", codIdentificador)
			// Não encontrou, salvar string vazia no cache
			cpfCacheLock.Lock()
			cpfCache[codIdentificador] = ""
			cpfCacheLock.Unlock()
			return "", nil
		}
		log.Printf("ERRO ao consultar CPF para código %s: %v", codIdentificador, err)
		return "", fmt.Errorf("erro ao consultar CPF: %w", err)
	}

	cpfValue := ""
	if cpf.Valid {
		cpfValue = cpf.String
		log.Printf("CPF encontrado para código %s: %s", codIdentificador, cpfValue)
	} else {
		log.Printf("CPF é NULL para código identificador: %s", codIdentificador)
	}

	// Salvar no cache
	cpfCacheLock.Lock()
	cpfCache[codIdentificador] = cpfValue
	cpfCacheLock.Unlock()

	return cpfValue, nil
}

func main() {
	// Carregar variáveis de ambiente do arquivo .env
	if err := godotenv.Load(); err != nil {
		// Se não encontrar .env, continua normalmente (pode estar usando variáveis do sistema)
		fmt.Println("Aviso: arquivo .env não encontrado, usando variáveis de ambiente do sistema")
	}

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://dadosdedemanda.vercel.app", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Endpoint de health check para testar conexão com banco
	router.GET("/health/db", func(c *gin.Context) {
		db, err := getDBConnection()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "error",
				"message": "Não foi possível conectar ao banco de dados",
				"error":   err.Error(),
			})
			return
		}

		// Testar query simples
		var result int
		err = db.QueryRow("SELECT 1").Scan(&result)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "error",
				"message": "Conexão estabelecida, mas query de teste falhou",
				"error":   err.Error(),
			})
			return
		}

		// Testar se a tabela pessoa existe
		var tableExists bool
		err = db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'pessoa'
			)
		`).Scan(&tableExists)

		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"status":      "connected",
				"message":     "Conexão OK, mas não foi possível verificar tabela pessoa",
				"query_test":  "OK",
				"table_check": "error",
				"error":       err.Error(),
			})
			return
		}

		// Contar registros na tabela pessoa se existir
		var count int
		if tableExists {
			err = db.QueryRow("SELECT COUNT(*) FROM pessoa").Scan(&count)
			if err != nil {
				count = -1
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"status":       "connected",
			"message":      "Conexão com banco de dados OK",
			"query_test":   "OK",
			"table_exists": tableExists,
			"pessoa_count": count,
		})
	})

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

		csvPath, err := ProcessXML(filepath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Header("Content-Disposition", "attachment; filename=output.csv")
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.File(csvPath)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "3333"
	}
	router.Run(":" + port)
}

func ProcessXML(filePath string) (string, error) {
	limb := Access()
	placas := PlacaV()

	file, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	var btcs Btcs
	err = xml.Unmarshal(file, &btcs)
	if err != nil {
		return "", err
	}

	var operacoesData []GroupedData
	linhaCount := make(map[string]int)

	for _, btc := range btcs.Btc {
		for _, operacao := range btc.Operacoes.Operacao {
			// Parse das datas
			dataInicio, err := time.Parse("2006-01-02 15:04:05", operacao.Datainicio)
			if err != nil {
				return "", err
			}

			dataFim, err := time.Parse("2006-01-02 15:04:05", operacao.Datafim)
			if err != nil {
				return "", err
			}

			// Calcular sentido
			linhaCount[operacao.Linha]++
			sentido := "GO-DF"
			if linhaCount[operacao.Linha]%2 == 0 {
				sentido = "DF-GO"
			}

			// Buscar informações da linha
			var linhaCerta, prefixoANTT string
			var latAbertura, lngAbertura, latFechamento, lngFechamento string
			if linha, existe := limb[operacao.Linha]; existe {
				linhaCerta = linha.Cod
				prefixoANTT = strings.ReplaceAll(linha.CodANTT, "-", "")

				// Preencher coordenadas baseado no sentido da viagem
				if sentido == "GO-DF" {
					// Sentido ida: Local1 → abertura, Local2 → fechamento
					latAbertura = linha.Lat1
					lngAbertura = linha.Lng1
					latFechamento = linha.Lat2
					lngFechamento = linha.Lng2
				} else {
					// Sentido volta (DF-GO): Local2 → abertura, Local1 → fechamento
					latAbertura = linha.Lat2
					lngAbertura = linha.Lng2
					latFechamento = linha.Lat1
					lngFechamento = linha.Lng1
				}
			}

			// Buscar placa do veículo
			veiculoPlaca := operacao.Veiculo
			if car, existe := placas[operacao.Veiculo]; existe {
				veiculoPlaca = car.Placa
			}

			// Buscar CPF do motorista no banco de dados usando código identificador
			cpfFormatado := ""
			cpf, err := getCPFByCodIdentificador(btc.Matdmtu)
			if err != nil {
				log.Printf("Erro ao buscar CPF para %s: %v", btc.Matdmtu, err)
			} else if cpf != "" {
				// Formatar CPF (remover pontos e traços, deixar apenas números)
				cpfFormatado = strings.ReplaceAll(cpf, ".", "")
				cpfFormatado = strings.ReplaceAll(cpfFormatado, "-", "")
			} else {
				log.Printf("CPF vazio ou não encontrado para código identificador: %s", btc.Matdmtu)
			}

			// Inicializar contadores
			qteTipo1 := 0 // VT (eletrônico)
			qteTipo2 := 0 // Comum (eletrônico)
			qteTipo3 := 0 // Passe Livre
			qteTipo4 := 0 // Dinheiro
			qteTipo5 := 0 // Idoso
			qteTipo6 := 0 // Funcionário

			// Processar passageiros
			for _, passageiro := range operacao.Passageiros.Passageiro {
				qtd, _ := strconv.Atoi(passageiro.Qtd)
				switch passageiro.Tipo {
				case "1":
					qteTipo1 += qtd
				case "2":
					qteTipo2 += qtd
				case "3":
					qteTipo3 += qtd
				case "4":
					qteTipo4 += qtd
				case "5":
					qteTipo5 += qtd
				case "6":
					qteTipo6 += qtd
				}
			}

			// Calcular totais
			qtePaxPagantes := qteTipo1 + qteTipo2 + qteTipo4
			qteTotalPax, _ := strconv.Atoi(operacao.TotalPassageiros)

			// Calcular tempo de viagem em formato hh:mm:ss
			duracao := dataFim.Sub(dataInicio)
			horas := int(duracao.Hours())
			minutos := int(duracao.Minutes()) % 60
			segundos := int(duracao.Seconds()) % 60
			tempoViagem := fmt.Sprintf("%02d:%02d:%02d", horas, minutos, segundos)

			// Extrair apenas a data (sem hora)
			dataInicioViagem := time.Date(dataInicio.Year(), dataInicio.Month(), dataInicio.Day(), 0, 0, 0, 0, dataInicio.Location())

			// Extrair apenas as horas
			horaInicioViagem := dataInicio.Format("15:04:05")
			horaFinalViagem := dataFim.Format("15:04:05")

			// Criar estrutura de dados
			operacaoData := GroupedData{
				Empresa:             "Amazonia Inter Turismo LTDA",
				PrefixoANTT:         prefixoANTT,
				Linha:               linhaCerta,
				Sentido:             sentido,
				DataInicioViagem:    dataInicioViagem,
				HoraInicioViagem:    horaInicioViagem,
				HoraFinalViagem:     horaFinalViagem,
				QtePaxPagantes:      qtePaxPagantes,
				Idoso:               qteTipo5,
				PasseLivre:          qteTipo3,
				QteOutrasGratuidade: qteTipo6,
				QteTotalPax:         qteTotalPax,
				QtePagoDinheiro:     qteTipo4,
				QtePagoEletronico:   qteTipo1 + qteTipo2,
				DistanciaViagem:     0,
				TempoViagem:         tempoViagem,
				VelocidadeMedia:     0,
				LtAberturaViagem:    latAbertura,
				LgAberturaViagem:    lngAbertura,
				LtFechamentoViagem:  latFechamento,
				LgFechamentoViagem:  lngFechamento,
				VeiculoNumero:       veiculoPlaca,
				CPFRodoviario:       cpfFormatado,
			}

			operacoesData = append(operacoesData, operacaoData)
		}
	}

	csvPath := "output.csv"
	csvFile, err := os.Create(csvPath)
	if err != nil {
		return "", err
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	writer.Comma = ';'
	defer writer.Flush()

	headers := []string{
		"EMPRESA",
		"PREFIXO",
		"CODIGO_LINHA",
		"SENTIDO",
		"DATA_INICIO_VIAGEM",
		"HORA_INICIO_VIAGEM",
		"HORA_FINAL_VIAGEM",
		"QTE_PAX_PAGANTES",
		"QTE_IDOSO",
		"QTE_PL",
		"QTE_OUTRAS_GRATUIDADE",
		"QTE_TOTAL_PAX",
		"QTE_PAGO_DINHEIRO",
		"QTE_PAGO_ELETRONICO",
		"DISTANCIA_VIAGEM",
		"TEMPO_VIAGEM",
		"VELOCIDADE_MEDIA",
		"LT_ABERTURA_VIAGEM",
		"LG_ABERTURA_VIAGEM",
		"LT_FECHAMENTO_VIAGEM",
		"LG_FECHAMENTO_VIAGEM",
		"VEICULO_NUMERO",
		"CPF_RODOVIARIO",
	}

	if err := writer.Write(headers); err != nil {
		return "", err
	}

	// Escrever dados - uma linha por operação
	for _, data := range operacoesData {
		values := []string{
			data.Empresa,
			data.PrefixoANTT,
			data.Linha,
			data.Sentido,
			data.DataInicioViagem.Format("02/01/2006"),
			data.HoraInicioViagem,
			data.HoraFinalViagem,
			strconv.Itoa(data.QtePaxPagantes),
			strconv.Itoa(data.Idoso),
			strconv.Itoa(data.PasseLivre),
			strconv.Itoa(data.QteOutrasGratuidade),
			strconv.Itoa(data.QteTotalPax),
			strconv.Itoa(data.QtePagoDinheiro),
			strconv.Itoa(data.QtePagoEletronico),
			strconv.Itoa(data.DistanciaViagem),
			data.TempoViagem,
			strconv.FormatFloat(data.VelocidadeMedia, 'f', 2, 64),
			data.LtAberturaViagem,
			data.LgAberturaViagem,
			data.LtFechamentoViagem,
			data.LgFechamentoViagem,
			data.VeiculoNumero,
			data.CPFRodoviario,
		}

		if err := writer.Write(values); err != nil {
			return "", err
		}
	}

	return csvPath, nil
}
