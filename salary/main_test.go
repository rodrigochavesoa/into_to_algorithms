package main

import "testing"

// Mock to simulate the behavior of the calendar interface
type mockCalendar struct{}

func (m mockCalendar) DaysPerMonth() float64 { return 20 }
func (m mockCalendar) HoursPerDay() float64  { return 8 }

func TestHourSalary(t *testing.T) {
    // Scenario of test
    // Salary: 1600, Days: 20, Hours: 8
    // Expexted calculation: 1600 / 20 = 80 (per day) / 8 = 10 (per hour)
    
    mockRules := mockCalendar{}
    salary := 1600.0
    expected := 10.0
    
    result, err := salaryHour(salary, mockRules)
    if err != nil {
        t.Fatalf("salaryHour returned error: %v", err)
    }
    
    // Verify if the result is equal to the expected value
    if result != expected {
        t.Errorf("Esperado %.2f, mas obteve %.2f", expected, result)
    }
}