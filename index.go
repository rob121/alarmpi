package main 



var Index =`

<!DOCTYPE html>
<html lang="en">

<head>

  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
  <meta name="description" content="">
  <meta name="author" content="">
  <title>Alarm Pi ON {{.Url}}</title>
  <!-- Bootstrap core CSS -->
  <link href="http://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/4.5.2/css/bootstrap.min.css" rel="stylesheet">
  <script>  
  
  var ws;
  var tm;
  var wsstate;
  function print(message) {
        console.log(message)
  }
  
  
  function ping() {
        if(ws){
        ws.send('__ping__');
        }
        tm = setTimeout(function () {

           wsstate="closed"
           ws=null;
      
           console.log("Ping closing socket");
           setupWs();     


        }, 5000);
  }

  function pong() {
    clearTimeout(tm);
  } 
  
  function setupWs(){ 
      
        if (ws) {
            return false;
        }
    
        wsstate = "open"
    
        ws = new WebSocket("{{.Ws}}");
        
        ws.onopen = function(evt) {
            print("Socket Open");
            wsstate = "open"
            ping();
            setInterval(ping, 30000);
        }
        
        ws.onclose = function(evt) {
            print("Socket Close");
            wsstate = "closed"
            ws = null;

        }
        ws.onmessage = function(evt) {
            
            if(evt.data=="__pong__"){ 
                
                   console.log("Received: "+evt.data)
                   pong();
                   return;  
                
            }
            
            
            jsobj = jQuery.parseJSON(evt.data)
            
            $("#state").html("");
            
       
            
            jQuery.each(jsobj,function(i,val){
                
            
                

                tr = "<tr>";
                tr += "<td>"+val.Label+"</td>";
                tr += "<td>"+val.Name+"</td>";
                tr += "<td>"+val.ActionType+"</td>";
                tr += "<td><div class='label'>"+val.State+"</div><div class='indicator "+val.State+"' >&nbsp;</div></td>";
                tr += "</tr>";
                
                $("#state").append(tr)
                

            });
            
           
        }
        ws.onerror = function(evt) {
            wsstate = "closed"
            ws = null;
            print("ERROR: " + evt.data);
        }
        return false;
        
  }
  
window.addEventListener("load", function(evt) {
  
    setupWs();
       
});


</script>
<style>

 .label { 
     height:30px;
     width:100px;
     display:inline-block;
 }
 
 .label::first-letter {
 
   text-transform: uppercase;
 }
 
 
 .indicator { 
    
     display:inline-block;
     height:20px;
     width:20px;
     border-radius:25px; 
     margin:5px;
     
 }

 .open { 
     
     border:5px solid #006699;
     background-color: #FFFFFF;
     
 }
 .closed { 
     border:5px solid #006699;
     background-color: #006699;
 }
</style>
</head>
<body>

  <!-- Navigation -->
  <nav class="navbar navbar-expand-lg navbar-dark bg-dark static-top">
    <div class="container">
      <a class="navbar-brand" href="#">Alarm Pi</a>
      <button class="navbar-toggler" type="button" data-toggle="collapse" data-target="#navbarResponsive" aria-controls="navbarResponsive" aria-expanded="false" aria-label="Toggle navigation">
        <span class="navbar-toggler-icon"></span>
      </button>
      <div class="collapse navbar-collapse" id="navbarResponsive">
        <ul class="navbar-nav ml-auto">
          <li class="nav-item">
            <a class="nav-link" href="#">Github</a>
          </li>
        </ul>
      </div>
    </div>
  </nav>

  <!-- Page Content -->
  <div class="container">
    <div class="row">
      <div class="col-lg-12 text-center">
        <h2 class="mt-5">Sensor Status</h2>
        <div class="table-responsive">
                <table class="table table-striped">
                 <thead class="thead-dark">
                 <tr>
                 <th scope="col" >
                  Label
                  </th>
                  <th>
                  Pin
                  </th>
                  <th>
                  Type
                  </th>
                  <th>
                  Status
                  </th>
                 </tr>
                 </thead> 
                 <tbody id="state">
                 
                {{range .Pins}}
                   <tr>
                    <td>
                     {{ .Label }}
                    </td>
                    
                    <td>
                     {{ .Name }}
                    </td>
                    
                    <td>
                     {{ .ActionType }}
                    </td>
                    
                    <td>
                      <div class="label">{{ .State }}</div><div class='indicator {{ .State }}'>&nbsp;</div>
                    </td>
                    
                   </tr>
                 {{end}}
            </tbody>
   </table>
   </div>
      </div>
    </div>
  </div>

  <!-- Bootstrap core JavaScript -->
  <script src="http://cdnjs.cloudflare.com/ajax/libs/jquery/3.5.1/jquery.min.js"></script>
  <script src="http://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/4.5.2/js/bootstrap.bundle.min.js"></script>

</body>

</html>

`

