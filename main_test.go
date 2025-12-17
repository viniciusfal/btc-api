package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestFormatCPF testa a formatação de CPF (remover pontos e traços)
func TestFormatCPF(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "CPF com pontos e traços",
			input:    "377.209.881-91",
			expected: "37720988191",
		},
		{
			name:     "CPF sem formatação",
			input:    "37720988191",
			expected: "37720988191",
		},
		{
			name:     "CPF vazio",
			input:    "",
			expected: "",
		},
		{
			name:     "CPF apenas com pontos",
			input:    "377.209.881.91",
			expected: "37720988191",
		},
		{
			name:     "CPF apenas com traços",
			input:    "377-209-881-91",
			expected: "37720988191",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCPF(tt.input)
			assert.Equal(t, tt.expected, result, "CPF formatado incorretamente")
		})
	}
}

// formatCPF é uma função auxiliar para testar a formatação de CPF
func formatCPF(cpf string) string {
	if cpf == "" {
		return ""
	}
	result := cpf
	result = strings.ReplaceAll(result, ".", "")
	result = strings.ReplaceAll(result, "-", "")
	return result
}

// TestCalculateSentido testa o cálculo de sentido da viagem
func TestCalculateSentido(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		expected string
	}{
		{
			name:     "Primeira ocorrência (ímpar)",
			count:    1,
			expected: "GO-DF",
		},
		{
			name:     "Segunda ocorrência (par)",
			count:    2,
			expected: "DF-GO",
		},
		{
			name:     "Terceira ocorrência (ímpar)",
			count:    3,
			expected: "GO-DF",
		},
		{
			name:     "Quarta ocorrência (par)",
			count:    4,
			expected: "DF-GO",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentido := calculateSentido(tt.count)
			assert.Equal(t, tt.expected, sentido, "Sentido calculado incorretamente")
		})
	}
}

// calculateSentido é uma função auxiliar para testar o cálculo de sentido
func calculateSentido(count int) string {
	if count%2 == 0 {
		return "DF-GO"
	}
	return "GO-DF"
}

// TestFormatDate testa a formatação de data
func TestFormatDate(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "Data válida",
			input:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			expected: "15/01/2024",
		},
		{
			name:     "Data com mês e dia de um dígito",
			input:    time.Date(2024, 3, 5, 10, 30, 0, 0, time.UTC),
			expected: "05/03/2024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.Format("02/01/2006")
			assert.Equal(t, tt.expected, result, "Data formatada incorretamente")
		})
	}
}

// TestFormatTime testa a formatação de hora
func TestFormatTime(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "Hora válida",
			input:    time.Date(2024, 1, 15, 8, 30, 45, 0, time.UTC),
			expected: "08:30:45",
		},
		{
			name:     "Hora com minutos e segundos de um dígito",
			input:    time.Date(2024, 1, 15, 9, 5, 3, 0, time.UTC),
			expected: "09:05:03",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.Format("15:04:05")
			assert.Equal(t, tt.expected, result, "Hora formatada incorretamente")
		})
	}
}

// TestCalculateTempoViagem testa o cálculo de tempo de viagem
func TestCalculateTempoViagem(t *testing.T) {
	tests := []struct {
		name         string
		dataInicio   time.Time
		dataFim      time.Time
		expected     string
		expectedHrs  int
		expectedMin  int
		expectedSeg  int
	}{
		{
			name:        "Viagem de 1h30min",
			dataInicio:  time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC),
			dataFim:     time.Date(2024, 1, 15, 9, 30, 0, 0, time.UTC),
			expected:    "01:30:00",
			expectedHrs: 1,
			expectedMin: 30,
			expectedSeg: 0,
		},
		{
			name:        "Viagem de 45 minutos",
			dataInicio:  time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			dataFim:     time.Date(2024, 1, 15, 10, 45, 0, 0, time.UTC),
			expected:    "00:45:00",
			expectedHrs: 0,
			expectedMin: 45,
			expectedSeg: 0,
		},
		{
			name:        "Viagem de 2h15min30seg",
			dataInicio:  time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC),
			dataFim:     time.Date(2024, 1, 15, 10, 15, 30, 0, time.UTC),
			expected:    "02:15:30",
			expectedHrs: 2,
			expectedMin: 15,
			expectedSeg: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duracao := tt.dataFim.Sub(tt.dataInicio)
			horas := int(duracao.Hours())
			minutos := int(duracao.Minutes()) % 60
			segundos := int(duracao.Seconds()) % 60
			result := formatTempoViagem(horas, minutos, segundos)

			assert.Equal(t, tt.expected, result, "Tempo de viagem calculado incorretamente")
			assert.Equal(t, tt.expectedHrs, horas, "Horas calculadas incorretamente")
			assert.Equal(t, tt.expectedMin, minutos, "Minutos calculados incorretamente")
			assert.Equal(t, tt.expectedSeg, segundos, "Segundos calculados incorretamente")
		})
	}
}

// formatTempoViagem é uma função auxiliar para formatar tempo de viagem
func formatTempoViagem(horas, minutos, segundos int) string {
	return fmt.Sprintf("%02d:%02d:%02d", horas, minutos, segundos)
}

// TestFormatPrefixANTT testa a formatação do prefixo ANTT
func TestFormatPrefixANTT(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Prefixo com traços",
			input:    "12-0730-70",
			expected: "12073070",
		},
		{
			name:     "Prefixo sem traços",
			input:    "12073070",
			expected: "12073070",
		},
		{
			name:     "Prefixo vazio",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strings.ReplaceAll(tt.input, "-", "")
			assert.Equal(t, tt.expected, result, "Prefixo ANTT formatado incorretamente")
		})
	}
}

