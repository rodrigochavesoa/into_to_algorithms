/*

The problem: average pace

The amateur runner completed the training and wants to know their pace, which is the time it takes to cover one kilometre.
The application must receive the total distance covered by the athlete and the total time taken to complete the training. 
Based on this, the system should provide the runner’s average pace.

O problema: ritmo médio

O corredor amador completou o treino e deseja saber seu ritmo, que é o tempo que ele leva para percorrer um quilômetro.
O aplicativo deve receber a distância total percorrida pelo atleta e o tempo total que ele levou para completar o treino. 
Com base nisso, o sistema precisa informar o ritmo médio do corredor.

*/
package main

import (
    "fmt"
    "regexp"
	"strconv"
	"strings"
	"math"
)

func main() {

	fmt.Println("---Início do Programa ---")
	tTotal := solicitarTempo()
	dTotal := solicitarDistancia()

	ritmoMedio := calcularRitmoMedio(tTotal, dTotal)

	fmt.Println("\n--- RESUMO DOS DADOS VALIDADOS ---")
	fmt.Printf("Tempo Total:     %s\n", tTotal)
	fmt.Printf("Distância Total: %.2f km\n", dTotal)
	fmt.Printf("Ritmo Médio:     %s/km\n", ritmoMedio)

}

func calcularRitmoMedio(tempo string, distancia float64) string {
	
	partes := strings.Split(tempo, ":")
	h, _ := strconv.Atoi(partes[0])
	m, _ := strconv.Atoi(partes[1])
	s, _ := strconv.Atoi(partes[2])

	segundosTotais := (h * 3600) + (m * 60) + s

	segundosPorKm := float64(segundosTotais) / distancia

	minutosPace := int(math.Floor(segundosPorKm / 60))
	segundosPace := int(math.Round(math.Mod(segundosPorKm, 60)))

	if segundosPace == 60 {
		minutosPace++
		segundosPace = 0
	}

	
	return fmt.Sprintf("%02d:%02d", minutosPace, segundosPace)
}

func solicitarDistancia() float64 {
	var d float64 

	for {
		fmt.Print("Informe a distância total (em km): ")
		_, err := fmt.Scanln(&d)
		
		if err == nil && d > 0 {
			return d 
		}
	
		var limpa string
		fmt.Scanln(&limpa)
		
		fmt.Println("Erro: Distância inválida! Deve ser um número maior que zero (use ponto para decimais, ex: 10.5).")
	}
}

func solicitarTempo() string {
    var t string

    for {
        fmt.Print("Informe o tempo total (HH:MM:SS): ")
        fmt.Scanln(&t)

        if validarFormatoTempo(t) && validarValoresTempo(t) {
            return t
        }
        fmt.Println("Erro: Tempo Inválido. Certifique-se do formato HH:MM:SS e se min/seg estão entre 00 e 59.")
    }
}

var timeRE = regexp.MustCompile("^[0-9]{2}:[0-9]{2}:[0-9]{2}$")

func validarFormatoTempo(tempo string) bool {
    return timeRE.MatchString(tempo)
}

func validarValoresTempo(tempo string) bool {

	partes := strings.Split(tempo, ":")

	h, _ := strconv.Atoi(partes[0])  
	m, _ := strconv.Atoi(partes[1])
	s, _ := strconv.Atoi(partes[2])

	if h < 0 || m < 0 || m > 59 || s < 0 || s > 59 {
		return false
}

return true

}
