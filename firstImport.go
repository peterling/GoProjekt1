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
	"syscall"
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
}


var	v = xmlConfig{}
var laufendeProzesse = make([] string,0)
var procSlice = make([]*exec.Cmd, 0)
var stopperProcSlice = make([]*exec.Cmd, 0)
var procSliceNameAndStopcmd = make ([] string,0)

func checkError(err error) {
    if err != nil {
        log.Fatalf("Error: %s", err)
    }
} 

func processRunning(pid int)bool{
	if asd,_:=os.FindProcess(pid);asd != nil{
fmt.Printf("%d läuft noch!",pid)}

return true
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
func killingProcessHard(pid int){
	//proc.Kill()
	// awer := exec.Command("ping", "localhost", "-t")
	//awer.Start()
	p,_:=os.FindProcess(pid)
	//p.Signal(os.Kill)
	p.Signal(syscall.Signal(9))
	//p.Signal(os.Kill)
	//_,err:= p.Wait()
//return p.Kill()
//return err
// err := awer.Wait()
//awer.Process.Kill()
 //   fmt.Println(err)
	
	/*if processRunning(pid){
	p,_:=	os.FindProcess(pid)
	p.Signal(os.Kill)
	p.Kill()*/
//	os.FindProcess(pid).Kill()
	}

func killingProcessSoftly(pid int){
	if processRunning(pid){
	p,_:=	os.FindProcess(pid)
	p.Signal(os.Interrupt)
	}
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
mutexProcSlice.Lock()
	procSlice = append(procSlice, cmd)
mutexProcSlice.Unlock()
runtime.Gosched()
mutexProcSliceNameAndStopcmd.Lock()
	procSliceNameAndStopcmd = append(procSliceNameAndStopcmd, v.ProgrammNamenListe[programmNr]+", "+v.ProgrammStopListe[programmNr])
mutexProcSliceNameAndStopcmd.Unlock()
runtime.Gosched()
	cmd.Run()
	//cmd.Start()
	//cmd.Wait()

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
	go watchFile()		//Veränderungen an der XML erkennen, ggfs. neu einlesen
	go programFlow()	//Aufrufe in Goroutine starten
	 fmt.Scanln()		//Programm weiterlaufen lassen ohne Ende
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
}//TODO Stoppbefehlliste bereinigen ?! wie bei startprogrammliste auch gemacht...


func programFlow(){
	//Ablauf über HTTPHandler

	//		go processeAufKonsoleAusgeben()
	processeAufKonsoleAusgeben()

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
	go programmKill(1)
	processeAufKonsoleAusgeben()
	go programmKill(2)
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
			
			
			
		//	fmt.Println("Folgende Prozesse laufen gerade: ")
		//	fmt.Println(procSlice)
		//	fmt.Println(procSliceNameAndStopcmd)
		
				//			fmt.Printf("Hier kann jetzt parallel anderer Kram passieren.\n\n")
//TODO: MUTEX AUF GLOBALES ZEUG [ok], TIME WARTENS EINBAUEN, PROCESS EXIT STATE FÜR PROCESSLISTE[atm], stopprocesses in aktuelle prozessliste
}
func processeAufKonsoleAusgeben(){
var procSliceNotExited = make([]string, 0)
mutexProcSlice.Lock()
mutexProcSliceNameAndStopcmd.Lock()
for k:= range procSlice {
	asd := procSlice[k].ProcessState.String()
//	asd:=procState.String()
	if strings.HasPrefix(asd, "exit")==false{
	procSliceNotExited = append(procSliceNotExited, strconv.Itoa(k)+", "+procSliceNameAndStopcmd[k])
    }
	
	//procSlice = procSliceNew
	//	fmt.Print(asd)
//	fmt.Println(procState)
}
//procSlice = nil
//procSlice = procSliceNew[0:]
//copy(procSlice,procSliceNew)
fmt.Println(procSlice)
mutexProcSlice.Unlock()
mutexProcSliceNameAndStopcmd.Unlock()
runtime.Gosched()
fmt.Println(procSliceNotExited) //diesen für die website zur anzeige der laufenden prozesse verwenden
}

/*	for {mutexProcSlice.Lock()
	if len(procSlice) > 0 {
		c := make([]*exec.Cmd, len(procSlice))
   		copy(c, procSlice)			
		for i := 0; i < len(c); i++ {
			if c[i] != nil && c[i].ProcessState.Exited()==false {	
    											fmt.Println(c[i])
												}
									}
						  }
time.Sleep(1 * time.Second)
	mutexProcSlice.Unlock()
	runtime.Gosched()
	}
	}*/
//	//for k.= range procSlice
//	mutexProcSlice.Lock()
//	fmt.Println(procSlice)
//	time.Sleep(1 * time.Second)
//for k:=range procSlice{
//	if procSlice[k].ProcessState.Exited(){
//		fmt.Println("Exited!")
//	}
	
//	}
//	mutexProcSlice.Unlock()
//	runtime.Gosched()
//for {mutexProcSlice.Lock()
//	if len(procSlice) > 0 {
//		c := make([]*exec.Cmd, len(procSlice))
//   		copy(c, procSlice)			
//		for i := 0; i < len(c); i++ {
//			if c[i] != nil && c[i].ProcessState.Exited()==false {	
//    											fmt.Println(c[i])
//												}
//									}
//						  }
//time.Sleep(1 * time.Second)
//	mutexProcSlice.Unlock()
//	runtime.Gosched()
//	}

//}

//func processeAufKonsoleAusgeben(){
//	//mutexProcSlice.Lock()
//	for {
//	if len(procSlice) > 0 {
	
//	c := make([]*exec.Cmd, len(procSlice))
//    copy(c, procSlice)
	
		
	
//for i := 0; i < len(c); i++ {
//	if c[i].ProcessState.Exited()!=true{
		
//    fmt.Println(c[i])
//}
//}
//}
//time.Sleep(1 * time.Second)
//}

//}




/*	fmt.Println(procSlice)
	time.Sleep(1 * time.Second)
for k:=range procSlice{
	if procSlice[k].ProcessState.Exited(){
		fmt.Println("Exited!")
	}
}
	}
	mutexProcSlice.Unlock()
		runtime.Gosched()
	}
	time.Sleep(2 * time.Second)
	}
	}
	*/
//	for{//fmt.Println(überwachteProzesse)
//	fmt.Print("Laufende Prozesse: ")
//	 cmasd := exec.Command("asd")
//	cmasd.Start()
//	fmt.Println(procSlice)
//	time.Sleep(1 * time.Second)
//	/*for k:= range procSlice{*/
//		procSlice = append(procSlice, cmasd)
//		fmt.Println("sad")
//		if procSlice[0].ProcessState.Exited() {
//			fmt.Println("Exited")
//		}
		/*
		if procSlice[k].ProcessState.Exited() == true {
			procSlice[k] = nil*/
	//	procSlice = append(procSlice[:k], procSlice[k+1:]...)
//	}
//	    keys := make([]int, 0, len(überwachteProzesse))
  //  for k := range überwachteProzesse {
    //    keys = append(keys, k)
   // }


	
//	stdout_peter, err_peter := cmd_peter.StdoutPipe()

//signal.Notify(stdin, os.Interrupt)
	
	

//unter Windows funktioniert nur Kill, nicht Interrupt.
//cmd.Process.Signal(os.Interrupt)
//cmd_peter.Process.Signal(os.Interrupt)
//cmd.Process.Signal(os.Kill)
//cmd_peter.Process.Signal(os.Kill)
