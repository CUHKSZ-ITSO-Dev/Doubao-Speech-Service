package main

import (
	_ "doubao-speech-service/internal/packed"
	_ "github.com/gogf/gf/contrib/drivers/sqlite/v2"

	"github.com/gogf/gf/v2/os/gctx"

	"doubao-speech-service/internal/cmd"
)

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
