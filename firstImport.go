package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type xmlConfig struct {
	XMLName              xml.Name `xml:"applications"`
	ProgrammNamenListe   []string `xml:"application>name"`
	ProgrammStartListe   []string `xml:"application>start"`
	ProgrammStopListe    []string `xml:"application>stop"`
	ProgrammRestartListe []bool   `xml:"application>restart"`
	ProgrammExitListe    []string `xml:"application>exit"`
}
type dummy struct { //for template execution
	Wait int
}

type process struct {
	Handle     *exec.Cmd
	Name       string
	StopCmd    string
	StartCmd   string
	Restart    bool
	Alive      bool
	StartCount int
	ExitCmd    string
	StdInPipe  io.WriteCloser
	StdOutPipe io.ReadCloser
	LogBuffer  []string
}

//Globale Variablen
var runningProcs = make([]process, 0) //all ever spawned processes in a struct
var runningProcsNew = make([]process, 0)
var v = xmlConfig{} //the read-in configuration struct
var runningProcsReorged time.Time = time.Now()
var programmListeReorged time.Time = time.Now()
var mutExRunningProcsReorged = &sync.RWMutex{}
var mutExProgrammListeReorged = &sync.RWMutex{}
var mutExRunningProcs = &sync.RWMutex{}

//Globale Konstanten
const logFileMaxSize = 10000 //maximum file size in bytes for program logging
const sliceMaxSize = 100
const configPath string = "config.xml"
const programmAuswahl int = 1             //wählen, welches Programm gestartet werden soll - IRL vom HTTPHandler
const runningProcsLengthTreshold int = 10 //maybe 10000
const runningProcsLengthInterval int = 5  //maybe 1000

func indexProgrammList(r int) int {
	index := -1		//dangerous!
	for k := range v.ProgrammNamenListe {

		if v.ProgrammNamenListe[k] == runningProcs[r].Name || strings.HasSuffix(runningProcs[r].Name, v.ProgrammNamenListe[k]) {		//for "[STOP] "-Processnames
			index = k
		}

	}
	fmt.Println("testindex")
	return index
}

func Download(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, configPath)
}

func ObserverHandler(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.New("control").Parse(uebersichtTemplate)) // Create a template.

	data := struct {
		Titel        string
		Programme    []string
		Prozesse     []process
		ProgrammHash string
		ProzessHash  string
	}{
		Titel:        "Observer",
		Programme:    v.ProgrammNamenListe,
		Prozesse:     runningProcs,
		ProgrammHash: hashOfProgrammListe(),
		ProzessHash:  hashOfRunningProcs(),
		/*[]string{
			"Programm1",
			"Programm2",
		},*/
	}

	t.Execute(w, data)
}

func checkForRestart() {
	mutExRunningProcs.Lock()
	for r := range runningProcs { //für alle einträge im restartslice
		if runningProcs[r].Restart == true && runningProcs[r].Alive == false { //wenn restart-switch on für appl, nur dann...
			go programmStart(r, r) //ohne GOROUTINE hängt ?!! wegen run/start in restartProc! sonst nicht überwacht wenn start statt run!
		}
	}
	mutExRunningProcs.Unlock()
	runtime.Gosched()
}

func openLogFile(progra string) *os.File {
	// open the out file for writing
	logFile, err := os.OpenFile("./log_"+progra+".txt", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666) //WRITE-ONLY
	//logFile, err := os.OpenFile("./log_"+progra+".txt", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)
	if err != nil {
		panic(err)
	}
	return logFile
}

func programmStart(programmNr int, option int) { //option -2 = programmstop (processnr), -1 = normal start (programmnr), +int = processIndex restart (processnr)
	var befehlKomplett string
	if option == -1 {
		befehlKomplett = v.ProgrammStartListe[programmNr]
	} else if option >= 0 {
		befehlKomplett = runningProcs[programmNr].StartCmd
	} else {
	//	programmNr = indexProgrammList(programmNr)
	//	befehlKomplett = v.ProgrammStopListe[programmNr]
	befehlKomplett = runningProcs[programmNr].StopCmd
	}

	if befehlKomplett != "" {
		befehlSplit := strings.Split(befehlKomplett, " ")
		cmd := exec.Command(befehlSplit[0], befehlSplit[1:]...)
		stdinPipe, _ := cmd.StdinPipe()

		cmdAusgabe, _ := cmd.StdoutPipe()
		logBuffer := make([]string, 0)

		mutExRunningProcs.Lock()
		switch option {
		case -2: //Programm Stop
			{
				//	programmNr = indexProgrammList(programmNr)
				var stopName string
				if strings.HasPrefix(runningProcs[programmNr].Name, "[STOP] "){
					stopName=runningProcs[programmNr].Name
				}else{
					stopName= "[STOP] " + runningProcs[programmNr].Name
				}
				
				
				if programmNr >= 0 {
					runningProcs = append(runningProcs, process{cmd,
						stopName,
						runningProcs[programmNr].StopCmd,
						runningProcs[programmNr].StopCmd, //for support of restart process procedure...
						false, //Stop-Command usually fired once!
						true,
						1,
						runningProcs[programmNr].ExitCmd,
						stdinPipe,
						cmdAusgabe,
						logBuffer})
						programmNr = indexProgrammList(programmNr)
				}
			}
		case -1: //first start
			{
				runningProcs = append(runningProcs, process{cmd,
					v.ProgrammNamenListe[programmNr],
					v.ProgrammStopListe[programmNr],
					befehlKomplett,
					v.ProgrammRestartListe[programmNr],
					true,
					1,
					v.ProgrammExitListe[programmNr],
					stdinPipe,
					cmdAusgabe,
					logBuffer})
			}
		default: //restart Process
			{		//option == programmNr
								programmNr = indexProgrammList(programmNr)
				runningProcs[option] = process{cmd,
					runningProcs[option].Name,
					runningProcs[option].StopCmd,
					runningProcs[option].StartCmd,
					runningProcs[option].Restart,
					true,
					runningProcs[option].StartCount + 1,
					runningProcs[option].ExitCmd,
					stdinPipe,
					cmdAusgabe,
					runningProcs[option].LogBuffer}
			}

		}

		mutExRunningProcs.Unlock()
		runtime.Gosched()

		logFile := openLogFile(v.ProgrammNamenListe[programmNr])
		defer logFile.Close()
		var procIndex int
		if len(runningProcs) > 0 {
			procIndex = len(runningProcs) - 1
		} else {
			procIndex = 0
		}
		
		if option >= 0{
			procIndex= option
		}
		
		
		scannen := bufio.NewScanner(cmdAusgabe)
		//	pid := runningProcs[procIndex].Handle.Process.Pid
		logFile.WriteString(time.Now().Format(time.RFC3339) + ": INFO[Instanz gestartet]\n") //CR for friends of Micro$oft Editors
		runningProcs[procIndex].LogBuffer = append(runningProcs[procIndex].LogBuffer, time.Now().Format(time.RFC3339)+": INFO[Instanz gestartet]\n")
		go func() {
			for scannen.Scan() {
				fileStat, _ := logFile.Stat()
				mutExRunningProcs.Lock()
				if len(runningProcs) > procIndex {
					if len(runningProcs[procIndex].LogBuffer) < sliceMaxSize {
						fmt.Println("Schreiben")
						runningProcs[procIndex].LogBuffer = append(runningProcs[procIndex].LogBuffer, scannen.Text()+"\n")
					} else {
						runningProcs[procIndex].LogBuffer = runningProcs[procIndex].LogBuffer[1:(sliceMaxSize - 2)]
						runningProcs[procIndex].LogBuffer = append(runningProcs[procIndex].LogBuffer, scannen.Text()+"\n")

					}
				}
				mutExRunningProcs.Unlock()
				runtime.Gosched()
				//fileStat, _ = logFile.Stat()
				if fileStat.Size() > logFileMaxSize {
					os.Truncate("./log_"+v.ProgrammNamenListe[programmNr]+".txt", 0)
					//logFile.WriteString(scannen.Text() + "\n")
					for r := range runningProcs[procIndex].LogBuffer {
						logFile.Write([]byte(runningProcs[procIndex].LogBuffer[r]))

					}
				} else {
					logFile.WriteString(scannen.Text() + "\n")

				}
				fmt.Printf("Ausgabe | %s\n", scannen.Text())
			}
		}()

		cmd.Run()
		fmt.Printf(v.ProgrammNamenListe[programmNr] + " wurde gestartet, ") //auf diese ausgabe ist kein verlass (kein real time, erst nach beendigung der funktion)
		fmt.Printf("PID %d\n", cmd.Process.Pid)
		//	cmd.Stdout.Write([]byte("\r\n"+time.Now().Format(time.RFC3339)+": INFO[Instanz gestartet]"))	//CR for friends of Micro$oft Editors
	}
}

func runningProcsLengthCheck() {
	fmt.Println("DEBUG: Lengthcheck started!")
	mutExRunningProcs.Lock()
	if len(runningProcs) > runningProcsLengthTreshold {
		mutExRunningProcsReorged.Lock()
		runningProcsReorged = time.Now()
		mutExRunningProcsReorged.Unlock()
		runningProcsNew = nil
		for r := range runningProcs {
			if runningProcs[r].Alive == true || runningProcs[r].Restart == true {
				runningProcsNew = append(runningProcsNew, runningProcs[r])
			}
		}
		runningProcs = nil
		for r := range runningProcsNew {
			runningProcs = append(runningProcs, runningProcsNew[r])
		}
		//	fmt.Println(runningProcsNew)
	}
	mutExRunningProcs.Unlock()
	runtime.Gosched()
}

func xmlReadIn() { //XML-Datei wird eingelesen
	mutExProgrammListeReorged.Lock()
	programmListeReorged = time.Now()
	mutExProgrammListeReorged.Unlock()
	runtime.Gosched()
	hashOfProgrammListe()
	xmlContent, _ := ioutil.ReadFile(configPath)
	//err:= xml.Unmarshal(nil, &v)		//bei Neu-Einlesen soll nicht appendet werden
	v = xmlConfig{} //leeren, damit nicht appendet wird bei Neu-Einlesen
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
	fmt.Println("DEBUG: Datei geprüft!")
	stat, err := os.Stat(configPath)
	if err != nil {
		return err
	}

	if stat.Size() != initialStat.Size() || stat.ModTime() != initialStat.ModTime() {
		xmlReadIn()
		log.Println("Datei neu eingelesen, da verändert")
		initialStat, err = os.Stat(configPath)
	}
	return nil
}

func hashOfProgrammListe() string { //Programmliste könnte reorganisiert worden sein -> index verschoben. Daher beim Ausführen über HTTP Hash abgleichen...
	h := sha1.New() //falls anderer Hash --> Anfrage verwerfen, Seite neu generieren
	mutExProgrammListeReorged.RLock()

	h.Write([]byte(programmListeReorged.String()))

	mutExProgrammListeReorged.RUnlock()
	runtime.Gosched()
	bs := h.Sum(nil)
	fmt.Printf("ProgrammListe: %x\n", bs)
	return hex.EncodeToString(bs)
}

func hashOfRunningProcs() string { //runningProcsListe könnte reorganisiert worden sein -> index verschoben. Daher beim Ausführen über HTTP Hash abgleichen...
	h := sha1.New() //falls anderer Hash --> Anfrage verwerfen, Seite neu generieren
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
	xmlReadIn()                //XML-Datei einlesen lassen
	go helperRoutinesStarter() //runs in the background for important tasks
	//go programFlow()	//Aufrufe in Goroutine starten, später HTTP, jetzt hardcoded
	webServer() //Webserver, you have control!
	//hier adden!

	//	fmt.Scanln() //Programm weiterlaufen lassen ohne Ende //endpoint
}

func ProcControl(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()       // parse arguments, you have to call this by yourself
	fmt.Println(r.Form) // print form information in server side
	for k, v := range r.Form {
		fmt.Println("key:", k)
		fmt.Println("val:", strings.Join(v, ""))
	}

	welchesProgramm := strings.Join(r.Form["program"], "")
	wasTun := strings.Join(r.Form["aktion"], "")
	hashProc := strings.Join(r.Form["hashproc"], "") //hash für Versionsabgleich der Listen(Processe)
	hashProg := strings.Join(r.Form["hashprog"], "") //hash für Versionsabgleich der Listen(Programme)
	procNr, _ := strconv.Atoi(welchesProgramm)

	t := template.Must(template.New("back").Parse(backTemplate))
	if welchesProgramm == "" {
		goto wrongHashOrValue
	}

	switch wasTun { //same identifier procNr for ProgrammID and ProcID...
	case "start":
		{
			if procNr >= 0 && procNr < len(v.ProgrammStartListe) && hashProg == hashOfProgrammListe() {

				go programmStart(procNr, -1) //evtl. goroutine ?!

				//t.Execute(w)
				t.Execute(w, dummy{3})
				fmt.Fprintln(w, "Programm "+v.ProgrammNamenListe[procNr]+" wurde gestartet")
			} else {
				goto wrongHashOrValue
			}
		}
	case "kill":
		{
			if procNr >= 0 && procNr < len(runningProcs) && hashProc == hashOfRunningProcs() {
				go programmKill(procNr) //routine ja oder nein ?

				t.Execute(w, dummy{3})
				fmt.Fprintf(w, "Prozess "+welchesProgramm+" ("+runningProcs[procNr].Name+") wurde hart beendet (SIGKILL/9).")
			} else {
				goto wrongHashOrValue
			}
		}
	case "term":
		{
			if procNr >= 0 && procNr < len(runningProcs) && hashProc == hashOfRunningProcs() {
				go programmTerminate(procNr) //routine ja oder nein ?
				t.Execute(w, dummy{3})
				fmt.Fprintln(w, "Beendigungsanfrage an Prozess "+welchesProgramm+" ("+runningProcs[procNr].Name+") wurde gesendet (SIGTERM/15). [ONLY NON-WINDOWS!]")
			} else {
				goto wrongHashOrValue
			}
		}
	case "stop":
		{
			if procNr >= 0 && procNr < len(runningProcs) && hashProc == hashOfRunningProcs() {
				go programmStart(procNr, -2) //evtl. goroutine!?
				t.Execute(w, dummy{3})
				fmt.Fprintln(w, "Stop-Befehl für "+runningProcs[procNr].Name+" (Prozess "+welchesProgramm+") wurde gestartet.")
			} else {
				goto wrongHashOrValue
			}
		}
	case "autostart":
		{
			if procNr >= 0 && procNr < len(runningProcs) && hashProc == hashOfRunningProcs() { //toggle Restartoption for running processes, for new processes wins the xml-config!
				runningProcs[procNr].Restart = !runningProcs[procNr].Restart //you are able to revive dead procs... or vice-versa
				t.Execute(w, dummy{3})
			} else {
				goto wrongHashOrValue
			}
		}
	case "log":
		{
			if procNr >= 0 && procNr < len(runningProcs) && hashProc == hashOfRunningProcs() {

				t.Execute(w, dummy{})
				//dat, _ := ioutil.ReadFile("log_" + v.ProgrammNamenListe[procNr] + ".txt")
				//fmt.Fprint(w, string(dat))
				for r := range runningProcs[procNr].LogBuffer {
					fmt.Fprint(w, runningProcs[procNr].LogBuffer[r])
					fmt.Fprint(w, "<html><br></html>")
				}
				//	fmt.Fprint(w, runningProcs[procNr].LogBuffer)

			} else {
				goto wrongHashOrValue
			}
		}
	case "exit":
		{
			if procNr >= 0 && procNr < len(runningProcs) && hashProc == hashOfRunningProcs() {
				go programmExit(procNr)
				t.Execute(w, dummy{3})
				fmt.Fprintln(w, "Exit-Befehl wurde an "+runningProcs[procNr].Name+" (Prozess "+welchesProgramm+") gesendet.")
			} else {
				goto wrongHashOrValue
			}
		}
	default:
		{
			t.Execute(w, dummy{3})
			fmt.Fprintln(w, "Ungültiger Aufruf! Bitte zurück, Seite neu laden und erneut versuchen!")
		} //nur wenn aktionskennung falsch, meldung. und meldung, dass befehl ausgeführt worden wäre. dies beides noch ändern
	}
	return
wrongHashOrValue:
	t.Execute(w, dummy{3})
	fmt.Fprintln(w, "Falscher Aufruf oder Hash ungültig! Bitte zurück, Seite neu laden und erneut versuchen!")
}

func webServer() {
	http.HandleFunc("/download", Download)
	http.HandleFunc("/", ObserverHandler)
	http.HandleFunc("/proccontrol", ProcControl)
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func programmTerminate(progra int) {
	mutExRunningProcs.RLock()
	if progra < len(runningProcs) && progra >= 0 {
		runningProcs[progra].Handle.Process.Signal(os.Interrupt) //signal == readonly ?

		//procSlice[progra].Process.Signal(syscall.Signal(9)) //9=SIGKILL(HARD), 15 = SIGTERM(SOFT)

		fmt.Printf("Beendigungsanfrage an " + runningProcs[progra].Name + " mit PID " + strconv.Itoa(runningProcs[progra].Handle.Process.Pid) + " gestellt\n")
	}
	mutExRunningProcs.RUnlock()
	runtime.Gosched()
}

func programmKill(progra int) {
	mutExRunningProcs.RLock()
	if progra < len(runningProcs) && progra >= 0 {
		runningProcs[progra].Restart = false
		runningProcs[progra].Handle.Process.Kill()
		runningProcs[progra].LogBuffer = append(runningProcs[progra].LogBuffer, "KILL-Anforderung gesendet")
		logFile := openLogFile(strings.TrimPrefix(runningProcs[progra].Name, "[STOP] "))
		defer logFile.Close()
		logFile.WriteString("KILL-Anforderung gesendet\n")
		fmt.Printf(runningProcs[progra].Name + " mit PID " + strconv.Itoa(runningProcs[progra].Handle.Process.Pid) + " wurde gekillt\n")
	}
	mutExRunningProcs.RUnlock()
	runtime.Gosched()
}

func programmExit(progra int) {
	mutExRunningProcs.RLock()
	if progra < len(runningProcs) && progra >= 0 {
		//	fmt.Println("schreibe gleich" + runningProcs[progra].ExitCmd)
		//			runningProcs[progra].StdInPipe.Write([]byte("mspaint.exe\n")) WORKS
		runningProcs[progra].StdInPipe.Write([]byte(runningProcs[progra].ExitCmd + "\n"))
		fmt.Printf(runningProcs[progra].Name + " mit PID " + strconv.Itoa(runningProcs[progra].Handle.Process.Pid) + " wurde Exit-Befehl geschickt\n")
	}
	mutExRunningProcs.RUnlock()
	runtime.Gosched()
}

func helperRoutinesStarter() {
	var i int = 0 //count helper-runs for firing the lengthCheck
	for {
		err := watchFile() //Veränderungen an der XML erkennen, ggfs. neu einlesen
		if err != nil {
			panic(err)
			//			break
		}
		fmt.Println("DEBUG: Helper re-run # " + strconv.Itoa(i))
		fmt.Println("DEBUG: Number of concurrent goroutines running at the moment: " + strconv.Itoa(runtime.NumGoroutine()))
		updateProcAliveState()
		time.Sleep(300 * time.Millisecond)
		checkForRestart()
		time.Sleep(300 * time.Millisecond)

		processeAufKonsoleAusgeben()
		if i > runningProcsLengthInterval {
			runningProcsLengthCheck()
			i = 0
		}
		i++
	}
}

func updateProcAliveState() {
	mutExRunningProcs.Lock()
	for r := range runningProcs {
		asd := runningProcs[r].Handle.ProcessState.String()
		if strings.HasPrefix(asd, "exit") == true { //exit status 0,1,... want all of them!
			runningProcs[r].Alive = false
		}
	}
	mutExRunningProcs.Unlock()
	runtime.Gosched()
}

func processeAufKonsoleAusgeben() {
	var procSliceNotExited = make([]string, 0) //copy, so we don't mix up the original list
	mutExRunningProcs.Lock()
	for r := range runningProcs {
		asd := runningProcs[r].Handle.ProcessState.String()
		//	asd := procSlice[k].ProcessState.String()
		//	asd:=procState.String()
		if strings.HasPrefix(asd, "exit") == false { //exit status 0,1,... don't want any of them!
			procSliceNotExited = append(procSliceNotExited, strconv.Itoa(r)+", "+runningProcs[r].Name+" "+runningProcs[r].StopCmd+runningProcs[r].ExitCmd)
			//procSliceNotExited = append(procSliceNotExited, strconv.Itoa(k)+", "+procSliceNameAndStopcmd[k])
		}
	}
	mutExRunningProcs.Unlock()
	runtime.Gosched()
	fmt.Print("noch laufende, not-exited processe: ")
	fmt.Println(procSliceNotExited) //diesen für die website zur anzeige der laufenden prozesse verwenden
}

//HTML Templates
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
	<input type="button" value="XML-Datei anzeigen" onClick="window.location.href='/download'">
	<a href="/download" download="config">XML-Datei herunterladen</a>
	<h1>Programme starten</h1>
	
		{{range $index, $results := .Programme}}<a href="/proccontrol?program={{$index}}&aktion=start&hashprog={{$.ProgrammHash}}">{{.}}</a><br>{{else}}<div><strong>keine Programme hinterlegt</strong></div>{{end}}
	<h1>Laufende Prozesse hart beenden (SIGKILL)</h1>	
		{{range $index, $results := .Prozesse}}{{if .Alive}}<a href="/proccontrol?program={{$index}}&aktion=kill&hashproc={{$.ProzessHash}}">{{.Name}}, PID: {{.Handle.Process.Pid}}, Autostart: {{.Restart}}, läuft: {{.Alive}}, {{.StartCount}} mal gestartet</a><br>{{end}}{{else}}<div><strong>keine überwachten Prozesse</strong></div>{{end}}
	<h1>Laufende Prozesse weich beenden (SIGTERM)</h1>	
		{{range $index, $results := .Prozesse}}{{if .Alive}}<a href="/proccontrol?program={{$index}}&aktion=term&hashproc={{$.ProzessHash}}">{{.Name}}, PID: {{.Handle.Process.Pid}}, Autostart: {{.Restart}}, läuft: {{.Alive}}, {{.StartCount}} mal gestartet</a><br>{{end}}{{else}}<div><strong>keine überwachten Prozesse</strong></div>{{end}}
	<h1>Laufende Prozesse mit hinterlegtem Exit-Befehl an STDIN beenden</h1>	
		{{range $index, $results := .Prozesse}}{{if .Alive}}<a href="/proccontrol?program={{$index}}&aktion=exit&hashproc={{$.ProzessHash}}">{{.Name}}, PID: {{.Handle.Process.Pid}}, Autostart: {{.Restart}}, läuft: {{.Alive}}, {{.StartCount}} mal gestartet</a><br>{{end}}{{else}}<div><strong>keine überwachten Prozesse</strong></div>{{end}}
	<h1>Laufende Prozesse mit hinterlegtem STOP-Befehl beenden</h1>	
		{{range $index, $results := .Prozesse}}{{if .Alive}}<a href="/proccontrol?program={{$index}}&aktion=stop&hashproc={{$.ProzessHash}}">{{.Name}}, PID: {{.Handle.Process.Pid}}, Autostart: {{.Restart}}, läuft: {{.Alive}}, {{.StartCount}} mal gestartet</a><br>{{end}}{{else}}<div><strong>keine überwachten Prozesse</strong></div>{{end}}
	<h1>Restart-Option laufender Prozesse (de-)aktivieren</h1>
		{{range $index, $results := .Prozesse}}{{if .Alive}}{{if .Restart}}<b>{{end}}<a href="/proccontrol?program={{$index}}&aktion=autostart&hashproc={{$.ProzessHash}}">{{.Name}}, PID: {{.Handle.Process.Pid}}, Autostart: {{.Restart}}, läuft: {{.Alive}}, {{.StartCount}} mal gestartet</a><br>{{if .Restart}}</b>{{end}}{{end}}{{else}}<div><strong>keine überwachten Prozesse</strong></div>{{end}}
	<h1>Restart-Option beendeter Prozesse (de-)aktivieren [revive/dismiss]</h1>
		{{range $index, $results := .Prozesse}}{{if not .Alive}}{{if .Restart}}<b>{{end}}<a href="/proccontrol?program={{$index}}&aktion=autostart&hashproc={{$.ProzessHash}}">{{.Name}}, PID: {{.Handle.Process.Pid}}, Autostart: {{.Restart}}, läuft: {{.Alive}}, {{.StartCount}} mal gestartet</a><br>{{if .Restart}}</b>{{end}}{{end}}{{else}}<div><strong>keine überwachten Prozesse</strong></div>{{end}}
	<h1>Logging</h1>
		{{range $index, $results := .Prozesse}}<a href="/proccontrol?program={{$index}}&aktion=log&hashproc={{$.ProzessHash}}">{{.Name}}, PID: {{.Handle.Process.Pid}}</a><br>{{else}}<div><strong>keine Programme hinterlegt</strong></div>{{end}}

	</body>
</html>`

//TODO
//Fehlerbehandlung
//Logging mit PID
//Tests!!!
//Diagramm
//SSL

/*
https://www.socketloop.com/tutorials/golang-securing-password-with-salt
https://astaxie.gitbooks.io/build-web-application-with-golang/content/en/09.5.html
*/
