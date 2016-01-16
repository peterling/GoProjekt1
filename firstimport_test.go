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

	"github.com/stretchr/testify/assert"
)

func TestError(t *testing.T) {

	//	go helperRoutinesStarter()
	//			if !(runningProcs[0].Handle.Process.Pid <= 65536 && runningProcs[0].Handle.Process.Pid > 0) {
	//			t.Error("Test failed")
	//Test Xml Read
	returnValue := xmlReadIn()
	if returnValue != nil {
		t.Error("Test: XML read failed")
	}

	//Test Programm start
	go programmStart(0, -1) //paint 1. mal
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

	//Test Process Terminate

	go programmStart(2, -1) //cmd 1. mal
	time.Sleep(1 * time.Second)

	programmTerminate(1)
	status = runningProcs[1].Handle.ProcessState.String()
	if status != "<nil>" {
		t.Error("Test: Process terminate failed")
	}

	programmKill(1) //Keine Spuren hinterlassen

	//Test watchFile
	returnValue = watchFile()
	if returnValue != nil {
		t.Error("Test: WatchFile failed")
	}

	//Test Process Stop
	go programmStart(2, -1) //cmd 1. mal
	time.Sleep(2 * time.Second)

	programmStart(2, -2)
	status = runningProcs[1].Handle.ProcessState.String()
	if strings.HasPrefix(status, "exit") != true {
		t.Error("Test: Process stop failed")
	}

	programmKill(2)

	//Test Process Exiz
	go programmStart(2, -1) //cmd 1. mal
	time.Sleep(2 * time.Second)

	programmExit(3)
	status = runningProcs[1].Handle.ProcessState.String()
	if strings.HasPrefix(status, "exit") != true {
		t.Error("Test: Process exit failed")
	}
	programmKill(3)

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
	handler := new(ObserverHandler)
	recorder := httptest.NewRecorder()
	url := fmt.Sprintf("http://example.com/echo?say=%s", expectedBody)
	req, err := http.NewRequest("GET", url, nil)
	assert.Nil(t, err)

	handler.ServeHTTP(recorder, req)

	assert.Equal(t, expectedBody, recorder.Body.String())

	fmt.Println("Test finished")

	//Test Killing Process Hard UNIX
	//Nicht unter Windows anwendbar !!!
	//	fmt.Println(runningProcs[0].Handle.Process.Pid)
	//	returnValue := killingProcessHardUnix(runningProcs[0].Handle.Process.Pid)

	//	if returnValue != true {
	//		t.Error("Test: KillingProcessHard failed")
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
	
		<a href="/proccontrol?program=0&aktion=start&hashprog=db7b60263b672e0cdbaea68ce466cf0c3daaf97e">Paint</a><br><a href="/proccontrol?program=1&aktion=start&hashprog=db7b60263b672e0cdbaea68ce466cf0c3daaf97e">Command Prompt</a><br><a href="/proccontrol?program=2&aktion=start&hashprog=db7b60263b672e0cdbaea68ce466cf0c3daaf97e">PING auf GoogleDNS</a><br><a href="/proccontrol?program=3&aktion=start&hashprog=db7b60263b672e0cdbaea68ce466cf0c3daaf97e">Rechner</a><br><a href="/proccontrol?program=4&aktion=start&hashprog=db7b60263b672e0cdbaea68ce466cf0c3daaf97e">Dauerping auf localhost</a><br>
	<h1>Laufende Prozesse hart beenden (SIGKILL)</h1>	
		<div><strong>keine überwachten Prozesse</strong></div>
	<h1>Laufende Prozesse weich beenden (SIGTERM)</h1>	
		<div><strong>keine überwachten Prozesse</strong></div>
	<h1>Laufende Prozesse mit hinterlegtem Exit-Befehl an STDIN beenden</h1>	
		<div><strong>keine überwachten Prozesse</strong></div>
	<h1>Laufende Prozesse mit hinterlegtem STOP-Befehl beenden</h1>	
		<div><strong>keine überwachten Prozesse</strong></div>
	<h1>Restart-Option laufender Prozesse (de-)aktivieren</h1>
		<div><strong>keine überwachten Prozesse</strong></div>
	<h1>Restart-Option beendeter Prozesse (de-)aktivieren [revive/dismiss]</h1>
		<div><strong>keine überwachten Prozesse</strong></div>
	<h1>Logging</h1>
		<div><strong>keine Programme hinterlegt</strong></div>
	</body>
</html>`

}
