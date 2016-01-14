	scannen := bufio.NewScanner(cmdAusgabe)
	go func() {
		for scannen.Scan() {
			fmt.Printf("Ausgabe | %s\n", scannen.Text())
		}
	}()
	err := cmd.Run()
