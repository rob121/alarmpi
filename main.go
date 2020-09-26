package main

import (
	"time"
	"log"
	"strings"
	"net/http"
    "github.com/spf13/viper"
	"github.com/warthog618/gpiod"
	"github.com/warthog618/gpiod/device/rpi"	
	"github.com/gorilla/mux"
	"os/exec"
	"html/template"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/olebedev/emitter"
)

var upgrader = websocket.Upgrader{} 
var pins map[int]*PinAssociation
var events chan gpiod.LineEvent
var done chan int
var chip *gpiod.Chip
var emt *emitter.Emitter


type PinAssociation struct{ 
    Name string
    Pin int
    Label string
    OnOpen string
    OnClose string
    ActionType string   
    PinLine *gpiod.Line
    State string
}

func main() {
    
    log.SetFlags(log.LstdFlags | log.Lshortfile)
    emt = &emitter.Emitter{}
    events = make(chan gpiod.LineEvent)
    done = make(chan int)
    pins = make(map[int]*PinAssociation)
    var err error
    setupConfig()
	// Chip Initialisation
	chip, err = gpiod.NewChip(viper.GetString("Chip"), gpiod.WithConsumer(viper.GetString("AppName")))
    defer chip.Close()

    if(err!=nil){
        
        panic(err)
        
    }
   
    //handl the events here
    go startWebServer()
    go eventWatcher()
    
    
    setupPins()
    
  
    
    //send the pins off to webserver
	
	select{} //block forever

}

func startWebServer(){
    
    log.Println("Starting Webserver")
    
    r := mux.NewRouter()
    r.HandleFunc("/ws", wsHandler)
    r.HandleFunc("/", indexHandler)
    http.Handle("/", r)
 
 
    for _,v := range pins {
        
        log.Printf("%#v",v)
        
    }
    
    
    srv := &http.Server{
        Handler:      r,
        Addr:         ":"+viper.GetString("Port"),
        // Good practice: enforce timeouts for servers you create!
        WriteTimeout: 15 * time.Second,
        ReadTimeout:  15 * time.Second,
    }
    
    log.Println("Listening on port",viper.GetString("Port"))
    log.Fatal(srv.ListenAndServe())
    
    
}

type PageData struct{
    Title string
    Url string
    Pins map[int]*PinAssociation
    Ws string
}

func wsHandler(w http.ResponseWriter, r *http.Request) {

	c, err := upgrader.Upgrade(w, r, nil)
	
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	
	defer c.Close()
	
	go func(){
    	
    	
       for range emt.On("change") {
         
          log.Println("Sending to WS")
          
          b, jerr := json.Marshal(pins)
    	  if jerr != nil {
    		log.Println("encoding error:", jerr)
    		continue
    	  }
          
          
          err = c.WriteMessage(websocket.TextMessage, b)
  
          if(err!=nil){
             log.Println("write:", err) 
          }
         
    
       }
    	
    	
	}()
	
	
	for {
    	
        mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		
        log.Printf("recv: %s", message)
		
		if(string(message)== "__ping__" ){
    		
          err = c.WriteMessage(mt, []byte("__pong__"))
  
          if(err!=nil){
             log.Println("write:", err) 
          }
    		
		}
		

	
	}
	
	
   
	
	
	
	
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
  //  vars := mux.Vars(r)
    //w.WriteHeader(http.StatusOK)
    

    data := PageData{
        "Test",
        viper.GetString("Port"),
        pins,
        template.HTMLEscapeString("ws://"+r.Host+"/ws"),
    }
    
    tmpl := template.New("Index")
    tmpl,_=tmpl.Parse(Index)
    
    tmpl.Execute(w, data)
    
    //fmt.Fprintf(w,Index)
}

func setupPins(){
    
    log.Println("Setting Up Pins")
    
    rawpins:=viper.GetStringMap("Pins")
    
    for k,_:= range rawpins {
   
       data:=viper.GetStringMap("Pins."+k)
   
       pinstr:=strings.ToLower(k)  
   
       pin,err:=rpi.Pin(pinstr)
       
       if(err!=nil){
           
           continue
       }
          
       l, rerr := chip.RequestLine(pin, gpiod.WithBothEdges(handler))
       
       if(rerr!=nil){
           
           
           continue
           
       }
   
       pa := &PinAssociation{k,pin,data["label"].(string),data["open"].(string),data["closed"].(string),data["type"].(string),l,""}

       var state gpiod.LineEventType
        
       r, _ := l.Value() // Read state from line (active/inactive)
	   
    

	   log.Println("Current Value",r)
	   
       state = gpiod.LineEventType(r)
	   
	   evt:=gpiod.LineEvent{pin,time.Second,state}
	   
	   pins[pin]=pa
	   
	   events <- evt
   
   
	   
          
    }
    
    
}

func eventWatcher(){
    
    
     for {
        select {
        case evt := <-events:
           // log.Printf("New Event %#v\n",evt)
            
            var obj *PinAssociation
            var ok bool
            
            if obj, ok = pins[evt.Offset]; !ok {
                
                   log.Printf("Pin Data not in configuration! %#v\n",evt)
                   break
        
            }
            
            pins[evt.Offset].State = "open"
            
            if int(evt.Type) == 1 {
                
                pins[evt.Offset].State  = "closed"
                
            }
            
            //pinchan <- pins
            
             <-emt.Emit("change", 1)
            
            switch obj.ActionType { 
                case "http":
                actionHttp(obj,int(evt.Type))
                break
                case "exec":
                actionExec(obj,int(evt.Type))
                break
                default:
                 log.Println("Action Not Implemented!")
                break
            }
            
       
        case do := <-done:
            log.Println("Received done:", do)
        }
      
       }

    
    
    
}

func actionHttp(obj *PinAssociation,state int){
    
    
    log.Println("Received http action")
    
    
    var url string
    
    dur,perr := time.ParseDuration(viper.GetString("HttpActionTimeout"))
    
    if(perr!=nil){
        
        log.Println("Invalid Duration")
        dur = 15 * time.Second
        
    }

    c := &http.Client{
    
    Timeout: dur,
    
    }
    
    if(state==1){

        log.Printf("Received event on %s (%s): %s",obj.Label,obj.Name,"Closed")
        
        url =  obj.OnClose

        
    }else{
        
        log.Printf("Received event on %s (%s): %s",obj.Label,obj.Name,"Open")
        
        url =  obj.OnOpen
  
        
    }
    
    if (len(url)<12){
        
        log.Println("Invalid Url")
        return
    }

    
    _, err := c.Get(url)

	// check for response error
	if err != nil {
		log.Println( err )
		return
	}
	
	log.Println("Action Success!")

    
    
}

func actionExec(obj *PinAssociation,state int){
    
        log.Println("Received event action")
        
        var statetype string
        var cstr string
        
        if(state==1){ 
            
            cstr = obj.OnClose
            statetype = "closed"
            
        }else{
            
            cstr = obj.OnOpen 
            statetype = "open"
            
        }
        
        log.Printf("Executing %s %s %s %s",cstr,obj.Name,statetype,obj.Label)
        
        cmd := exec.Command(cstr,obj.Name,statetype,obj.Label)
        
        out, err := cmd.CombinedOutput()
        
        if err != nil {
                log.Fatalf("cmd.Run() failed with %s\n", err)
        }
        
        log.Printf("combined out:\n%s\n", string(out))
        
        
        log.Println("Action Success!")
    
}

func handler(evt gpiod.LineEvent) {
	// handle change in line state
	//New Event gpiod.LineEvent{Offset:16, Timestamp:1601067131395882654, Type:2}
	events <- evt
}

func setupConfig(){
    
   log.Println("Setting Configuration") 
    
   viper.SetDefault("AppName","alarmpi") 
   viper.SetDefault("HttpActionTimeout","15s")
   viper.SetDefault("Chip","gpiochip0")
   viper.SetDefault("Debug",false)  
   viper.SetDefault("Port","8000")
   viper.SetConfigType("json")
   viper.SetConfigName("config") // name of config file (without extension)
   viper.AddConfigPath("/etc/alarmpi/")   // path to look for the config file in
   viper.AddConfigPath("$HOME/.alarmpi")  // call multiple times to add many search paths
   viper.AddConfigPath(".")               // optionally look for config in the working directory
   err := viper.ReadInConfig() // Find and read the config file
   
   if err != nil { // Handle errors reading the config file
	//panic(fmt.Errorf("Fatal error config file: %s \n", err))
	panic(err)
   }
   
   
}