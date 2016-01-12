// main_test project main_test.go
package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestError(t *testing.T) {

	//	go helperRoutinesStarter()
	//			if !(runningProcs[0].Handle.Process.Pid <= 65536 && runningProcs[0].Handle.Process.Pid > 0) {
	//			t.Error("Test failed")
	//Test Xml Read
	actual := xmlReadIn()
	if actual != nil {
		t.Error("Test: XML read failed")
	}

	//Test Programm start
	go programmStart(0) //paint 1. mal
	time.Sleep(2 * time.Second)
	//1.Version
	if len(runningProcs) == 0 {
		t.Error("Test: Programm start failed")
	}

	//2. Version
	// Programm muss manuell beendet werden
	//	err := programmStart(0)
	//	time.Sleep(2 * time.Second)
	//	if err != nil {
	//		t.Error("Test: Programm start failed")
	//	}

	//Test Killing Process & Update Process Status
	programmKill(0) //1. paint killen
	time.Sleep(2 * time.Second)

	//Update Process Alive
	updateProcAliveState()
	if runningProcs[0].Alive != false {
		t.Error("Test: update Process Status failed")
	}
	//Killing Process
	status := runningProcs[0].Handle.ProcessState.String()
	if strings.HasPrefix(status, "exit") != true {
		t.Error("Test: KillingProcess failed")
	}

	//Test Process Stop

	go programmStart(2) //paint 1. mal
	time.Sleep(2 * time.Second)

	programmStop(1)
	status = runningProcs[1].Handle.ProcessState.String()
	if strings.HasPrefix(status, "exit") != true {
		t.Error("Test: Process stop failed")
	}

	programmKill(1) //Keine Spuren hinterlassen

	fmt.Println("Test finished")

	//Test Killing Process Hard UNIX
	//Nicht unter Windows anwendbar !!!
	//	fmt.Println(runningProcs[0].Handle.Process.Pid)
	//	returnValue := killingProcessHardUnix(runningProcs[0].Handle.Process.Pid)

	//	if returnValue != true {
	//		t.Error("Test: KillingProcessHard failed")
	//	}

	//	actual = watchFile()
	//	if actual != nil {
	//		t.Error("Test failed")
	//	}

}
