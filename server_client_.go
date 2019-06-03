package main // templates

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

//Client represents a client
// password: password to be searched
// nickname: identifier
// passwordStatus: channel to track the status of a password i.e if it is in any files that are searched or not
// current: files currently being searched by slaves
// visited: files already searched by slaves

type Client struct {
	password       string
	nickname       string
	passwordStatus chan string
	current        []string
	visited        []string
}

//Slave represents a slave

// conn: connection
// nickname: identifier
// ch_cllients: channel with all the current clients
// files: files this slave has
// file_to_process: file assigned by server to be searched
// total_current_files: all files accross all slaves

type Slave struct {
	conn                net.Conn
	stopping            chan string
	nickname            string
	ch_cllients         chan []Client
	files               []string
	file_to_process     string
	total_current_files chan []string
}

// unique function takes an array of strings and returns the unique strings with their respective count
func unique(arr []string) ([]string, map[string]int) {
	var unique []string
	m := map[string]int{}

	for _, v := range arr {
		if m[v] == 0 {
			m[v] = 1
			unique = append(unique, v)
		} else {
			m[v] += 1
		}
	}

	return unique, m

}

// currentfiles function takes a list of slaves and returns filesnames accross all the slaves
func currentfiles(slaveList []Slave) []string {
	c_files := make([]string, 1)

	for _, slave := range slaveList {
		c_files = append(c_files, slave.files...)
	}
	return c_files //[]string {"somebigfile_0.txt","somebigfile_1.txt","somebigfile_3.txt"}
}

//ReadLinesInto is a method on Slave type
//it keeps waiting for user to input a line, ch chan is the msgchannel
//it formats and writes the message to the channel
// func (c Slave) ReadLinesInto(ch chan<- string) {
// 	bufc := bufio.NewReader(c.conn)
// 	for {
// 		line, err := bufc.ReadString('\n')
// 		if err != nil {
// 			break
// 		}
// 		ch <- fmt.Sprintf("%s: %s", c.nickname, line)
// 	}
// }

//WriteLinesFrom is a method
//each slave routine is writing to channel
// func (c Slave) WriteLinesFrom(ch <-chan string) {
// 	for msg := range ch {
// 		_, err := io.WriteString(c.conn, msg)
// 		if err != nil {
// 			return
// 		}
// 	}
// }

// stringInSlice function checks if a string is present in a slice or not
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// channel to list conversion for clients
func retrieveClientList(clientChan chan Client) []Client {
	clientList := make([]Client, 1)
	for client := range clientChan {
		clientList = append(clientList, client)
	}
	return clientList
}

// channel to list conversion for slaves
func retrieveSlaveList(slaveChan <-chan Slave) []Slave {
	slaveList := make([]Slave, 1)
	for slave := range slaveChan {
		slaveList = append(slaveList, slave)
	}

	return slaveList
}

// sortslaves function takes slave's files and sort themin accending order according to the count provided by map
func sortslaves(files []string, map_count map[string]int) []string {
	for i, _ := range files {
		i = i
		for j := 0; j < len(files)-1; j++ {

			if map_count[files[j]] > map_count[files[j+1]] {
				files[j], files[j+1] = files[j+1], files[j]

			}
		}
	}
	return files
}

// removes client from client list
func delete_Client(c []Client, r_client Client) []Client {

	for i, cl := range c {
		if cl.nickname == r_client.nickname && i < len(c)-1 {
			copy(c[i:], c[i+1:])
			break
		}
	}

	c = c[:len(c)-1]
	return c // Truncate slice.

}


// removes slave from slave list
func delete_Slave(c []Slave, r_client Slave) []Slave {

	for i, cl := range c {
		if cl.nickname == r_client.nickname {
			copy(c[i:], c[i+1:])
			break
		}
	}

	c = c[:len(c)-1]
	return c // Truncate slice.

}

// method responsible to handle a slave
func (s Slave) scheduling(rmchan chan Client, s_rmchan chan Slave) {
	defer func() {
		// log.Printf("Connection from %v closed.\n", s.nickname)
		s_rmchan <- s
	}()

	c_files := <-s.total_current_files
	localClientList := make([]Client, 1)
	
	for { 

		select {
		// if number of clients are updated
		case list := <-s.ch_cllients:
			localClientList = list
		default:
		}

		clientList := localClientList
		
		// if a client is connected
		if len(clientList) > 0 {
			
			select {

			// if number of slaves are updated and thus total current files to be searched changes
			case updatedfiles := <-s.total_current_files:
				
				c_files = updatedfiles
				
			default:
			}
			

			// return count of files to find rareness
			_, map_count := unique(c_files)


			passwords_to_be_searched := make([]string, 1)

			// all the passwords to be searched
			for _, client := range clientList {
				passwords_to_be_searched = append(passwords_to_be_searched, client.password)
			}

			clientbool := false
			filemin := ""

			// first come first serve clients
			for _, client := range clientList {

				// sort slave files according to their rareness sfs = slave_files_sorted
				sfs := sortslaves(s.files, map_count)
				
				// seacrh if the chosen rarest file is already searhed or not
				for _, f := range sfs {
					if !stringInSlice(f, client.current) && !stringInSlice(f, client.visited) {
						filemin = f
						clientbool = true
						break
					}
				}
				if clientbool == true {
					break
				}
			}

			if filemin != "" {

				
				// heartbeat
				termination := make([]byte, 50)
				_, err := s.conn.Read(termination)
				// fmt.Println("TERMINTATION ??? >", err, "<")
				termination = bytes.Trim(termination, "\x00") // Trimming null values

				if err != nil {
					break
				}

				//setting chosen file to be processed
				s.file_to_process = filemin

				//creating data to be send to slave including filename choosen and passwords to be searched
				input := s.file_to_process + " "
				for _, p := range passwords_to_be_searched {

					input = input + p + " "
				}

				// sending data
				s.conn.Write([]byte(input))

				// upon processing updating current list of clients
				for i, _ := range clientList {
					clientList[i].current = append(clientList[i].current, s.file_to_process)
				}

				// recieveing confirmation from slave
				isfound := make([]byte, 50)
				s.conn.Read(isfound)

				isfound = bytes.Trim(isfound, "\x00") // Trimming null values
				isfound_list := strings.Fields(string(isfound))

				// fmt.Println("\nDid you find it? Report your Status ", s.nickname, isfound_list)

				// found_or_not := false

				for i, _ := range isfound_list {
					if isfound_list[i] == "1" {
						fmt.Println("\tAffirmative, I have found it for client ", clientList[i].nickname, " password ", clientList[i].password, " is in use. Stopping Search. ")
						rmchan <- clientList[i]
						clientList[i].passwordStatus <- "IN USE\n"
						clientList = <-s.ch_cllients
						

					} else {
						// fmt.Println("\tNegative for", clientList[i].nickname)
						// fmt.Println("Heading to the next location.\n")

					}
				}

				// upon confirmation updating visiting list of clients
				for i, _ := range clientList {
					clientList[i].visited = append(clientList[i].visited, s.file_to_process)
				}
			}
		}

		// updating local client list
		localClientList = clientList
		

	}

}


// function that creates a go routine for slaves
// add new clients and slaves
// delete clients and slaves
func AssigningFileFromChunk(slaveChan <-chan Slave, clientChan chan Client, visited []string) {

	slaveList := make([]Slave, 1)
	slaveList[0] = <-slaveChan

	clientList := make([]Client, 1)
	clientList[0] = <-clientChan
	

	// client remove channel
	rmchan := make(chan Client)
	// slave remove channel
	s_rmchan := make(chan Slave)

	c_files := currentfiles(slaveList)

	for i, _ := range slaveList {
		slaveList[i].total_current_files <- c_files
		slaveList[i].ch_cllients <- clientList
		// new go routine to handle slave, one for each
		go slaveList[i].scheduling(rmchan, s_rmchan)
	}

	for {

		select {
		case s := <-slaveChan:
			// fmt.Println(" ----------------- New Slave has Entered --------------------\n")
			s.ch_cllients <- clientList

			//update slavelist
			slaveList = append(slaveList, s)

			//update current files
			c_files = currentfiles(slaveList)

			for i, _ := range slaveList {
				slaveList[i].total_current_files <- c_files
			}

			// new go routine to handle new slave
			go slaveList[len(slaveList)-1].scheduling(rmchan, s_rmchan)

		case c := <-clientChan:
			// fmt.Println(" ----------------- New Client has Entered --------------------\n")

			//update client list
			clientList = append(clientList, c)
			
			//update client info to each slave
			for i, _ := range slaveList {
				slaveList[i].ch_cllients <- clientList

			}

		case cc := <-rmchan:
			// fmt.Println(" ----------------- Client needs to be Removed --------------------\n")
			clientList = delete_Client(clientList, cc)
			for i, _ := range slaveList {
				slaveList[i].ch_cllients <- clientList
			}

		case s := <-s_rmchan:
			// fmt.Println(" ----------------- Slave needs to be Removed --------------------\n")
			slaveList = delete_Slave(slaveList, s)
			c_files := currentfiles(slaveList)
			for i, _ := range slaveList {
				slaveList[i].total_current_files <- c_files
			}
		default:
			// fmt.Println(" ----------------- No Updated Status --------------------\n")

			uniq, _ := unique(c_files)
			for _, client := range clientList {
				presence := 0
				for _, file_searched := range client.visited {
					if stringInSlice(file_searched, uniq) == true {
						presence++
					}
				}

				// if all files seached for client
				if presence == len(uniq) {
					fmt.Println("Password not found for Client", client.nickname)
					// fmt.Println(" ----------------- Client needs to be Removed --------------------\n")
					// fmt.Println("CLIENT LIST::: ", client, client.passwordStatus)
					client.passwordStatus <- "NOT IN USE\n"
					// fmt.Println("CLIENT LIST::: ")
					// rmchan <-client
					clientList = delete_Client(clientList, client)

					for i, _ := range slaveList {
						slaveList[i].ch_cllients <- clientList
					}
				}
			}

		}
	}

}


func promptNick(c net.Conn, bufc *bufio.Reader) string {
	nick, _, _ := bufc.ReadLine()
	fmt.Println("\nNickname given: ", string(nick))
	return string(nick)
}

//Page represents a Page
type Page struct {
	Title string
	Body  []byte
}

func (pageToSave Page) savePage() error {
	fileName := pageToSave.Title + ".txt"
	err := ioutil.WriteFile(fileName, pageToSave.Body, 0700)
	return err
}

func loadPage(pageToLoad string) (*Page, error) {
	fileName := pageToLoad + ".txt"
	fileBody, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	return &Page{Title: pageToLoad, Body: fileBody}, err
}


func defaultHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, This is defult page%s!", r.URL.Path[1:])
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	pageTitle := r.URL.Path[len("/view/"):]
	loadedPage, err := loadPage(pageTitle)
	if err != nil {
		http.Redirect(w, r, "/edit/"+pageTitle, http.StatusFound)

	} else {
		viewTemplate, _ := template.ParseFiles("view.html")
		viewTemplate.Execute(w, loadedPage)
		//fmt.Fprintf(w, "<html><body>%s</body></html> ", loadedPage.Body)
	}
}

func editHandler(w http.ResponseWriter, r *http.Request) {
	pageTitle := r.URL.Path[len("/edit/"):]
	loadedPage, err := loadPage(pageTitle)

	if err != nil {
		loadedPage = &Page{Title: pageTitle}
	}

	editTemplate, _ := template.ParseFiles("edit.html")
	editTemplate.Execute(w, loadedPage)
}

func promptFilenames(c net.Conn, bufc *bufio.Reader) []string {

	filenames, _, _ := bufc.ReadLine()
	// log.Println("Filenames of slave fetched!\n")
	output := strings.Fields(string(filenames))
	return output

}



func acceptLoop(l net.Listener, clientChan chan Client, searchan chan string) {
	defer l.Close()
	// fmt.Println("In Accept LOop: ")

	visited := make([]string, 1)

	// msgchan := make(chan string)
	// procchan := make(chan string) // to be processed filename

	// //A channel to keep track of Slaves, slaves are added to this channel
	// //handleMessages then iterates through slaves and for each appends to its channel
	// //the messages sent by other users, broadcat
	// addchan := make(chan Slave)

	// defer close(msgchan)
	// defer close(procchan)
	// defer close(msgchan)
	// defer close(addchan)
	// defer close (rmchan)

	
	slavechan := make(chan Slave)

	// a go routine for scheduling 
	go AssigningFileFromChunk(slavechan, clientChan, visited)

	// accpeting new slaves
	for {
		c, err := l.Accept()
		// fmt.Println("apple")
		if err != nil {
			fmt.Println(err)
			continue
		}

		bufc := bufio.NewReader(c)
		defer c.Close()

		slave := Slave{
			conn:                c,
			stopping:            searchan,
			nickname:            promptNick(c, bufc),
			ch_cllients:         make(chan []Client, 10),
			files:               promptFilenames(c, bufc),
			file_to_process:     "",
			total_current_files: make(chan []string, 500),
		}

		
		slavechan <- slave
		// close(slavechan)

	}

}



func Save(clientChan chan Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// logger <- r.Host
		// io.WriteString(w, "Hello world!")

		// pageTitle := r.URL.Path[len("/save/"):]
		pageBody := r.FormValue("body")

		// passwordchan <- fmt.Sprintf("%s",pageBody)

		client := Client{
			// conn     net.Conn
			// stopping chan string
			password: pageBody,
			nickname: pageBody,
			// ch       chan string
			current:        make([]string, 1),
			visited:        make([]string, 1),
			passwordStatus: make(chan string, 20),
		}
		clientChan <- client

		password := <-client.passwordStatus
		// fmt.Println(">>> ", password)
		io.WriteString(w, "Password: "+client.password+" is "+string(password)) // in use or not in use

	}
}

func main() {

	arr := os.Args[1:]
	// ip := ""
	clport := ""
	slport := ""
	// var directory = "./data/"

	if len(arr) == 0 {
		slport = ":3000"
		clport = ":8081"

	} else if len(arr) == 1 {
		slport = ":" + arr[0]
		clport = ":8081"
	} else if len(arr) == 2 {

		slport = ":" + arr[0]
		clport = ":" + arr[1]
	}
	fmt.Println(clport, slport)

	clientChan := make(chan Client)
	
	searchan := make(chan string, 1)

	saveHandler := Save(clientChan)

	http.HandleFunc("/", defaultHandler)
	http.HandleFunc("/view/", viewHandler)
	http.HandleFunc("/edit/", editHandler)
	http.HandleFunc("/save/", saveHandler)
	

	listener, err := net.Listen("tcp", slport)
	if err != nil {
		log.Fatal(err)
		fmt.Println("Error: ", err)
	}
	
	go acceptLoop(listener, clientChan, searchan) // slaves wait
	http.ListenAndServe(clport, nil)
	

}
