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

// ParametroViagem representa os dados da tabela parametro_viagem
type ParametroViagem struct {
	CodLinha         int
	Local1           string
	Local2           string
	Linha            string
	CodANTT          string
	Lat1             string
	Long1            string
	Lat2             string
	Long2            string
	DistanciaKm      sql.NullInt64
	DistanciaMinutos sql.NullInt64
}

var (
	cpfCache       = make(map[string]string)
	cpfCacheLock   sync.RWMutex
	linhaCache     = make(map[string]*ParametroViagem)
	linhaCacheLock sync.RWMutex
	dbPool         *sql.DB
	dbPoolOnce     sync.Once
	dbPoolInitErr  error
)

// getDBConnection retorna o pool de conexões com o banco de dados PostgreSQL
func getDBConnection() (*sql.DB, error) {
	dbPoolOnce.Do(func() {
		// Usar apenas variáveis de ambiente do sistema (Railway)
		// Não tentar carregar .env - Railway usa variáveis de ambiente diretamente

		// Tentar múltiplas variáveis de ambiente na ordem de prioridade
		// DATABASE_URL: Para produção no Railway (rede privada, mais rápida e segura)
		// DATABASE_PUBLIC_URL: Para desenvolvimento local ou acesso externo
		var databaseURL string
		envVars := []string{"DATABASE_URL", "DATABASE_PUBLIC_URL", "POSTGRES_URL"}
		var foundVar string

		for _, envVar := range envVars {
			databaseURL = os.Getenv(envVar)
			if databaseURL != "" {
				foundVar = envVar
				break
			}
		}

		// Fallback: usar URL hardcoded se nenhuma variável de ambiente for encontrada
		if databaseURL == "" {
			databaseURL = "postgresql://postgres:XNcPMtKXjbHZWIboytPDPQkOlQsNqjEL@yamanote.proxy.rlwy.net:45628/railway"
			foundVar = "hardcoded"
			log.Printf("AVISO: Nenhuma variável de ambiente encontrada, usando URL hardcoded")
		}

		// Log apenas uma vez na inicialização (ocultar senha)
		urlForLog := databaseURL
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
		log.Printf("Banco de dados: variável %s encontrada - %s", foundVar, urlForLog)

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
		// Se não conseguir conectar, retornar string vazia (não bloquear processamento)
		// Erro já foi logado na inicialização
		return "", nil
	}

	if db == nil {
		// Banco não inicializado, erro já foi logado
		return "", nil
	}

	// Testar conexão com um ping rápido (silencioso)
	if err := db.Ping(); err != nil {
		return "", nil
	}

	// Verificar se código identificador está vazio
	if codIdentificador == "" {
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
		queryParam = codInt
	} else {
		queryParam = codIdentificador
	}

	err = db.QueryRow(query, queryParam).Scan(&cpf)
	if err != nil {
		if err == sql.ErrNoRows {
			// Não encontrou, salvar string vazia no cache
			cpfCacheLock.Lock()
			cpfCache[codIdentificador] = ""
			cpfCacheLock.Unlock()
			return "", nil
		}
		return "", fmt.Errorf("erro ao consultar CPF: %w", err)
	}

	cpfValue := ""
	if cpf.Valid {
		cpfValue = cpf.String
	}

	// Salvar no cache
	cpfCacheLock.Lock()
	cpfCache[codIdentificador] = cpfValue
	cpfCacheLock.Unlock()

	return cpfValue, nil
}

// getParametroViagemByCodLinha busca informações da linha na tabela parametro_viagem usando o código da linha
func getParametroViagemByCodLinha(codLinha string) (*ParametroViagem, error) {
	// Verificar cache primeiro
	linhaCacheLock.RLock()
	if linha, exists := linhaCache[codLinha]; exists {
		linhaCacheLock.RUnlock()
		return linha, nil
	}
	linhaCacheLock.RUnlock()

	// Buscar no banco de dados
	db, err := getDBConnection()
	if err != nil {
		// Se não conseguir conectar, retornar nil (não bloquear processamento)
		return nil, nil
	}

	if db == nil {
		return nil, nil
	}

	// Testar conexão com um ping rápido (silencioso)
	if err := db.Ping(); err != nil {
		return nil, nil
	}

	// Converter código da linha para inteiro
	codInt, errConv := strconv.Atoi(codLinha)
	if errConv != nil {
		// Código inválido, salvar nil no cache
		linhaCacheLock.Lock()
		linhaCache[codLinha] = nil
		linhaCacheLock.Unlock()
		return nil, nil
	}

	var param ParametroViagem
	query := `
		SELECT cod_linha, local1, local2, linha, cod_antt, 
		       lat1, long1, lat2, long2, distancia_km, distancia_minutos
		FROM parametro_viagem 
		WHERE cod_linha = $1
		LIMIT 1
	`

	err = db.QueryRow(query, codInt).Scan(
		&param.CodLinha,
		&param.Local1,
		&param.Local2,
		&param.Linha,
		&param.CodANTT,
		&param.Lat1,
		&param.Long1,
		&param.Lat2,
		&param.Long2,
		&param.DistanciaKm,
		&param.DistanciaMinutos,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Não encontrou, salvar nil no cache
			linhaCacheLock.Lock()
			linhaCache[codLinha] = nil
			linhaCacheLock.Unlock()
			return nil, nil
		}
		return nil, fmt.Errorf("erro ao consultar parametro_viagem: %w", err)
	}

	// Salvar no cache
	linhaCacheLock.Lock()
	linhaCache[codLinha] = &param
	linhaCacheLock.Unlock()

	return &param, nil
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
		log.Printf("AVISO: Nenhuma variável de ambiente de banco encontrada. Configure DATABASE_URL (produção) ou DATABASE_PUBLIC_URL (desenvolvimento) nas variáveis de ambiente do Railway.")
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
		envVars := []string{"DATABASE_URL", "DATABASE_PUBLIC_URL", "POSTGRES_URL"}
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

			// Buscar informações da linha do banco de dados
			var linhaCerta, prefixoANTT string
			var latAbertura, lngAbertura, latFechamento, lngFechamento string
			param, err := getParametroViagemByCodLinha(operacao.Linha)
			if err == nil && param != nil {
				linhaCerta = strconv.Itoa(param.CodLinha)
				prefixoANTT = strings.ReplaceAll(param.CodANTT, "-", "")

				// Preencher coordenadas baseado no sentido da viagem
				if sentido == "GO-DF" {
					// Sentido ida: Local1 → abertura, Local2 → fechamento
					latAbertura = param.Lat1
					lngAbertura = param.Long1
					latFechamento = param.Lat2
					lngFechamento = param.Long2
				} else {
					// Sentido volta (DF-GO): Local2 → abertura, Local1 → fechamento
					latAbertura = param.Lat2
					lngAbertura = param.Long2
					latFechamento = param.Lat1
					lngFechamento = param.Long1
				}
			}

			// Buscar placa do veículo
			veiculoPlaca := operacao.Veiculo
			if car, existe := placas[operacao.Veiculo]; existe {
				veiculoPlaca = car.Placa
			}

			// Buscar CPF do motorista no banco de dados usando código identificador
			cpfFormatado := ""

			// Buscar CPF do motorista (sem logs excessivos)
			if btc.Matdmtu != "" {
				cpf, err := getCPFByCodIdentificador(btc.Matdmtu)
				if err == nil && cpf != "" {
					// Formatar CPF (remover pontos e traços, deixar apenas números)
					cpfFormatado = strings.ReplaceAll(cpf, ".", "")
					cpfFormatado = strings.ReplaceAll(cpfFormatado, "-", "")
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

			// Calcular distância da viagem - priorizar dados da tabela
			var distanciaKm float64
			if param != nil {
				// Priorizar distância da tabela se disponível
				if param.DistanciaKm.Valid {
					distanciaKm = float64(param.DistanciaKm.Int64)
				} else {
					// Fallback: calcular usando coordenadas geográficas se distância não estiver na tabela
					var lat1, lng1, lat2, lng2 string
					if sentido == "GO-DF" {
						// Sentido ida: Local1 → abertura, Local2 → fechamento
						lat1 = param.Lat1
						lng1 = param.Long1
						lat2 = param.Lat2
						lng2 = param.Long2
					} else {
						// Sentido volta (DF-GO): Local2 → abertura, Local1 → fechamento
						lat1 = param.Lat2
						lng1 = param.Long2
						lat2 = param.Lat1
						lng2 = param.Long1
					}

					// Verificar se coordenadas estão preenchidas
					if lat1 != "" && lng1 != "" && lat2 != "" && lng2 != "" {
						distanciaKm = calculateGeographicDistance(lat1, lng1, lat2, lng2)
					} else {
						log.Printf("AVISO: Coordenadas incompletas para linha %s (sentido %s): lat1=%s, lng1=%s, lat2=%s, lng2=%s",
							operacao.Linha, sentido, lat1, lng1, lat2, lng2)
					}
				}
			}

			// Calcular velocidade média usando distancia_minutos da tabela quando disponível
			var velocidadeMedia float64

			if param != nil && param.DistanciaKm.Valid && param.DistanciaMinutos.Valid {
				// Usar dados da tabela: velocidade = distância (km) / tempo (horas)
				// distancia_minutos está em minutos, converter para horas
				distanciaKmTabela := float64(param.DistanciaKm.Int64)
				distanciaMinutosTabela := float64(param.DistanciaMinutos.Int64)

				if distanciaMinutosTabela > 0 {
					// Converter minutos para horas
					tempoHoras := distanciaMinutosTabela / 60.0
					velocidadeMedia = distanciaKmTabela / tempoHoras
				} else {
					// Se distancia_minutos for 0 ou inválido, usar velocidade média esperada
					velocidadeMedia = 45.0
				}
			} else if distanciaKm > 0 {
				// Fallback: calcular usando tempo real da viagem se não houver dados na tabela
				tempoHorasCalculado := duracao.Hours()

				if tempoHorasCalculado > 0 {
					velocidadeCalculada := distanciaKm / tempoHorasCalculado

					// Validar se a velocidade calculada é razoável
					const VELOCIDADE_MAXIMA_PERMITIDA = 70.0 // km/h (limite legal para ônibus)
					const VELOCIDADE_MINIMA_ACEITAVEL = 15.0 // km/h (abaixo disso o tempo inclui pausas)

					if velocidadeCalculada >= VELOCIDADE_MINIMA_ACEITAVEL && velocidadeCalculada <= VELOCIDADE_MAXIMA_PERMITIDA {
						velocidadeMedia = velocidadeCalculada
					} else {
						// Velocidade fora da faixa = tempo incorreto (inclui pausas)
						velocidadeMedia = 45.0 // Velocidade média esperada
					}
				} else {
					velocidadeMedia = 45.0
				}
			} else {
				// Distância zero, não pode calcular
				velocidadeMedia = 0
			}

			// Extrair apenas a data (sem hora)
			dataInicioViagem := time.Date(dataInicio.Year(), dataInicio.Month(), dataInicio.Day(), 0, 0, 0, 0, dataInicio.Location())

			// Extrair apenas as horas
			horaInicioViagem := dataInicio.Format("15:04:05")
			horaFinalViagem := dataFim.Format("15:04:05")

			// Usar distância da tabela se disponível, senão usar a calculada
			var distanciaFinal float64
			if param != nil && param.DistanciaKm.Valid {
				distanciaFinal = float64(param.DistanciaKm.Int64)
			} else {
				distanciaFinal = distanciaKm
			}

			// Arredondar distância e velocidade para cima e converter para inteiro
			distanciaViagemInt := int(math.Ceil(distanciaFinal))
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
			strconv.Itoa(int(data.DistanciaViagem)),
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
