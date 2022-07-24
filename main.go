package main

import (
  "os"
  "os/signal"
  "context"
  "time"
  "net/http"
  "io/ioutil"
  "encoding/json"
  "github.com/hashicorp/go-hclog"
  "log"
  "github.com/gorilla/mux"
  mgo "gopkg.in/mgo.v2"
  "gopkg.in/mgo.v2/bson"

)


// The struct below holds our database session information
type DBSession struct {
    session      *mgo.Session
    collection   *mgo.Collection
}


//Book struct holds book data
type Book struct {
    ID            bson.ObjectId    `json: "id"          bson:"_id,omitempty"`
    Title         string           `json: "title"       bson:"title"`
    Authors       []string         `json: "authors"     bson: "authors"`
    Genre         []string         `json: "genre"       bson: "genre"`
    PublishDate   string           `json: "publishdate" bson: "publishdate"`
    Characters    []string         `json: "characters"  bson: "characters"`
    Publisher     Publisher                 `json: "publisher"   bson:"publisher"`  
}


// Publisher is nested in Movie
type Publisher struct {
    Name     string   `json: "budget"    bson:"name"`
    Country  string   `json: "country"   bson:"country"`
    website  string   `json: "website"   bson:"website"`
}


// GetBook fetches a book with a given ID
func (db *DBSession) GetBook (w http.ResponseWriter, r *http.Request){
    vars := mux.Vars(r)

    w.WriteHeader(http.StatusOK)
    var book Book
    err := db.collection.Find(bson.M{"_id": bson.ObjectIdHex(vars["id"])}).One(&book)
    if err != nil {
        w.Write([]byte(err.Error()))
    } else {
        w.Header().Set("Content-Type", "application/json")
        response, _ := json.Marshal(book)
        w.Write(response)
    }
}


//PostBook adds a new book to our MongoDB collection
func (db *DBSession) PostBook (w http.ResponseWriter, r *http.Request){
    var book Book
    postBody, _ := ioutil.ReadAll(r.Body)
    json.Unmarshal(postBody, &book)

    //Create a Hash ID to insert
    book.ID = bson.NewObjectId()
    err := db.collection.Insert(book)
    if err != nil {
        w.Write([]byte(err.Error()))
    } else {
        w.Header().Set("Content-Type","application/json")
        response, _ := json.Marshal(book)
        w.Write(response)
    }
}




//UpdateBook modifies the data of an existing book resource
func (db *DBSession) UpdateBook(w http.ResponseWriter, r *http.Request){
    vars := mux.Vars(r)
    var book Book

    putBody, _ := ioutil.ReadAll(r.Body)
    json.Unmarshal(putBody, &book)
    err := db.collection.Update(bson.M{"_id": 
        bson.ObjectIdHex(vars["id"])}, bson.M{"$set": &book})

    if err != nil {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(err.Error()))
    } else {
        w.Header().Set("Content-Type","text")
        w.Write([]byte("Update succesfully"))
    }
}


//DeleteBook removes the data from the db
func (db *DBSession) DeleteBook (w http.ResponseWriter, r *http.Request){
     vars := mux.Vars(r)
     err := db.collection.Remove(bson.M{"_id":
         bson.ObjectIdHex(vars["id"])})
     if err != nil {
         w.WriteHeader(http.StatusOK)
         w.Write([]byte(err.Error()))
     } else {
         w.Header().Set("Content-Type", "text")
         w.Write([]byte("Delete Succesfully"))
     }
}

func main() {
    l := hclog.Default()
    session, err := mgo.Dial("127.0.0.1")
    c := session.DB("booksdb").C("books")
    db := &DBSession{session: session, collection:c}
    addr := "127.0.0.1:8000"
    if err != nil {
        panic(err)
    }
    defer session.Close()
    
    //logger := log.New(os.Stdout, "", log.Ldate | log.Ltime)
    // Create a new router
    r := mux.NewRouter()

    //Attach an elegant path with handler
    r.HandleFunc("/api/books/{id:[a-zA-Z0-9]*}", db.GetBook).Methods("GET")
    r.HandleFunc("/api/books", db.PostBook).Methods("POST")
    r.HandleFunc("/api/books/{id:[a-zA-Z0-9]*}", db.UpdateBook).Methods("PUT")
    r.HandleFunc("/api/books/{id:[a-zA-Z0-9]*}", db.DeleteBook).Methods("DELETE")

    srv := &http.Server{
        Handler:       r,
        Addr:          addr,
        ErrorLog:      l.StandardLogger(&hclog.StandardLoggerOptions{}),
        IdleTimeout:   time.Minute,
        WriteTimeout:  15 * time.Second,
        ReadTimeout:   15 * time.Second,
    }

    //start the server
    go func() {
      l.Info("Starting server on port 8000 ")
      err := srv.ListenAndServe()
      if err != nil {
          l.Error("Error starting the server :", "error", err)
          os.Exit(1)
      }        
    }()


    //Trap sigterm or interupt and gracefully shutdown the server
    ch := make(chan os.Signal, 1)
    signal.Notify(ch, os.Interrupt)
    signal.Notify(ch, os.Kill)
    
    //Block untill a signal is received
    sig := <- ch
    log.Println("Got signal :", sig)


    //gracefully shutdown the server, waiting max 30 for current operations to complete
    ctx, _ := context.WithTimeout(context.Background(), 30 * time.Second)
    srv.Shutdown(ctx)

}


