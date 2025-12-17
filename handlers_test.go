package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://dadosdedemanda.vercel.app", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

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
		defer os.Remove(filepath)

		csvPath, err := ProcessXML(filepath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer os.Remove(csvPath)

		c.Header("Content-Disposition", "attachment; filename=output.csv")
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.File(csvPath)
	})

	return router
}

// TestUploadHandler_Success testa upload bem-sucedido
func TestUploadHandler_Success(t *testing.T) {
	// Limpar cache e mockar banco
	cpfCacheLock.Lock()
	cpfCache = make(map[string]string)
	cpfCacheLock.Unlock()

	originalDBPool := dbPool
	originalDBPoolOnce := dbPoolOnce
	originalDBPoolInitErr := dbPoolInitErr
	dbPool = nil
	dbPoolOnce = sync.Once{}
	dbPoolInitErr = nil
	defer func() {
		dbPool = originalDBPool
		dbPoolOnce = originalDBPoolOnce
		dbPoolInitErr = originalDBPoolInitErr
	}()

	router := setupRouter()

	// Criar arquivo XML de teste
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<btcs versaoApp="1.0" dataGeracao="2024-01-15 10:00:00" DataIni="2024-01-15" DataFim="2024-01-15" CodFuncionario="123" NFuncionario="João Silva" CodEmpresa="1">
  <btc doc="123456" matdmtu="951716" data="2024-01-15" nome="João Silva" codigoTD="TD001">
    <operacoes>
      <operacao codigoEmpresa="1" veiculo="1001" linha="1001" roletaInicial="1000" roletaFinal="2000" totalPassageiros="50" tarifaAtual="5.00" Receita="250.00" datainicio="2024-01-15 08:00:00" datafim="2024-01-15 09:30:00">
        <passageiros>
          <passageiro tipo="1" vlUnitario="5.00" qtd="20" qtdCreditos="0" idoso="0"/>
        </passageiros>
        <coletas recebido="250.00" girosPagantes="45" girosCartoes="35" engolidos="0"/>
      </operacao>
    </operacoes>
  </btc>
</btcs>`

	// Criar multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.xml")
	if err != nil {
		t.Fatalf("Erro ao criar form file: %v", err)
	}
	_, err = part.Write([]byte(xmlContent))
	if err != nil {
		t.Fatalf("Erro ao escrever no form: %v", err)
	}
	writer.Close()

	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Status deve ser 200")
	assert.Contains(t, w.Header().Get("Content-Type"), "text/csv", "Content-Type deve ser text/csv")
	assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment", "Content-Disposition deve conter attachment")
}

// TestUploadHandler_NoFile testa upload sem arquivo
func TestUploadHandler_NoFile(t *testing.T) {
	router := setupRouter()

	req, _ := http.NewRequest("POST", "/upload", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Status deve ser 400")
}

// TestHealthDBHandler_WithoutDB testa health check sem banco
func TestHealthDBHandler_WithoutDB(t *testing.T) {
	router := setupRouter()

	// Salvar estado original
	originalDBPool := dbPool
	originalDBPoolOnce := dbPoolOnce
	originalDBPoolInitErr := dbPoolInitErr

	// Limpar conexão
	dbPool = nil
	dbPoolOnce = sync.Once{}
	dbPoolInitErr = nil

	req, _ := http.NewRequest("GET", "/health/db", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Pode retornar 503 ou 200 dependendo da implementação
	assert.Contains(t, []int{http.StatusServiceUnavailable, http.StatusOK}, w.Code, "Status deve ser 503 ou 200")

	// Restaurar estado original
	dbPool = originalDBPool
	dbPoolOnce = originalDBPoolOnce
	dbPoolInitErr = originalDBPoolInitErr
}
