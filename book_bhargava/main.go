/*

Forneça o tempo de execução para cada um dos casos em termos da notação Big O.

1.3 Você tem um nome e deseja encontrar o número de telefone para esse nome em uma agenda telefônica

*/

package main

import "fmt"

type Contato struct {
	Nome string
	Telefone string
}

func buscarTelefone(agenda []Contato, nomeBuscado string) Contato {
	baixo := 0 
	alto := len(agenda) - 1

	for baixo <= alto {
		meio := baixo + (alto-baixo)/2
		if agenda[meio].Nome == nomeBuscado {
			//return fmt.Sprintf("Nome: %s | Telefone: %s", agenda[meio].Nome, agenda[meio].Telefone)
			return agenda[meio]
		} else if agenda[meio].Nome < nomeBuscado {
			baixo = meio + 1
		} else {
			alto = meio - 1
		}
	}
	return Contato {}

}

func main() {

	agenda := []Contato{
	{Nome: "Ana", Telefone: "1111-1111"},
	{Nome: "Carlos", Telefone: "2222-2222"},
	{Nome: "Daniel", Telefone: "3333-3333"},
	{Nome: "Helena", Telefone: "4444-4444"},
	{Nome: "Rodrigo", Telefone: "75997066216"},
}

result :=  buscarTelefone(agenda, "Rodrigo") 

if result.Nome == "" {
fmt.Println("Resultado da pesquisa: Nome não encontrado")
} else {
fmt.Printf("Resultado da pesquisa - Nome: %s, Telefone: %s\n", result.Nome, result.Telefone)
}

}