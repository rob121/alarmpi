# Alarm Pi

A pi based program for monitoring contact inputs from a wired alarm system and sending those events to a home automation platform

## Features

* Watches for contact closure on a configured gpio pin
* Register a callback type on open/close
* Web ui to see status
* Web api request to see status (GET /status)

![Image of Home](alarmpi_home.png?raw=true)


## Configuration

See config example.   The config.json may be placed in /etc/alarmpi, .alarmpi in your $HOME directory, or in the current working directory

### Extra settings

In extreme cases you may need to set the following, the default values are below

```
   "AppName": "alarmpi",
   "HttpActionTimeout": "15s",
   "Chip": "gpiochip0"
```

## Install

Copy binary to a suitable location, service file included for linux/raspi


### Attaching Sensors to you pi

#### Contact sensors (Doors/Windows)

Attach one leg of the sensor to 3.3v and the other to a GPIO PIN (Care should be taken here, if your pins are in some weird state and not reading input you can damage things)

#### Motion sensors

Coming soon

## Run

To run execute on the command line, the default port is 8000, but can be changed in the configuration

```
./alarmpi
```
 
### Pin Entries

The configuration has a top level "Pins" each top level contains a association to a GPIO pin and within each block  you have the following labels

```
	"Pins":{
		"GPIO16":{
    		"Label": "Contact 1",
    		"Type": "http",
			"Open": "http://example.com/on",
			"Closed": "example.com/off"
		},
```

#### Label

The label to be presented i.e Hallway, Front Door, etc.

#### Type

The type of event to trigger on, options are "http" for making a HTTP GET request or "exec" for executing a local program

#### Http 

Http will execute the url provided in the "Open" and "Close fields"

#### Exec

Exec will execute a script named in "Open" or Closed and pass in the following arguments


* Name: gpio16
* State: open|closed
* Label: Front Door

It will essentially execute as if you were running the following

```
./callback_script.sh gpio16 open "Front Door"
```




