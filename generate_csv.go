package main

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

var firstNamesM = []string{"Aarav", "Vihaan", "Vivaan", "Advik"}
var firstNamesF = []string{"Saanvi", "Aanya", "Aadhya", "Aaradhya"}
var lastNames = []string{"Sharma", "Verma", "Gupta", "Malhotra"}
var cities = [][2]string{{"Mumbai", "Maharashtra"}, {"Delhi", "Delhi"}}
var tagsList = []string{"VIP", "Customer", "Lead"}

func main() {
	rand.Seed(time.Now().UnixNano())
	generateCSV("demo_set_100.csv", 6000000, 100)
	generateCSV("demo_set_1k.csv", 5000000, 1000)
	generateCSV("demo_set_50k.csv", 3000000, 50000)
	generateCSV("demo_set_100k.csv", 4000000, 100000)
	generateMixedCSV("demo_set_mixed.csv", 490, 10)
	fmt.Println("Successfully generated all demo CSV sets, including demo_set_100.csv!")
}

func generateCSV(filename string, startPhone, count int) {
	file, _ := os.Create(filename)
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Write([]string{"First Name", "Last Name", "Email", "Mobile Number", "Gender", "Date of Birth", "City", "State", "Country", "Tags"})

	for i := 0; i < count; i++ {
		gender := "male"
		if rand.Intn(2) == 1 {
			gender = "female"
		}
		first := firstNamesM[rand.Intn(len(firstNamesM))]
		if gender == "female" {
			first = firstNamesF[rand.Intn(len(firstNamesF))]
		}
		last := lastNames[rand.Intn(len(lastNames))]

		email := fmt.Sprintf("%s.%s.%d@demo.com", strings.ToLower(first), strings.ToLower(last), rand.Intn(9000)+1000)

		// FIXED: Removed +91 as requested, generating exactly 10 digits
		phone := fmt.Sprintf("987%07d", startPhone+i)

		dob := "1990-01-01"
		loc := cities[rand.Intn(len(cities))]
		tags := "VIP"
		
		writer.Write([]string{first, last, email, phone, gender, dob, loc[0], loc[1], "India", tags})
	}
}

func generateMixedCSV(filename string, validCount, invalidCount int) {
	file, _ := os.Create(filename)
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Write([]string{"First Name", "Last Name", "Email", "Mobile Number", "Gender", "Date of Birth", "City", "State", "Country", "Tags"})

	startPhone := 8000000

	// 1. Generate perfectly valid rows
	for i := 0; i < validCount; i++ {
		first := "ValidUser"
		last := fmt.Sprintf("Last%d", i)
		email := fmt.Sprintf("valid.%d@demo.com", i)
		phone := fmt.Sprintf("999%07d", startPhone+i)
		writer.Write([]string{first, last, email, phone, "male", "1990-01-01", "Mumbai", "Maharashtra", "India", "VIP"})
	}

	// 2. Generate exactly 10 invalid rows
	for i := 0; i < invalidCount; i++ {
		first := "InvalidUser"
		last := "Bad"
		email := fmt.Sprintf("bademail%d", i) // Missing @
		phone := "123" // Too short

		if i < 3 {
			// Invalid Email
			writer.Write([]string{first, last, email, "9870000001", "male", "1990-01-01", "Mumbai", "MH", "India", ""})
		} else if i < 6 {
			// Invalid Phone
			writer.Write([]string{first, last, "good@demo.com", phone, "male", "1990-01-01", "Mumbai", "MH", "India", ""})
		} else {
			// Missing First Name
			writer.Write([]string{"", last, "good2@demo.com", "9870000002", "male", "1990-01-01", "Mumbai", "MH", "India", ""})
		}
	}
}
