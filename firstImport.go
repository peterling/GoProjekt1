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
//	"bufio"
//	"syscall"
	"strings"
	//"path/filepath"
	"encoding/xml"
	"time"
	"sync"
	"runtime"
)

type xmlConfig struct {
	XMLName xml.Name `xml:"applications"`
	ProgrammNamenListe		[]string `xml:"application>name"`
	ProgrammStartListe		[]string `xml:"application>start"`
	ProgrammStopListe		[]string `xml:"application>stop"`
	ProgrammRestartListe	[]bool `xml:"application>restart"`
}

type process struct {
	Handle		*exec.Cmd
	Name		string
	StopCmd		string
	StartCmd	string
	Restart		bool
	Alive		bool
}

var runningProcs = make([] process,0)		//all ever spawned processes in a struct
var runningProcsNew = make([] process,0)
var	v = xmlConfig{}							//the read-in configuration struct

func restartProc(r int){
	//mutExRunningProcs.Lock()
		var befehlKomplett string = runningProcs[r].StartCmd
			befehlSplit :=strings.Split(befehlKomplett," ")	
			cmd := exec.Command(befehlSplit[0], befehlSplit[1:]...)

			runningProcs = append(runningProcs, process{cmd,
										runningProcs[r].Name,
										runningProcs[r].StopCmd,
										runningProcs[r].StartCmd,
										//true})
										runningProcs[r].Restart,
										true})		//Restart should be true
			runningProcs[r].Restart=false			//Beim bisherigen Eintrag Restart deaktivieren
//mutExRunningProcs.Unlock()
//runtime.Gosched()

			logFile:=openLogFile("len(runningProcs)")

			defer logFile.Close()	//sinnvoll?
			cmd.Stdout=logFile
			cmd.Run()		//ARGH!""§
			//cmd.Start()
			//cmd.Wait()
}

func checkForRestart(){
	mutExRunningProcs.Lock()
	fmt.Println("CHECK FOR RESTART BEGINN")
	for r:=range runningProcs{		//für alle einträge im restartslice
	asd := runningProcs[r].Handle.ProcessState.String()
		if	runningProcs[r].Restart==true && strings.HasPrefix(asd, "exit")==true{//wenn restart-switch on für appl, nur dann...	
			//exit status 0,1,... want them all!
			fmt.Println("CHECK FOR RESTART vor restartPROC")
			go restartProc(r)		//ohne GOROUTINE hängt ?!! wegen run/start in restartProc! sonst nicht überwacht wenn start statt run!
				fmt.Println("CHECK FOR RESTART nach restartPROC")
		
		}
	
	}
mutExRunningProcs.Unlock()
runtime.Gosched()
fmt.Println("CHECK FOR RESTART ENDE")
}
/*
func checkForRestart(){
	for r:=range runningProcs{		//für alle einträge im restartslice
		if	runningProcs[r].Restart==true{//wenn restart-switch on für appl, nur dann...	
			asd := runningProcs[r].Handle.ProcessState.String()
			if strings.HasPrefix(asd, "exit")==true{ //exit status 0,1,... want them all!
				var befehlKomplett string = runningProcs[r].StartCmd//procSliceStartCmd[r]
				befehlSplit :=strings.Split(befehlKomplett," ")	
				cmd := exec.Command(befehlSplit[0], befehlSplit[1:]...)
	
				runningProcs = append(runningProcs, process{cmd,
											runningProcs[r].Name,
											runningProcs[r].StopCmd,
											runningProcs[r].StartCmd,
											//true})
											runningProcs[r].Restart,
											true})		//Restart should be true
				runningProcs[r].Restart=false			//Beim bisherigen Eintrag Restart deaktivieren


				logFile:=openLogFile("len(runningProcs)")
	
				defer logFile.Close()	//sinnvoll?
				cmd.Stdout=logFile
				cmd.Run()		//ARGH!""§
				//cmd.Start()
				//cmd.Wait()

			}
		
		}
	
	}

}
*/
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
	logFile, err := os.OpenFile("./log_"+progra+".txt",os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666 )
    if err != nil {
        panic(err)
    }
	logFile.WriteString("LogFile created")

	return logFile
}

func programmStart(programmNr int){
	var befehlKomplett string = v.ProgrammStartListe[programmNr]	
	befehlSplit :=strings.Split(befehlKomplett," ")	
	cmd := exec.Command(befehlSplit[0], befehlSplit[1:]...)
	//	stdout, err := cmd.StdoutPipe()
	//    checkError(err)
	//    stderr, err := cmd.StderrPipe()
	//    checkError(err)
	//	go io.Copy(os.Stdout, stdout)
	//   go io.Copy(os.Stderr, stderr)
	mutExRunningProcs.Lock()
	runningProcs = append(runningProcs, process{cmd,
					v.ProgrammNamenListe[programmNr],
					v.ProgrammStopListe[programmNr],
					befehlKomplett,
					v.ProgrammRestartListe[programmNr],
					true})
mutExRunningProcs.Unlock()
runtime.Gosched()
	logFile:=openLogFile(v.ProgrammNamenListe[programmNr])

	defer logFile.Close()
	cmd.Stdout=logFile
	cmd.Run()
	
	//defer logFile.Close()	//sinnvoll?
	cmd.Stdout=logFile
	cmd.Run()
	//	cmd.Start() //??besser?? weil bei start kein processstate abrufbar ?!
	fmt.Printf(v.ProgrammNamenListe[programmNr]+" wurde gestartet, ") //auf diese ausgabe ist kein verlass (kein real time, erst nach beendigung der funktion)
	fmt.Printf("PID %d\n",cmd.Process.Pid)
	cmd.Stdout.Write([]byte("\r\n"+time.Now().Format(time.RFC3339)+": INFO[Instanz gestartet]"))	//CR for friends of Micro$oft Editors
	logFile.Sync()		//at this point ?
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
const runningProcsLengthTreshold int = 5	//maybe 10000
const runningProcsLengthInterval time.Duration = time.Second*1	//maybe 1000

func runningProcsLengthCheck(){
	mutExRunningProcs.Lock()
	
	if len(runningProcs) > runningProcsLengthTreshold{
		fmt.Println("Hey, ist zu lang!")
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
	xmlContent, _ := ioutil.ReadFile(configPath)
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
        stat, err := os.Stat(configPath)
        if err != nil {
            return err
        }

        if stat.Size() != initialStat.Size() || stat.ModTime() != initialStat.ModTime() {
        xmlReadIn()
		log.Println("asd")
		initialStat, err = os.Stat(configPath)
        }

        time.Sleep(2 * time.Second)
    }

    return nil
}

func main() {
	//Programminitialisierung
	xmlReadIn()			//XML-Datei einlesen lassen
	go watchFile()		//Veränderungen an der XML erkennen, ggfs. neu einlesen [background]
	go helperRoutinesStarter()
	go programFlow()	//Aufrufe in Goroutine starten, später HTTP, jetzt hardcoded

	//hier adden!

	 fmt.Scanln()		//Programm weiterlaufen lassen ohne Ende //endpoint
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
		//	fmt.Println(progra)

		//	var nameAndStopcmd string = procSliceNameAndStopcmd[progra]
		//var nameAndStopcmd string = runningProcs[progra]
		//	befehlKomplett :=strings.Split(nameAndStopcmd,", ")
		befehlKomplett :=runningProcs[progra].StopCmd
		befehlSplit :=strings.Split(befehlKomplett," ")	
		cmd := exec.Command(befehlSplit[0], befehlSplit[1:]...)
		stdout, err := cmd.StdoutPipe()
	    checkError(err)
	    stderr, err := cmd.StderrPipe()
	    checkError(err)
		go io.Copy(os.Stdout, stdout)
	    go io.Copy(os.Stderr, stderr)
//		mutexStopperProcSlice.Lock()
		//stopperProcSlice = append(stopperProcSlice, cmd) //Befehl zum Stoppen könnte auch hängen, Name wird aber nicht gespeichert (overKill)
		runningProcs = append(runningProcs, process{cmd,
											"STOP: "+runningProcs[progra].Name,
											runningProcs[progra].StopCmd,
											runningProcs[progra].StopCmd,		//for support of restart process procedure...
											false,							//Stop-Command usually fired once!
											true})
	
		cmd.Run()		//dann Status erst nach Abschluss des Prozesses
		//cmd.Wait()
		//cmd.Start()		//dann Status direkt bei Feuern des Prozesses
		fmt.Println("Der für das Programm "+befehlKomplett+" mit PID "+ strconv.Itoa(runningProcs[progra].Handle.Process.Pid)+" ursprünglich hinterlegte Stop-Befehl wurde ausgeführt")
	}
	mutExRunningProcs.Unlock()
	runtime.Gosched()
}
func helperRoutinesStarter(){
	
for {
	updateProcAliveState()	//blockt wenn keine goroutine
	time.Sleep(1 * time.Second)
	checkForRestart()	//	((HIER MIT GO SONST HÄNGT))
	time.Sleep(3 * time.Second)
	processeAufKonsoleAusgeben()
	time.Sleep(runningProcsLengthInterval)
	runningProcsLengthCheck()
	}
}
func programFlow(){
	//Ablauf über HTTPHandler, momentan simuliert...
	//Einfach mal ein paar willkürliche Aufrufe...

	//go checkForRestart()
	go programmStart(0)		
	go programmStart(0)		//goroutine not necessary for programmSTart
	go programmStart(1)
	time.Sleep(2 * time.Second)
	programmKill(0)			//goroutine not necessary for programmKill
	programmKill(5)			//goroutine not necessary for programmKill doesn't block even with wrong param :)
	programmKill(1)			//goroutine not necessary for programmKill
	programmKill(2)			//goroutine not necessary for programmKill
//	go programmTerminate(1)
//	processeAufKonsoleAusgeben()
//	go programmKill(2)
//	go programmTerminate(2)
//	go programmTerminate(-1)
//	time.Sleep(2 * time.Second)
//	go programmStop(0)
//	go programmStop(0)
//	go programmStop(0)
	
	//			fmt.Printf("Hier kann jetzt parallel anderer Kram passieren.\n\n")
//TODO: MUTEX AUF GLOBALES ZEUG [ok], TIME WARTENS EINBAUEN, PROCESS EXIT STATE FÜR PROCESSLISTE[atm], stopprocesses in aktuelle prozessliste
}

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
			procSliceNotExited = append(procSliceNotExited, strconv.Itoa(r)+", "+runningProcs[r].Name+" "+runningProcs[r].StopCmd)
			//procSliceNotExited = append(procSliceNotExited, strconv.Itoa(k)+", "+procSliceNameAndStopcmd[k])
		}
	}
mutExRunningProcs.Unlock()
runtime.Gosched()
	fmt.Print("noch laufende, not-exited processe: ")
	fmt.Println(procSliceNotExited) //diesen für die website zur anzeige der laufenden prozesse verwenden
//	fmt.Print("zusätzlich noch laufende, [todo:nur not-exited!] STOP-processe: ")
//	fmt.Println(stopperProcSlice) //diesen für die website zur anzeige der laufenden prozesse verwenden
}

func checkError(err error) {
    if err != nil {
        log.Fatalf("Error: %s", err)
    }
}

//	stdout_peter, err_peter := cmd_peter.StdoutPipe()	//in buffer loaden, pro process ODER PRO PROGRAMM... ?!

//signal.Notify(stdin, os.Interrupt)

//unter Windows funktioniert nur Kill, nicht Interrupt.
//cmd.Process.Signal(os.Interrupt)
//cmd_peter.Process.Signal(os.Interrupt)
//cmd.Process.Signal(os.Kill)
//cmd_peter.Process.Signal(os.Kill)
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

*/
