/*

Problem: Hourly Salary Calculation with External Configuration

Create a Go program that calculates an employee’s hourly salary based on their monthly salary and calendar 
rules provided in a JSON configuration file.  

The program should:  
- Define an abstraction that represents calendar rules (days per month and hours per day).  
- Implement this abstraction so that values are dynamically loaded from a JSON file.  
- Use these rules to compute the hourly salary, ensuring that values are valid.  
- Read the monthly salary entered by the user and display the calculation result.  
- Also show the total working hours per month based on the provided rules.  
- Include unit tests that validate the hourly salary calculation using simulated values.  

#

Problema: Cálculo de Salário por Hora com Configuração Externa

Crie um programa em Go que seja capaz de calcular o salário por hora de um funcionário a partir do salário mensal e de 
regras de calendário fornecidas em um arquivo de configuração JSON.  

O programa deve:  
- Definir uma abstração que represente regras de calendário (dias por mês e horas por dia).  
- Implementar essa abstração de forma que os valores sejam carregados dinamicamente de um arquivo JSON.  
- Utilizar essas regras para calcular o valor do salário por hora, garantindo que os valores sejam válidos.  
- Ler o salário mensal informado pelo usuário e exibir o resultado do cálculo.  
- Mostrar também o total de horas trabalhadas por mês com base nas regras fornecidas.  
- Incluir testes unitários que validem o cálculo do salário por hora utilizando valores simulados.  

*/

package main // package main is the entry point of the program

import (

    "encoding/json" // the library json is used to read the configuration file

    "fmt"           // the library fmt is used to print in the console

    "log"           // the library log is used to log errors

    "os"            // the library os is used to read the configuration file

)

// contract
type Calendar interface { // interface is a contract that defines the methods that a type must implement

    DaysPerMonth() float64 // method that returns the number of days per month

    HoursPerDay() float64  // method that returns the number of hours per day

}

// dynamic implementation
type ConfigRules struct { // ConfigRules is a struct that implements the Calendar interface

    Days  float64 `json:"days_per_month"` // json tag is used to map the JSON field to the struct field

    Hours float64 `json:"hours_per_day"`  // json tag is used to map the JSON field to the struct field

}

func (c ConfigRules) DaysPerMonth() float64 { return c.Days }  // method that returns the number of days per month

func (c ConfigRules) HoursPerDay() float64  { return c.Hours } // method that returns the number of hours per day

// genericg function
func salaryHour(salarym float64, rules Calendar) (float64, error){ // salaryHour is a function that calculates the salary per hour based on the salary per month and the rules defined in the Calendar interface

    if rules.DaysPerMonth() <= 0 || rules.HoursPerDay() <= 0 {
        return 0, fmt.Errorf("Configuration of calendar is invalid: values must be greater than zero.")
    }

    dayValue := salarym / rules.DaysPerMonth() // method that returns the number of days per month 
    return dayValue / rules.HoursPerDay(), nil // method that returns the number of hours per day

}

func main() { // main is the entry point of the program

    // reading company configuration
    file, err := os.Open("configs/config_salary.json") // open the configuration file

    if err != nil {                             // check if there is an error opening the configuration file

        log.Fatalf("Error opening configuration: %v", err) // log the error and exit the program

    }

    defer file.Close() // close the configuration file when the function returns

    var cfg ConfigRules // variable that will hold the configuration read from the JSON file

    if err := json.NewDecoder(file).Decode(&cfg); err != nil { // decode the JSON file and store the result in the cfg variable

        log.Fatalf("Error parsing JSON: %v", err) // log the error and exit the program

    }

    var salarym float64                             // variable that will hold the salary per month

    fmt.Println("What is your Salary per month?")   // ask the user for the salary per month

    if _, err := fmt.Scanln(&salarym); err != nil { // check if there is an error reading the salary per month

        log.Fatalf("Error reading salary %v", err) // log the error and exit the program

    }

    result, err := salaryHour(salarym, cfg)
    if err != nil {
        log.Fatalf("Error calculating salaray per hour: %v", err) // log the error and exit the program
    }                  

    fmt.Printf("Your Salary Per Hour is : R$ %.2f\n", result) // print the salary per hour

    totalHours := cfg.DaysPerMonth() * cfg.HoursPerDay()                // calculate the total hours per month based on the rules defined in the Calendar interface

    fmt.Printf("Hour Job Per Month is : %.0f hour/month\n", totalHours) // print the total hours per month

} 

