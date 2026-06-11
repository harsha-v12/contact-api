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
	generateCSV("demo_set_1.csv", 1000000, 150)
	generateCSV("demo_set_2.csv", 2000000, 150)
	fmt.Println("Successfully generated demo_set_1.csv and demo_set_2.csv!")
}

func generateCSV(filename string, startPhone, count int) {
	file, _ := os.Create(filename)
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Write([]string{"First Name", "Last Name", "Email", "Mobile Number", "Gender", "Date of Birth", "City", "State", "Country", "Tags"})

	for i := 0; i < count; i++ {
		gender := "male"
		if rand.Intn(2) == 1 { gender = "female" }
		first := firstNamesM[rand.Intn(len(firstNamesM))]
		if gender == "female" { first = firstNamesF[rand.Intn(len(firstNamesF))] }
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
