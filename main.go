package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/EcclesiaTechStudio/bible-cli/internal/model"
	"github.com/EcclesiaTechStudio/bible-cli/internal/shell"
	"github.com/EcclesiaTechStudio/bible-cli/internal/ui"
)

func main() {
	// 1. Load Data
	db, err := model.LoadDatabase()
	if err != nil {
		fmt.Printf("%sCRITICAL: %v%s\n", ui.ColorRed, err, ui.ColorReset)
		return
	}

	// 2. Initialize Engine
	app := shell.New(db)

	// 3. Command Line Args Mode
	if len(os.Args) > 1 {
		fullCommand := strings.Join(os.Args[1:], " ")
		app.RunCommand(fullCommand)
		return
	}

	// 4. Interactive Mode
	scanner := bufio.NewScanner(os.Stdin)
	ui.PrintHeader()

	for {
		pathStr := app.GetPathString()
		fmt.Printf("%s📖 %s%s $ ", ui.ColorBlue, ui.ColorGreen, pathStr)
		fmt.Print(ui.ColorReset)

		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		app.RunCommand(input)
	}
}
