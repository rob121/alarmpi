# Alarm Pi

A pi based program for monitoring contact inputs from a wired alarm system

## Overview



## Configuration

 
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


* 1Name: gpio16
* State: open|closed
* Label: Front Door





