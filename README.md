# GoProjekt1
Start-URL: https://localhost:4443

Erläuterungen zu Funktionsweisen, um erwartetes Verhalten zu kennen.

Aktuell wird mit dem Reload-Parameter festgelegt, ob ein Programm >nachdem< es ein erstes Mal gestartet wurde, überwacht und bei Beendigung automatisch >neu< gestartet werden soll.
Beim Start des Observers wird >kein< (also auch nicht Programme, bei denen der Restart-Parameter aktiviert wurde!) Programm automatisch ausgeführt.

Eine Änderung des Restart-Parameters innerhalb der Weboberfläche hat keine Auswirkung auf die in der XML-Datei hinterlegte Konfiguration. Die Änderung in der Weboberfläche wirkt sich nur auf die momentan ausgeführte Umgebung aus.

Durch eine Änderung der XML-Datei kann sich die Reihenfolge der Programme oder auch die Anzahl und der Inhalt der Einträge geändert haben. Deshalb wird zum Zeitpunkt der Ausführung eines Pogramms dessen korrespondierender STOP-Befehl zwischengespeichert. --> Eine Änderung des STOP-Befehls innerhalb der XML-Datei, nachdem ein Programm bereits gestartet wurde, beeinflusst also nicht den für den Prozess hinterlegten STOP-Befehl!

Da die Prozessliste gelegentlich reorganisiert wird, wird beim Zugriff auf Prozesse überprüft, ob die Indizes client- und serverseitig identisch sind (hash über timestamp der Erstellung der Liste). Andernfalls könnte "der falsche" Prozess bearbeitet werden.
Dasselbe gilt für die Programmliste, die nach Änderungen der XML-Datei ebenfalls neu generiert wird.

Der URL-Parameter kann sowohl die Prozess- als auch die Programmnummer beschreiben (kontextabhängig).

Die Anzahl der Starts gilt pro Prozess und bleibt auch nach Deaktivierung (und ggfs. erneuter Aktivierung) der Autostart-Funktion erhalten.
