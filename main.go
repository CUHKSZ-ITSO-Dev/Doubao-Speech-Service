package main

import (
	_ "doubao-speech-service/internal/packed"

	"github.com/gogf/gf/v2/os/gctx"

	"doubao-speech-service/internal/cmd"
)

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
