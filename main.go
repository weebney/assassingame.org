package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	app := pocketbase.New()

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/*", apis.StaticDirectoryHandler(os.DirFS("./public"), false))
		return nil
	})

	// Elimination handler --- gets new target when eliminated
	app.OnRecordBeforeUpdateRequest("players").Add(func(e *core.RecordUpdateEvent) error {
		if e.Record.OriginalCopy().GetBool("is_alive") && !e.Record.GetBool("is_alive") {
			selfID := e.Record.GetString("user")
			gameID := e.Record.GetString("game")
			assassin, err := app.Dao().FindFirstRecordByFilter("players", fmt.Sprintf("target.id = '%s' && game.id = '%s'", selfID, gameID), nil)
			if err != nil {
				return err
			}
			targetID := e.Record.GetString("target")
			assassin.Set("target", targetID)
			if err := app.Dao().SaveRecord(assassin); err != nil {
				return err
			}
		}
		return nil
	})

	// TODO: elimintaion confirmation

	// TODO: winner handler

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
