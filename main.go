package main

import (
	"fmt"
	"time"
)

//dd:log
func calculateSum(a, b int) int {
	time.Sleep(50 * time.Millisecond)
	return a + b
}

//dd:log
func processData(name string, items []string) error {
	time.Sleep(100 * time.Millisecond)
	if len(items) == 0 {
		return fmt.Errorf("no items to process for %s", name)
	}
	fmt.Printf("Processed %d items for %s\n", len(items), name)
	return nil
}

//dd:log
func simpleOperation() {
	time.Sleep(25 * time.Millisecond) // Simulate work
	fmt.Println("Simple operation completed")
}

func main() {
	fmt.Println("Hello GopherCon UK '25")

	result := calculateSum(10, 20)
	fmt.Printf("Sum: %d\n", result)

	items := []string{"item1", "item2", "item3"}
	if err := processData("demo", items); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	simpleOperation()
}
