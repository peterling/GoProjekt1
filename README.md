# GoProjekt1
Erläuterungen zu Funktionsweisen, um erwartetes Verhalten zu kennen.

Aktuell wird mit dem Reload-Parameter festgelegt, ob ein Programm >nachdem< es ein erstes Mal gestartet wurde, überwacht und bei Beendigung automatisch >neu< gestartet werden soll.
Beim Start des Observers wird >kein< (also auch nicht Programme, bei denen der Restart-Parameter aktiviert wurde!) Programm automatisch ausgeführt.

Eine Änderung des Restart-Parameters innerhalb der Weboberfläche hat keine Auswirkung auf die in der XML-Datei hinterlegte Konfiguration. Die Änderung in der Weboberfläche wirkt sich nur auf die momentan ausgeführte Umgebung aus.
