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

Using pir motion sensors that have 3 leges, connect in the following manner:

* leg 1 v+ to 5v on the pi
* leg 2 high/low to GPIO pin of choice
* leg 3 gnd to gnd on pi

[This is an example of a product that works](https://www.amazon.com/HiLetgo-Pyroelectric-Sensor-Infrared-Detector/dp/B07RT7MK7C/ref=sr_1_1?dchild=1&keywords=pir+motion+hiletgo&qid=1601258474&sr=8-1)

## Run

To run execute on the command line, the default port is 8000, but can be changed in the configuration

```
./alarmpi
```
 
### Pin Entries

Fill out a entry for each pin in the interface

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




