package main

import "os/exec"

func openInBrowser(url string) {
	exec.Command("xdg-open", url).Start()
}
