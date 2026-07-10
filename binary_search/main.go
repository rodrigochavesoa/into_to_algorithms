/*

# Enunciado em Inglês (Canada)

Problem: Binary Search with Benchmark Tests

Create a Go package that implements the `binary search` algorithm to locate an element within a sorted list of integers.  

The program should:  
- Implement a correct iterative version of binary search with O(log n) time complexity and O(1) space complexity.  
- Ensure that computing the middle index does not cause integer overflow.  
- Include unit tests in main_test.go to verify the correctness of the algorithm.  
- Add benchmarks in main_test.go to measure performance using the Go testing framework (`go test -bench=. -benchmem`).  
- Make sure benchmarks use b.ResetTimer() for accurate timing and separate setup from measured code.  
- Allow the main.go program to include a short demonstration (non-benchmark) that runs binary search 
  and prints results, but all performance measurements must be done through benchmark tests.  


# 

Problema: Busca Binária com Testes de Benchmark

Crie um pacote em Go que implemente o algoritmo de busca binária para localizar um elemento dentro de 
uma lista ordenada de inteiros.  

O programa deve:  
- Implementar uma versão iterativa correta da busca binária com complexidade O(log n) em tempo e O(1) em espaço.  
- Garantir que o cálculo do índice do meio não cause overflow de inteiros.  
- Incluir testes unitários em main_test.go que validem a correção do algoritmo.  
- Adicionar benchmarks em main_test.go para medir o desempenho utilizando o framework de testes do Go 
  (`go test -bench=. -benchmem`).  
- Assegurar que os benchmarks utilizem b.ResetTimer() para medições precisas e que a preparação seja separada do 
  código medido.  
- Permitir que o programa main.go contenha uma breve demonstração (não-benchmark) que execute a busca binária e 
  imprima os resultados, mas todas as medições de desempenho devem ser feitas via testes de benchmark.  

*/

package main

import (
	"fmt"
)

// binarySearch performs a search for an item in a sorted slice.
// It returns the index if found, or -1 otherwise.
func binarySearch(list []int, item int) int {
	low := 0
	high := len(list) - 1

	for low <= high {
		mid := low + (high-low)/2  
		// Attention! Integer Overflow Prevention: Calculate mid without potential overflow 
		// "mid := (low + high) / 2". Using mid := low + (high-low)/2 avoids this issue.
		if list[mid] == item { 
			return mid
		} else if list[mid] < item {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}
	return -1
}

func main() {
	itens := []int{1, 2, 3, 4, 5, 8, 50, 100, 200, 300, 301, 302, 304, 305, 405, 505, 605, 705, 890, 1000}

	result := binarySearch(itens, 505)

	if result == -1 {
		fmt.Println("Number not found")
	} else {
		fmt.Println("Found Idex:", result)
	}
}
