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
	//Test Xml Read
	actual := xmlReadIn()
	if actual != nil {
		t.Error("Test: XML read failed")
	}

	//Test Programm start
	go programmStart(0) //paint 1. mal
	time.Sleep(2 * time.Second)
	if len(runningProcs) != 0 {
		if !(runningProcs[0].Handle.Process.Pid <= 65536 && runningProcs[0].Handle.Process.Pid > 0) {
			t.Error("Test failed")
		}
		//	err := programmStart(0)
		//	if err != nil {
		//		t.Error("Test: Programm start failed")
		//	}
		//Test Programm kill
		programmKill(0) //1. paint killen
		time.Sleep(2 * time.Second)
		status := runningProcs[0].Handle.ProcessState.String()
		if strings.HasPrefix(status, "exit") != true {
			t.Error("Test: KillingProcessHard failed")
		}
	}
	//Test Killing Process
	//	programmStop(0)

	//	fmt.Println(string(runningProcs[0].Handle.Process.Pid))
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
