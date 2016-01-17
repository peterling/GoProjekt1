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
	//"de/vorlesung/projekt/a04"
)

type xmlConfig struct { //these parameters are read from the xml configuration file
	XMLName              xml.Name `xml:"applications"`
	ProgrammNamenListe   []string `xml:"application>name"`    //indexed list of all configured program names
	ProgrammStartListe   []string `xml:"application>start"`   //indexed list of the corresponding start commands
	ProgrammStopListe    []string `xml:"application>stop"`    //indexed list of the corresponding stop commands
	ProgrammRestartListe []bool   `xml:"application>restart"` //indexed list of automatic restart configuration
	ProgrammExitListe    []string `xml:"application>exit"`    //indexed list of commands that are sent to the program's STDIN
}
type dummy struct { //for template execution
	Wait int //waiting time in seconds before redirection to the main page
}

type process struct {
	Handle     *exec.Cmd      //unique handle of the process
	Name       string         //name of the program that is executed, identifier
	StopCmd    string         //stopcommand with its arguments, trying to end process by starting another command
	StartCmd   string         //startcommand with its arguments
	Restart    bool           //whether the automatic restart option is activated or not
	Alive      bool           //has the process already been exited ?
	StartCount int            //how often has the program been started in this session ?
	ExitCmd    string         //this string can be sent to the STDIN of the process
	StdInPipe  io.WriteCloser //pipe to the STDIN of the process
	StdOutPipe io.ReadCloser  //pipe from STDOUT
	LogBuffer  []string       //for keeping some logging entries
}

//Globale Variablen
var runningProcs = make([]process, 0)           //all ever spawned child processes (since start of session, will be cleaned up from time to time)
var runningProcsNew = make([]process, 0)        //temporary slice for copying while reorganisation
var v = xmlConfig{}                             //the read-in configuration struct
var runningProcsReorged time.Time = time.Now()  //timestamp of when the processList has been reorganized
var programmListeReorged time.Time = time.Now() //timestamp of when the programmList has been reorganized
var mutExRunningProcsReorged = &sync.RWMutex{}  //mutual exclusion lock for accessing the runningProcessesReorgTimestamp
var mutExProgrammListeReorged = &sync.RWMutex{} //mutual exclusion lock for accessing the programListReorgTimestamp
var mutExRunningProcs = &sync.RWMutex{}         //mutual exclusion lock for accessing the runningProcessesList

//Globale Konstanten
const logFileMaxSize = 10000                //maximum file size in bytes for program logging to files
const sliceMaxSize = 100                    //how many entries are allowed in the temporary slice-based logging
const configPath string = "config.xml"      //where is the xml configuration file located ?
const runningProcsLengthTreshold int = 1000 //after the runningProcessesList exceeds X entries, it is reorganized for performance reasons
const runningProcsLengthInterval int = 300  //how often the length of the runningProcessesList is checked for its length

func indexProgrammList(r int) int { //get the programListeEntry from a given processNr
	index := -1 //dangerous!
	for k := range v.ProgrammNamenListe {

		if v.ProgrammNamenListe[k] == runningProcs[r].Name || strings.HasSuffix(runningProcs[r].Name, v.ProgrammNamenListe[k]) { //for "[STOP] "-Processnames
			index = k
		}

	}
	return index
}

func Download(w http.ResponseWriter, r *http.Request) { //function for download of the configuration file
	http.ServeFile(w, r, configPath)
}

func ObserverHandler(w http.ResponseWriter, r *http.Request) { //main http handler for webGUI
	t := template.Must(template.New("control").Parse(uebersichtTemplate)) //take our main template and parse it

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
	}
	t.Execute(w, data)
}

func checkForRestart() { //are there any exited processes that need to be restarted ?
	mutExRunningProcs.Lock() //try not to interfere with other routines
	for r := range runningProcs {
		if runningProcs[r].Restart == true && runningProcs[r].Alive == false {
			go programmStart(r, r)
		}
	}
	mutExRunningProcs.Unlock()
	runtime.Gosched()
}

func openLogFile(progra string) *os.File { //open log files for writing persistent logging to disk (observer itself could crash)
	logFile, err := os.OpenFile("./log_"+progra+".txt", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666) //WRITE-ONLY is sufficient
	//logFile, err := os.OpenFile("./log_"+progra+".txt", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)	//testing purposes
	if err != nil {
		panic(err)
	}
	return logFile //for every program there is *one* logging file that is limited in size
}

func programmStart(programmNr int, option int) { //option -2 = programmstop (processnr), -1 = normal start (programmnr), +int = processIndex restart (processnr)
	var befehlKomplett string //dependent on the option int, we start a new process, execute a stop command or restart an exited process.
	if option == -1 {         //the interpretation of programmNr depens on the chosen option! programmNr can also refer to a runningProcessesListIndex
		befehlKomplett = v.ProgrammStartListe[programmNr] //whole start command including arguments and parameters from programmListe
	} else if option >= 0 {
		befehlKomplett = runningProcs[programmNr].StartCmd //...from the process that has to be restarted
	} else {
		befehlKomplett = runningProcs[programmNr].StopCmd //...from the StopCommandEntry of the running process in order to stop the program.
	}

	if befehlKomplett != "" {
		befehlSplit := strings.Split(befehlKomplett, " ") //the command entry is a single string, containing the command, its arguments and parameters, separated by commata
		cmd := exec.Command(befehlSplit[0], befehlSplit[1:]...)
		stdinPipe, _ := cmd.StdinPipe()   //pipe the processes' STDIN for later use
		cmdAusgabe, _ := cmd.StdoutPipe() //pipe the processes' STDOUT for later use, it is also possible to redirect the STDERR, too
		logBuffer := make([]string, 0)    //create a slice for logging entries

		mutExRunningProcs.Lock() //co-op, team!
		switch option {
		case -2: //we chose to stop a process, so run its STOP-Command
			{
				var stopName string
				if strings.HasPrefix(runningProcs[programmNr].Name, "[STOP] ") { //user could have tried to stop a stopping process...
					stopName = runningProcs[programmNr].Name
				} else {
					stopName = "[STOP] " + runningProcs[programmNr].Name //mark it with the [STOP]-Prefix for quick recognition
				}

				if programmNr >= 0 {
					runningProcs = append(runningProcs, process{cmd,
						stopName,
						runningProcs[programmNr].StopCmd, //when you try to restart a stopping-process, you probably want to give a second try *to stop*
						runningProcs[programmNr].StopCmd, //for support of restart process procedure...
						false, //Stop-Command usually fired once!
						true,  //usually it should run
						1,     //first try of stopping
						runningProcs[programmNr].ExitCmd, //take it from the existing process, for new entries it could have been changed in config...
						stdinPipe,                        //so we can speak to the process...
						cmdAusgabe,                       //...and also listen to it
						logBuffer})                       //log what happens
					programmNr = indexProgrammList(programmNr) //programmNr was the Index of runningProcessList but ahead we need the programListIndex!
				}
			}
		case -1: //first/normal start
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
		default: //restart Process, usually option == programmNr
			{
				programmNr = indexProgrammList(programmNr)
				runningProcs[option] = process{cmd,
					runningProcs[option].Name,
					runningProcs[option].StopCmd,
					runningProcs[option].StartCmd,
					runningProcs[option].Restart,
					true,
					runningProcs[option].StartCount + 1, //increment the start count
					runningProcs[option].ExitCmd,
					stdinPipe,
					cmdAusgabe,
					runningProcs[option].LogBuffer}
			}
		}

		mutExRunningProcs.Unlock() //cope with others
		runtime.Gosched()          //share

		logFile := openLogFile(v.ProgrammNamenListe[programmNr]) //logging to disk
		defer logFile.Close()                                    //close the file, whatever happens
		var procIndex int                                        //where in the processList are we?
		if len(runningProcs) > 0 {
			procIndex = len(runningProcs) - 1 //the last appended process
		} else {
			procIndex = 0 //first entry (right after start or reorganisation)
		}

		if option >= 0 { //take the old slot when restarting
			procIndex = option
		}

		scannen := bufio.NewScanner(cmdAusgabe)                                                                                                      //scanner for log output
		logFile.WriteString(time.Now().Format(time.RFC3339) + ": INFO[Instanz gestartet]\n")                                                         //create a logging entry on processStart in file...
		runningProcs[procIndex].LogBuffer = append(runningProcs[procIndex].LogBuffer, time.Now().Format(time.RFC3339)+": INFO[Instanz gestartet]\n") //...and in slice
		go func() {
			for scannen.Scan() { //let the scanner run and wait for stdout-outputs
				fileStat, _ := logFile.Stat() //properties of logfile
				mutExRunningProcs.Lock()
				if len(runningProcs) > procIndex { //otherwise it would be out of range
					if len(runningProcs[procIndex].LogBuffer) < sliceMaxSize { //we have space left
						runningProcs[procIndex].LogBuffer = append(runningProcs[procIndex].LogBuffer, scannen.Text()+"\n")
					} else {
						runningProcs[procIndex].LogBuffer = runningProcs[procIndex].LogBuffer[1:(sliceMaxSize - 2)] //simulate a ring buffer
						runningProcs[procIndex].LogBuffer = append(runningProcs[procIndex].LogBuffer, scannen.Text()+"\n")

					}
				}
				mutExRunningProcs.Unlock()
				runtime.Gosched()
				if fileStat.Size() > logFileMaxSize { //is the file size limit for logging to disk exceeded ?
					os.Truncate("./log_"+v.ProgrammNamenListe[programmNr]+".txt", 0) //if so, empty it
					for r := range runningProcs[procIndex].LogBuffer {
						logFile.Write([]byte(runningProcs[procIndex].LogBuffer[r])) //and fill it with the logs we already have in ram
					}
				} else {
					logFile.WriteString(scannen.Text() + "\n")

				}
				fmt.Printf("Ausgabe | %s\n", scannen.Text())
			}
		}()
		cmd.Run() //execute the command
		fmt.Printf(v.ProgrammNamenListe[programmNr] + " wurde gestartet, ")
		fmt.Printf("PID %d\n", cmd.Process.Pid)
	}
}

func runningProcsLengthCheck() { //periodically check the slice's length
	fmt.Println("DEBUG: Lengthcheck started!")
	mutExRunningProcs.Lock()
	if len(runningProcs) > runningProcsLengthTreshold {
		mutExRunningProcsReorged.Lock()
		runningProcsReorged = time.Now() //others have to know if the list has been reorganized. indexes probably have changed!
		mutExRunningProcsReorged.Unlock()
		runningProcsNew = nil
		for r := range runningProcs {
			if runningProcs[r].Alive == true || runningProcs[r].Restart == true { //the others are exited and don't matter much to us
				runningProcsNew = append(runningProcsNew, runningProcs[r]) //copy those we still want in our new, cleaned list
			}
		}
		runningProcs = nil
		for r := range runningProcsNew {
			runningProcs = append(runningProcs, runningProcsNew[r]) //copy them back
		}
	}
	mutExRunningProcs.Unlock()
	runtime.Gosched()
}

func xmlReadIn() error { //read in XML configuration file
	mutExProgrammListeReorged.Lock()
	programmListeReorged = time.Now() //memorize when the last read-in took place
	mutExProgrammListeReorged.Unlock()
	runtime.Gosched()
	hashOfProgrammListe()                          //a hash is more comfy to submit than a plain timestamp
	xmlContent, err := ioutil.ReadFile(configPath) //open the whole file
	if err != nil {
		fmt.Printf("error: %v", err)
		return err
	}
	v = xmlConfig{} //empty it before we read another time
	err = xml.Unmarshal(xmlContent, &v)
	if err != nil {
		fmt.Printf("error: %v", err)
		return err
	}
	return nil
}

func watchFile() error { //check if the config file has changed
	initialStat, err := os.Stat(configPath)
	if err != nil {
		return err
	}
	fmt.Println("DEBUG: Datei geprüft!")
	stat, err := os.Stat(configPath)
	if err != nil {
		return err
	}

	if stat.Size() != initialStat.Size() || stat.ModTime() != initialStat.ModTime() { //do size or modification date differ ?
		xmlReadIn()
		log.Println("Datei neu eingelesen, da verändert")
		initialStat, err = os.Stat(configPath)
	}
	return nil
}

func hashOfProgrammListe() string { //client and server should have the same version of the programmListe, so that they refer to the same element!
	h := sha1.New() //if versions differ (because of xml configuration changes), the wrong commands could be executed
	mutExProgrammListeReorged.RLock()
	h.Write([]byte(programmListeReorged.String()))
	mutExProgrammListeReorged.RUnlock()
	runtime.Gosched()
	bs := h.Sum(nil)
	fmt.Printf("ProgrammListe: %x\n", bs)
	return hex.EncodeToString(bs) //otherwise hex-encoded. for better human readability
}

func hashOfRunningProcs() string { //client and server should have the same version of the processListe, so that they refer to the same element!
	h := sha1.New() //if versions differ (because of runningProcsList reorganisation), the wrong process could be stopped/killed/...
	mutExRunningProcsReorged.RLock()
	h.Write([]byte(runningProcsReorged.String()))
	mutExRunningProcsReorged.RUnlock()
	runtime.Gosched()
	bs := h.Sum(nil)

	fmt.Printf("RunningProcs: %x\n", bs)
	return hex.EncodeToString(bs) //otherwise hex-encoded. for better human readability
}

func main() {
	//mandatory steps to get the program up and running...
	xmlReadIn()                //read in the configured applications and commands
	go helperRoutinesStarter() //runs important tasks in the background
	webServer()                //Webserver, you have control!
	//here you could add additional tasks!
}

func ProcControl(w http.ResponseWriter, r *http.Request) {
	r.ParseForm() // parse http arguments
	//fmt.Println(r.Form) // print form information to console
	for k, v := range r.Form {
		fmt.Println("key:", k)
		fmt.Println("val:", strings.Join(v, ""))
	}

	welchesProgramm := strings.Join(r.Form["program"], "") //can refer to a programmListeIndex or runningProcessesListIndex, depending on context
	wasTun := strings.Join(r.Form["aktion"], "")           //which function should be used
	hashProc := strings.Join(r.Form["hashproc"], "")       //hash für Versionsabgleich der Listen(Processe)
	hashProg := strings.Join(r.Form["hashprog"], "")       //hash für Versionsabgleich der Listen(Programme)
	procNr, _ := strconv.Atoi(welchesProgramm)

	t := template.Must(template.New("back").Parse(backTemplate))
	if welchesProgramm == "" {
		goto wrongHashOrValue
	}

	switch wasTun { //same identifier procNr used for either ProgrammID or ProcID...
	case "start":
		{
			if procNr >= 0 && procNr < len(v.ProgrammStartListe) && hashProg == hashOfProgrammListe() {

				go programmStart(procNr, -1)
				t.Execute(w, dummy{3}) //show result and go back to main page after 3 seconds
				fmt.Fprintln(w, "Programm "+v.ProgrammNamenListe[procNr]+" wurde gestartet")
			} else {
				goto wrongHashOrValue
			}
		}
	case "kill":
		{
			if procNr >= 0 && procNr < len(runningProcs) && hashProc == hashOfRunningProcs() {
				go programmKill(procNr)
				t.Execute(w, dummy{3})
				fmt.Fprintf(w, "Prozess "+welchesProgramm+" ("+runningProcs[procNr].Name+") wurde hart beendet (SIGKILL/9).")
			} else {
				goto wrongHashOrValue
			}
		}
	case "term":
		{
			if procNr >= 0 && procNr < len(runningProcs) && hashProc == hashOfRunningProcs() {
				go programmTerminate(procNr)
				t.Execute(w, dummy{3})
				fmt.Fprintln(w, "Beendigungsanfrage an Prozess "+welchesProgramm+" ("+runningProcs[procNr].Name+") wurde gesendet (SIGTERM/15). [ONLY NON-WINDOWS!]")
			} else {
				goto wrongHashOrValue
			}
		}
	case "stop":
		{
			if procNr >= 0 && procNr < len(runningProcs) && hashProc == hashOfRunningProcs() {
				go programmStart(procNr, -2)
				t.Execute(w, dummy{3})
				fmt.Fprintln(w, "Stop-Befehl für "+runningProcs[procNr].Name+" (Prozess "+welchesProgramm+") wurde gestartet.")
			} else {
				goto wrongHashOrValue
			}
		}
	case "autostart":
		{
			if procNr >= 0 && procNr < len(runningProcs) && hashProc == hashOfRunningProcs() { //toggle restart option for running processes, for new processes the xml-config wins!
				runningProcs[procNr].Restart = !runningProcs[procNr].Restart //you are able to revive dead procs... or deactivate the autostart feature if it was set
				t.Execute(w, dummy{3})
			} else {
				goto wrongHashOrValue
			}
		}
	case "log":
		{
			if procNr >= 0 && procNr < len(runningProcs) && hashProc == hashOfRunningProcs() {
				t.Execute(w, dummy{})
				for r := range runningProcs[procNr].LogBuffer {
					fmt.Fprint(w, runningProcs[procNr].LogBuffer[r])
					fmt.Fprint(w, "<html><br></html>") //html-encoded newline after each log entry (output-encoding could be changed for some special characters...)
				}
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

func webServer() { //served by different handlers depending on the given url
	http.HandleFunc("/download", Download)       //serve xml configuration file to browser
	http.HandleFunc("/", ObserverHandler)        //main handler
	http.HandleFunc("/proccontrol", ProcControl) //processControl, handles all tasks
	err := http.ListenAndServeTLS(":4443", "cert.pem", "key.pem", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func programmTerminate(processNr int) {
	mutExRunningProcs.RLock()
	if processNr < len(runningProcs) && processNr >= 0 {
		runningProcs[processNr].Handle.Process.Signal(os.Interrupt) //send an signal to terminate from the os
		fmt.Printf("Beendigungsanfrage an " + runningProcs[processNr].Name + " mit PID " + strconv.Itoa(runningProcs[processNr].Handle.Process.Pid) + " gestellt\n")
	}
	mutExRunningProcs.RUnlock()
	runtime.Gosched()
}

func programmKill(processNr int) { //immediately kill a process and therefore write a log entry so that the reason for that can be determined afterwards
	mutExRunningProcs.Lock()
	if processNr < len(runningProcs) && processNr >= 0 {
		runningProcs[processNr].Restart = false
		runningProcs[processNr].Handle.Process.Kill()
		runningProcs[processNr].LogBuffer = append(runningProcs[processNr].LogBuffer, "KILL-Anforderung gesendet")
		logFile := openLogFile(strings.TrimPrefix(runningProcs[processNr].Name, "[STOP] "))
		defer logFile.Close()
		logFile.WriteString("KILL-Anforderung gesendet\n")
		fmt.Printf(runningProcs[processNr].Name + " mit PID " + strconv.Itoa(runningProcs[processNr].Handle.Process.Pid) + " wurde gekillt\n")
	}
	mutExRunningProcs.Unlock()
	runtime.Gosched()
}

func programmExit(processNr int) {
	mutExRunningProcs.RLock()
	if processNr < len(runningProcs) && processNr >= 0 {
		runningProcs[processNr].StdInPipe.Write([]byte(runningProcs[processNr].ExitCmd + "\n"))
		fmt.Printf(runningProcs[processNr].Name + " mit PID " + strconv.Itoa(runningProcs[processNr].Handle.Process.Pid) + " wurde Exit-Befehl geschickt\n")
	}
	mutExRunningProcs.RUnlock()
	runtime.Gosched()
}

func helperRoutinesStarter() {
	var i int = 0 //count helper-runs for firing the lengthCheck when its interval is reached
	for {
		err := watchFile() //check for config file changes
		if err != nil {
			panic(err)
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

func updateProcAliveState() { //looking for processes that have been exited
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
		if strings.HasPrefix(asd, "exit") == false { //exit status 0,1,... don't want any of them!
			procSliceNotExited = append(procSliceNotExited, strconv.Itoa(r)+", "+runningProcs[r].Name+" "+runningProcs[r].StopCmd+" "+runningProcs[r].ExitCmd)
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

//inspired by golang.org and also to be mentioned:
/*
https://jan.newmarch.name/go/template/chapter-template.html
https://gowalker.org/os/exec
https://gobyexample.com/reading-files
https://github.com/bradfitz/runsit
http://golangtutorials.blogspot.de/2011/06/control-structures-go-switch-case.html

not implemented:
https://www.socketloop.com/tutorials/golang-securing-password-with-salt
https://astaxie.gitbooks.io/build-web-application-with-golang/content/en/09.5.html
*/
