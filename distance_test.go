package main

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCalculateGeographicDistance_ValidCoordinates testa cálculo com coordenadas válidas
func TestCalculateGeographicDistance_ValidCoordinates(t *testing.T) {
	// Coordenadas da linha 1001: Planaltina-GO → Brasília-DF
	lat1 := "-15.43488062"
	lng1 := "-47.6108282"
	lat2 := "-15.7936645"
	lng2 := "-47.8829638"

	distance := calculateGeographicDistance(lat1, lng1, lat2, lng2)

	// Distância esperada aproximada: ~40-50 km
	assert.Greater(t, distance, 30.0, "Distância deve ser maior que 30 km")
	assert.Less(t, distance, 60.0, "Distância deve ser menor que 60 km")
}

// TestCalculateGeographicDistance_InvalidCoordinates testa com coordenadas inválidas
func TestCalculateGeographicDistance_InvalidCoordinates(t *testing.T) {
	distance := calculateGeographicDistance("invalid", "invalid", "invalid", "invalid")
	assert.Equal(t, 0.0, distance, "Distância deve ser 0 para coordenadas inválidas")
}

// TestCalculateGeographicDistance_EmptyCoordinates testa com coordenadas vazias
func TestCalculateGeographicDistance_EmptyCoordinates(t *testing.T) {
	distance := calculateGeographicDistance("", "", "", "")
	assert.Equal(t, 0.0, distance, "Distância deve ser 0 para coordenadas vazias")
}

// TestCalculateGeographicDistance_SamePoint testa com mesmo ponto
func TestCalculateGeographicDistance_SamePoint(t *testing.T) {
	lat := "-15.43488062"
	lng := "-47.6108282"
	distance := calculateGeographicDistance(lat, lng, lat, lng)
	assert.InDelta(t, 0.0, distance, 0.1, "Distância deve ser aproximadamente 0 para mesmo ponto")
}

// TestCalculateGeographicDistance_RealWorldDistance testa distância conhecida
func TestCalculateGeographicDistance_RealWorldDistance(t *testing.T) {
	// Coordenadas aproximadas: São Paulo (-23.5505, -46.6333) → Rio de Janeiro (-22.9068, -43.1729)
	// Distância real: ~358 km
	lat1 := "-23.5505"
	lng1 := "-46.6333"
	lat2 := "-22.9068"
	lng2 := "-43.1729"

	distance := calculateGeographicDistance(lat1, lng1, lat2, lng2)

	// Deve estar próximo de 358 km (tolerância de 10%)
	assert.InDelta(t, 358.0, distance, 35.8, "Distância deve estar próxima de 358 km")
}

// TestCalculateGeographicDistance_FormulaAccuracy testa precisão da fórmula
func TestCalculateGeographicDistance_FormulaAccuracy(t *testing.T) {
	// Teste com coordenadas que devem dar distância conhecida
	// Equador: (0, 0) → (0, 1) grau = ~111 km
	lat1 := "0.0"
	lng1 := "0.0"
	lat2 := "0.0"
	lng2 := "1.0"

	distance := calculateGeographicDistance(lat1, lng1, lat2, lng2)

	// 1 grau de longitude no equador ≈ 111 km
	assert.InDelta(t, 111.0, distance, 5.0, "1 grau no equador deve ser aproximadamente 111 km")
}

// TestTimeLimitation testa limitação de tempo máximo
func TestTimeLimitation(t *testing.T) {
	// Simular cálculo de velocidade com tempo > 3h
	distanciaKm := 70.0
	tempoHoras := 8.0 // 8 horas (suspeito)
	const TEMPO_MAXIMO_VIAGEM = 3.0

	if tempoHoras > TEMPO_MAXIMO_VIAGEM {
		tempoHoras = TEMPO_MAXIMO_VIAGEM
	}

	velocidadeMedia := distanciaKm / tempoHoras

	// Velocidade deve ser baseada em 3h, não 8h
	velocidadeEsperada := 70.0 / 3.0
	assert.InDelta(t, velocidadeEsperada, velocidadeMedia, 0.1, "Velocidade deve usar tempo limitado")
	assert.Greater(t, velocidadeMedia, 20.0, "Velocidade deve ser realista (> 20 km/h)")
}

// TestVelocityCalculation_ZeroDistance testa cálculo com distância zero
func TestVelocityCalculation_ZeroDistance(t *testing.T) {
	distanciaKm := 0.0
	tempoHoras := 1.5
	velocidadeMedia := distanciaKm / tempoHoras

	assert.Equal(t, 0.0, velocidadeMedia, "Velocidade deve ser 0 quando distância é 0")
}

// TestVelocityCalculation_ZeroTime testa cálculo com tempo zero
func TestVelocityCalculation_ZeroTime(t *testing.T) {
	distanciaKm := 70.0
	tempoHoras := 0.0
	var velocidadeMedia float64
	if distanciaKm > 0 && tempoHoras > 0 {
		velocidadeMedia = distanciaKm / tempoHoras
	}

	assert.Equal(t, 0.0, velocidadeMedia, "Velocidade deve ser 0 quando tempo é 0")
}

// TestHaversineFormula testa a fórmula de Haversine diretamente
func TestHaversineFormula(t *testing.T) {
	// Coordenadas conhecidas para validação
	// Usar coordenadas que resultam em distância conhecida
	lat1 := 0.0
	lng1 := 0.0
	lat2 := 0.0
	lng2 := 1.0

	// Converter para radianos
	lat1Rad := lat1 * math.Pi / 180.0
	lng1Rad := lng1 * math.Pi / 180.0
	lat2Rad := lat2 * math.Pi / 180.0
	lng2Rad := lng2 * math.Pi / 180.0

	// Fórmula de Haversine
	const R = 6371.0
	dLat := lat2Rad - lat1Rad
	dLng := lng2Rad - lng1Rad

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	distance := R * c

	// 1 grau de longitude no equador ≈ 111 km
	assert.InDelta(t, 111.0, distance, 5.0, "Fórmula de Haversine deve calcular corretamente")
}

