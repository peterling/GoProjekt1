// main_test project main_test.go
package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	//	"github.com/stretchr/testify/assert"
)

func TestError(t *testing.T) {

	//Test Xml Read
	returnError := xmlReadIn()
	if returnError != nil {
		t.Error("Test: XML read failed")
	}

	//Test Programm start
	go programmStart(0, -1) //paint 1. mal

	indexProgramm := 0
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
	programmKill(indexProgramm) //1. paint killen
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

	//Test Process Terminate

	go programmStart(2, -1) //cmd 1. mal
	indexProgramm++
	time.Sleep(1 * time.Second)

	programmTerminate(indexProgramm)
	status = runningProcs[1].Handle.ProcessState.String()
	if status != "<nil>" {
		t.Error("Test: Process terminate failed")
	}

	programmKill(indexProgramm) //Keine Spuren hinterlassen

	//Test watchFile
	returnError = watchFile()
	if returnError != nil {
		t.Error("Test: WatchFile failed")
	}

	//Test Process Stop
	go programmStart(2, -1) //cmd 1. mal
	indexProgramm++
	time.Sleep(2 * time.Second)

	programmStart(2, -2)
	indexProgramm++
	status = runningProcs[1].Handle.ProcessState.String()
	if strings.HasPrefix(status, "exit") != true {
		t.Error("Test: Process stop failed")
	}

	programmKill(indexProgramm)

	//Test Process Exit
	go programmStart(2, -1) //cmd 1. mal
	indexProgramm++
	time.Sleep(2 * time.Second)

	programmExit(indexProgramm)
	status = runningProcs[1].Handle.ProcessState.String()
	if strings.HasPrefix(status, "exit") != true {
		t.Error("Test: Process exit failed")
	}
	programmKill(indexProgramm)

	//Test Programm Index
	go programmStart(0, -1)
	time.Sleep(2 * time.Second)
	indexProgramm++
	fmt.Println(len(runningProcs))
	returnValue := indexProgrammList(indexProgramm)
	fmt.Println(string(returnValue))
	if returnValue != 0 {
		t.Error("Test: Programm index failed")
	}
	programmKill(indexProgramm)

	//Test: Programm Restart
	runningProcs[indexProgramm].Restart = true
	runningProcs[indexProgramm].Alive = false

	checkForRestart()
	time.Sleep(2 * time.Second)
	status = runningProcs[indexProgramm].Handle.ProcessState.String()
	if strings.HasPrefix(status, "exit") == true {
		t.Error("Test: Process restart failed")
	}

	programmKill(indexProgramm)

	//Test Hash Programm List
	h := sha1.New()
	h.Write([]byte(programmListeReorged.String()))
	bs := h.Sum(nil)
	actual := hex.EncodeToString(bs)
	expected := hashOfProgrammListe()

	if actual != expected {
		t.Error("Test: Hash Programm List failed")
	}
	//Test Hash of Process
	h = sha1.New()
	h.Write([]byte(runningProcsReorged.String()))
	bs = h.Sum(nil)
	actual = hex.EncodeToString(bs)
	expected = hashOfRunningProcs()

	if actual != expected {
		t.Error("Test: Hash Programm List failed")
	}
	time.Sleep(2 * time.Second)

	// Server tests
	go webServer()
	time.Sleep(2 * time.Second)
	request, _ := http.NewRequest("POST", "/", nil)
	response := httptest.NewRecorder()
	ObserverHandler(response, request)
	responsebody := response.Body
	stringbody := string(responsebody.Bytes())

	boolreturn := strings.Contains(stringbody, expectedBody)

	if boolreturn != true {
		t.Error("Server Site failed to load")
	}
	request, _ = http.NewRequest("POST", "/proccontrol", nil)
	response = httptest.NewRecorder()
	ProcControl(response, request)
	fmt.Println(response.Code)
	if response.Code != 200 {
		t.Fatalf("Non-expected status code %v:\n\tbody: %v", "200", response.Code)
	}
}

//	returnFile := openLogFile("Paint")
//	returnFile.Close()
//	expectedFile, err := os.OpenFile("./log_Paint.txt", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
//	expectedFile.Close()
//	if (returnFile != expectedFile) && err != nil {
//		t.Error("Test: Open Logfile failed")
//	}

const expectedBody = `<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>Observer</title>
	</head>
	<body>
	<input type="button" value="Seite aktualisieren" onClick="window.location.reload()">
	<input type="button" value="XML-Datei anzeigen" onClick="window.location.href='/download'">
	<a href="/download" download="config">XML-Datei herunterladen</a>
	<h1>Programme starten</h1>
`

//}
