package main

import (
	"database/sql"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_GetCPFByCodIdentificador testa a busca de CPF no banco real
// Este teste requer que DATABASE_PUBLIC_URL esteja configurada
func TestIntegration_GetCPFByCodIdentificador(t *testing.T) {
	// Carregar variáveis de ambiente
	godotenv.Load()

	// Verificar se a variável de ambiente está configurada
	databaseURL := os.Getenv("DATABASE_PUBLIC_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_PUBLIC_URL não está configurada, pulando teste de integração")
	}

	// Resetar o pool de conexões para forçar nova inicialização
	originalDBPool := dbPool
	originalDBPoolInitErr := dbPoolInitErr

	// Resetar variáveis globais
	dbPool = nil
	dbPoolOnce = sync.Once{} // nolint:staticcheck // Reset necessário para teste
	dbPoolInitErr = nil

	// Limpar cache
	cpfCacheLock.Lock()
	cpfCache = make(map[string]string)
	cpfCacheLock.Unlock()

	// Testar conexão
	db, err := getDBConnection()
	require.NoError(t, err, "Deve conseguir conectar ao banco de dados")
	require.NotNil(t, db, "Conexão não deve ser nil")

	// Testar ping
	err = db.Ping()
	require.NoError(t, err, "Ping deve funcionar")

	// Buscar um código identificador real do banco para testar
	var codIdentificadorTest string
	var cpfEsperado string
	err = db.QueryRow(`
		SELECT cod_identificador::text, cpf 
		FROM pessoa 
		WHERE cpf IS NOT NULL AND cpf != '' 
		LIMIT 1
	`).Scan(&codIdentificadorTest, &cpfEsperado)

	if err == sql.ErrNoRows {
		t.Skip("Não há registros com CPF no banco, pulando teste")
	}
	require.NoError(t, err, "Deve conseguir buscar um registro de teste")

	t.Logf("Testando com cod_identificador: %s, CPF esperado: %s", codIdentificadorTest, cpfEsperado)

	// Testar a função getCPFByCodIdentificador
	result, err := getCPFByCodIdentificador(codIdentificadorTest)

	// Verificar resultados
	assert.NoError(t, err, "Não deve retornar erro")
	assert.NotEmpty(t, result, "CPF não deve estar vazio")
	assert.Equal(t, cpfEsperado, result, "CPF deve corresponder ao esperado")

	// Testar cache - segunda chamada deve usar cache
	result2, err2 := getCPFByCodIdentificador(codIdentificadorTest)
	assert.NoError(t, err2)
	assert.Equal(t, result, result2, "Segunda chamada deve retornar o mesmo valor (cache)")

	// Restaurar variáveis globais
	dbPool = originalDBPool
	dbPoolOnce = sync.Once{} // nolint:staticcheck // Reset necessário para teste
	dbPoolInitErr = originalDBPoolInitErr
}

// TestIntegration_DatabaseConnection testa a conexão com o banco
func TestIntegration_DatabaseConnection(t *testing.T) {
	godotenv.Load()

	databaseURL := os.Getenv("DATABASE_PUBLIC_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_PUBLIC_URL não está configurada, pulando teste de integração")
	}

	// Resetar pool
	originalDBPool := dbPool
	originalDBPoolInitErr := dbPoolInitErr

	dbPool = nil
	dbPoolOnce = sync.Once{} // nolint:staticcheck // Reset necessário para teste
	dbPoolInitErr = nil

	db, err := getDBConnection()
	require.NoError(t, err)
	require.NotNil(t, db)

	// Testar ping
	err = db.Ping()
	require.NoError(t, err)

	// Testar query simples
	var result int
	err = db.QueryRow("SELECT 1").Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, 1, result)

	// Verificar se a tabela pessoa existe
	var tableExists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'pessoa'
		)
	`).Scan(&tableExists)
	require.NoError(t, err)
	assert.True(t, tableExists, "Tabela pessoa deve existir")

	// Contar registros
	var totalRecords int
	err = db.QueryRow("SELECT COUNT(*) FROM pessoa").Scan(&totalRecords)
	require.NoError(t, err)
	t.Logf("Total de registros na tabela pessoa: %d", totalRecords)

	// Verificar registros com CPF
	var recordsWithCPF int
	err = db.QueryRow("SELECT COUNT(*) FROM pessoa WHERE cpf IS NOT NULL AND cpf != ''").Scan(&recordsWithCPF)
	require.NoError(t, err)
	t.Logf("Registros com CPF preenchido: %d", recordsWithCPF)

	// Restaurar
	dbPool = originalDBPool
	dbPoolOnce = sync.Once{} // nolint:staticcheck // Reset necessário para teste
	dbPoolInitErr = originalDBPoolInitErr
}

// TestIntegration_QueryCPFDirect testa query direta de CPF
func TestIntegration_QueryCPFDirect(t *testing.T) {
	godotenv.Load()

	databaseURL := os.Getenv("DATABASE_PUBLIC_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_PUBLIC_URL não está configurada, pulando teste de integração")
	}

	// Resetar pool
	originalDBPool := dbPool
	originalDBPoolInitErr := dbPoolInitErr

	dbPool = nil
	dbPoolOnce = sync.Once{} // nolint:staticcheck // Reset necessário para teste
	dbPoolInitErr = nil

	db, err := getDBConnection()
	require.NoError(t, err)

	// Buscar alguns registros com CPF
	rows, err := db.Query(`
		SELECT cod_identificador, cpf 
		FROM pessoa 
		WHERE cpf IS NOT NULL AND cpf != '' 
		LIMIT 5
	`)
	if err == sql.ErrNoRows {
		t.Skip("Não há registros com CPF")
	}
	require.NoError(t, err)
	defer rows.Close()

	var testCases []struct {
		codIdentificador int
		cpf              string
	}

	for rows.Next() {
		var cod int
		var cpf sql.NullString
		err := rows.Scan(&cod, &cpf)
		require.NoError(t, err)

		if cpf.Valid && cpf.String != "" {
			testCases = append(testCases, struct {
				codIdentificador int
				cpf              string
			}{
				codIdentificador: cod,
				cpf:              cpf.String,
			})
		}
	}

	if len(testCases) == 0 {
		t.Skip("Não há casos de teste com CPF válido")
	}

	// Limpar cache
	cpfCacheLock.Lock()
	cpfCache = make(map[string]string)
	cpfCacheLock.Unlock()

	// Testar cada caso
	for _, tc := range testCases {
		// Converter int para string usando strconv
		codStr := strconv.Itoa(tc.codIdentificador)
		expectedCPF := tc.cpf
		
		t.Run("cod_"+codStr, func(t *testing.T) {
			result, err := getCPFByCodIdentificador(codStr)
			assert.NoError(t, err)
			assert.Equal(t, expectedCPF, result, 
				"CPF para cod_identificador %s deve ser %s", codStr, expectedCPF)
		})
	}

	// Restaurar
	dbPool = originalDBPool
	dbPoolOnce = sync.Once{} // nolint:staticcheck // Reset necessário para teste
	dbPoolInitErr = originalDBPoolInitErr
}

