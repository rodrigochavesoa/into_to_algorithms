/*

Problem: Average of Three Numbers

Create a program that takes three numeric values and calculates their arithmetic average.

The program should:
- Read three numbers provided by the user.
- Add the received values.
- Divide the sum by the number of values.
- Display the calculated average.

#

Problema: Cálculo da Média de Três Números

Crie um programa que receba três valores numéricos e calcule a média aritmética entre eles.

O programa deve:
- Ler três números fornecidos pelo usuário.
- Somar os valores recebidos.
- Dividir o resultado da soma pela quantidade de números.
- Exibir o valor da média calculada.

*/

package main

import "fmt"

func main() {
	valuesNum(45, 60, 70)
}

func valuesNum(a, b, c int) {
 sum := a + b + c
	avg := 	sum / 3
	fmt.Printf("A soma dos três números é: %d\n", sum)
	fmt.Printf("A média calculada é: %d\n", avg)
}
