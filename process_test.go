package main

import (
	"encoding/csv"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestProcessXML_ValidXML testa processamento de XML válido
func TestProcessXML_ValidXML(t *testing.T) {
	// Limpar cache de CPF e mockar banco para não falhar
	cpfCacheLock.Lock()
	cpfCache = make(map[string]string)
	cpfCacheLock.Unlock()

	// Mockar banco para não tentar conectar
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

	// Criar arquivo XML temporário
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<btcs versaoApp="1.0" dataGeracao="2024-01-15 10:00:00" DataIni="2024-01-15" DataFim="2024-01-15" CodFuncionario="123" NFuncionario="João Silva" CodEmpresa="1">
  <btc doc="123456" matdmtu="951716" data="2024-01-15" nome="João Silva" codigoTD="TD001">
    <operacoes>
      <operacao codigoEmpresa="1" veiculo="1001" linha="1001" roletaInicial="1000" roletaFinal="2000" totalPassageiros="50" tarifaAtual="5.00" Receita="250.00" datainicio="2024-01-15 08:00:00" datafim="2024-01-15 09:30:00">
        <passageiros>
          <passageiro tipo="1" vlUnitario="5.00" qtd="20" qtdCreditos="0" idoso="0"/>
          <passageiro tipo="2" vlUnitario="5.00" qtd="15" qtdCreditos="0" idoso="0"/>
          <passageiro tipo="3" vlUnitario="0.00" qtd="5" qtdCreditos="0" idoso="0"/>
          <passageiro tipo="4" vlUnitario="5.00" qtd="10" qtdCreditos="0" idoso="0"/>
          <passageiro tipo="5" vlUnitario="0.00" qtd="0" qtdCreditos="0" idoso="3"/>
          <passageiro tipo="6" vlUnitario="0.00" qtd="0" qtdCreditos="0" idoso="0"/>
        </passageiros>
        <coletas recebido="250.00" girosPagantes="45" girosCartoes="35" engolidos="0"/>
      </operacao>
    </operacoes>
  </btc>
</btcs>`

	tmpFile, err := os.CreateTemp("", "test_*.xml")
	if err != nil {
		t.Fatalf("Erro ao criar arquivo temporário: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(xmlContent)
	if err != nil {
		t.Fatalf("Erro ao escrever no arquivo: %v", err)
	}
	tmpFile.Close()

	// Processar XML
	csvPath, err := ProcessXML(tmpFile.Name())
	if err != nil {
		t.Fatalf("Erro ao processar XML: %v", err)
	}
	defer os.Remove(csvPath)

	// Verificar se CSV foi criado
	assert.FileExists(t, csvPath, "CSV deve ser criado")

	// Ler e verificar conteúdo do CSV
	file, err := os.Open(csvPath)
	if err != nil {
		t.Fatalf("Erro ao abrir CSV: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';'

	// Ler headers
	headers, err := reader.Read()
	if err != nil {
		t.Fatalf("Erro ao ler headers: %v", err)
	}

	expectedHeaders := []string{
		"EMPRESA", "PREFIXO", "CODIGO_LINHA", "SENTIDO", "DATA_INICIO_VIAGEM",
		"HORA_INICIO_VIAGEM", "HORA_FINAL_VIAGEM", "QTE_PAX_PAGANTES", "QTE_IDOSO",
		"QTE_PL", "QTE_OUTRAS_GRATUIDADE", "QTE_TOTAL_PAX", "QTE_PAGO_DINHEIRO",
		"QTE_PAGO_ELETRONICO", "DISTANCIA_VIAGEM", "TEMPO_VIAGEM", "VELOCIDADE_MEDIA",
		"LT_ABERTURA_VIAGEM", "LG_ABERTURA_VIAGEM", "LT_FECHAMENTO_VIAGEM",
		"LG_FECHAMENTO_VIAGEM", "VEICULO_NUMERO", "CPF_RODOVIARIO",
	}

	assert.Equal(t, expectedHeaders, headers, "Headers do CSV devem estar corretos")

	// Ler primeira linha de dados
	row, err := reader.Read()
	if err != nil {
		t.Fatalf("Erro ao ler linha de dados: %v", err)
	}

	// Verificar alguns campos importantes
	assert.Equal(t, "Amazonia Inter Turismo LTDA", row[0], "Empresa deve estar correta")
	assert.Equal(t, "1001", row[2], "Código da linha deve estar correto")
	assert.Equal(t, "GO-DF", row[3], "Sentido deve ser GO-DF na primeira ocorrência")
	assert.Equal(t, "15/01/2024", row[4], "Data deve estar no formato DD/MM/AAAA")
	assert.Equal(t, "08:00:00", row[5], "Hora de início deve estar no formato hh:mm:ss")
	assert.Equal(t, "09:30:00", row[6], "Hora final deve estar no formato hh:mm:ss")
	assert.Equal(t, "45", row[7], "QtePaxPagantes deve ser 45 (20+15+10)")
	assert.Equal(t, "3", row[8], "QteIdoso deve ser 3")
	assert.Equal(t, "5", row[9], "QtePL deve ser 5")
	assert.Equal(t, "01:30:00", row[15], "Tempo de viagem deve ser 01:30:00")
	
	// Verificar distância e velocidade (devem estar preenchidas)
	distanciaStr := row[14]
	assert.NotEmpty(t, distanciaStr, "Distância deve estar preenchida")
	velocidadeStr := row[16]
	assert.NotEmpty(t, velocidadeStr, "Velocidade média deve estar preenchida")
}

// TestProcessXML_SentidoAlternado testa alternância de sentido
func TestProcessXML_SentidoAlternado(t *testing.T) {
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
      <operacao codigoEmpresa="1" veiculo="1001" linha="1001" roletaInicial="2000" roletaFinal="3000" totalPassageiros="40" tarifaAtual="5.00" Receita="200.00" datainicio="2024-01-15 10:00:00" datafim="2024-01-15 11:15:00">
        <passageiros>
          <passageiro tipo="1" vlUnitario="5.00" qtd="15" qtdCreditos="0" idoso="0"/>
        </passageiros>
        <coletas recebido="200.00" girosPagantes="40" girosCartoes="25" engolidos="0"/>
      </operacao>
    </operacoes>
  </btc>
</btcs>`

	tmpFile, err := os.CreateTemp("", "test_*.xml")
	if err != nil {
		t.Fatalf("Erro ao criar arquivo temporário: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(xmlContent)
	if err != nil {
		t.Fatalf("Erro ao escrever no arquivo: %v", err)
	}
	tmpFile.Close()

	csvPath, err := ProcessXML(tmpFile.Name())
	if err != nil {
		t.Fatalf("Erro ao processar XML: %v", err)
	}
	defer os.Remove(csvPath)

	file, err := os.Open(csvPath)
	if err != nil {
		t.Fatalf("Erro ao abrir CSV: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';'

	// Pular header
	_, _ = reader.Read()

	// Primeira linha - deve ser GO-DF
	row1, err := reader.Read()
	if err != nil {
		t.Fatalf("Erro ao ler primeira linha: %v", err)
	}
	assert.Equal(t, "GO-DF", row1[3], "Primeira ocorrência deve ser GO-DF")

	// Segunda linha - deve ser DF-GO
	row2, err := reader.Read()
	if err != nil {
		t.Fatalf("Erro ao ler segunda linha: %v", err)
	}
	assert.Equal(t, "DF-GO", row2[3], "Segunda ocorrência deve ser DF-GO")
}

// TestProcessXML_Coordenadas testa preenchimento de coordenadas
func TestProcessXML_Coordenadas(t *testing.T) {
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

	tmpFile, err := os.CreateTemp("", "test_*.xml")
	if err != nil {
		t.Fatalf("Erro ao criar arquivo temporário: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(xmlContent)
	if err != nil {
		t.Fatalf("Erro ao escrever no arquivo: %v", err)
	}
	tmpFile.Close()

	csvPath, err := ProcessXML(tmpFile.Name())
	if err != nil {
		t.Fatalf("Erro ao processar XML: %v", err)
	}
	defer os.Remove(csvPath)

	file, err := os.Open(csvPath)
	if err != nil {
		t.Fatalf("Erro ao abrir CSV: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';'

	// Pular header
	_, _ = reader.Read()

	row, err := reader.Read()
	if err != nil {
		t.Fatalf("Erro ao ler linha: %v", err)
	}

	// Verificar coordenadas para sentido GO-DF (primeira ocorrência)
	// Linha 1001: Lat1=-15.43488062, Lng1=-47.6108282, Lat2=-15.7936645, Lng2=-47.8829638
	// Sentido GO-DF: abertura = Lat1/Lng1, fechamento = Lat2/Lng2
	assert.Equal(t, "-15.43488062", row[17], "Latitude de abertura deve ser Lat1")
	assert.Equal(t, "-47.6108282", row[18], "Longitude de abertura deve ser Lng1")
	assert.Equal(t, "-15.7936645", row[19], "Latitude de fechamento deve ser Lat2")
	assert.Equal(t, "-47.8829638", row[20], "Longitude de fechamento deve ser Lng2")
}

// TestProcessXML_PrefixoANTT testa formatação do prefixo ANTT
func TestProcessXML_PrefixoANTT(t *testing.T) {
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

	tmpFile, err := os.CreateTemp("", "test_*.xml")
	if err != nil {
		t.Fatalf("Erro ao criar arquivo temporário: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(xmlContent)
	if err != nil {
		t.Fatalf("Erro ao escrever no arquivo: %v", err)
	}
	tmpFile.Close()

	csvPath, err := ProcessXML(tmpFile.Name())
	if err != nil {
		t.Fatalf("Erro ao processar XML: %v", err)
	}
	defer os.Remove(csvPath)

	file, err := os.Open(csvPath)
	if err != nil {
		t.Fatalf("Erro ao abrir CSV: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';'

	// Pular header
	_, _ = reader.Read()

	row, err := reader.Read()
	if err != nil {
		t.Fatalf("Erro ao ler linha: %v", err)
	}

	// Linha 1001 tem CodANTT "12-0730-70", deve ser "12073070" no CSV
	assert.Equal(t, "12073070", row[1], "Prefixo ANTT deve estar sem traços")
}

// TestProcessXML_PlacaVeiculo testa busca de placa do veículo
func TestProcessXML_PlacaVeiculo(t *testing.T) {
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

	tmpFile, err := os.CreateTemp("", "test_*.xml")
	if err != nil {
		t.Fatalf("Erro ao criar arquivo temporário: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(xmlContent)
	if err != nil {
		t.Fatalf("Erro ao escrever no arquivo: %v", err)
	}
	tmpFile.Close()

	csvPath, err := ProcessXML(tmpFile.Name())
	if err != nil {
		t.Fatalf("Erro ao processar XML: %v", err)
	}
	defer os.Remove(csvPath)

	file, err := os.Open(csvPath)
	if err != nil {
		t.Fatalf("Erro ao abrir CSV: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';'

	// Pular header
	_, _ = reader.Read()

	row, err := reader.Read()
	if err != nil {
		t.Fatalf("Erro ao ler linha: %v", err)
	}

	// Veículo 1001 deve ter placa "JHX-0E23"
	assert.Equal(t, "JHX-0E23", row[21], "Placa do veículo deve estar correta")
}

// TestProcessXML_InvalidFile testa erro com arquivo inválido
func TestProcessXML_InvalidFile(t *testing.T) {
	_, err := ProcessXML("arquivo_inexistente.xml")
	assert.Error(t, err, "Deve retornar erro para arquivo inexistente")
}

// TestProcessXML_InvalidXML testa erro com XML inválido
func TestProcessXML_InvalidXML(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_*.xml")
	if err != nil {
		t.Fatalf("Erro ao criar arquivo temporário: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("XML inválido")
	if err != nil {
		t.Fatalf("Erro ao escrever no arquivo: %v", err)
	}
	tmpFile.Close()

	_, err = ProcessXML(tmpFile.Name())
	assert.Error(t, err, "Deve retornar erro para XML inválido")
}

// TestProcessXML_TimeLimitation testa limitação de tempo máximo
func TestProcessXML_TimeLimitation(t *testing.T) {
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

	// XML com tempo suspeito (8 horas - turno não invertido)
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<btcs versaoApp="1.0" dataGeracao="2024-01-15 10:00:00" DataIni="2024-01-15" DataFim="2024-01-15" CodFuncionario="123" NFuncionario="João Silva" CodEmpresa="1">
  <btc doc="123456" matdmtu="951716" data="2024-01-15" nome="João Silva" codigoTD="TD001">
    <operacoes>
      <operacao codigoEmpresa="1" veiculo="1001" linha="1001" roletaInicial="1000" roletaFinal="2000" totalPassageiros="50" tarifaAtual="5.00" Receita="250.00" datainicio="2024-01-15 08:00:00" datafim="2024-01-15 16:00:00">
        <passageiros>
          <passageiro tipo="1" vlUnitario="5.00" qtd="20" qtdCreditos="0" idoso="0"/>
        </passageiros>
        <coletas recebido="250.00" girosPagantes="45" girosCartoes="35" engolidos="0"/>
      </operacao>
    </operacoes>
  </btc>
</btcs>`

	tmpFile, err := os.CreateTemp("", "test_*.xml")
	if err != nil {
		t.Fatalf("Erro ao criar arquivo temporário: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(xmlContent)
	if err != nil {
		t.Fatalf("Erro ao escrever no arquivo: %v", err)
	}
	tmpFile.Close()

	csvPath, err := ProcessXML(tmpFile.Name())
	if err != nil {
		t.Fatalf("Erro ao processar XML: %v", err)
	}
	defer os.Remove(csvPath)

	file, err := os.Open(csvPath)
	if err != nil {
		t.Fatalf("Erro ao abrir CSV: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';'

	// Pular header
	_, _ = reader.Read()

	row, err := reader.Read()
	if err != nil {
		t.Fatalf("Erro ao ler linha: %v", err)
	}

	// Verificar que velocidade foi calculada (deve usar tempo limitado de 3h, não 8h)
	velocidadeStr := row[16]
	assert.NotEmpty(t, velocidadeStr, "Velocidade deve estar preenchida")
	velocidade, err := strconv.ParseFloat(velocidadeStr, 64)
	assert.NoError(t, err, "Velocidade deve ser um número válido")
	
	// Com tempo limitado para 3h, velocidade deve ser realista (> 20 km/h)
	// Se distância for ~40 km e tempo 3h, velocidade ≈ 13 km/h
	// Mas se distância for maior, velocidade será maior
	assert.Greater(t, velocidade, 10.0, "Velocidade deve ser realista mesmo com tempo limitado")
}

// TestProcessXML_ZeroTime testa caso de tempo zero
func TestProcessXML_ZeroTime(t *testing.T) {
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

	// XML com tempo zero (mesma data/hora início e fim)
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<btcs versaoApp="1.0" dataGeracao="2024-01-15 10:00:00" DataIni="2024-01-15" DataFim="2024-01-15" CodFuncionario="123" NFuncionario="João Silva" CodEmpresa="1">
  <btc doc="123456" matdmtu="951716" data="2024-01-15" nome="João Silva" codigoTD="TD001">
    <operacoes>
      <operacao codigoEmpresa="1" veiculo="1001" linha="1001" roletaInicial="1000" roletaFinal="2000" totalPassageiros="50" tarifaAtual="5.00" Receita="250.00" datainicio="2024-01-15 08:00:00" datafim="2024-01-15 08:00:00">
        <passageiros>
          <passageiro tipo="1" vlUnitario="5.00" qtd="20" qtdCreditos="0" idoso="0"/>
        </passageiros>
        <coletas recebido="250.00" girosPagantes="45" girosCartoes="35" engolidos="0"/>
      </operacao>
    </operacoes>
  </btc>
</btcs>`

	tmpFile, err := os.CreateTemp("", "test_*.xml")
	if err != nil {
		t.Fatalf("Erro ao criar arquivo temporário: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(xmlContent)
	if err != nil {
		t.Fatalf("Erro ao escrever no arquivo: %v", err)
	}
	tmpFile.Close()

	csvPath, err := ProcessXML(tmpFile.Name())
	if err != nil {
		t.Fatalf("Erro ao processar XML: %v", err)
	}
	defer os.Remove(csvPath)

	file, err := os.Open(csvPath)
	if err != nil {
		t.Fatalf("Erro ao abrir CSV: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';'

	// Pular header
	_, _ = reader.Read()

	row, err := reader.Read()
	if err != nil {
		t.Fatalf("Erro ao ler linha: %v", err)
	}

	// Com tempo zero, velocidade deve ser 0
	velocidadeStr := row[16]
	velocidade, err := strconv.ParseFloat(velocidadeStr, 64)
	if err == nil {
		assert.Equal(t, 0.0, velocidade, "Velocidade deve ser 0 quando tempo é zero")
	}
}
