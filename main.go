package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
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
	DistanciaViagem     float64
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
		// Usar apenas variáveis de ambiente do sistema (Railway)
		// Não tentar carregar .env - Railway usa variáveis de ambiente diretamente

		// Tentar múltiplas variáveis de ambiente na ordem de prioridade
		var databaseURL string
		envVars := []string{"DATABASE_URL", "POSTGRES_URL", "DATABASE_PUBLIC_URL"}

		for _, envVar := range envVars {
			databaseURL = os.Getenv(envVar)
			if databaseURL != "" {
				log.Printf("Variável de ambiente encontrada: %s", envVar)
				break
			} else {
				log.Printf("Variável de ambiente %s não encontrada ou vazia", envVar)
			}
		}

		if databaseURL == "" {
			dbPoolInitErr = fmt.Errorf("nenhuma variável de ambiente de banco encontrada (DATABASE_URL, POSTGRES_URL, DATABASE_PUBLIC_URL)")
			log.Printf("ERRO CRÍTICO: %v", dbPoolInitErr)
			log.Printf("ERRO: Verifique se a variável DATABASE_PUBLIC_URL está configurada no ambiente ou no arquivo .env")
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

		// Adicionar parâmetros SSL se não estiverem presentes na URL
		// lib/pq só suporta: require (default), verify-full, verify-ca, e disable
		// Adicionar sslmode=require e sslrootcert para evitar avisos de ALPN
		if !strings.Contains(databaseURL, "sslmode=") {
			separator := "?"
			if strings.Contains(databaseURL, "?") {
				separator = "&"
			}
			databaseURL = databaseURL + separator + "sslmode=require"
			log.Printf("Parâmetro SSL adicionado à connection string (sslmode=require)")
		} else {
			log.Printf("URL já contém parâmetros SSL, usando configuração original")
		}

		// Adicionar fallback_application_name para melhorar compatibilidade
		if !strings.Contains(databaseURL, "fallback_application_name=") {
			separator := "&"
			if !strings.Contains(databaseURL, "?") {
				separator = "?"
			}
			databaseURL = databaseURL + separator + "fallback_application_name=btc-api"
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

		// Criar tabela pessoa se não existir (estrutura real: id_pessoa como PK, cod_identificador como campo)
		createTableSQL := `
			CREATE TABLE IF NOT EXISTS pessoa (
				id_pessoa SERIAL PRIMARY KEY,
				cod_identificador INTEGER NOT NULL,
				cpf VARCHAR(14),
				funcao VARCHAR(100),
				status BOOLEAN DEFAULT true
			);
			CREATE INDEX IF NOT EXISTS idx_pessoa_cod_identificador ON pessoa(cod_identificador);
		`
		_, err = dbPool.Exec(createTableSQL)
		if err != nil {
			log.Printf("AVISO: Erro ao criar tabela pessoa (pode já existir): %v", err)
		} else {
			log.Printf("Tabela pessoa verificada/criada com sucesso")
		}

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

// calculateGeographicDistance calcula a distância entre duas coordenadas geográficas usando a fórmula de Haversine
// Retorna a distância em quilômetros
func calculateGeographicDistance(lat1, lng1, lat2, lng2 string) float64 {
	// Converter strings para float64
	lat1Float, err1 := strconv.ParseFloat(lat1, 64)
	lng1Float, err2 := strconv.ParseFloat(lng1, 64)
	lat2Float, err3 := strconv.ParseFloat(lat2, 64)
	lng2Float, err4 := strconv.ParseFloat(lng2, 64)

	// Se houver erro na conversão, retornar 0
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		log.Printf("Erro ao converter coordenadas: lat1=%s, lng1=%s, lat2=%s, lng2=%s", lat1, lng1, lat2, lng2)
		return 0
	}

	// Raio médio da Terra em quilômetros
	const R = 6371.0

	// Converter graus para radianos
	lat1Rad := lat1Float * math.Pi / 180.0
	lng1Rad := lng1Float * math.Pi / 180.0
	lat2Rad := lat2Float * math.Pi / 180.0
	lng2Rad := lng2Float * math.Pi / 180.0

	// Diferenças
	dLat := lat2Rad - lat1Rad
	dLng := lng2Rad - lng1Rad

	// Fórmula de Haversine
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	// Distância em quilômetros
	distance := R * c

	return distance
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
		log.Printf("ERRO CRÍTICO ao obter conexão com banco para código %s: %v", codIdentificador, err)
		log.Printf("ERRO: Verificar variáveis de ambiente DATABASE_URL, POSTGRES_URL ou DATABASE_PUBLIC_URL")
		// Se não conseguir conectar, retornar string vazia (não bloquear processamento)
		return "", nil
	}

	if db == nil {
		log.Printf("ERRO CRÍTICO: conexão com banco é nil para código %s. Banco não foi inicializado.", codIdentificador)
		// Se db for nil, retornar string vazia
		return "", nil
	}

	// Testar conexão com um ping rápido
	if err := db.Ping(); err != nil {
		log.Printf("ERRO: Conexão com banco está inativa (Ping falhou) para código %s: %v", codIdentificador, err)
		return "", nil
	}

	// Verificar se código identificador está vazio
	if codIdentificador == "" {
		log.Printf("ERRO: código identificador está vazio")
		cpfCacheLock.Lock()
		cpfCache[codIdentificador] = ""
		cpfCacheLock.Unlock()
		return "", nil
	}

	// Converter código identificador para inteiro
	codInt, errConv := strconv.Atoi(codIdentificador)

	var cpf sql.NullString
	query := "SELECT cpf FROM pessoa WHERE cod_identificador = $1"

	// Usar inteiro se a conversão foi bem-sucedida, senão usar string
	var queryParam interface{}
	if errConv == nil {
		// Conversão bem-sucedida, usar inteiro
		queryParam = codInt
		log.Printf("Consultando CPF para código identificador: '%s' (convertido para INTEGER: %d)", codIdentificador, codInt)
	} else {
		// Falha na conversão, tentar como string
		queryParam = codIdentificador
		log.Printf("AVISO: Não foi possível converter código '%s' para inteiro, tentando como STRING: %v", codIdentificador, errConv)
	}

	log.Printf("Executando query: %s com parâmetro: %v (tipo: %T)", query, queryParam, queryParam)

	err = db.QueryRow(query, queryParam).Scan(&cpf)
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
		log.Printf("CPF encontrado para código %s: '%s' (tamanho: %d)", codIdentificador, cpfValue, len(cpfValue))
	} else {
		log.Printf("CPF é NULL para código identificador: %s", codIdentificador)
	}

	// Verificar se CPF está vazio mesmo após encontrar registro
	if cpfValue == "" {
		log.Printf("AVISO: CPF encontrado mas está vazio para código %s. Verificar se o campo CPF está preenchido no banco.", codIdentificador)
	}

	// Salvar no cache
	cpfCacheLock.Lock()
	cpfCache[codIdentificador] = cpfValue
	cpfCacheLock.Unlock()

	return cpfValue, nil
}

func main() {
	// Usar apenas variáveis de ambiente do sistema (Railway)
	// Não carregar .env - Railway configura variáveis diretamente no ambiente

	// Verificar se as variáveis de ambiente estão disponíveis
	envVars := []string{"DATABASE_URL", "POSTGRES_URL", "DATABASE_PUBLIC_URL"}
	envFound := false
	for _, envVar := range envVars {
		value := os.Getenv(envVar)
		if value != "" {
			// Ocultar senha no log
			urlForLog := value
			if strings.Contains(urlForLog, "@") {
				parts := strings.Split(urlForLog, "@")
				if len(parts) > 0 {
					userPass := strings.Split(parts[0], "://")
					if len(userPass) > 1 {
						userParts := strings.Split(userPass[1], ":")
						if len(userParts) > 1 {
							urlForLog = userPass[0] + "://" + userParts[0] + ":***@" + parts[1]
						}
					}
				}
			}
			log.Printf("Variável de ambiente %s encontrada: %s", envVar, urlForLog)
			envFound = true
			break
		}
	}
	if !envFound {
		log.Printf("AVISO: Nenhuma variável de ambiente de banco encontrada. Verifique se DATABASE_PUBLIC_URL está configurada.")
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

	// Endpoint de diagnóstico completo do banco
	router.GET("/debug/db", func(c *gin.Context) {
		envVars := []string{"DATABASE_URL", "POSTGRES_URL", "DATABASE_PUBLIC_URL"}
		envStatus := make(map[string]string)
		for _, envVar := range envVars {
			value := os.Getenv(envVar)
			if value != "" {
				// Ocultar senha
				if strings.Contains(value, "@") {
					parts := strings.Split(value, "@")
					if len(parts) > 0 {
						userPass := strings.Split(parts[0], "://")
						if len(userPass) > 1 {
							userParts := strings.Split(userPass[1], ":")
							if len(userParts) > 1 {
								value = userPass[0] + "://" + userParts[0] + ":***@" + parts[1]
							}
						}
					}
				}
				envStatus[envVar] = value
			} else {
				envStatus[envVar] = "não definida"
			}
		}

		db, err := getDBConnection()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":       "error",
				"error":        err.Error(),
				"env_vars":     envStatus,
				"db_connected": false,
			})
			return
		}

		if db == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":       "error",
				"error":        "dbPool é nil",
				"env_vars":     envStatus,
				"db_connected": false,
			})
			return
		}

		// Testar ping
		pingErr := db.Ping()
		if pingErr != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":       "error",
				"error":        pingErr.Error(),
				"env_vars":     envStatus,
				"db_connected": false,
			})
			return
		}

		// Contar registros
		var totalRecords int
		db.QueryRow("SELECT COUNT(*) FROM pessoa").Scan(&totalRecords)

		// Buscar alguns registros com CPF
		rows, _ := db.Query("SELECT id_pessoa, cod_identificador, cpf, funcao, status FROM pessoa WHERE cpf IS NOT NULL AND cpf != '' LIMIT 5")
		var sampleRecords []map[string]interface{}
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id int
				var cod int
				var cpfSample sql.NullString
				var funcao sql.NullString
				var status bool
				if err := rows.Scan(&id, &cod, &cpfSample, &funcao, &status); err == nil {
					cpfStr := ""
					if cpfSample.Valid {
						cpfStr = cpfSample.String
					}
					funcaoStr := ""
					if funcao.Valid {
						funcaoStr = funcao.String
					}
					sampleRecords = append(sampleRecords, map[string]interface{}{
						"id_pessoa":         id,
						"cod_identificador": cod,
						"cpf":               cpfStr,
						"cpf_length":        len(cpfStr),
						"funcao":            funcaoStr,
						"status":            status,
					})
				}
			}
		}

		// Buscar alguns registros sem CPF
		rowsNoCPF, _ := db.Query("SELECT id_pessoa, cod_identificador, cpf, funcao FROM pessoa WHERE cpf IS NULL OR cpf = '' LIMIT 5")
		var recordsNoCPF []map[string]interface{}
		if rowsNoCPF != nil {
			defer rowsNoCPF.Close()
			for rowsNoCPF.Next() {
				var id int
				var cod int
				var cpfSample sql.NullString
				var funcao sql.NullString
				if err := rowsNoCPF.Scan(&id, &cod, &cpfSample, &funcao); err == nil {
					cpfStr := ""
					if cpfSample.Valid {
						cpfStr = cpfSample.String
					}
					funcaoStr := ""
					if funcao.Valid {
						funcaoStr = funcao.String
					}
					recordsNoCPF = append(recordsNoCPF, map[string]interface{}{
						"id_pessoa":         id,
						"cod_identificador": cod,
						"cpf":               cpfStr,
						"cpf_length":        len(cpfStr),
						"funcao":            funcaoStr,
					})
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"status":           "success",
			"db_connected":     true,
			"env_vars":         envStatus,
			"total_records":    totalRecords,
			"records_with_cpf": sampleRecords,
			"records_no_cpf":   recordsNoCPF,
		})
	})

	// Endpoint de debug para testar consulta de CPF
	router.GET("/debug/cpf/:codigo", func(c *gin.Context) {
		codigo := c.Param("codigo")
		log.Printf("DEBUG: Testando consulta de CPF para código: %s", codigo)

		// Limpar cache para forçar nova busca
		cpfCacheLock.Lock()
		delete(cpfCache, codigo)
		cpfCacheLock.Unlock()

		db, err := getDBConnection()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "error",
				"message": "Não foi possível conectar ao banco de dados",
				"error":   err.Error(),
			})
			return
		}

		// Converter para inteiro
		codInt, errConv := strconv.Atoi(codigo)
		var queryParam interface{}
		if errConv == nil {
			queryParam = codInt
		} else {
			queryParam = codigo
		}

		var cpf sql.NullString
		query := "SELECT cpf FROM pessoa WHERE cod_identificador = $1"
		err = db.QueryRow(query, queryParam).Scan(&cpf)

		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusOK, gin.H{
					"status":           "not_found",
					"message":          fmt.Sprintf("CPF não encontrado para código: %s", codigo),
					"codigo":           codigo,
					"query_param":      queryParam,
					"query_param_type": fmt.Sprintf("%T", queryParam),
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":      "error",
				"message":     "Erro ao consultar CPF",
				"error":       err.Error(),
				"codigo":      codigo,
				"query_param": queryParam,
			})
			return
		}

		cpfValue := ""
		if cpf.Valid {
			cpfValue = cpf.String
		}

		// Testar também uma consulta para ver todos os registros
		var totalRecords int
		db.QueryRow("SELECT COUNT(*) FROM pessoa").Scan(&totalRecords)

		rows, _ := db.Query("SELECT cod_identificador, cpf FROM pessoa LIMIT 10")
		var sampleRecords []map[string]interface{}
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var cod int
				var cpfSample sql.NullString
				if err := rows.Scan(&cod, &cpfSample); err == nil {
					cpfStr := ""
					if cpfSample.Valid {
						cpfStr = cpfSample.String
					}
					sampleRecords = append(sampleRecords, map[string]interface{}{
						"cod_identificador": cod,
						"cpf":               cpfStr,
						"cpf_length":        len(cpfStr),
					})
				}
			}
		}

		// Testar também usando a função getCPFByCodIdentificador
		cpfFromFunction, errFromFunction := getCPFByCodIdentificador(codigo)

		c.JSON(http.StatusOK, gin.H{
			"status":            "success",
			"total_records":     totalRecords,
			"codigo":            codigo,
			"cpf_direct_query":  cpfValue,
			"cpf_from_function": cpfFromFunction,
			"cpf_valid":         cpf.Valid,
			"query_param":       queryParam,
			"query_param_type":  fmt.Sprintf("%T", queryParam),
			"sample_records":    sampleRecords,
			"function_error": func() string {
				if errFromFunction != nil {
					return errFromFunction.Error()
				}
				return ""
			}(),
		})
	})

	// Endpoint para limpar cache de CPF (útil para debug)
	router.DELETE("/debug/cpf/cache", func(c *gin.Context) {
		cpfCacheLock.Lock()
		cpfCache = make(map[string]string)
		cpfCacheLock.Unlock()
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Cache de CPF limpo",
		})
	})

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

			// Verificar se Matdmtu está vazio
			if btc.Matdmtu == "" {
				log.Printf("AVISO: Matdmtu está vazio para motorista %s, não é possível buscar CPF", btc.Nome)
			} else {
				log.Printf("Buscando CPF para Matdmtu: '%s' (motorista: %s)", btc.Matdmtu, btc.Nome)
				cpf, err := getCPFByCodIdentificador(btc.Matdmtu)
				if err != nil {
					log.Printf("ERRO ao buscar CPF para Matdmtu %s: %v", btc.Matdmtu, err)
				} else if cpf != "" {
					// Formatar CPF (remover pontos e traços, deixar apenas números)
					cpfOriginal := cpf
					cpfFormatado = strings.ReplaceAll(cpf, ".", "")
					cpfFormatado = strings.ReplaceAll(cpfFormatado, "-", "")
					log.Printf("CPF formatado para Matdmtu %s: '%s' -> '%s'", btc.Matdmtu, cpfOriginal, cpfFormatado)
				} else {
					log.Printf("CPF vazio ou não encontrado para Matdmtu: %s (motorista: %s). Verificar se existe registro na tabela pessoa com cod_identificador = %s", btc.Matdmtu, btc.Nome, btc.Matdmtu)
				}
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

			// Dividir tipo 2 (gratuidade que inclui idoso e passe livre)
			// 1/3 vai para Passe Livre (tipo 3), 2/3 fica como Idoso (tipo 2)
			qtePasseLivre := qteTipo2 / 3
			qteIdoso := qteTipo2 - qtePasseLivre

			// Atualizar contadores: tipo 2 agora é apenas idoso, tipo 3 recebe passe livre
			qteTipo2 = qteIdoso
			qteTipo3 = qteTipo3 + qtePasseLivre

			// Calcular totais
			qtePaxPagantes := qteTipo1 + qteTipo2 + qteTipo4
			qteTotalPax, _ := strconv.Atoi(operacao.TotalPassageiros)

			// Calcular tempo de viagem em formato hh:mm:ss
			duracao := dataFim.Sub(dataInicio)
			horas := int(duracao.Hours())
			minutos := int(duracao.Minutes()) % 60
			segundos := int(duracao.Seconds()) % 60
			tempoViagem := fmt.Sprintf("%02d:%02d:%02d", horas, minutos, segundos)

			// Calcular distância da viagem usando coordenadas geográficas
			var distanciaKm float64
			if linha, existe := limb[operacao.Linha]; existe {
				// Determinar coordenadas baseado no sentido da viagem
				var lat1, lng1, lat2, lng2 string
				if sentido == "GO-DF" {
					// Sentido ida: Local1 → abertura, Local2 → fechamento
					lat1 = linha.Lat1
					lng1 = linha.Lng1
					lat2 = linha.Lat2
					lng2 = linha.Lng2
				} else {
					// Sentido volta (DF-GO): Local2 → abertura, Local1 → fechamento
					lat1 = linha.Lat2
					lng1 = linha.Lng2
					lat2 = linha.Lat1
					lng2 = linha.Lng1
				}

				// Verificar se coordenadas estão preenchidas
				if lat1 == "" || lng1 == "" || lat2 == "" || lng2 == "" {
					log.Printf("AVISO: Coordenadas incompletas para linha %s (sentido %s): lat1=%s, lng1=%s, lat2=%s, lng2=%s",
						operacao.Linha, sentido, lat1, lng1, lat2, lng2)
				} else {
					distanciaKm = calculateGeographicDistance(lat1, lng1, lat2, lng2)
					log.Printf("Distância calculada para linha %s (sentido %s): %.2f km (coords: %s,%s → %s,%s)",
						operacao.Linha, sentido, distanciaKm, lat1, lng1, lat2, lng2)
				}
			} else {
				log.Printf("ERRO: Linha %s não encontrada em access.go, distância será 0", operacao.Linha)
			}

			// Calcular tempo de viagem real
			// Problema: motorista não inverte turno, então o tempo calculado inclui pausas entre viagens
			tempoHorasCalculado := duracao.Hours()

			// Limites de velocidade para ônibus
			const VELOCIDADE_MAXIMA_PERMITIDA = 70.0 // km/h (limite legal para ônibus - não pode passar disso)
			const VELOCIDADE_MEDIA_ESPERADA = 45.0   // km/h (velocidade média típica em rodovia)
			const VELOCIDADE_MINIMA_ACEITAVEL = 25.0 // km/h (mínimo realista considerando trânsito)

			var tempoHoras float64
			var velocidadeMedia float64

			if distanciaKm > 0 && tempoHorasCalculado > 0 {
				// Calcular velocidade com o tempo informado
				velocidadeCalculada := distanciaKm / tempoHorasCalculado

				// Validar velocidade calculada
				if velocidadeCalculada > VELOCIDADE_MAXIMA_PERMITIDA {
					// Velocidade acima do permitido = tempo muito curto (impossível)
					// Recalcular tempo mínimo baseado na velocidade máxima permitida
					tempoHoras = distanciaKm / VELOCIDADE_MAXIMA_PERMITIDA
					velocidadeMedia = VELOCIDADE_MAXIMA_PERMITIDA
					log.Printf("Velocidade impossível para linha %s: %.2f km/h (acima de %.0f km/h). Tempo calculado muito curto (%.2fh). Recalculando com tempo mínimo baseado em %.0f km/h: %.2fh",
						operacao.Linha, velocidadeCalculada, VELOCIDADE_MAXIMA_PERMITIDA, tempoHorasCalculado, VELOCIDADE_MAXIMA_PERMITIDA, tempoHoras)
				} else if velocidadeCalculada < VELOCIDADE_MINIMA_ACEITAVEL {
					// Velocidade muito baixa = tempo muito longo (inclui pausas)
					// Usar tempo esperado baseado na velocidade média esperada
					tempoHoras = distanciaKm / VELOCIDADE_MEDIA_ESPERADA
					velocidadeMedia = VELOCIDADE_MEDIA_ESPERADA
					log.Printf("Velocidade muito baixa para linha %s: %.2f km/h (tempo calculado: %.2fh parece incluir pausas). Usando tempo esperado baseado em velocidade média de %.0f km/h: %.2fh",
						operacao.Linha, velocidadeCalculada, tempoHorasCalculado, VELOCIDADE_MEDIA_ESPERADA, tempoHoras)
				} else {
					// Velocidade dentro da faixa aceitável, usar tempo calculado
					tempoHoras = tempoHorasCalculado
					velocidadeMedia = velocidadeCalculada
					log.Printf("Velocidade média calculada para linha %s: %.2f km/h (distância: %.2f km, tempo: %.2f h)",
						operacao.Linha, velocidadeMedia, distanciaKm, tempoHoras)
				}
			} else if distanciaKm > 0 {
				// Tempo zero ou inválido, usar tempo esperado baseado na velocidade média
				tempoHoras = distanciaKm / VELOCIDADE_MEDIA_ESPERADA
				velocidadeMedia = VELOCIDADE_MEDIA_ESPERADA
				log.Printf("Tempo inválido (%.2fh) para linha %s, usando tempo esperado baseado em velocidade média de %.0f km/h: %.2fh",
					tempoHorasCalculado, operacao.Linha, VELOCIDADE_MEDIA_ESPERADA, tempoHoras)
			} else {
				// Distância zero, não pode calcular
				tempoHoras = 0
				velocidadeMedia = 0
				log.Printf("AVISO: Distância é 0 para linha %s, velocidade não pode ser calculada", operacao.Linha)
			}

			// Extrair apenas a data (sem hora)
			dataInicioViagem := time.Date(dataInicio.Year(), dataInicio.Month(), dataInicio.Day(), 0, 0, 0, 0, dataInicio.Location())

			// Extrair apenas as horas
			horaInicioViagem := dataInicio.Format("15:04:05")
			horaFinalViagem := dataFim.Format("15:04:05")

			// Arredondar distância e velocidade para cima e converter para inteiro
			distanciaViagemInt := int(math.Ceil(distanciaKm))
			velocidadeMediaInt := int(math.Ceil(velocidadeMedia))

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
				Idoso:               qteTipo2, // Tipo 2 após divisão (2/3 do tipo 2 original)
				PasseLivre:          qteTipo3, // Tipo 3 + 1/3 do tipo 2 original
				QteOutrasGratuidade: qteTipo6,
				QteTotalPax:         qteTotalPax,
				QtePagoDinheiro:     qteTipo4,
				QtePagoEletronico:   qteTipo1 + qteTipo2,
				DistanciaViagem:     float64(distanciaViagemInt),
				TempoViagem:         tempoViagem,
				VelocidadeMedia:     float64(velocidadeMediaInt),
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
			strconv.FormatFloat(data.DistanciaViagem, 'f', 2, 64),
			data.TempoViagem,
			strconv.Itoa(int(data.VelocidadeMedia)),
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
