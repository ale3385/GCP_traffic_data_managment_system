package main

import (
	"fmt"
	"log"
	"net/http"
  "io"
  "io/ioutil"
  "encoding/json"
  _ "strconv"
  _ "html"
  _ "html/template"
  "os"
  "sync"
	"time"
  _ "encoding/base64"
	"google.golang.org/appengine"
  "cloud.google.com/go/pubsub"
	"github.com/gocql/gocql"

	"github.com/newrelic/go-agent"
)

var (
	messagesMu sync.Mutex
  countMu sync.Mutex
	count   int
  subscription *pubsub.Subscription

	datasetKeyspace string
	sUsername string
	sPassword string
	sHost string
)

func main() {

	datasetKeyspace = getENV("CASSANDRA_KEYSPACE")
	sUsername = getENV("CASSANDRA_UNAME")
	sPassword = getENV("CASSANDRA_UPASS")
	sHost = getENV("CASSANDRA_HOST")

	//  newrelic part
	config := newrelic.NewConfig("cassandra-service", "df553dd04a541579cffd9a3a60c7afa9ca692cc7")
	app, err := newrelic.NewApplication(config)
	if err != nil {
    log.Printf("ERROR: Issue with initializing newrelic application ")
	}

	http.HandleFunc(newrelic.WrapHandleFunc(app, "/_ah/health", healthCheckHandler))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/push", pushHandler))
  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "This is main entry for endpoints..")
	})

	log.Print("Starting service.....")
	appengine.Main()
}


type pushRequest struct {
    Message struct {
        Attributes map[string]string
        Data       string `json:"data"`
        message_id     string `json:"message_id"`
        messageId      string `json:"messageId"`
        publish_time   string `json:"publish_time"`
        publishTime    string `json:"publishTime"`
    }
    Subscription string
}

type entityEntryJSONStruct struct {
	Direction string `json:"_direction"`
	Fromst string `json:"_fromst"`
	Last_updt string `json:"_last_updt"`
	Length string `json:"_length"`
	Lif_lat string `json:"_lif_lat"`
	Lit_lat string `json:"_lit_lat"`
	Lit_lon string `json:"_lit_lon"`
	Strheading string `json:"_strheading"`
	Tost string `json:"_tost"`
	Traffic string `json:"_traffic"`
	Segmentid string `json:"segmentid"`
	Start_lon string `json:"start_lon"`
	Street string `json:"street"`
}


func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "ok")
}

func pushHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

  if r.Body == nil {
      log.Print("ERROR: Please send a request body")
      return
  }
  body, err := ioutil.ReadAll(r.Body)
  defer r.Body.Close()
  if err != nil {
    log.Printf("INFO:  Can't read http body ioutil.ReadAll... ")
		return
	}
  //var msg pushRequest
  //if err := json.Unmarshal([]byte(body), &msg); err != nil {
  //  log.Printf("ERROR: Could not decode body with Unmarshal: %s \n", string(body))
  //}
  //log.Printf("DEBUG:  >>>>>  body: %s \n", string(body))
  //log.Printf("DEBUG:  >>>>>  messageId: "    + msg.Message.messageId + "\n")
  // sDec, _  := b64.StdEncoding.DecodeString( msg.Message.Data )
  //log.Printf("DEBUG:  >>>>> Message.Data:" + string(sDec) + "\n")

	log.Printf("DEBUG:  >>>>>  body: %s \n", string(body))

  var data entityEntryJSONStruct
  if err := json.Unmarshal(body /*sDec*/, &data); err != nil {
		errmsg := "ERROR: Could not decode Message.Data into Entry type with Unmarshal: " + string(body)
    log.Printf(errmsg + "\n")
		io.WriteString(w, "{\"status\":\"1\", \"message\":\"" + errmsg + "\"}")
  }

	cassandraWriter(w, r, data)

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "{\"status\":\"0\", \"message\":\"ok\"}")
}

func getENV(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("%s environment variable not set.", k)
	}
	return v
}

//-----------------------------------------------------------------------------------------------
var thesession *gocql.Session

func initSession() error {
    var err error
    if thesession == nil || thesession.Closed() {
        thesession, err = getCluster().CreateSession()
    }
    return err
}

func getCluster() *gocql.ClusterConfig {
     cluster := gocql.NewCluster(sHost)
     cluster.Keyspace = datasetKeyspace
		 cluster.Timeout = 3 * time.Second
		 cluster.NumConns = 16
	   cluster.Authenticator = gocql.PasswordAuthenticator{
	 		Username: sUsername,
	 		Password: sPassword,
	 	}
     cluster.Consistency = gocql.One
     cluster.Port = 9042   // default port
     return cluster
}

func cassandraWriter(w http.ResponseWriter, r *http.Request, e entityEntryJSONStruct) {

//  const cConsistency gocql.Consistency = gocql.One
//	cluster := gocql.NewCluster(sHost)
//	cluster.ProtoVersion = 4
//	//cluster.RetryPolicy = &gocql.SimpleRetryPolicy{NumRetries:3}
//	//cluster.Timeout = 2 * time.Second
//	cluster.NumConns = 1
//  cluster.Authenticator = gocql.PasswordAuthenticator{
//		Username: sUsername,
//		Password: sPassword,
//	}
//	//log.Println("INFO: sHost: ", sHost)
//	cluster.Keyspace = datasetKeyspace
//	cluster.Consistency = cConsistency

//	session, err := cluster.CreateSession()
	err := initSession()
	if err != nil {
		msg := "Error creating session: " + err.Error()
		log.Printf(msg)
		io.WriteString(w, "{\"status\":\"1\", \"message\":\""+ msg +"\"}")
		//log.Fatalf(msg)
	}
	defer thesession.Close()

	err = thesession.Query(
		`INSERT INTO northamerica.datasetentry (id, Direction, Fromst, Last_updt, Length, Lif_lat, Lit_lat, Lit_lon, Strheading, Tost, Traffic, Segmentid, Start_lon, Street) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		gocql.TimeUUID(), e.Direction, e.Fromst, e.Last_updt, e.Length, e.Lif_lat, e.Lit_lat, e.Lit_lon, e.Strheading, e.Tost, e.Traffic, e.Segmentid, e.Start_lon, e.Street ).Exec()
  if err != nil {
		msg := "Error Authentication: " + err.Error()
		io.WriteString(w, msg)
		log.Printf(msg)
		//log.Fatalf(msg)
		return
	}

}