package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/asdine/storm/v3"
        "github.com/lithammer/shortuuid/v3"
	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/gorilla/websocket"
	"github.com/kirsle/configdir"
	"github.com/olebedev/emitter"
	"github.com/spf13/viper"
	"github.com/warthog618/gpiod"
	"github.com/warthog618/gpiod/device/rpi"
	"html/template"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var upgrader = websocket.Upgrader{}
var pins map[int]*PinAssociation

//var events chan LineEvent
var done chan int
var emt *emitter.Emitter
var events chan gpiod.LineEvent
var chip *gpiod.Chip
var attributes Attributes
var db *storm.DB

//go:embed tmpl/index.html
var Index string

//go:embed tmpl/form.html
var Form string

type PinAssociation struct {
	Name       string `storm:"id"`
	Pin        int
	Label      string `storm:"unique"`
	OnOpen     string
	OnClose    string
	ActionType string
	PinLine    *gpiod.Line `json:"-"`
	State      string
	Device     string
	Active     bool
}

func main() {
	var err error

	configPath := configdir.LocalConfig("alarmpi")
	err = configdir.MakePath(configPath) // Ensure it exists.

	if err != nil {
		panic(err)
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	emt = &emitter.Emitter{}
	events = make(chan gpiod.LineEvent)
	done = make(chan int)
	pins = make(map[int]*PinAssociation)

	setupConfig()

	db, err = storm.Open(filepath.Join(configPath, "alarmpi.db"))

	if err != nil {

		panic(err)

	}

	defer db.Close()

	// Chip Initialisation
	chip, err = gpiod.NewChip(viper.GetString("Chip"), gpiod.WithConsumer(viper.GetString("AppName")))
	defer chip.Close()

	if err != nil {
	}

	attributes = loadAttributes()
	go eventWatcher()

	setupPins()

	//handle the events here
	go startWebServer()

	select {} //block forever

}

func startWebServer() {

	log.Println("Starting Webserver")

	r := mux.NewRouter()
	r.HandleFunc("/ping", pingHandler)
	r.HandleFunc("/config/{pin}", configHandler)
	r.HandleFunc("/remove/{pin}", removeHandler)
	r.HandleFunc("/status", sensorsHandler)
	r.HandleFunc("/ws", wsHandler)
	r.HandleFunc("/save", saveHandler)
	r.HandleFunc("/", indexHandler)
	http.Handle("/", r)

	srv := &http.Server{
		Handler: r,
		Addr:    ":" + viper.GetString("Port"),
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Println("Listening on port", viper.GetString("Port"))
	log.Fatal(srv.ListenAndServe())

}

type PageData struct {
	Title      string
	Url        string
	Pins       map[int]*PinAssociation
	Ws         string
	Attributes Attributes
	Pin        PinAssociation
}

func pingHandler(w http.ResponseWriter, r *http.Request) {

	<-emt.Emit("change", 1)
	fmt.Fprintf(w, "pong")
}

func activePins(pins map[int]*PinAssociation) map[int]*PinAssociation {

	var out = make(map[int]*PinAssociation)

	for k, v := range pins {

		if v.Active == true {

			out[k] = v

		}

	}

	return out

}

func sensorsHandler(w http.ResponseWriter, r *http.Request) {

	out := activePins(pins)

	b, jerr := json.Marshal(out)
	if jerr != nil {
		log.Println("encoding error:", jerr)
		fmt.Fprintf(w, `{ "error": true }`)
		return
	}

	fmt.Fprintf(w, string(b))
}

func setupPins() {

	log.Println("Setting Up Pins")

	var rawpins []PinAssociation

	err := db.All(&rawpins)

	if err != nil {

		log.Println(err)

	}

	ii := 0
	for _, par := range rawpins {

		ii++

		pinstr := strings.ToLower(par.Name)

		pin, _ := rpi.Pin(pinstr)

		pinstr = strings.ToUpper(par.Name)

		var pa *PinAssociation
		var ok bool

		if pa, ok = pins[pin]; ok {

		} else {

			l, rerr := chip.RequestLine(pin, gpiod.WithBothEdges, gpiod.WithEventHandler(handler))

			if rerr != nil {
				log.Println(rerr)
				continue
			}

			var state gpiod.LineEventType

			r, err := l.Value() // Read state from line (active/inactive)

			if(err!=nil){

				log.Println(err)

			}

			state = gpiod.LineEventType(r)

			evt := gpiod.LineEvent{pin, time.Second, state}

			pa = &PinAssociation{}
			pa.PinLine = l
			<-emt.Emit("gpioevent", evt)
		}

		pa.Active = par.Active
		pa.ActionType = par.ActionType
		pa.Device = par.Device
		pa.OnClose = par.OnClose
		pa.OnOpen = par.OnOpen
		pa.Label = par.Label
		pa.Name = pinstr
		pa.Pin = pin

		pins[pin] = pa

	}

}

func removeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pinraw := vars["pin"]

	var pa PinAssociation

	err := db.One("Name", pinraw, &pa)

	if err != nil {
		fmt.Println(err)
	}

	pa.Active = false

        uui := shortuuid.New()

        pa.Name = uui //allow us to overwrite whatever is here

	err = db.Save(&pa)

	if err != nil {
		fmt.Println(err)
	}

	setupPins()

	http.Redirect(w, r, fmt.Sprintf("/"), 302)

}

var decoder = schema.NewDecoder()

func saveHandler(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()

	var pa PinAssociation

	err := decoder.Decode(&pa, r.PostForm)

	if err != nil {
		log.Println("Error in POST parameters : ", err)
	}

	pa.Active = true

	db.Save(&pa)

	setupPins()

	http.Redirect(w, r, fmt.Sprintf("/"), 302)

}

func wsHandler(w http.ResponseWriter, r *http.Request) {

	c, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	defer c.Close()

	go func() {

		for range emt.On("change") {

			log.Println("Sending to WS")

			out := activePins(pins)
			b, jerr := json.Marshal(out)
			if jerr != nil {
				log.Println("encoding error:", jerr)
				continue
			}

			err = c.WriteMessage(websocket.TextMessage, b)

			if err != nil {
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

		if string(message) == "__ping__" {

			err = c.WriteMessage(mt, []byte("__pong__"))

			if err != nil {
				log.Println("write:", err)
			}

		}

	}

}

func indexHandler(w http.ResponseWriter, r *http.Request) {

	var rawpins []PinAssociation

	db.All(&rawpins)

	data := PageData{
		"Test",
		viper.GetString("Port"),
		pins,
		template.HTMLEscapeString("ws://" + r.Host + "/ws"),
		attributes,
		PinAssociation{},
	}

	tmpl := template.New("Index")
	tmpl, _ = tmpl.Parse(Index)

	tmpl.Execute(w, data)

	//fmt.Fprintf(w,Index)
}

func configHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	//w.WriteHeader(http.StatusOK)

	var pa PinAssociation

	derr := db.One("Name", vars["pin"], &pa)

	if derr != nil {

		log.Println("Not Found", derr)

	}

	data := PageData{
		Title:      "Form",
		Url:        viper.GetString("Port"),
		Pins:       nil,
		Ws:         template.HTMLEscapeString("ws://" + r.Host + "/ws"),
		Attributes: attributes,
		Pin:        pa,
	}

	var err error
	tmpl := template.New("Index")
	tmpl, err = tmpl.Parse(Form)

	if err != nil {

		fmt.Println(err)
	}

	tmpl.Execute(w, data)

	//fmt.Fprintf(w,Index)
}

func eventWatcher() {

	//the below code debounces events from gpi, change timeout to smooth things out
	eventChan := make(chan gpiod.LineEvent)

	go debounceEvent(50*time.Millisecond, eventChan, func(evt gpiod.LineEvent) {

		log.Println("Got Debounce Event")

		go parseEvent(evt)

	})



	for evtraw := range emt.On("gpioevent") {
		// so the sending is done

		evt := evtraw.Args[0].(gpiod.LineEvent)


		eventChan <- evt
	}

}


func debounceEvent(interval time.Duration, input chan gpiod.LineEvent, cb func(arg gpiod.LineEvent)) {

	var itemhold gpiod.LineEvent
	var lister = make(map[int]gpiod.LineEvent)
	timer := time.NewTimer(interval)

	for {


		select {
		case itemhold = <-input:

			lister[itemhold.Offset]=itemhold

		case <-timer.C:

			for key,item := range lister{

				cb(item)

				delete(lister,key)
			}
			timer.Reset(interval)
		}
	}
}

func parseEvent(evt gpiod.LineEvent) {

	var obj *PinAssociation
	var ok bool

	if obj, ok = pins[evt.Offset]; !ok {

		log.Printf("Pin Data not in configuration! %#v\n", evt)
		return

	}

	if pins[evt.Offset].Active == false {

		return
	}

	var states []string

	switch pins[evt.Offset].Device {
	case "motion":
		states = []string{"inactive", "active"}
		break
        case "motion invert":
                states = []string{"active", "inactive"}
              break
	case "contact":
		states = []string{"open", "closed"}
		break
        case "contact invert":
                states = []string{"closed", "open"}
                break
	default:
		states = []string{"open", "closed"}
	}

	pins[evt.Offset].State = states[0]

	if int(evt.Type) == 1 {

		pins[evt.Offset].State = states[1]

	}

	obj = pins[evt.Offset]

	//pinchan <- pins
	//refresh the webui
	<-emt.Emit("change", 1)

	switch obj.ActionType {
	case "http":
		actionHttp(obj, int(evt.Type))
		break
	case "hubitat":
		actionHubitat(obj, int(evt.Type))
		break
	case "exec":
		actionExec(obj, int(evt.Type))
		break
	default:
		log.Println("Action Type Not Implemented!")
		break
	}

}

func actionHubitat(obj *PinAssociation, state int) {

	//log.Println("Received hubitat action")

	var url string

	dur, perr := time.ParseDuration(viper.GetString("HttpActionTimeout"))

	if perr != nil {

		log.Println("Invalid Duration")
		dur = 15 * time.Second

	}

	rawurl := viper.GetString("Integrations.Hubitat")

	c := &http.Client{

		Timeout: dur,
	}

	if state == 1 {

		log.Printf("Received event on %s (%s): %s", obj.Label, obj.Name, "Closed")

		st := obj.State
		//hack for fixing the naming convention on hubitat
		if obj.State == "closed" {

			st = "close"
		}

		url = fmt.Sprintf(rawurl, obj.OnClose, st)

	} else {

		log.Printf("Received event on %s (%s): %s", obj.Label, obj.Name, "Open")

		url = fmt.Sprintf(rawurl, obj.OnOpen, obj.State)

	}

	if len(url) < 10 {

		log.Println("Invalid Url")
		return
	}

	_, err := c.Get(url)

	// check for response error
	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("Action Success on %s\n", url)

}

func actionHttp(obj *PinAssociation, state int) {

	log.Println("Received http action")

	var url string

	dur, perr := time.ParseDuration(viper.GetString("HttpActionTimeout"))

	if perr != nil {

		log.Println("Invalid Duration")
		dur = 15 * time.Second

	}

	c := &http.Client{

		Timeout: dur,
	}

	if state == 1 {

		log.Printf("Received event on %s (%s): %s", obj.Label, obj.Name, "Closed")

		url = obj.OnClose

	} else {

		log.Printf("Received event on %s (%s): %s", obj.Label, obj.Name, "Open")

		url = obj.OnOpen

	}

	if len(url) < 12 {

		log.Println("Invalid Url")
		return
	}

	_, err := c.Get(url)

	// check for response error
	if err != nil {
		log.Println(err)
		return
	}

	log.Println("Action Success!")

}

func actionExec(obj *PinAssociation, state int) {

	log.Println("Received event action")

	var statetype string
	var cstr string

	if state == 1 {

		cstr = obj.OnClose
		statetype = "closed"

	} else {

		cstr = obj.OnOpen
		statetype = "open"

	}

	log.Printf("Executing %s %s %s %s", cstr, obj.Name, statetype, obj.Label)

	cmd := exec.Command(cstr, obj.Name, statetype, obj.Label)

	out, err := cmd.CombinedOutput()

	if err != nil {
		log.Println("cmd.Run() failed with %s\n", err)
	}

	log.Printf("combined out:\n%s\n", string(out))

	log.Println("Action Success!")

}

func handler(evt gpiod.LineEvent) {
	// handle change in line state
	//New Event gpiod.LineEvent{Offset:16, Timestamp:1601067131395882654, Type:2}
	//events <- evt

	<-emt.Emit("gpioevent", evt)
}

func setupConfig() {

	log.Println("Setting Configuration")

	viper.SetDefault("AppName", "alarmpi")
	viper.SetDefault("HttpActionTimeout", "15s")
	viper.SetDefault("Chip", "gpiochip0")
	viper.SetDefault("Debug", false)
	viper.SetDefault("Port", "8000")
	viper.SetConfigType("json")
	viper.SetConfigName("config")         // name of config file (without extension)
	viper.AddConfigPath("/etc/alarmpi/")  // path to look for the config file in
	viper.AddConfigPath("$HOME/.alarmpi") // call multiple times to add many search paths
	viper.AddConfigPath(".")              // optionally look for config in the working directory
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {

		fmt.Println("Config file changed:", e.Name)

		<-emt.Emit("configchange", 1)
	})

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			panic(err)
		}
	}

}
