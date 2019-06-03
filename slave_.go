package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
)

func main() {

	arr := os.Args[1:]
	ip := ""
	port := ""
	nickname := ""
	var directory = "./data/"

	if len(arr) == 0 {
		ip = ""
		port = "3000"
		nickname = "Slave"

	} else if len(arr) == 1 {
		ip = ""
		port = "3000"
		nickname = arr[0]
	} else if len(arr) == 2 {
		ip = ""
		port = arr[1]
		nickname = arr[0]
	} else if len(arr) == 3 {
		ip = arr[2]
		port = arr[1]
		nickname = arr[0]
	}

	fmt.Println("Connections : ", ip+":"+port)
	conn, err := net.Dial("tcp", ip+":"+port)
	if err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}

	log.Printf("Connection from %v closed.\n", conn.RemoteAddr())
	io.WriteString(conn, fmt.Sprintf("Slave"+nickname+"\n"))

	filename := ""

	files, err := ioutil.ReadDir(directory)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		fmt.Println(file.Name())
		filename = filename + string(file.Name()) + " "
	}

	io.WriteString(conn, fmt.Sprintf("%v\n", (filename)))

	for { 

		//send heartbeat
		conn.Write([]byte("1\n"))

		output := make([]byte, 500) // using small tmo buffer for demonstrating
		//recieve data
		conn.Read(output)
		output = bytes.Trim(output, "\x00") // Trimming null values

		
		//split data
		output_list := strings.Fields(string(output))

		// search this file
		filename_to_be_searched := output_list[0]

		//search these passwords
		passwords := output_list[1:]


		// boolen array for the response of passwords' search
		passwordFound := make([]string, len(passwords))


		log.Println("File Directory : ", directory+string(filename_to_be_searched))


		//open file to be searched
		f, _ := os.Open(directory + string(filename_to_be_searched))
		defer f.Close()

		// ----------------------SEARCHING---------------------------------------------------
		passwordinUSE := false
		// Splits on newlines by default.
		scanner := bufio.NewScanner(f)
		for i, _ := range passwords {
			for scanner.Scan() {
				if scanner.Text() == string(passwords[i]) { // Complete match
					
					fmt.Println(fmt.Sprintf("Password In Use!!!! [ %v, %v ]", scanner.Text(), string(passwords[i])))
					passwordinUSE = true
					
				}
			}
			
			if passwordinUSE == false {
				passwordFound[i] = "0"
			} else {
				passwordFound[i] = "1"
			}
		}

		
		outputResult := ""
		for _, result := range passwordFound {
			outputResult = outputResult + string(result) + " "
		}

		// responding with search result to server
		io.WriteString(conn, fmt.Sprintf("%v\n", (outputResult)))

	}

	// if err != nil {
	// 	retcode = 2
	// 	return
	// }
	// panic(Exit{1})

}
