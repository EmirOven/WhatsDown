package app

import (
	"log"
	"os/exec"
	"path/filepath"
)

func compileBlueprints() {
	files, _ := filepath.Glob(filepath.Join(uiDir, "*.blp"))
	subFiles, _ := filepath.Glob(filepath.Join(uiDir, "components", "*.blp"))
	allFiles := append(files, subFiles...)
	if len(allFiles) == 0 {
		return
	}
	log.Printf("[live-reload] compiling %d blueprints", len(allFiles))
	for _, f := range allFiles {
		out := f[:len(f)-len(".blp")] + ".ui"
		res, err := exec.Command("blueprint-compiler", "compile", f, "--output", out).CombinedOutput()
		if err != nil {
			log.Printf("[live-reload] blueprint-compiler failed for %s: %v\n%s", f, err, res)
		}
	}
}

func reloadWindow() {
	if err := showPage(currentPage); err != nil {
		log.Printf("[live-reload] UI error, keeping previous window: %v", err)
		return
	}
	log.Printf("[live-reload] window rebuilt from %s", currentPage.uiPath())
}
