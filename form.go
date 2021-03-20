package main 

import (
     "strconv"
    "github.com/spf13/viper"
)

type Attributes struct {
    Devices []string
    Pins []string
    Types []string
}


func loadAttributes() Attributes {
    
    var defpins []string
    
    for i:=1;i<=viper.GetInt("GpioPinCt");i++ {
        
        
        defpins = append(defpins,"GPIO"+strconv.Itoa(i))
        
        
    }
    
    return Attributes{Devices: []string{"contact","contact invert","motion","motion invert"},Pins: defpins,Types: []string{"http","hubitat","exec"}}
    
    
}
