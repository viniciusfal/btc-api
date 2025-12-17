package main

import (
	"database/sql"
	"sync"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

// TestGetCPFByCodIdentificador_Success testa busca de CPF com sucesso
func TestGetCPFByCodIdentificador_Success(t *testing.T) {
	// Criar mock do banco
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Erro ao criar mock: %v", err)
	}
	defer db.Close()

	// Substituir temporariamente o dbPool
	originalDBPool := dbPool
	originalDBPoolOnce := dbPoolOnce
	originalDBPoolInitErr := dbPoolInitErr

	// Resetar variáveis globais completamente
	dbPool = db
	dbPoolOnce = sync.Once{}
	dbPoolInitErr = nil
	
	// Executar o Once manualmente para marcar como executado
	// Isso faz com que getDBConnection retorne dbPool diretamente sem tentar inicializar
	dbPoolOnce.Do(func() {
		// Não fazer nada, apenas marcar como executado
	})

	// Limpar cache
	cpfCacheLock.Lock()
	cpfCache = make(map[string]string)
	cpfCacheLock.Unlock()

	// Configurar expectativas do mock
	codIdentificador := "951716"
	cpfEsperado := "377.209.881-91"

	rows := sqlmock.NewRows([]string{"cpf"}).AddRow(cpfEsperado)
	mock.ExpectQuery("SELECT cpf FROM pessoa WHERE cod_identificador = \\$1").
		WithArgs(951716).
		WillReturnRows(rows)

	// Executar função
	result, err := getCPFByCodIdentificador(codIdentificador)

	// Verificar resultados
	assert.NoError(t, err)
	assert.Equal(t, cpfEsperado, result)

	// Verificar se todas as expectativas foram atendidas
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// Restaurar variáveis globais
	dbPool = originalDBPool
	dbPoolOnce = originalDBPoolOnce
	dbPoolInitErr = originalDBPoolInitErr
}

// TestGetCPFByCodIdentificador_NotFound testa quando CPF não é encontrado
func TestGetCPFByCodIdentificador_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Erro ao criar mock: %v", err)
	}
	defer db.Close()

	originalDBPool := dbPool
	originalDBPoolOnce := dbPoolOnce
	originalDBPoolInitErr := dbPoolInitErr

	dbPool = db
	dbPoolOnce = sync.Once{}
	dbPoolInitErr = nil
	dbPoolOnce.Do(func() {})

	cpfCacheLock.Lock()
	cpfCache = make(map[string]string)
	cpfCacheLock.Unlock()

	codIdentificador := "999999"

	mock.ExpectQuery("SELECT cpf FROM pessoa WHERE cod_identificador = \\$1").
		WithArgs(999999).
		WillReturnError(sql.ErrNoRows)

	result, err := getCPFByCodIdentificador(codIdentificador)

	assert.NoError(t, err)
	assert.Equal(t, "", result)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	dbPool = originalDBPool
	dbPoolOnce = originalDBPoolOnce
	dbPoolInitErr = originalDBPoolInitErr
}

// TestGetCPFByCodIdentificador_NullCPF testa quando CPF é NULL no banco
func TestGetCPFByCodIdentificador_NullCPF(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Erro ao criar mock: %v", err)
	}
	defer db.Close()

	originalDBPool := dbPool
	originalDBPoolOnce := dbPoolOnce
	originalDBPoolInitErr := dbPoolInitErr

	dbPool = db
	dbPoolOnce = sync.Once{}
	dbPoolInitErr = nil
	dbPoolOnce.Do(func() {})

	cpfCacheLock.Lock()
	cpfCache = make(map[string]string)
	cpfCacheLock.Unlock()

	codIdentificador := "951716"

	rows := sqlmock.NewRows([]string{"cpf"}).AddRow(nil)
	mock.ExpectQuery("SELECT cpf FROM pessoa WHERE cod_identificador = \\$1").
		WithArgs(951716).
		WillReturnRows(rows)

	result, err := getCPFByCodIdentificador(codIdentificador)

	assert.NoError(t, err)
	assert.Equal(t, "", result)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	dbPool = originalDBPool
	dbPoolOnce = originalDBPoolOnce
	dbPoolInitErr = originalDBPoolInitErr
}

// TestGetCPFByCodIdentificador_Cache testa se o cache está funcionando
func TestGetCPFByCodIdentificador_Cache(t *testing.T) {
	// Limpar cache
	cpfCacheLock.Lock()
	cpfCache = make(map[string]string)
	cpfCache["951716"] = "377.209.881-91"
	cpfCacheLock.Unlock()

	// Não deve chamar o banco se estiver no cache
	result, err := getCPFByCodIdentificador("951716")

	assert.NoError(t, err)
	assert.Equal(t, "377.209.881-91", result)

	// Limpar cache após teste
	cpfCacheLock.Lock()
	cpfCache = make(map[string]string)
	cpfCacheLock.Unlock()
}

// TestGetCPFByCodIdentificador_StringConversion testa conversão de string para int
func TestGetCPFByCodIdentificador_StringConversion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Erro ao criar mock: %v", err)
	}
	defer db.Close()

	originalDBPool := dbPool
	originalDBPoolOnce := dbPoolOnce
	originalDBPoolInitErr := dbPoolInitErr

	dbPool = db
	dbPoolOnce = sync.Once{}
	dbPoolInitErr = nil
	dbPoolOnce.Do(func() {})

	cpfCacheLock.Lock()
	cpfCache = make(map[string]string)
	cpfCacheLock.Unlock()

	// Testar com código que não pode ser convertido para int
	codIdentificador := "abc123"
	cpfEsperado := "123.456.789-00"

	rows := sqlmock.NewRows([]string{"cpf"}).AddRow(cpfEsperado)
	mock.ExpectQuery("SELECT cpf FROM pessoa WHERE cod_identificador = \\$1").
		WithArgs("abc123").
		WillReturnRows(rows)

	result, err := getCPFByCodIdentificador(codIdentificador)

	assert.NoError(t, err)
	assert.Equal(t, cpfEsperado, result)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	dbPool = originalDBPool
	dbPoolOnce = originalDBPoolOnce
	dbPoolInitErr = originalDBPoolInitErr
}
