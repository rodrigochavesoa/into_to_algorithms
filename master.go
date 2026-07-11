package inttoalgorithms


import "fmt"

func naturalNumbers(n int) (sum int) {
	for i := 1; i <= n; i++ {
		sum += i
	}
	return
}

func main() {
	var n int
	fmt.Println("Digite n:")
	fmt.Scanln(&n)
	fmt.Printf("A soma é: %v\n", naturalNumbers(n))
}