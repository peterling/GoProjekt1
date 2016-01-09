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
	ProgrammNamenListe	[]string `xml:"application>name"`
	ProgrammStartListe  []string `xml:"application>start"`
	ProgrammStopListe  []string `xml:"application>stop"`
	ProgrammRestartListe	[]bool `xml:"application>restart"`
}


var	v = xmlConfig{}
var laufendeProzesse = make([] string,0)
var procSlice = make([]*exec.Cmd, 0)
var stopperProcSlice = make([]*exec.Cmd, 0)
var procSliceNameAndStopcmd = make ([] string,0)
var procSliceStartCmd = make ([] string,0)		//für jeden process auch das startcommando hinterlegen für ggfs. restart
var procRestartEnabled = make([] bool,0)		//für jeden process auch autostart hinterlegen

func checkError(err error) {
    if err != nil {
        log.Fatalf("Error: %s", err)
    }
} 
func helperCheckRestart(){
for true{
	go checkForRestart()
	time.Sleep(3 * time.Second)
	//fmt.Println(procRestartEnabled)
	}
}
func checkForRestart(){
	//for {			//Dauerschleife, daher timer am ende!
		fmt.Println("Schleife begonnen!")
	
		fmt.Println(procRestartEnabled)
		for r:=range procRestartEnabled{		//für alle einträge im restartslice
		
	if	procRestartEnabled[r]==true{//wenn restart-switch on für appl, nur dann...
				//k= von if oben das entsprechende nur nehmen
					fmt.Println("BIS HIERHER")
					fmt.Println(len(procSlice))
					fmt.Println(r)
					fmt.Println(procSlice[r].ProcessState.String())
		asd := procSlice[r].ProcessState.String()
	//	asd:=procState.String()
		if strings.HasPrefix(asd, "exit")==true{ //exit status 0,1,... want them all!
		fmt.Println("ASDASDASD")
		fmt.Println(procRestartEnabled[r])
	//	programmStart(1)
		//hier aus programmstart genommen, da startprogrammnummer nix gut! könnte sich ja geändert haben.
		
var befehlKomplett string = procSliceStartCmd[r]
befehlSplit :=strings.Split(befehlKomplett," ")	
	cmd := exec.Command(befehlSplit[0], befehlSplit[1:]...)
procSliceStartCmd = append(procSliceStartCmd, befehlKomplett)		//mutex noch einbauen
procRestartEnabled = append(procRestartEnabled, true)		//mutex noch einbauen
procRestartEnabled[r]=false			//beim bereits verendeten process den restart wegnehmen!
mutexProcSlice.Lock()
	logFile:=createLogFile(len(procSlice))
	procSlice = append(procSlice, cmd)
mutexProcSlice.Unlock()
//runtime.Gosched()
mutexProcSliceNameAndStopcmd.Lock()
	procSliceNameAndStopcmd = append(procSliceNameAndStopcmd, procSliceNameAndStopcmd[r])
mutexProcSliceNameAndStopcmd.Unlock()
runtime.Gosched()
defer logFile.Close()
	cmd.Stdout=logFile
	cmd.Run()
	//cmd.Start()
	
	
		cmd.Stdout.Write([]byte("Here is a string...."))
		fmt.Println("BIS HIERHER")
//		fmt.Printf(v.ProgrammNamenListe[programmNr]+" wurde gestartet, ") //auf diese ausgabe ist kein verlass (kein real time, erst nach beendigung der funktion)
		fmt.Printf("PID %d\n",cmd.Process.Pid)
	
		
		
		
		
		

		
		
		}
		
}
	
}
//time.Sleep(3 * time.Second)
//}

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

/*
func killingProcessSoftly(pid int){
	if processRunning(pid){
	p,_:=	os.FindProcess(pid)
	p.Signal(os.Interrupt)
	}
}*/

func createLogFile(progra int)*os.File{
	  // open the out file for writing
	asd :=strconv.Itoa(progra)
    logFile, err := os.Create("./logProcSlice"+asd+".txt")
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
procSliceStartCmd = append(procSliceStartCmd, befehlKomplett)		//mutex noch einbauen
procRestartEnabled = append(procRestartEnabled, false)		//mutex noch einbauen
mutexProcSlice.Lock()
	logFile:=createLogFile(len(procSlice))
	procSlice = append(procSlice, cmd)
mutexProcSlice.Unlock()
runtime.Gosched()
mutexProcSliceNameAndStopcmd.Lock()
	procSliceNameAndStopcmd = append(procSliceNameAndStopcmd, v.ProgrammNamenListe[programmNr]+", "+v.ProgrammStopListe[programmNr])
mutexProcSliceNameAndStopcmd.Unlock()
runtime.Gosched()
defer logFile.Close()
	cmd.Stdout=logFile
	cmd.Run()
	
	
		cmd.Stdout.Write([]byte("Here is a string...."))
/*	
	
	writer := bufio.NewWriter(logFile)
    defer writer.Flush()
	 stdoutPipe, err := cmd.StdoutPipe()
    if err != nil {
        panic(err)
    }
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
        panic(err)
    }

	go io.Copy(writer,stdoutPipe)
	go io.Copy(writer,stderrPipe)*/
//	cmd.stdoutPipe = logFile
//	cmd.Stderr = logFile
//		logFile.WriteString(cmd.ProcessState.String())
	//logFile.WriteString("asdfghjgfdsd")

	//cmd.Start()	//kein processstate abrufbar
	//cmd.Wait()	//crash
//	writer.WriteString("asfsfstringwirter")
//	writer.ReadFrom(stdoutPipe)
//	logFile.WriteString(cmd.ProcessState.String())

	//cmd.CombinedOutput()

//	createLogFile(cmd.Process.Pid)	//logfile namens der pid anlegen [wird erst nach beendigung von run gemacht - zu spät!]
	
	

	//p,_:=os.FindProcess(cmd.Process.Pid)
		fmt.Printf(v.ProgrammNamenListe[programmNr]+" wurde gestartet, ") //auf diese ausgabe ist kein verlass (kein real time, erst nach beendigung der funktion)
		fmt.Printf("PID %d\n",cmd.Process.Pid)
	//fmt.Printf("PID lautet: %d",p.Pid)
	//return cmd.Process
}
var mutexProcSlice = &sync.Mutex{}
var mutexStopperProcSlice = &sync.Mutex{}
var mutexProcSliceNameAndStopcmd = &sync.Mutex{}

var configPath = "config.xml"
var programmAuswahl int = 1	//wählen, welches Programm gestartet werden soll - von HTTPHandler
func xmlReadIn(){			//XML-Datei wird eingelesen
	xmlContent, _ := ioutil.ReadFile(configPath)
	err := xml.Unmarshal(xmlContent, &v)
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}
}
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
	go programFlow()	//Aufrufe in Goroutine starten, später HTTP, jetzt hardcoded

//hier adden!

	 fmt.Scanln()		//Programm weiterlaufen lassen ohne Ende
	}
	
func programmTerminate(progra int){
	procSlice[progra].Process.Signal(os.Interrupt) //no need for syscall. still doesn't work under win-app TODO:CHECK WHY, app-specific?
	//	procSlice[progra].Process.Signal(syscall.SIGTERM) //doesn't work under win-app TODO:CHECK WHY, app-specific?
	//	procSlice[progra].Process.Signal(syscall.SIGINT) //doesn't work under win-app TODO:CHECK WHY, app-specific? cmd[cli] OK, calc[gui] NOK
	//			procSlice[progra].Process.Signal(syscall.SIGKILL)
	//procSlice[progra].Process.Signal(syscall.Signal(9)) //9=SIGKILL(HARD), 15 = SIGTERM(SOFT)
	mutexProcSliceNameAndStopcmd.Lock()
	var nameAndStopcmd string = procSliceNameAndStopcmd[progra]
	mutexProcSliceNameAndStopcmd.Unlock()
	runtime.Gosched()
		
		//Konsolenrückmeldung
	befehlSplit :=strings.Split(nameAndStopcmd,", ")	
	fmt.Printf("Beendigungsanfrage an "+befehlSplit[0]+" mit PID "+strconv.Itoa(procSlice[progra].Process.Pid)+" gestellt\n")
}

func cleanProcSlice(progra int){
	mutexProcSlice.Lock()
	if procSlice[progra].ProcessState.Exited(){
	fmt.Println("ASDLKGLDFLKDFLGK")
//	procSlice[progra].ProcessState.String() ggfs. hiermit, vgl. startprocess
	}
	mutexProcSlice.Unlock()
	runtime.Gosched()
}
func programmKill(progra int){
//	defer cleanProcSlice(progra)
	procSlice[progra].Process.Kill()
	mutexProcSliceNameAndStopcmd.Lock()
	var nameAndStopcmd string = procSliceNameAndStopcmd[progra]
	mutexProcSliceNameAndStopcmd.Unlock()
	runtime.Gosched()
	befehlSplit :=strings.Split(nameAndStopcmd,", ")	
	fmt.Printf(befehlSplit[0]+" mit PID "+strconv.Itoa(procSlice[progra].Process.Pid)+" wurde gekillt\n")
	}

	
func programmStop(progra int){
	//defer if processstate war nicht exit... dann .kill ?
	var nameAndStopcmd string = procSliceNameAndStopcmd[progra]
	befehlKomplett :=strings.Split(nameAndStopcmd,", ")
	befehlSplit :=strings.Split(befehlKomplett[1]," ")	
	cmd := exec.Command(befehlSplit[0], befehlSplit[1:]...)
	stdout, err := cmd.StdoutPipe()
    checkError(err)
    stderr, err := cmd.StderrPipe()
    checkError(err)
	go io.Copy(os.Stdout, stdout)
    go io.Copy(os.Stderr, stderr)
	mutexStopperProcSlice.Lock()
	stopperProcSlice = append(stopperProcSlice, cmd) //Befehl zum Stoppen könnte auch hängen, Name wird aber nicht gespeichert (overKill)
	mutexStopperProcSlice.Unlock()
	runtime.Gosched()
	cmd.Run()
	//cmd.Wait()
	//cmd.Start()
	fmt.Println("Der für das Programm "+befehlKomplett[0]+" mit PID "+ strconv.Itoa(procSlice[progra].Process.Pid)+" ursprünglich hinterlegte Stop-Befehl wurde ausgeführt")
	fmt.Println(stopperProcSlice)
	fmt.Println(stopperProcSlice[0].ProcessState.String())
}	//TODO Stoppbefehlliste bereinigen ?! wie bei startprogrammliste auch gemacht...


func programFlow(){
	//Ablauf über HTTPHandler

	//		go processeAufKonsoleAusgeben()
	processeAufKonsoleAusgeben()
go helperCheckRestart()
//go checkForRestart()
	go programmStart(0)
//		go programmStart(2)
	//go programmStart(1)
	go programmStart(1)
	processeAufKonsoleAusgeben()
	go programmStart(1)
	go programmStart(1)
	go programmStart(1)
	time.Sleep(2 * time.Second)
	go programmKill(0)
	time.Sleep(1 * time.Second)
	//go programmKill(1)
	go programmTerminate(1)
	processeAufKonsoleAusgeben()
	//go programmKill(2)
//	go programmTerminate(2)
//	go programmKill(3)
	//go programmKill(4)
	//	go programmKill(1) //check if exists/running und ob erfolgreich...
		time.Sleep(3 * time.Second)
	//	programmStop(1)
		processeAufKonsoleAusgeben()
			processeAufKonsoleAusgeben()
				
				
			go	programmStop(0)
			go	programmStop(0)
			programmStop(0)
			time.Sleep(3 * time.Second)
			processeAufKonsoleAusgeben()
			procRestartEnabled[0]=true	//mal für ein programm den restart enablen
		//	fmt.Println("Folgende Prozesse laufen gerade: ")
		//	fmt.Println(procSlice)
		//	fmt.Println(procSliceNameAndStopcmd)
		
				//			fmt.Printf("Hier kann jetzt parallel anderer Kram passieren.\n\n")
//TODO: MUTEX AUF GLOBALES ZEUG [ok], TIME WARTENS EINBAUEN, PROCESS EXIT STATE FÜR PROCESSLISTE[atm], stopprocesses in aktuelle prozessliste
}

func processeAufKonsoleAusgeben(){
	var procSliceNotExited = make([]string, 0)	//copy, so we don't mix up the original list
	mutexProcSlice.Lock()
	mutexProcSliceNameAndStopcmd.Lock()
	for k:= range procSlice {
		asd := procSlice[k].ProcessState.String()
	//	asd:=procState.String()
		if strings.HasPrefix(asd, "exit")==false{ //exit status 0,1,... don't want them all!
			procSliceNotExited = append(procSliceNotExited, strconv.Itoa(k)+", "+procSliceNameAndStopcmd[k])
		}
	}
	fmt.Println(procSlice)
	mutexProcSlice.Unlock()
	mutexProcSliceNameAndStopcmd.Unlock()
	runtime.Gosched()

	fmt.Print("noch laufende, not-exited processe: ")
	fmt.Println(procSliceNotExited) //diesen für die website zur anzeige der laufenden prozesse verwenden
	fmt.Print("zusätzlich noch laufende, [todo:nur not-exited!] STOP-processe: ")
	fmt.Println(stopperProcSlice) //diesen für die website zur anzeige der laufenden prozesse verwenden
}

	
//	stdout_peter, err_peter := cmd_peter.StdoutPipe()	//in buffer loaden, pro process

//signal.Notify(stdin, os.Interrupt)

//unter Windows funktioniert nur Kill, nicht Interrupt.
//cmd.Process.Signal(os.Interrupt)
//cmd_peter.Process.Signal(os.Interrupt)
//cmd.Process.Signal(os.Kill)
//cmd_peter.Process.Signal(os.Kill)
