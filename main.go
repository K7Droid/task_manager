
package main

import (

	//websocket
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"
	"github.com/gorilla/websocket"

	//ram monitor
	"fmt"
	
	//cpu monitor
	"strings"
	
	//graficar
	chart "github.com/wcharczuk/go-chart"
	"bufio"
    "encoding/base64"
)

const (
	writeWait = 2 * time.Second
	pongWait = 14 * time.Second
	pingPeriod = (pongWait * 9) / 10
	filePeriod = 2 * time.Second
)

var (
	addr      = flag.String("addr", ":3000", "http service address")
	homeTempl1 = template.Must(template.New("").Parse(htmlBody))
	homeTempl2 = template.Must(template.New("").Parse(htmlBodyProcesos))
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	opc = 0
	tiempo1 = 0.0
	tiempo2 = 0.0
	contprocesos = 0
	contrunning = 0
	contsleeping = 0
	contstoped = 0
	contzombies = 0
)

var graficaX1 []float64
var graficaY1 []float64

var graficaX2 []float64
var graficaY2 []float64


type procStruct struct{
	pid 	string
	nombre 	string
	usuario string
	estado 	string
	percram string
	hijos 	[]procStruct
}


func dealwithErr(err error) {
	if err != nil {
			fmt.Println(err)
			//os.Exit(-1)
	}
}

func getData(i int, lastMod time.Time) ([]byte, time.Time, error) {
	
	switch i {
	case 1:
		tiempo1 +=  1.0
		contenido := ""
		b, err := ioutil.ReadFile("/proc/meminfo")

		str := string(b)
		listaInfo := strings.Split(string(str),"\n")

		memTotal := strings.Replace((listaInfo[0])[10:24]," ","",-1)
		memLibre := strings.Replace((listaInfo[1])[10:24]," ","",-1)

		ramTotal, err1 := strconv.Atoi(memTotal)
		ramLibre, err2 := strconv.Atoi(memLibre)
		
		if err1 == nil && err2 == nil{
			ramTotal1 := ramTotal / 1024
			ramLibre1 := ramLibre / 1024
			//fmt.Println(strconv.Itoa(ramTotal1)+" - "+strconv.Itoa(ramLibre1))
			contenido = "Memoria Total: " + strconv.Itoa(ramTotal1) + " MB\n"
			contenido = contenido + "Memoria Libre: " + strconv.Itoa(ramLibre1) + " MB\n"
			porcentaje1 := float64(ramLibre1) / float64(ramTotal1) * 100
			contenido = contenido + "Porcentaje de memoria utilizado: " + fmt.Sprintf("%f", porcentaje1) + "%\n"
		
			graficaY1 = append(graficaY1,porcentaje1)
			graficaX1 = append(graficaX1,tiempo1)

			mainSeries := chart.ContinuousSeries{
				Name:    "A test series",
				XValues: graficaX1,
				YValues: graficaY1,
			}

			polyRegSeries := &chart.PolynomialRegressionSeries{
				Degree:      3,
				InnerSeries: mainSeries,
			}
		
			graph := chart.Chart{
				Series: []chart.Series{
					mainSeries,
					polyRegSeries,
				},
			}
		
			f, _ := os.Create("graficaresultante.png")
			defer f.Close()
			graph.Render(chart.PNG, f)
		
			imgFile, err := os.Open("graficaresultante.png") // a QR code image
		
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		
			defer imgFile.Close()
		
			fInfo, _ := imgFile.Stat()
			var size int64 = fInfo.Size()
			buf := make([]byte, size)
		
			fReader := bufio.NewReader(imgFile)
			fReader.Read(buf)
		
			imgBase64Str := base64.StdEncoding.EncodeToString(buf)
			contenido = contenido + imgBase64Str
			return []byte(contenido), lastMod, err
		}else{
			return []byte("Ocurrió\nun\nerror\n"), lastMod, err
		}

	case 2:
		tiempo2 +=  1.0
		var err error
		err = nil
		idle0, total0 := getCPUSample()
		time.Sleep(1 * time.Second)
		idle1, total1 := getCPUSample()

		idleTicks := float64(idle1 - idle0)
		totalTicks := float64(total1 - total0)
		cpuUsage := 100 * (totalTicks - idleTicks) / totalTicks
		

		contenido := "CPU Total: " + fmt.Sprintf("%f",totalTicks) + " bytes\n"
		contenido = contenido + "CPU Ocupado: " + fmt.Sprintf("%f",totalTicks-idleTicks) + " bytes\n"
		contenido = contenido + "Porcentaje del CPU utilizado: " + fmt.Sprintf("%f",cpuUsage) + "%\n"

		graficaY2 = append(graficaY2,cpuUsage)
		graficaX2 = append(graficaX2,tiempo2)

		mainSeries := chart.ContinuousSeries{
			Name:    "A test series 2",
			XValues: graficaX2,
			YValues: graficaY2,
		}

		polyRegSeries := &chart.PolynomialRegressionSeries{
			Degree:      3,
			InnerSeries: mainSeries,
		}
	
		graph := chart.Chart{
			Series: []chart.Series{
				mainSeries,
				polyRegSeries,
			},
		}
	
		f, _ := os.Create("graficaresultante2.png")
		defer f.Close()
		graph.Render(chart.PNG, f)
	
		imgFile, err := os.Open("graficaresultante2.png") // a QR code image
	
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	
		defer imgFile.Close()

		fInfo, _ := imgFile.Stat()
		var size int64 = fInfo.Size()
		buf := make([]byte, size)
	
		fReader := bufio.NewReader(imgFile)
		fReader.Read(buf)
	
		imgBase64Str := base64.StdEncoding.EncodeToString(buf)
		contenido = contenido + imgBase64Str	
		return []byte(contenido), lastMod, err
	
	case 3:
		contenido:=""
		var procesosSt []procStruct
		files, err := ioutil.ReadDir("/proc")
		if err != nil {
			log.Fatal(err)
		}
		contenido = contenido + "Procesos en ejecucion: " + strconv.Itoa(contrunning) + "\n"
		contenido = contenido + "Procesos suspendidos: " + strconv.Itoa(contsleeping) + "\n"
		contenido = contenido + "Procesos detenidos: " + strconv.Itoa(contstoped) + "\n"
		contenido = contenido + "Procesos zombie: " + strconv.Itoa(contzombies) + "\n"
		contenido = contenido + "Procesos en ejecucion: " + strconv.Itoa(contprocesos) + "\n"
		contenido = contenido + "%%"
		contprocesos = 0
		contrunning = 0
		contsleeping = 0
		contstoped = 0
		contzombies = 0
		contenido= contenido + `<table id="tablaprocesos" class="table table-striped">
		<thead class="thead-dark">
			<tr>
				<th scope="col">PID</th>
				<th scope="col">Nombre</th>
				<th scope="col">Usuario</th>
				<th scope="col">Estado</th>
				<th scope="col">% RAM</th>
				<th scope="col"></th>
				<th scope="col">Hijos</th>
			</tr>
		</thead>
		<tbody>`
		for _, f := range files {
			b, err := ioutil.ReadFile("/proc/"+f.Name()+"/status")
			if err == nil {
				contprocesos += 1
				str := string(b)
				listaInfo := strings.Split(string(str),"\n")
				nombre := strings.Replace((listaInfo[0])[5:],"	","",-1)
				usuario1 := strings.Replace((listaInfo[8])[4:9],"	","",-1)
				usuario2,err1 := exec.Command("getent" ,"passwd", usuario1).Output()
				usuario3 := strings.Split(string(usuario2),":")
				usuario :=usuario3[0]
				if err1 != nil {
					//Entrá aqui cuando no encuentra el username de acuerdo al UID
				}
				estado := strings.Replace((listaInfo[2])[6:],"	","",-1)
				if estado == "S (sleeping)" {
					contsleeping += 1
				}else if estado == "R (running)" {
					contrunning += 1
				}else if estado == "I (idle)" {
					contstoped += 1
				}else {
					contzombies += 1
				}

				var nuevoproc = procStruct{f.Name(),nombre,usuario,estado,"",[]procStruct{}}
				procesosSt = append(procesosSt,nuevoproc);			
			}
			
		}
		for _, f := range files {
			b, err := ioutil.ReadFile("/proc/"+f.Name()+"/status")
			if err == nil {
				str := string(b)
				listaInfo := strings.Split(string(str),"\n")
				nombre := strings.Replace((listaInfo[0])[5:],"	","",-1)
				usuario1 := strings.Replace((listaInfo[8])[4:9],"	","",-1)
				padre := strings.Replace((listaInfo[6])[5:],"	","",-1)
				usuario2,err1 := exec.Command("getent" ,"passwd", usuario1).Output()
				usuario3 := strings.Split(string(usuario2),":")
				usuario :=usuario3[0]
				if err1 != nil {
					//Entrá aqui cuando no encuentra el username de acuerdo al UID
				}
				estado := strings.Replace((listaInfo[2])[6:],"	","",-1)
				

				var nuevoproc = procStruct{f.Name(),nombre,usuario,estado,"",[]procStruct{}}
				for i:=0; i < len(procesosSt); i++ {
					if padre == procesosSt[i].pid && padre != "0" {
						procesosSt[i].hijos = append(procesosSt[i].hijos,nuevoproc)
					}
				}
				
				
			}
			
		}
		for j:=0;j<len(procesosSt);j++ {
			//ram := strings.Replace((listaInfo[0])[10:24]," ","",-1)
			contenido= contenido+ `<tr>`
			contenido= contenido+ "<td>" + procesosSt[j].pid + "</td>"		//PID
			contenido= contenido+ "<td>" + procesosSt[j].nombre + "</td>"		//Nombre
			contenido= contenido+ "<td>" + procesosSt[j].usuario + "</td>"		//Usuario
			contenido= contenido+ "<td>" + procesosSt[j].estado + "</td>"		//Estado
			contenido= contenido+ "<td>" + procesosSt[j].percram + "</td>"			//% RAM
			contenido= contenido+ 
			`<td>
			<form method="POST" action="/procesos"> 
			<input name="idproceso" type="hidden" id="idproceso" value="`+ procesosSt[j].pid +`"/>
			<input type="submit" value="KILL"/>
			</form>
			</td>`
			contenido = contenido+ "<td>"
			if len(procesosSt[j].hijos) > 0{
				contenido= contenido + `<table id="tablahijos" class="table table-striped">
				<thead class="thead-dark">
					<tr>
						<th scope="col">PID</th>
						<th scope="col">Nombre</th>
					</tr>
				</thead>
				<tbody>`
				for k:=0;k<len(procesosSt[j].hijos);k++ {
					contenido= contenido+ `<tr>`
					contenido= contenido+ "<td>" + procesosSt[j].hijos[k].pid + "</td>"		//PID
					contenido= contenido+ "<td>" + procesosSt[j].hijos[k].nombre + "</td>"		//Nombre
					contenido= contenido+ `<tr>`
				}
				contenido= contenido+ "</tbody></table>"
			}
			contenido = contenido+ "</td>"
			contenido = contenido+ "</tr>"
		}
		contenido= contenido+ "</tbody></table>"
		return []byte(contenido), lastMod, err
	default:
		var err error
		err = nil
		contenido := "Ocurrió\nun\nerror\n"
		return []byte(contenido), lastMod, err

	}

}

func getCPUSample() (idle, total uint64) {
    contents, err := ioutil.ReadFile("/proc/stat")
    if err != nil {
        return
    }
    lines := strings.Split(string(contents), "\n")
    for _, line := range(lines) {
        fields := strings.Fields(line)
        if fields[0] == "cpu" {
            numFields := len(fields)
            for i := 1; i < numFields; i++ {
                val, err := strconv.ParseUint(fields[i], 10, 64)
                if err != nil {
                    fmt.Println("Error: ", i, fields[i], err)
                }
                total += val
                if i == 4 {
                    idle = val
                }
            }
            return
        }
    }
    return
}

func getInfoCPU(lastMod2 time.Time) ([]byte, time.Time) {
	idle0, total0 := getCPUSample()
    time.Sleep(3 * time.Second)
    idle1, total1 := getCPUSample()

    idleTicks := float64(idle1 - idle0)
    totalTicks := float64(total1 - total0)
	cpuUsage := 100 * (totalTicks - idleTicks) / totalTicks
	

	contenido2 := "CPU Total: " + fmt.Sprintf("%f",totalTicks) + " bytes\n"
	contenido2 = contenido2 + "CPU Ocupado: " + fmt.Sprintf("%f",totalTicks-idleTicks) + " bytes\n"
	contenido2 = contenido2 + "Porcentaje del CPU utilizado: " + fmt.Sprintf("%f",cpuUsage) + "%\n"
	return []byte(contenido2), lastMod2
}

func reader(ws *websocket.Conn) {
	defer ws.Close()
	ws.SetReadLimit(512)
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}
}

func writer(ws *websocket.Conn, lastMod time.Time) {
	lastError := ""
	pingTicker := time.NewTicker(pingPeriod)
	fileTicker := time.NewTicker(filePeriod)
	defer func() {
		pingTicker.Stop()
		fileTicker.Stop()
		ws.Close()
	}()
	for {
		select {
		case <-fileTicker.C:
			var p []byte
			var err error

			p, lastMod, err = getData(opc,lastMod)
			if err != nil {
				if s := err.Error(); s != lastError {
					lastError = s
					p = []byte(lastError)
				}
			} else {
				lastError = ""
			}

			if p != nil {
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				if err := ws.WriteMessage(websocket.TextMessage, p); err != nil {
					return
				}
			}
		case <-pingTicker.C:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}

	var lastMod time.Time
	if n, err := strconv.ParseInt(r.FormValue("lastMod"), 16, 64); err == nil {
		lastMod = time.Unix(0, n)
	}

	go writer(ws, lastMod)
	reader(ws)
}


func BytesToString(data []byte) string {
	return string(data[:])
}

func index(w http.ResponseWriter, r *http.Request) {    
    file, err := os.Open("index.html")
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()
    b, err := ioutil.ReadAll(file)
    fmt.Fprintf(w, BytesToString(b))
}

func ramm(w http.ResponseWriter, r *http.Request) {
	opc = 1
	if r.URL.Path != "/rammonitor" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	p, lastMod, err := getData(1,time.Time{})
	if err != nil {
		p = []byte(err.Error())
		lastMod = time.Unix(0, 0)
	}
	var v = struct {
		Host    	string
		Data    	string
		LastMod 	string
	}{
		r.Host,
		string(p),
		strconv.FormatInt(lastMod.UnixNano(), 16),
	}
	homeTempl1.Execute(w, &v)
}

func cpum(w http.ResponseWriter, r *http.Request) {
	opc = 2
	if r.URL.Path != "/cpumonitor" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	p, lastMod, err := getData(2,time.Time{})
	if err != nil {
		p = []byte(err.Error())
		lastMod = time.Unix(0, 0)
	}
	var v = struct {
		Host    	string
		Data    	string
		LastMod 	string
	}{
		r.Host,
		string(p),
		strconv.FormatInt(lastMod.UnixNano(), 16),
	}
	homeTempl1.Execute(w, &v)
}

func procesos(w http.ResponseWriter, r *http.Request) {
	opc = 3
	if r.URL.Path != "/procesos" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
            fmt.Fprintf(w, "ParseForm() err: %v", err)
            return
		}
		idproceso := r.FormValue("idproceso")
		if idproceso !="" {
			corrercomando := exec.Command("kill","-15",idproceso).Run()
			if corrercomando != nil{
			fmt.Fprintf(w, "Error al eliminar proceso!")
			return
			}
			fmt.Println("Se elimino el proceso: "+ idproceso);
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	p, lastMod, err := getData(3,time.Time{})
	if err != nil {
		p = []byte(err.Error())
		lastMod = time.Unix(0, 0)
	}
	var v = struct {
		Host    	string
		Data    	string
		LastMod 	string
	}{
		r.Host,
		string(p),
		strconv.FormatInt(lastMod.UnixNano(), 16),
	}
	homeTempl2.Execute(w, &v)
}


func main() {
	http.HandleFunc("/",index);
	http.HandleFunc("/procesos",procesos);
	http.HandleFunc("/rammonitor", ramm)
	http.HandleFunc("/cpumonitor", cpum)
	http.HandleFunc("/ws", serveWs)

	fmt.Printf("Corriendo correctamente el proyecto en el puerto 3000...\n")
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal(err)
	}
}

const htmlBody = `<!DOCTYPE html>
<html lang="en">
	<head>
		<meta charset=”UTF-8”>
		<title>SO1 P1 WEB</title>
		<link rel="stylesheet" href="https://stackpath.bootstrapcdn.com/bootstrap/4.4.1/css/bootstrap.min.css" integrity="sha384-Vkoo8x4CGsO3+Hhxv8T/Q5PaXtkKtu6ug5TOeNV6gBiFeWPGFN9MuhOf23Q9Ifjh" crossorigin="anonymous">
		<script src="https://code.jquery.com/jquery-3.4.1.slim.min.js" integrity="sha384-J6qa4849blE2+poT4WnyKhv5vZF5SrPo0iEjwBvKU7imGFAV0wwj1yYfoRSJoZ+n" crossorigin="anonymous"></script>
		<script src="https://stackpath.bootstrapcdn.com/bootstrap/4.4.1/js/bootstrap.min.js" integrity="sha384-wfSDF2E50Y2D1uUdj0O3uMBJnjuUD4Ih7YwaYd1iqfktj0Uod8GCExl3Og8ifwB6" crossorigin="anonymous"></script>
		<script src="https://cdn.jsdelivr.net/npm/popper.js@1.16.0/dist/umd/popper.min.js" integrity="sha384-Q6E9RHvbIyZFJoft+2mJbHaEWldlvI9IOYy5n3zV9zzTtmI3UksdQRVvoxMfooAo" crossorigin="anonymous"></script>
    </head>
	<body>
		<nav class="navbar navbar-expand-lg navbar-dark bg-primary">
			<a class="navbar-brand" href="#">P1 Processes Monitor</a>
			<button class="navbar-toggler" type="button" data-toggle="collapse" data-target="#navbarSupportedContent" aria-controls="navbarSupportedContent" aria-expanded="false" aria-label="Toggle navigation">
				<span class="navbar-toggler-icon"></span>
			</button>
			<div class="collapse navbar-collapse" id="navbarSupportedContent">
				<ul class="navbar-nav mr-auto">
					<li class="nav-item active">
						<a class="nav-link" href="/">Home <span class="sr-only">(current)</span></a>
					</li>
					<li class="nav-item">
						<a class="nav-link" href="/procesos">Processes</a>
					</li>
					<li class="nav-item">
						<a class="nav-link" href="/cpumonitor">CPU Monitor</a>
					</li>
					<li class="nav-item">
						<a class="nav-link" href="/rammonitor">RAM Monitor</a>
					</li>
				</ul>
			</div>
		</nav>
		<pre id="fileData"></pre>
		<img id="img1">
        <script type="text/javascript">
            (function() {
				var data = document.getElementById("fileData");
				var img11 = document.getElementById('img1');

                var conn = new WebSocket("ws://{{.Host}}/ws?lastMod={{.LastMod}}");
                conn.onclose = function(evt) {
                    data.textContent = 'Connection closed';
                }
                conn.onmessage = function(evt) {
					console.log('file updated');
					var contenido = evt.data.split("\n");
					data.textContent = contenido[0] + "\n" + contenido[1]+ "\n"+ contenido[2]+ "\n";
					img11.src = "data:image/png;base64," + contenido[3]; 
                }
            })();
		</script>
		
    </body>
</html>
`

const htmlBodyProcesos = `<!DOCTYPE html>
<html lang="en">
	<head>
		<meta charset=”UTF-8”>
		<title>SO1 P1 WEB</title>
		<link rel="stylesheet" href="https://stackpath.bootstrapcdn.com/bootstrap/4.4.1/css/bootstrap.min.css" integrity="sha384-Vkoo8x4CGsO3+Hhxv8T/Q5PaXtkKtu6ug5TOeNV6gBiFeWPGFN9MuhOf23Q9Ifjh" crossorigin="anonymous">
		<script src="https://code.jquery.com/jquery-3.4.1.slim.min.js" integrity="sha384-J6qa4849blE2+poT4WnyKhv5vZF5SrPo0iEjwBvKU7imGFAV0wwj1yYfoRSJoZ+n" crossorigin="anonymous"></script>
		<script src="https://stackpath.bootstrapcdn.com/bootstrap/4.4.1/js/bootstrap.min.js" integrity="sha384-wfSDF2E50Y2D1uUdj0O3uMBJnjuUD4Ih7YwaYd1iqfktj0Uod8GCExl3Og8ifwB6" crossorigin="anonymous"></script>
		<script src="https://cdn.jsdelivr.net/npm/popper.js@1.16.0/dist/umd/popper.min.js" integrity="sha384-Q6E9RHvbIyZFJoft+2mJbHaEWldlvI9IOYy5n3zV9zzTtmI3UksdQRVvoxMfooAo" crossorigin="anonymous"></script>
    </head>
	<body>
		<nav class="navbar navbar-expand-lg navbar-dark bg-primary">
			<a class="navbar-brand" href="#">P1 Processes Monitor</a>
			<button class="navbar-toggler" type="button" data-toggle="collapse" data-target="#navbarSupportedContent" aria-controls="navbarSupportedContent" aria-expanded="false" aria-label="Toggle navigation">
				<span class="navbar-toggler-icon"></span>
			</button>
			<div class="collapse navbar-collapse" id="navbarSupportedContent">
				<ul class="navbar-nav mr-auto">
					<li class="nav-item active">
						<a class="nav-link" href="/">Home <span class="sr-only">(current)</span></a>
					</li>
					<li class="nav-item">
						<a class="nav-link" href="/procesos">Processes</a>
					</li>
					<li class="nav-item">
						<a class="nav-link" href="/cpumonitor">CPU Monitor</a>
					</li>
					<li class="nav-item">
						<a class="nav-link" href="/rammonitor">RAM Monitor</a>
					</li>
				</ul>
			</div>
		</nav>
		<pre id="fileData"></pre>
		<div id="espaciotabla"></div>
        <script type="text/javascript">
            (function() {
				var data2 = document.getElementById("fileData");
				var table = document.getElementById("espaciotabla");
				
                var conn = new WebSocket("ws://{{.Host}}/ws?lastMod={{.LastMod}}");
                conn.onclose = function(evt) {
					data2.textContent = 'Connection closed';
                }
                conn.onmessage = function(evt) {
                    console.log('file updated');
					//data.textContent = evt.data;
					var contenido = evt.data.split("%%");
					data2.textContent = contenido[0];
					var lineas = contenido[1];
					table.innerHTML=contenido[1];
					
                }
            })();
		</script>
		
    </body>
</html>
`
