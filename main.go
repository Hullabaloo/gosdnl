package main

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/lib/pq"
	//_ "github.com/lib/pq"
)

var isRunning bool

//var myMutex sync.Mutex

type SdnList struct {
	XMLName  xml.Name   `xml:"sdnList"`
	SdnEntry []SdnEntry `xml:"sdnEntry"`
}

type SdnEntry struct {
	XMLName   xml.Name `xml:"sdnEntry"`
	Uid       string   `xml:"uid"`
	LastName  string   `xml:"lastName"`
	FirstName string   `xml:"firstName"`
	SdnType   string   `xml:"sdnType"`
}

// upade database from the remote xml file
func appUpdate(w http.ResponseWriter, r *http.Request) {
	// myMutex.Lock()
	// defer myMutex.Unlock()

	if isRunning {
		// method is already running, do not execute again
		return
	}

	isRunning = true
	defer func() { isRunning = false }()

	runResult := processRemoteXml()

	// Return JSON response
	jsonResp := map[string]interface{}{
		"result": true,
		"info":   "",
		"code":   200,
	}
	if !runResult {
		jsonResp = map[string]interface{}{
			"result": false,
			"info":   "service unavailable",
			"code":   503,
		}
	}
	json.NewEncoder(w).Encode(jsonResp)
}

// check the app state
func appState(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"result": false,
		"info":   "empty",
	}
	// Return JSON response
	if isRunning {
		resp = map[string]interface{}{
			"result": false,
			"info":   "updating",
		}
	} else {
		db, err := sql.Open("postgres", "user=postgres password=postgres dbname=sdnl host=db port=5432 sslmode=disable")
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		var count int

		// Execute the query to get the count of records in the individuals table
		err = db.QueryRow("SELECT COUNT(*) FROM individuals").Scan(&count)

		// Check if there was an error executing the query
		if err != nil {
			log.Fatal(err)
		}

		// Check if the count of records is greater than 0
		if count > 0 {
			resp = map[string]interface{}{
				"result": true,
				"info":   "ok",
			}
		} else {
			resp = map[string]interface{}{
				"result": false,
				"info":   "empty",
			}
		}
	}
	json.NewEncoder(w).Encode(resp)
}

// get the JSON array of data
// get params: name, type
func appGetNames(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	name := r.URL.Query().Get("name")
	reqType := r.URL.Query().Get("type")
	qOption := r.URL.Query().Get("option")

	db, err := sql.Open("postgres", "user=postgres password=postgres dbname=sdnl host=localhost port=5432 sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("")

	if reqType != "strong" && reqType != "superstrong" {
		reqType = "weak"
	}
	if reqType == "weak" {
		// split the name parameter into parts
		nameParts := strings.Fields(name)
		var newA []string
		// get the query words array and concatenate the '%' chars with each element, put element to new array "newA"
		for _, elem := range nameParts {
			newA = append(newA, fmt.Sprintf("%%%s%%", elem))
		}

		if qOption == "full" {
			// full word search query using regexp word split by whitespace
			rows, err = db.Query("SELECT uid, firstname, lastname FROM"+
				"(SELECT uid, firstname, lastname, regexp_split_to_table(firstname || ' ' || lastname, '\\s+') as splitted from individuals"+
				") individuals WHERE splitted ILIKE ANY($1)", pq.Array(nameParts))
		} else {
			// partial word search query
			rows, err = db.Query("SELECT uid, firstname, lastname FROM individuals WHERE firstname ILIKE ANY($1) OR lastname ILIKE ANY($1)", pq.Array(newA))
		}

	} else if reqType == "strong" {
		rows, err = db.Query("SELECT uid, firstname, lastname FROM individuals WHERE firstname ILIKE $1 OR lastname ILIKE $1 OR firstname || ' ' || lastname ILIKE $1", name)
	} else if reqType == "superstrong" {
		rows, err = db.Query("SELECT uid, firstname, lastname FROM individuals WHERE firstname || ' ' || lastname = $1", name)
	}

	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var uid string
		var firstName string
		var lastName string

		err = rows.Scan(&uid, &firstName, &lastName)
		if err != nil {
			log.Fatal(err)
		}

		results = append(results, map[string]interface{}{
			"uid":       uid,
			"firstname": firstName,
			"lastname":  lastName,
		})
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	if results == nil {
		resp := map[string]interface{}{
			"result": false,
			"info":   "Nothing found",
			"code":   "404",
		}
		json.NewEncoder(w).Encode(resp)

	} else {
		json.NewEncoder(w).Encode(results)
	}
}

func processRemoteXml() bool {
	url := "https://www.treasury.gov/ofac/downloads/sdn.xml"

	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return false
	}
	defer resp.Body.Close()
	var entries = 0
	var entriesnsdn = 0

	db, err := sql.Open("postgres", "user=postgres password=postgres dbname=sdnl host=localhost port=5432 sslmode=disable")
	if err != nil {
		log.Println(err)
		return false
	}
	defer db.Close()

	if resp.StatusCode != 200 {
		log.Println("xml not found")
		return false
	}

	nuidsList := []string{}
	decoder := xml.NewDecoder(resp.Body)
	for {
		token, err := decoder.Token()
		if token == nil || err != nil {
			break
		}

		// process only StartElement tokens with name "sdnEntry"
		se, ok := token.(xml.StartElement)
		if !ok || se.Name.Local != "sdnEntry" {
			entriesnsdn++
			continue
		}

		// decode the sdnEntry element
		var sdnEntry SdnEntry
		err = decoder.DecodeElement(&sdnEntry, &se)
		if err != nil {
			log.Println(err)
			return false
		}

		// check if sdnType is "Individual"
		if sdnEntry.SdnType == "Individual" {
			nuidsList = append(nuidsList, sdnEntry.Uid) // check if UID is in the database)
			//fmt.Printf("UID: %s, Name: %s %s\n", sdnEntry.Uid, sdnEntry.FirstName, sdnEntry.LastName)
			entries++

			var fn, ln string
			err = db.QueryRow("SELECT firstname, lastname FROM individuals WHERE uid = $1", sdnEntry.Uid).Scan(&fn, &ln)
			if err != nil {
				// insert new data if no UID user found in the database
				if err == sql.ErrNoRows {
					sqlStatement := `INSERT INTO individuals (uid, firstname, lastname) VALUES ($1, $2, $3) RETURNING id`
					id := 0 // use isert_id if required
					err = db.QueryRow(sqlStatement, sdnEntry.Uid, sdnEntry.FirstName, sdnEntry.LastName).Scan(&id)
					if err != nil {
						log.Println(err)
						return false
					}
				} else {
					log.Println(err)
					return false
				}
			} else {
				// data update if UID found in the database
				// we can update data if firstName or lastName is different
				//
				// !!!! as an option we can store a hash of required data as an additional column in the same table and update
				// the record only if hash for the xml entry data is different than one in the current database record

				if sdnEntry.FirstName != fn || sdnEntry.LastName != ln {

					// Prepare the SQL statement
					sqlStatement := "UPDATE individuals SET firstname=$1, lastname=$2, updated_at=$3 WHERE uid=$4"

					// prepare the timestamp in the desired format
					updatedAt := time.Now().Format("2006-01-02 15:04:05")

					// Execute the SQL statement with the given parameters
					_, err := db.Exec(sqlStatement, sdnEntry.FirstName, sdnEntry.LastName, updatedAt, sdnEntry.Uid)
					if err != nil {
						log.Println(err)
						return false
					}
				}
			}

		}
		// if entries == 50 {
		// 	break
		// }
	}

	// delete entries from database if db UID record is not found in the xml data source
	if len(nuidsList) > 0 {
		rows, err := db.Query("SELECT id, uid FROM individuals")
		if err != nil {
			log.Println(err)
			return false
		}
		defer rows.Close()

		// Loop through the records and check if the uid is in the persons array
		for rows.Next() {
			var id int
			var uid string
			err = rows.Scan(&id, &uid)
			if err != nil {
				log.Println(err)
				return false
			}
			if !contains(nuidsList, uid) {
				// delete the record from the database
				deleteRecord(db, id)
			}
		}
		err = rows.Err()
		if err != nil {
			log.Println(err)
			return false
		}
	}

	return true
}

// Helper function to check if a string is in a string slice
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func deleteRecord(db *sql.DB, id int) error {
	// Prepare the SQL statement
	stmt, err := db.Prepare("DELETE FROM individuals WHERE id=$1")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Execute the statement with id param
	_, err = stmt.Exec(id)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	http.HandleFunc("/update", appUpdate)
	http.HandleFunc("/state", appState)
	http.HandleFunc("/get_names", appGetNames)

	fmt.Println("Server started at http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
