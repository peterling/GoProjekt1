package main

import (
    "fmt"
    "io"
	"io/ioutil"
    "log"
    "os"
	"strconv"
    "os/exec"
	//"os/signal"
	"bufio"
//	"syscall"
	"strings"
	//"path/filepath"
	"encoding/xml"
	"time"
	"sync"
	"runtime"
	"crypto/sha1"
	"net/http"
	"encoding/hex"
	"html/template"
//	"bytes"
)

type xmlConfig struct {
	XMLName xml.Name `xml:"applications"`
	ProgrammNamenListe		[]string `xml:"application>name"`
	ProgrammStartListe		[]string `xml:"application>start"`
	ProgrammStopListe		[]string `xml:"application>stop"`
	ProgrammRestartListe	[]bool `xml:"application>restart"`
	ProgrammExitListe		[]string `xml:"application>exit"`
}
type dummy struct {		//for template execution
	Wait int
}

type process struct {
	Handle		*exec.Cmd
	Name		string
	StopCmd		string
	StartCmd	string
	Restart		bool
	Alive		bool
	StartCount	int
	ExitCmd		string
	StdInPipe	io.WriteCloser
	StdOutPipe	io.ReadCloser
	LogBuffer	[]string
	buRio		*bufio.Reader
}

var runningProcs = make([] process,0)		//all ever spawned processes in a struct
var runningProcsNew = make([] process,0)
var	v = xmlConfig{}							//the read-in configuration struct

func restartProc(r int){
	//mutExRunningProcs.Lock()
		var befehlKomplett string = runningProcs[r].StartCmd
			befehlSplit :=strings.Split(befehlKomplett," ")	
			cmd := exec.Command(befehlSplit[0], befehlSplit[1:]...)
			stdinPipe,_:=cmd.StdinPipe()
			stdoutPipe,_ :=cmd.StdoutPipe()
			//logBuffer := new(bytes.Buffer)
			runningProcs = append(runningProcs, process{cmd,
										runningProcs[r].Name,
										runningProcs[r].StopCmd,
										runningProcs[r].StartCmd,
										//true})
										runningProcs[r].Restart,
										true,
										runningProcs[r].StartCount+1,
										runningProcs[r].ExitCmd,
										stdinPipe,
										stdoutPipe,
										runningProcs[r].LogBuffer,
										nil})		//Restart should be true
			runningProcs[r].Restart=false			//Beim bisherigen Eintrag Restart deaktivieren
//mutExRunningProcs.Unlock()
//runtime.Gosched()

//			logFile:=openLogFile(runningProcs[r].Name)
//			defer logFile.Close()	//sinnvoll?
			
//			cmd.Stdout=logFile
			cmd.Run()
}
func Download(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "config.xml")
	}

const backTemplate = `
<!DOCTYPE html>
<html>
<head>
{{if .Wait}}<meta http-equiv="refresh" content="{{.Wait}}; url=./" />{{end}}
</head>
<body>

<button onclick="goBack()">Zurück</button>
{{if not .Wait}}<br><br>{{end}}
{{if .Wait}}<p>Zurück in {{.Wait}} Sekunden...</p>{{end}}

<script>
function goBack() {
	window.location.href = "/";
}
</script>

</body>
</html>
`
	const uebersichtTemplate = `
	
<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>{{.Titel}}</title>
	</head>
	<body>
	<input type="button" value="Seite aktualisieren" onClick="window.location.reload()">
	<input type="button" value="XML-Datei herunterladen" onClick="window.location.href='/download'">
	<a href="/download">XML-Datei (Rechtsklick zum Speichern)</a>
	<h1>Programme starten</h1>
	
		{{range $index, $results := .Programme}}<a href="/proccontrol?program={{$index}}&aktion=start&hashprog={{$.ProgrammHash}}">{{.}}</a><br>{{else}}<div><strong>keine Programme hinterlegt</strong></div>{{end}}
	<h1>Laufende Prozesse hart beenden (SIGKILL)</h1>	
		{{range $index, $results := .Prozesse}}{{if .Alive}}<a href="/proccontrol?program={{$index}}&aktion=kill&hashproc={{$.ProzessHash}}">{{.Name}}, Autostart: {{.Restart}}, läuft: {{.Alive}}, {{.StartCount}} mal gestartet</a><br>{{end}}{{else}}<div><strong>keine überwachten Prozesse</strong></div>{{end}}
	<h1>Laufende Prozesse weich beenden (SIGTERM)</h1>	
		{{range $index, $results := .Prozesse}}{{if .Alive}}<a href="/proccontrol?program={{$index}}&aktion=term&hashproc={{$.ProzessHash}}">{{.Name}}, Autostart: {{.Restart}}, läuft: {{.Alive}}, {{.StartCount}} mal gestartet</a><br>{{end}}{{else}}<div><strong>keine überwachten Prozesse</strong></div>{{end}}
	<h1>Laufende Prozesse mit hinterlegtem Exit-Befehl an STDIN beenden</h1>	
		{{range $index, $results := .Prozesse}}{{if .Alive}}<a href="/proccontrol?program={{$index}}&aktion=exit&hashproc={{$.ProzessHash}}">{{.Name}}, Autostart: {{.Restart}}, läuft: {{.Alive}}, {{.StartCount}} mal gestartet</a><br>{{end}}{{else}}<div><strong>keine überwachten Prozesse</strong></div>{{end}}
	<h1>Laufende Prozesse mit hinterlegtem STOP-Befehl beenden</h1>	
		{{range $index, $results := .Prozesse}}{{if .Alive}}<a href="/proccontrol?program={{$index}}&aktion=stop&hashproc={{$.ProzessHash}}">{{.Name}}, Autostart: {{.Restart}}, läuft: {{.Alive}}, {{.StartCount}} mal gestartet</a><br>{{end}}{{else}}<div><strong>keine überwachten Prozesse</strong></div>{{end}}
	<h1>Restart-Option laufender Prozesse (de-)aktivieren</h1>
		{{range $index, $results := .Prozesse}}{{if .Alive}}{{if .Restart}}<b>{{end}}<a href="/proccontrol?program={{$index}}&aktion=autostart&hashproc={{$.ProzessHash}}">{{.Name}}, Autostart: {{.Restart}}, läuft: {{.Alive}}, {{.StartCount}} mal gestartet</a><br>{{if .Restart}}</b>{{end}}{{end}}{{else}}<div><strong>keine überwachten Prozesse</strong></div>{{end}}
	<h1>Restart-Option beendeter Prozesse (de-)aktivieren [revive/dismiss]</h1>
		{{range $index, $results := .Prozesse}}{{if not .Alive}}{{if .Restart}}<b>{{end}}<a href="/proccontrol?program={{$index}}&aktion=autostart&hashproc={{$.ProzessHash}}">{{.Name}}, Autostart: {{.Restart}}, läuft: {{.Alive}}, {{.StartCount}} mal gestartet</a><br>{{if .Restart}}</b>{{end}}{{end}}{{else}}<div><strong>keine überwachten Prozesse</strong></div>{{end}}
	<h1>Logging</h1>
		{{range $index, $results := .Programme}}<a href="/proccontrol?program={{$index}}&aktion=log&hashprog={{$.ProgrammHash}}">{{.}}</a><br>{{else}}<div><strong>keine Programme hinterlegt</strong></div>{{end}}

	</body>
</html>`
	//	{{range $index, $results := .Programme}}<a href="/proccontrol?program={{$index}}&aktion=start&hashprog=">{{.}}</a><br>{{end}}{{.StartingLink}}
	//		{{range $index, $results := .Programme}}<a class="postlink" href="/proccontrol?program={{$index}}&aktion=start&hashprog={{$.ProgrammHash}}" onClick="window.location.reload();return false;">{{.}}</a><br>{{else}}<div><strong>keine Programme hinterlegt</strong></div>{{end}}

	//		{{$p := .ProzessHash}}{{range $index, $results := .Programme}}<a href="/proccontrol?program={{$index}}&aktion=start&hashprog={{$p}}">{{.}}</a><br>{{end}}{{.StartingLink}}
//HEAD <meta http-equiv="refresh" content="3" />
func ObserverHandler(w http.ResponseWriter, r *http.Request) {
	 t:= template.Must(template.New("control").Parse(uebersichtTemplate)) // Create a template.

	data := struct {
		Titel string
		Programme []string
		Prozesse []process
		ProgrammHash string
		ProzessHash string
	}{
		Titel: "Observer",
		Programme: v.ProgrammNamenListe,
		Prozesse: runningProcs ,
		ProgrammHash: hashOfProgrammListe(),
		ProzessHash: hashOfRunningProcs(),
		/*[]string{
			"Programm1",
			"Programm2",
		},*/
	}

	t.Execute(w,data)
}

func checkForRestart(){
	mutExRunningProcs.Lock()
	for r:=range runningProcs{		//für alle einträge im restartslice
		if	runningProcs[r].Restart==true && runningProcs[r].Alive==false{//wenn restart-switch on für appl, nur dann...	
			go restartProc(r)		//ohne GOROUTINE hängt ?!! wegen run/start in restartProc! sonst nicht überwacht wenn start statt run!
		}
	}
	mutExRunningProcs.Unlock()
	runtime.Gosched()
}

func killingProcessHardUnix(pid int) bool {
    out, err := exec.Command("kill", "-9", strconv.Itoa(pid)).CombinedOutput()
    if err != nil {
        log.Println(err)
    }

    if string(out) == "" {
        return true // pid exist
    }
    return false
}
/*				//ANSATZ ÜBER PID... verworfen... evtl. wieder aufgreifen wenn MAP statt SLICE
func killingProcessSoftly(pid int){
	if processRunning(pid){
	p,_:=	os.FindProcess(pid)
	p.Signal(os.Interrupt)
	}
}*/

func openLogFile(progra string)*os.File{
	// open the out file for writing
	// logFile, err := os.Create("./logProcSlice"+progra+".txt")
	logFile, err := os.OpenFile("./log_"+progra+".txt",os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666 ) //WRITE-ONLY
	//logFile, err := os.OpenFile("./log_"+progra+".txt",os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777 )
    if err != nil {
        panic(err)
    }
	logFile.WriteString("LogFile created")

	return logFile
}
//var b bytes.Buffer
//var gfdsf = make([]string,0)
func programmStart(programmNr int){
	var befehlKomplett string = v.ProgrammStartListe[programmNr]	
	befehlSplit :=strings.Split(befehlKomplett," ")	
	cmd := exec.Command(befehlSplit[0], befehlSplit[1:]...)
	stdinPipe,_:=cmd.StdinPipe()
//	stdoutPipe, _ := cmd.StdoutPipe()
	logBuffer := make([]string,0)

	//wroider := bufio.NewScanner(stdout).Scan()
	stdoutPipe,_:=cmd.StdoutPipe()
		reader := bufio.NewReader(stdoutPipe)
//	scanner := bufio.NewScanner(stdoutPipe)
//	bufio.NewWriter().ReadFrom(stdoutPipe)
	//    checkError(err)
//	logFile:=openLogFile(v.ProgrammNamenListe[programmNr])
	//io.Pipe()
//	asd,_ := ioutil.ReadAll(stdout)
//	gfdsf = append(gfdsf,string(asd))
//	time.Sleep(2 * time.Second)
//	fmt.Println(gfdsf)
	//ioutil.ReadAll()
	//    stderr, err := cmd.StderrPipe()
	//    checkError(err)
//		go io.Copy(logFile, stdout)
//	go io.Copy(logBuffer,stdoutPipe)	
	//   go io.Copy(os.Stderr, stderr)
	mutExRunningProcs.Lock()
	runningProcs = append(runningProcs, process{cmd,
					v.ProgrammNamenListe[programmNr],
					v.ProgrammStopListe[programmNr],
					befehlKomplett,
					v.ProgrammRestartListe[programmNr],
					true,
					1,
					v.ProgrammExitListe[programmNr],
					stdinPipe,
					stdoutPipe,
					logBuffer,
					reader})
mutExRunningProcs.Unlock()
runtime.Gosched()
//	logFile:=openLogFile(v.ProgrammNamenListe[programmNr])
	//cmd.Stdout=wroider
//	defer logFile.Close()
//	cmd.Stdout=logFile
	cmd.Run()
	
	//defer logFile.Close()	//sinnvoll?

//	cmd.Run()
	//	cmd.Start() //??besser?? weil bei start kein processstate abrufbar ?!
	fmt.Printf(v.ProgrammNamenListe[programmNr]+" wurde gestartet, ") //auf diese ausgabe ist kein verlass (kein real time, erst nach beendigung der funktion)
	fmt.Printf("PID %d\n",cmd.Process.Pid)
//	cmd.Stdout.Write([]byte("\r\n"+time.Now().Format(time.RFC3339)+": INFO[Instanz gestartet]"))	//CR for friends of Micro$oft Editors
//	logFile.Sync()		//at this point ?
	/*
	writer := bufio.NewWriter(logFile)
	 defer writer.Flush()
	stdoutPipe, err := cmd.StdoutPipe()
	 if err != nil {
	     panic(err)
	}
	*/
}

var mutExRunningProcs = &sync.RWMutex{}
//var mutexStopperProcSlice = &sync.Mutex{}
const configPath string = "config.xml"
const programmAuswahl int = 1	//wählen, welches Programm gestartet werden soll - IRL vom HTTPHandler
const runningProcsLengthTreshold int = 10	//maybe 10000
const runningProcsLengthInterval int = 5	//maybe 1000

func runningProcsLengthCheck(){
	fmt.Println("Lengthcheck started!")
	mutExRunningProcs.Lock()
	if len(runningProcs) > runningProcsLengthTreshold{
	mutExRunningProcsReorged.Lock()
	runningProcsReorged = time.Now()
	mutExRunningProcsReorged.Unlock()
		runningProcsNew = nil
		for r:=range runningProcs{
			if runningProcs[r].Alive == true || runningProcs[r].Restart == true {
				runningProcsNew = append(runningProcsNew, runningProcs[r])
			}
		}
		//copy (runningProcsNew, runningProcs)
		runningProcs = nil
		for r:=range runningProcsNew{
			runningProcs = append(runningProcs, runningProcsNew[r])
		}
		fmt.Println(runningProcsNew)
	}
	mutExRunningProcs.Unlock()
	runtime.Gosched()
}
func xmlReadIn(){			//XML-Datei wird eingelesen
mutExProgrammListeReorged.Lock()
programmListeReorged = time.Now()
mutExProgrammListeReorged.Unlock()
runtime.Gosched()
hashOfProgrammListe()
	xmlContent, _ := ioutil.ReadFile(configPath)
	//err:= xml.Unmarshal(nil, &v)		//bei Neu-Einlesen soll nicht appendet werden
	v=xmlConfig{}	//leeren, damit nicht appendet wird bei Neu-Einlesen
	err := xml.Unmarshal(xmlContent, &v)
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}
}

func watchFile() error {
    initialStat, err := os.Stat(configPath)
    if err != nil {
        return err
    }

    for {
		fmt.Println("Datei geprüft!")
        stat, err := os.Stat(configPath)
        if err != nil {
            return err
        }

        if stat.Size() != initialStat.Size() || stat.ModTime() != initialStat.ModTime() {
        xmlReadIn()
		log.Println("Datei neu eingelesen, da verändert")
		initialStat, err = os.Stat(configPath)
        }
        time.Sleep(2 * time.Second)
    }
    return nil
}
var runningProcsReorged time.Time = time.Now()
var programmListeReorged time.Time = time.Now()
var mutExRunningProcsReorged = &sync.RWMutex{}
var mutExProgrammListeReorged = &sync.RWMutex{}

func hashOfProgrammListe()string{		//Programmliste könnte reorganisiert worden sein -> index verschoben. Daher beim Ausführen über HTTP Hash abgleichen...
	h := sha1.New()				//falls anderer Hash --> Anfrage verwerfen, Seite neu generieren
	mutExProgrammListeReorged.RLock()

		h.Write([]byte(programmListeReorged.String()))

	mutExProgrammListeReorged.RUnlock()
	runtime.Gosched()
	    bs := h.Sum(nil)
fmt.Printf("ProgrammListe: %x\n", bs)
return hex.EncodeToString(bs)
}

func hashOfRunningProcs()string{		//runningProcsListe könnte reorganisiert worden sein -> index verschoben. Daher beim Ausführen über HTTP Hash abgleichen...
	h := sha1.New()				//falls anderer Hash --> Anfrage verwerfen, Seite neu generieren
	mutExRunningProcsReorged.RLock()

		h.Write([]byte(runningProcsReorged.String()))

	mutExRunningProcsReorged.RUnlock()
	runtime.Gosched()
	    bs := h.Sum(nil)

fmt.Printf("RunningProcs: %x\n", bs)
return hex.EncodeToString(bs)
}

func main() {
	//Programminitialisierung
	xmlReadIn()			//XML-Datei einlesen lassen
	go helperRoutinesStarter()		//runs in the background for important tasks
	//go programFlow()	//Aufrufe in Goroutine starten, später HTTP, jetzt hardcoded
webServer()				//Webserver, you have control!
	//hier adden!

	 fmt.Scanln()		//Programm weiterlaufen lassen ohne Ende //endpoint
}




func ProcControl(w http.ResponseWriter, r *http.Request) {
    r.ParseForm()  // parse arguments, you have to call this by yourself
    fmt.Println(r.Form)  // print form information in server side
    for k, v := range r.Form {
        fmt.Println("key:", k)
        fmt.Println("val:", strings.Join(v, ""))
    }
	
	welchesProgramm := strings.Join(r.Form["program"],"")
	wasTun:=strings.Join(r.Form["aktion"],"")
	hashProc:=strings.Join(r.Form["hashproc"],"")		//hash für Versionsabgleich der Listen(Processe)
	hashProg:=strings.Join(r.Form["hashprog"],"")		//hash für Versionsabgleich der Listen(Programme)
	procNr,_:=strconv.Atoi(welchesProgramm)
	
	t:= template.Must(template.New("back").Parse(backTemplate))
	if welchesProgramm=="" {goto wrongHashOrValue}
	
	 switch wasTun {		//same identifier procNr for ProgrammID and ProcID...
     case "start":{ if procNr >=0 && procNr <len(v.ProgrammStartListe) && hashProg==hashOfProgrammListe() {

					go programmStart(procNr)		//evtl. goroutine ?!
					
					//t.Execute(w)
					t.Execute(w,dummy{3})
					fmt.Fprintln(w, "Programm "+ v.ProgrammNamenListe[procNr] +" wurde gestartet");
					}else {goto wrongHashOrValue}
					}
    case "kill":	{ if procNr >=0 && procNr <len(runningProcs) && hashProc==hashOfRunningProcs(){
					go programmKill(procNr)	//routine ja oder nein ?
					t.Execute(w,dummy{3})
					fmt.Fprintf(w,"Prozess "+welchesProgramm+" ("+runningProcs[procNr].Name+") wurde hart beendet (SIGKILL/9).")
					}else {goto wrongHashOrValue}}
	case "term":	{ if procNr >=0 && procNr <len(runningProcs) && hashProc==hashOfRunningProcs(){			
					go programmTerminate(procNr)	//routine ja oder nein ?
					t.Execute(w,dummy{3})
					fmt.Fprintln(w,"Beendigungsanfrage an Prozess "+welchesProgramm+" ("+runningProcs[procNr].Name+") wurde gesendet (SIGTERM/15). [ONLY NON-WINDOWS!]")
					}else {goto wrongHashOrValue}}
    case "stop":	{ if procNr >=0 && procNr <len(runningProcs) && hashProc==hashOfRunningProcs(){			
					go programmStop(procNr)	//evtl. goroutine!?
					t.Execute(w,dummy{3})
					fmt.Fprintln(w,"Stop-Befehl für "+runningProcs[procNr].Name+" (Prozess "+welchesProgramm+") wurde gestartet.")
				    }else {goto wrongHashOrValue}}
	case "autostart":{	if procNr >=0 && procNr <len(runningProcs) && hashProc==hashOfRunningProcs(){		//toggle Restartoption for running processes, for new processes wins the xml-config!
					runningProcs[procNr].Restart=!runningProcs[procNr].Restart	//you can also revive dead procs... or vice-versa
					t.Execute(w,dummy{3})
					}else {goto wrongHashOrValue}}
	case "log":		{if procNr >=0 && procNr <len(v.ProgrammStartListe) && hashProg==hashOfProgrammListe(){
					t.Execute(w,dummy{})
					dat, _:=ioutil.ReadFile("log_"+v.ProgrammNamenListe[procNr]+".txt")
					fmt.Fprint(w,string(dat))
					//g,_ :=os.Open("log_"+runningProcs[procNr].Name+".txt")
					//g.Read(b1)
					//openLogFile(v.ProgrammNamenListe[procNr]).Read(b1)
					//fmt.Fprint(w, string(b1))
					
					//fmt.Fprint(w,"HIER LOG AUSGEBEN")
					//fmt.Print(w,openLogFile(runningProcs[procNr].Name))
					//b2 := make([]byte, 200)
					//openLogFile(v.ProgrammNamenListe[0]).Read(b2)
					//fmt.Fprint(w,string(b2))

					}else {goto wrongHashOrValue}}
	case "exit":	{if procNr >=0 && procNr <len(runningProcs) && hashProc==hashOfRunningProcs(){
					go programmExit(procNr)
					t.Execute(w,dummy{3})
					fmt.Fprintln(w,"Exit-Befehl wurde an "+runningProcs[procNr].Name+" (Prozess "+welchesProgramm+") gesendet.")
					}}				
	default: 		{t.Execute(w,dummy{3})
					fmt.Fprintln(w,"Ungültiger Aufruf! Bitte zurück, Seite neu laden und erneut versuchen!")
	}			//nur wenn aktionskennung falsch, meldung. und meldung, dass befehl ausgeführt worden wäre. dies beides noch ändern
	}
	return
	wrongHashOrValue:	fmt.Fprintln(w,"Falscher Aufruf oder Hash ungültig! Bitte zurück, Seite neu laden und erneut versuchen!")
}
func webServer(){
	http.HandleFunc("/download", Download)
	http.HandleFunc("/", ObserverHandler)
	http.HandleFunc("/proccontrol", ProcControl)
    err := http.ListenAndServe(":8000", nil)
    if err != nil {
        log.Fatal("ListenAndServe: ", err)
    }
}

func programmTerminate(progra int){
	mutExRunningProcs.RLock()
	if progra < len(runningProcs) && progra >= 0{
		runningProcs[progra].Handle.Process.Signal(os.Interrupt)		//signal == readonly ?
		
		//	procSlice[progra].Process.Signal(os.Interrupt) //no need for syscall. still doesn't work under win-app TODO:CHECK WHY, app-specific?
		//	procSlice[progra].Process.Signal(syscall.SIGTERM) //doesn't work under win-app TODO:CHECK WHY, app-specific?
		//	procSlice[progra].Process.Signal(syscall.SIGINT) //doesn't work under win-app TODO:CHECK WHY, app-specific? cmd[cli] OK, calc[gui] NOK
		//			procSlice[progra].Process.Signal(syscall.SIGKILL)
		//procSlice[progra].Process.Signal(syscall.Signal(9)) //9=SIGKILL(HARD), 15 = SIGTERM(SOFT)
			
		fmt.Printf("Beendigungsanfrage an "+runningProcs[progra].Name+" mit PID "+strconv.Itoa(runningProcs[progra].Handle.Process.Pid)+" gestellt\n")
	}
	mutExRunningProcs.RUnlock()
	runtime.Gosched()
}

func programmKill(progra int){
	mutExRunningProcs.RLock()
	if progra <len(runningProcs) && progra >= 0{
	//	defer cleanProcSlice(progra)
	runningProcs[progra].Handle.Process.Kill()
	//	procSlice[progra].Process.Kill()
	fmt.Printf(runningProcs[progra].Name+" mit PID "+strconv.Itoa(runningProcs[progra].Handle.Process.Pid)+" wurde gekillt\n")
	}
	mutExRunningProcs.RUnlock()
	runtime.Gosched()
}
	
func programmStop(progra int){
	mutExRunningProcs.Lock()
	if progra <len(runningProcs) && progra >= 0{
		//defer if processstate war nicht exit... dann .kill ?
		befehlKomplett :=runningProcs[progra].StopCmd
		befehlSplit :=strings.Split(befehlKomplett," ")	
		cmd := exec.Command(befehlSplit[0], befehlSplit[1:]...)
		stdinPipe,_:=cmd.StdinPipe()
		stdoutPipe,_:=cmd.StdoutPipe()
		
	//	stdoutPipe, _ := cmd.StdoutPipe()
	logBuffer := make([]string,0)
	reader := bufio.NewReader(stdoutPipe)
	line, _, _ := reader.ReadLine()
	logBuffer = append(logBuffer, string(line))
	
	

	/*	stdout, err := cmd.StdoutPipe()
	    checkError(err)
	    stderr, err := cmd.StderrPipe()
	    checkError(err)
		io.Copy(os.Stdout, stdout)		//with GO or otherwise the second time it crashes
	    io.Copy(os.Stderr, stderr)*/		//with GO or otherwise the second time it crashes
		runningProcs = append(runningProcs, process{cmd,
											"STOP: "+runningProcs[progra].Name,
											runningProcs[progra].StopCmd,
											runningProcs[progra].StopCmd,		//for support of restart process procedure...
											false,							//Stop-Command usually fired once!
											true,
											1,
											runningProcs[progra].ExitCmd,
											stdinPipe,
											stdoutPipe,
											logBuffer,
											nil})
											
		fmt.Println("KZUUKSTILL ALIVE123")
		go cmd.Run()		//dann Status erst nach Abschluss des Prozesses routine important for long-lasting tasks!!!
		//cmd.Wait()
		//cmd.Start()		//dann Status direkt bei Feuern des Prozesses
		fmt.Println("Der für das Programm "+befehlKomplett+" mit PID "+ strconv.Itoa(runningProcs[progra].Handle.Process.Pid)+" ursprünglich hinterlegte Stop-Befehl wurde ausgeführt")
	}
	mutExRunningProcs.Unlock()
	runtime.Gosched()
}
func programmExit(progra int){
	mutExRunningProcs.RLock()
	if progra <len(runningProcs) && progra >= 0{
		fmt.Println("schreibe gleich"+runningProcs[progra].ExitCmd)
		//asdgf:="mspaint.exe\n"
		//runningProcs[progra].StdInPipe.Write([]byte(asdgf+"\n")) //WORKS
	//			runningProcs[progra].StdInPipe.Write([]byte("mspaint.exe\n")) WORKS

				runningProcs[progra].StdInPipe.Write([]byte(runningProcs[progra].ExitCmd+"\n"))

	//	io.WriteString(runningProcs[progra].StdInPipe,runningProcs[progra].ExitCmd)
			//	io.WriteString(runningProcs[progra].StdInPipe,"mspaint.exe\n") WORKS

	fmt.Printf(runningProcs[progra].Name+" mit PID "+strconv.Itoa(runningProcs[progra].Handle.Process.Pid)+" wurde Exit-Befehl geschickt\n")
	}
	mutExRunningProcs.RUnlock()
	runtime.Gosched()
}
	var logcomplete = make([]string,0)
func helperRoutinesStarter(){
var i int = 0			//count helper-runs for firing the lengthCheck
go watchFile()		//Veränderungen an der XML erkennen, ggfs. neu einlesenr
	for {
		//fmt.Println(v)
		//hashOfRunningProcs()	//placed in runningProcsLengthCheck
		//hashOfProgrammListe()	//placed in XMLRead
	fmt.Println("Helper re-run "+strconv.Itoa(i))
	fmt.Println("Number of goroutines running "+strconv.Itoa(runtime.NumGoroutine()))
	updateProcAliveState()
	time.Sleep(300 * time.Millisecond)
	checkForRestart()	//	((HIER MIT GO SONST HÄNGT))
	time.Sleep(300 * time.Millisecond)
//	fmt.Println(&b)

for r:=range runningProcs{
	//if len(runningProcs[r].LogBuffer) >0{
//	logcomplete = append(logcomplete, runningProcs[r].LogBuffer)
	//fmt.Println(runningProcs[r].LogBuffer.String())
//	runningProcs[r].LogBuffer.Reset()
line, _, _ := runningProcs[r].buRio.ReadLine()
if len(line) > 0 {
if len(runningProcs[r].LogBuffer) < 100{
	runningProcs[r].LogBuffer = append(runningProcs[r].LogBuffer, string(line))
} else{
runningProcs[r].LogBuffer = runningProcs[r].LogBuffer[1:98]
runningProcs[r].LogBuffer = append(runningProcs[r].LogBuffer,string(line))
}

	}

	fmt.Println((runningProcs[r].LogBuffer))
}

	processeAufKonsoleAusgeben()
		if i>runningProcsLengthInterval{
		runningProcsLengthCheck()
		i=0
		}
	i++
	}
}

//func programFlow(){
//	//Ablauf über HTTPHandler, momentan simuliert...
//	//Einfach mal ein paar willkürliche Aufrufe...
//	go programmStart(0)		
//	go programmStart(0)		//goroutine necessary for programmSTart
//	go programmStart(1)
//	time.Sleep(2 * time.Second)
//	programmKill(0)			//goroutine not necessary for programmKill
//	programmKill(5)			//goroutine not necessary for programmKill doesn't block even with wrong param :)
//	programmKill(1)			//goroutine not necessary for programmKill
//	programmKill(2)			//goroutine not necessary for programmKill
//	time.Sleep(2 * time.Second)
//	programmTerminate(1)
//	programmTerminate(2)
//	programmTerminate(3)
//	programmTerminate(2)
//	programmTerminate(-1)
//	time.Sleep(2 * time.Second)
//	programmStop(0)
//	programmStop(0)
	
	//			fmt.Printf("Hier kann jetzt parallel anderer Kram passieren.\n\n")
//TODO: MUTEX AUF GLOBALES ZEUG [ok], TIME WARTENS EINBAUEN, PROCESS EXIT STATE FÜR PROCESSLISTE[atm], stopprocesses in aktuelle prozessliste
//}

func updateProcAliveState(){
	mutExRunningProcs.Lock()
	for r:= range runningProcs{
		asd := runningProcs[r].Handle.ProcessState.String()
		if strings.HasPrefix(asd, "exit")==true{ //exit status 0,1,... want all of them!
		runningProcs[r].Alive=false
		}
	}
	mutExRunningProcs.Unlock()
	runtime.Gosched()
}

func processeAufKonsoleAusgeben(){
	var procSliceNotExited = make([]string, 0)	//copy, so we don't mix up the original list
	mutExRunningProcs.Lock()
	for r:= range runningProcs {
		asd := runningProcs[r].Handle.ProcessState.String()
		//	asd := procSlice[k].ProcessState.String()
		//	asd:=procState.String()
		if strings.HasPrefix(asd, "exit")==false{ //exit status 0,1,... don't want any of them!
			procSliceNotExited = append(procSliceNotExited, strconv.Itoa(r)+", "+runningProcs[r].Name+" "+runningProcs[r].StopCmd+runningProcs[r].ExitCmd)
			//procSliceNotExited = append(procSliceNotExited, strconv.Itoa(k)+", "+procSliceNameAndStopcmd[k])
		}
	}
	mutExRunningProcs.Unlock()
	runtime.Gosched()
	fmt.Print("noch laufende, not-exited processe: ")
	fmt.Println(procSliceNotExited) //diesen für die website zur anzeige der laufenden prozesse verwenden
}

func checkError(err error) {
    if err != nil {
        log.Fatalf("Error: %s", err)
    }
}
 
//TODO 
//Fehlerbehandlung
//Logging mit PID
//Tests!!!
//Diagramm
//SSL

/*
type EntetiesClass struct {
    Name string
    Value *exec.Cmd
	startCmd string
}
 data := map[string][]EntetiesClass{
        "Yoga": {{"Yoga", 15}, {"Yoga", 51}},
        "Pilates": {{"Pilates", 3}, {"Pilates", 6}, {"Pilates", 9}},
    }
https://www.socketloop.com/tutorials/golang-securing-password-with-salt
https://astaxie.gitbooks.io/build-web-application-with-golang/content/en/09.5.html
*/