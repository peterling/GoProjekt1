# GoProjekt1
Erläuterungen zu Funktionsweisen, um erwartetes Verhalten zu kennen.

Aktuell wird mit dem Reload-Parameter festgelegt, ob ein Programm >nachdem< es ein erstes Mal gestartet wurde, überwacht und bei Beendigung automatisch >neu< gestartet werden soll.
Beim Start des Observers wird >kein< (also auch nicht Programme, bei denen der Restart-Parameter aktiviert wurde!) Programm automatisch ausgeführt.

Eine Änderung des Restart-Parameters innerhalb der Weboberfläche hat keine Auswirkung auf die in der XML-Datei hinterlegte Konfiguration. Die Änderung in der Weboberfläche wirkt sich nur auf die momentan ausgeführte Umgebung aus.

Durch eine Änderung der XML-Datei kann sich die Reihenfolge der Programme oder auch die Anzahl und der Inhalt der Einträge geändert haben. Deshalb wird zum Zeitpunkt der Ausführung eines Pogramms dessen korrespondierender STOP-Befehl zwischengespeichert.

Da die Prozessliste gelegentlich reorganisiert wird, wird beim Zugriff auf Prozesse überprüft, ob die Indizes client- und serverseitig identisch sind (hash über timestamp der Erstellung der Liste). Andernfalls könnte "der falsche" Prozess bearbeitet werden.
//TODO: Meldung, wenn Hashes sich unterscheiden.
Dasselbe gilt für die Programmliste.

Der URL-Parameter kann sowohl die Prozess- als auch die Programmnummer beschreiben (kontextabhängig).
