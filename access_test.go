package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAccess_ReturnsMap testa se Access() retorna um mapa
func TestAccess_ReturnsMap(t *testing.T) {
	lines := Access()
	assert.NotNil(t, lines, "Access() não deve retornar nil")
	assert.Greater(t, len(lines), 0, "Access() deve retornar pelo menos uma linha")
}

// TestAccess_SpecificLines testa linhas específicas
func TestAccess_SpecificLines(t *testing.T) {
	lines := Access()

	// Testar linha 1001
	line1001, exists := lines["1001"]
	assert.True(t, exists, "Linha 1001 deve existir")
	assert.Equal(t, "1001", line1001.Cod, "Código da linha 1001 deve estar correto")
	assert.Equal(t, "12-0730-70", line1001.CodANTT, "CodANTT da linha 1001 deve estar correto")
	assert.NotEmpty(t, line1001.Lat1, "Lat1 da linha 1001 deve estar preenchido")
	assert.NotEmpty(t, line1001.Lng1, "Lng1 da linha 1001 deve estar preenchido")
	assert.NotEmpty(t, line1001.Lat2, "Lat2 da linha 1001 deve estar preenchido")
	assert.NotEmpty(t, line1001.Lng2, "Lng2 da linha 1001 deve estar preenchido")

	// Testar linha 9901
	line9901, exists := lines["9901"]
	assert.True(t, exists, "Linha 9901 deve existir")
	assert.Equal(t, "9901", line9901.Cod, "Código da linha 9901 deve estar correto")
	assert.Equal(t, "12-0338-70", line9901.CodANTT, "CodANTT da linha 9901 deve estar correto")

	// Testar linha 1057
	line1057, exists := lines["1057"]
	assert.True(t, exists, "Linha 1057 deve existir")
	assert.Equal(t, "1057", line1057.Cod, "Código da linha 1057 deve estar correto")
}

// TestAccess_AllFields testa se todos os campos estão preenchidos
func TestAccess_AllFields(t *testing.T) {
	lines := Access()

	for cod, line := range lines {
		assert.NotEmpty(t, line.Cod, "Código não deve estar vazio para linha %s", cod)
		assert.NotEmpty(t, line.Local1, "Local1 não deve estar vazio para linha %s", cod)
		assert.NotEmpty(t, line.Local2, "Local2 não deve estar vazio para linha %s", cod)
		assert.NotEmpty(t, line.Linha, "Linha não deve estar vazio para linha %s", cod)
		assert.NotEmpty(t, line.CodANTT, "CodANTT não deve estar vazio para linha %s", cod)
		// Km pode estar vazio, então não testamos
		// Coordenadas podem estar vazias para algumas linhas
	}
}

// TestPlacaV_ReturnsMap testa se PlacaV() retorna um mapa
func TestPlacaV_ReturnsMap(t *testing.T) {
	placas := PlacaV()
	assert.NotNil(t, placas, "PlacaV() não deve retornar nil")
	assert.Greater(t, len(placas), 0, "PlacaV() deve retornar pelo menos uma placa")
}

// TestPlacaV_SpecificPlacas testa placas específicas
func TestPlacaV_SpecificPlacas(t *testing.T) {
	placas := PlacaV()

	// Testar veículo 1001
	car1001, exists := placas["1001"]
	assert.True(t, exists, "Veículo 1001 deve existir")
	assert.Equal(t, "JHX-0E23", car1001.Placa, "Placa do veículo 1001 deve estar correta")

	// Testar veículo 1002
	car1002, exists := placas["1002"]
	assert.True(t, exists, "Veículo 1002 deve existir")
	assert.Equal(t, "JHX-4G03", car1002.Placa, "Placa do veículo 1002 deve estar correta")

	// Testar veículo 1054
	car1054, exists := placas["1054"]
	assert.True(t, exists, "Veículo 1054 deve existir")
	assert.Equal(t, "JHJ-7372", car1054.Placa, "Placa do veículo 1054 deve estar correta")
}

// TestPlacaV_AllFields testa se todos os campos estão preenchidos
func TestPlacaV_AllFields(t *testing.T) {
	placas := PlacaV()

	for cod, car := range placas {
		assert.NotEmpty(t, car.Placa, "Placa não deve estar vazia para veículo %s", cod)
		// Verificar formato básico da placa (deve ter pelo menos 7 caracteres)
		assert.GreaterOrEqual(t, len(car.Placa), 7, "Placa deve ter pelo menos 7 caracteres para veículo %s", cod)
	}
}

// TestAccess_Coordenadas testa coordenadas de linhas específicas
func TestAccess_Coordenadas(t *testing.T) {
	lines := Access()

	// Linha 1001 deve ter coordenadas preenchidas
	line1001, exists := lines["1001"]
	assert.True(t, exists, "Linha 1001 deve existir")
	assert.Equal(t, "-15.43488062", line1001.Lat1, "Lat1 da linha 1001 deve estar correto")
	assert.Equal(t, "-47.6108282", line1001.Lng1, "Lng1 da linha 1001 deve estar correto")
	assert.Equal(t, "-15.7936645", line1001.Lat2, "Lat2 da linha 1001 deve estar correto")
	assert.Equal(t, "-47.8829638", line1001.Lng2, "Lng2 da linha 1001 deve estar correto")
}
